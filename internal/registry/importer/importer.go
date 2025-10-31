package importer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/agentregistry-dev/agentregistry/internal/registry/database"
	"github.com/agentregistry-dev/agentregistry/internal/registry/service"
	"github.com/agentregistry-dev/agentregistry/internal/registry/validators"
	apiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
)

// Service handles importing seed data into the registry
type Service struct {
	registry       service.RegistryService
	httpClient     *http.Client
	requestHeaders map[string]string
	updateIfExists bool
	githubToken    string
}

// NewService creates a new importer service with sane defaults
func NewService(registry service.RegistryService) *Service {
	return &Service{
		registry:       registry,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		requestHeaders: map[string]string{},
	}
}

// (Deprecated) NewServiceWithHTTP was removed; use NewService() and setters instead.

// SetRequestHeaders replaces headers used for HTTP fetches
func (s *Service) SetRequestHeaders(headers map[string]string) {
	s.requestHeaders = headers
}

// SetHTTPClient overrides the HTTP client used for fetches
func (s *Service) SetHTTPClient(client *http.Client) {
	if client != nil {
		s.httpClient = client
	}
}

// SetUpdateIfExists toggles replacing existing name/version entries instead of skipping
func (s *Service) SetUpdateIfExists(update bool) {
	s.updateIfExists = update
}

// SetGitHubToken sets a token used only for GitHub enrichment calls
func (s *Service) SetGitHubToken(token string) {
	s.githubToken = strings.TrimSpace(token)
}

// ImportFromPath imports seed data from various sources:
// 1. Local file paths (*.json files) - expects ServerJSON array format
// 2. Direct HTTP URLs to seed.json files - expects ServerJSON array format
// 3. Registry root URLs (automatically appends /v0/servers and paginates)
func (s *Service) ImportFromPath(ctx context.Context, path string) error {
	servers, err := s.readSeedFile(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to read seed data: %w", err)
	}

	// Import each server using registry service CreateServer
	var successfullyCreated []string
	var failedCreations []string
	total := len(servers)
	processed := 0

	for _, server := range servers {
		processed++
		log.Printf("Importing %d/%d: %s@%s", processed, total, server.Name, server.Version)

		// Best-effort enrichment
		if err := s.enrichServer(ctx, server); err != nil {
			log.Printf("Warning: enrichment failed for %s@%s: %v", server.Name, server.Version, err)
		}

		_, err := s.registry.CreateServer(ctx, server)
		if err != nil {
			// If duplicate version and update is enabled, try update path
			if s.updateIfExists && errors.Is(err, database.ErrInvalidVersion) {
				if _, uerr := s.registry.UpdateServer(ctx, server.Name, server.Version, server, nil); uerr != nil {
					failedCreations = append(failedCreations, fmt.Sprintf("%s: %v", server.Name, uerr))
					log.Printf("Failed to update existing server %s: %v", server.Name, uerr)
				} else {
					successfullyCreated = append(successfullyCreated, server.Name)
					continue
				}
			} else {
				failedCreations = append(failedCreations, fmt.Sprintf("%s: %v", server.Name, err))
				log.Printf("Failed to create server %s: %v", server.Name, err)
			}
		} else {
			successfullyCreated = append(successfullyCreated, server.Name)
		}
	}

	// Report import results after actual creation attempts
	if len(failedCreations) > 0 {
		log.Printf("Import completed with errors: %d servers created successfully, %d servers failed",
			len(successfullyCreated), len(failedCreations))
		log.Printf("Failed servers: %v", failedCreations)
		return fmt.Errorf("failed to import %d servers", len(failedCreations))
	}

	log.Printf("Import completed successfully: all %d servers created", len(successfullyCreated))
	return nil
}

// readSeedFile reads seed data from various sources
func (s *Service) readSeedFile(ctx context.Context, path string) ([]*apiv0.ServerJSON, error) {
	var data []byte
	var err error

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		// Handle HTTP URLs
		if strings.HasSuffix(path, "/v0/servers") || strings.Contains(path, "/v0/servers") {
			// This is a registry API endpoint - fetch paginated data
			return s.fetchFromRegistryAPI(ctx, path)
		}
		// This is a direct file URL
		data, err = s.fetchFromHTTP(ctx, path)
	} else {
		// Handle local file paths
		data, err = os.ReadFile(path)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read seed data from %s: %w", path, err)
	}

	// Parse ServerJSON array format
	var serverResponses []apiv0.ServerJSON
	if err := json.Unmarshal(data, &serverResponses); err != nil {
		return nil, fmt.Errorf("failed to parse seed data as ServerJSON array format: %w", err)
	}

	if len(serverResponses) == 0 {
		return []*apiv0.ServerJSON{}, nil
	}

	// Validate servers and collect warnings instead of failing the whole batch
	var validRecords []*apiv0.ServerJSON
	var invalidServers []string
	var validationFailures []string

	for _, response := range serverResponses {
		if err := validators.ValidateServerJSON(&response); err != nil {
			// Log warning and track invalid server instead of failing
			invalidServers = append(invalidServers, response.Name)
			validationFailures = append(validationFailures, fmt.Sprintf("Server '%s': %v", response.Name, err))
			log.Printf("Warning: Skipping invalid server '%s': %v", response.Name, err)
			continue
		}

		// Add valid ServerJSON to records
		validRecords = append(validRecords, &response)
	}

	// Print summary of validation results
	if len(invalidServers) > 0 {
		log.Printf("Validation summary: %d servers passed validation, %d invalid servers skipped", len(validRecords), len(invalidServers))
		log.Printf("Invalid servers: %v", invalidServers)
		for _, failure := range validationFailures {
			log.Printf("  - %s", failure)
		}
	} else {
		log.Printf("Validation summary: All %d servers passed validation", len(validRecords))
	}

	return validRecords, nil
}

func (s *Service) fetchFromHTTP(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	// apply custom headers if provided
	for k, v := range s.requestHeaders {
		req.Header.Set(k, v)
	}

	client := s.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (s *Service) fetchFromRegistryAPI(ctx context.Context, baseURL string) ([]*apiv0.ServerJSON, error) {
	var allRecords []*apiv0.ServerJSON
	cursor := ""

	for {
		url := baseURL
		if cursor != "" {
			if strings.Contains(url, "?") {
				url += "&cursor=" + cursor
			} else {
				url += "?cursor=" + cursor
			}
		}

		data, err := s.fetchFromHTTP(ctx, url)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch page from registry API: %w", err)
		}

		var response struct {
			Servers  []apiv0.ServerResponse `json:"servers"`
			Metadata *struct {
				NextCursor string `json:"nextCursor,omitempty"`
			} `json:"metadata,omitempty"`
		}

		if err := json.Unmarshal(data, &response); err != nil {
			return nil, fmt.Errorf("failed to parse registry API response: %w", err)
		}

		// Extract ServerJSON from each ServerResponse
		for _, serverResponse := range response.Servers {
			allRecords = append(allRecords, &serverResponse.Server)
		}

		// Check if there's a next page
		if response.Metadata == nil || response.Metadata.NextCursor == "" {
			break
		}
		cursor = response.Metadata.NextCursor
	}

	return allRecords, nil
}

// enrichServer augments ServerJSON with vendor metadata under _meta.publisher-provided
// Key: agentregistry.solo.io/metadata { stars: <int> }
func (s *Service) enrichServer(ctx context.Context, server *apiv0.ServerJSON) error {
	if server == nil || server.Repository == nil || server.Repository.URL == "" {
		return nil
	}
	owner, repo := parseGitHubRepo(server.Repository.URL)
	if owner == "" || repo == "" {
		return nil
	}

	stars, err := s.fetchGitHubStars(ctx, owner, repo)
	if err != nil {
		return err
	}

	if server.Meta == nil {
		server.Meta = &apiv0.ServerMeta{}
	}
	if server.Meta.PublisherProvided == nil {
		server.Meta.PublisherProvided = map[string]interface{}{}
	}

	server.Meta.PublisherProvided["agentregistry.solo.io/metadata"] = map[string]interface{}{
		"stars": stars,
	}
	return nil
}

// parseGitHubRepo extracts owner/repo from common GitHub URL formats
func parseGitHubRepo(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, ".git")
	if strings.Contains(raw, "github.com/") {
		parts := strings.Split(raw, "github.com/")
		path := parts[len(parts)-1]
		segs := strings.Split(strings.Trim(path, "/"), "/")
		if len(segs) >= 2 {
			return segs[0], segs[1]
		}
	}
	sshRe := regexp.MustCompile(`github\.com:([^/]+)/([^/]+)$`)
	m := sshRe.FindStringSubmatch(raw)
	if len(m) == 3 {
		return m[1], m[2]
	}
	return "", ""
}

// fetchGitHubStars queries the GitHub repo API for stargazers_count
func (s *Service) fetchGitHubStars(ctx context.Context, owner, repo string) (int, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	// Do NOT forward arbitrary registry headers to GitHub.
	// Only apply an explicit GitHub token if provided.
	if s.githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.githubToken)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/vnd.github+json")
	}
	client := s.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("github api status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Stars int `json:"stargazers_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}
	return payload.Stars, nil
}
