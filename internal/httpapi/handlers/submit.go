package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SubmitHandler handles resource submission from external repositories
type SubmitHandler struct {
	client client.Client
	logger zerolog.Logger
}

// NewSubmitHandler creates a new submit handler
func NewSubmitHandler(c client.Client, logger zerolog.Logger) *SubmitHandler {
	return &SubmitHandler{
		client: c,
		logger: logger.With().Str("handler", "submit").Logger(),
	}
}

// SubmitRequest is the request body for submitting a resource
type SubmitRequest struct {
	RepositoryURL string `json:"repositoryUrl"`
}

// SubmitResponse is the response for a submission
type SubmitResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Name    string `json:"name,omitempty"`
	Kind    string `json:"kind,omitempty"`
	Version string `json:"version,omitempty"`
	Status  string `json:"status,omitempty"`
}

// AgentRegistryManifest represents the .agentregistry.yaml file structure
type AgentRegistryManifest struct {
	Kind        string `yaml:"kind"` // mcp-server, agent, skill
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Title       string `yaml:"title,omitempty"`
	Description string `yaml:"description,omitempty"`

	// Package info
	Packages []ManifestPackage `yaml:"packages,omitempty"`

	// Remote endpoints
	Remotes []ManifestRemote `yaml:"remotes,omitempty"`

	// Agent-specific
	Agent *ManifestAgent `yaml:"agent,omitempty"`

	// Skill-specific
	Skill *ManifestSkill `yaml:"skill,omitempty"`

	// Config requirements
	Config *ManifestConfig `yaml:"config,omitempty"`
}

type ManifestPackage struct {
	Type       string `yaml:"type"` // oci, npm, pypi
	Image      string `yaml:"image,omitempty"`
	Identifier string `yaml:"identifier,omitempty"`
	Transport  string `yaml:"transport,omitempty"`
}

type ManifestRemote struct {
	URL  string `yaml:"url"`
	Type string `yaml:"type,omitempty"`
}

type ManifestAgent struct {
	Framework     string `yaml:"framework,omitempty"`
	Language      string `yaml:"language,omitempty"`
	ModelProvider string `yaml:"modelProvider,omitempty"`
	ModelName     string `yaml:"modelName,omitempty"`
}

type ManifestSkill struct {
	Category string `yaml:"category,omitempty"`
}

type ManifestConfig struct {
	Required []ManifestConfigVar `yaml:"required,omitempty"`
	Optional []ManifestConfigVar `yaml:"optional,omitempty"`
}

type ManifestConfigVar struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Default     string `yaml:"default,omitempty"`
}

// Submit handles POST /admin/v0/submit
func (h *SubmitHandler) Submit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.RepositoryURL == "" {
		h.respondError(w, http.StatusBadRequest, "Repository URL is required")
		return
	}

	h.logger.Info().Str("repositoryUrl", req.RepositoryURL).Msg("processing submission")

	// Parse and validate repository URL
	repoInfo, err := parseRepositoryURL(req.RepositoryURL)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid repository URL: %v", err))
		return
	}

	// Fetch .agentregistry.yaml from repository
	manifest, err := h.fetchManifest(ctx, repoInfo)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("Failed to fetch manifest: %v", err))
		return
	}

	// Validate manifest
	if err := validateManifest(manifest); err != nil {
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid manifest: %v", err))
		return
	}

	switch manifest.Kind {
	case "mcp-server", "agent", "skill":
		// supported
	default:
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("Unsupported resource kind: %s", manifest.Kind))
		return
	}

	// Submission is a PROPOSE flow: it fetches and validates the manifest from a
	// public repository but does NOT write anything to the cluster. Publishing to
	// the catalog is a privileged operation handled by the admin push endpoints.
	//
	// TODO: open a pull request against the registry repo with the validated
	// manifest instead of returning it inline.
	h.logger.Info().
		Str("kind", manifest.Kind).
		Str("name", manifest.Name).
		Str("version", manifest.Version).
		Msg("submission validated (no cluster write)")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SubmitResponse{
		Success: true,
		Message: "Manifest validated. Submission does not modify the cluster; an admin must publish it (or a PR will be opened once integration is available).",
		Name:    manifest.Name,
		Kind:    manifest.Kind,
		Version: manifest.Version,
		Status:  "validated",
	})
}

type repoInfo struct {
	Host   string // github.com, gitlab.com
	Owner  string
	Repo   string
	Branch string
	URL    string
}

func parseRepositoryURL(rawURL string) (*repoInfo, error) {
	// Handle different URL formats
	// https://github.com/owner/repo
	// https://github.com/owner/repo.git
	// git@github.com:owner/repo.git

	rawURL = strings.TrimSuffix(rawURL, ".git")
	rawURL = strings.TrimSuffix(rawURL, "/")

	// Convert SSH format to HTTPS
	if strings.HasPrefix(rawURL, "git@") {
		rawURL = strings.Replace(rawURL, ":", "/", 1)
		rawURL = strings.Replace(rawURL, "git@", "https://", 1)
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("URL must include owner and repository")
	}

	return &repoInfo{
		Host:   u.Host,
		Owner:  parts[0],
		Repo:   parts[1],
		Branch: "main", // Default to main, could be made configurable
		URL:    rawURL,
	}, nil
}

func (h *SubmitHandler) fetchManifest(ctx context.Context, repo *repoInfo) (*AgentRegistryManifest, error) {
	// Construct raw content URL based on host
	var rawURL string
	switch repo.Host {
	case "github.com":
		rawURL = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/.agentregistry.yaml", repo.Owner, repo.Repo, repo.Branch)
	case "gitlab.com":
		rawURL = fmt.Sprintf("https://gitlab.com/%s/%s/-/raw/%s/.agentregistry.yaml", repo.Owner, repo.Repo, repo.Branch)
	default:
		return nil, fmt.Errorf("unsupported repository host: %s", repo.Host)
	}

	h.logger.Debug().Str("rawURL", rawURL).Msg("fetching manifest")

	// Create HTTP request with timeout
	httpCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, "GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf(".agentregistry.yaml not found in repository root (branch: %s)", repo.Branch)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch manifest: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest AgentRegistryManifest
	if err := yaml.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest YAML: %w", err)
	}

	return &manifest, nil
}

func validateManifest(m *AgentRegistryManifest) error {
	if m.Kind == "" {
		return fmt.Errorf("kind is required")
	}
	if m.Kind != "mcp-server" && m.Kind != "agent" && m.Kind != "skill" {
		return fmt.Errorf("kind must be one of: mcp-server, agent, skill")
	}
	if m.Name == "" {
		return fmt.Errorf("name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("version is required")
	}
	return nil
}

func sanitizeCRName(name string) string {
	// Convert to lowercase and replace invalid characters
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")

	// Truncate to 63 characters (K8s limit)
	if len(name) > 63 {
		name = name[:63]
	}

	// Remove trailing hyphens
	name = strings.TrimSuffix(name, "-")

	return name
}

func (h *SubmitHandler) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"message": message,
	})
}
