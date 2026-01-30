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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
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

	// Create catalog entry based on kind
	var resourceName string
	switch manifest.Kind {
	case "mcp-server":
		resourceName, err = h.createMCPServerCatalog(ctx, manifest, repoInfo)
	case "agent":
		resourceName, err = h.createAgentCatalog(ctx, manifest, repoInfo)
	case "skill":
		resourceName, err = h.createSkillCatalog(ctx, manifest, repoInfo)
	default:
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("Unsupported resource kind: %s", manifest.Kind))
		return
	}

	if err != nil {
		h.logger.Error().Err(err).Msg("failed to create catalog entry")
		h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create catalog entry: %v", err))
		return
	}

	h.logger.Info().
		Str("kind", manifest.Kind).
		Str("name", manifest.Name).
		Str("version", manifest.Version).
		Str("resourceName", resourceName).
		Msg("submission created successfully")

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(SubmitResponse{
		Success: true,
		Message: "Resource submitted for review",
		Name:    manifest.Name,
		Kind:    manifest.Kind,
		Version: manifest.Version,
		Status:  "pending_review",
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

func (h *SubmitHandler) createMCPServerCatalog(ctx context.Context, m *AgentRegistryManifest, repo *repoInfo) (string, error) {
	// Generate a sanitized name for the CR
	crName := sanitizeCRName(fmt.Sprintf("submit-%s-%s-%s", repo.Owner, m.Name, m.Version))

	// Convert manifest packages to catalog packages
	var packages []agentregistryv1alpha1.Package
	for _, p := range m.Packages {
		pkg := agentregistryv1alpha1.Package{
			RegistryType: p.Type,
			Transport: agentregistryv1alpha1.Transport{
				Type: p.Transport,
			},
		}
		if p.Image != "" {
			pkg.Identifier = p.Image
		} else {
			pkg.Identifier = p.Identifier
		}
		packages = append(packages, pkg)
	}

	// Convert remotes
	var remotes []agentregistryv1alpha1.Transport
	for _, r := range m.Remotes {
		remotes = append(remotes, agentregistryv1alpha1.Transport{
			Type: r.Type,
			URL:  r.URL,
		})
	}

	catalog := &agentregistryv1alpha1.MCPServerCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: crName,
			Labels: map[string]string{
				"agentregistry.dev/submitted":     "true",
				"agentregistry.dev/review-status": "pending",
			},
			Annotations: map[string]string{
				"agentregistry.dev/submitted-at":   time.Now().UTC().Format(time.RFC3339),
				"agentregistry.dev/repository-url": repo.URL,
			},
		},
		Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
			Name:        m.Name,
			Version:     m.Version,
			Title:       m.Title,
			Description: m.Description,
			Repository: &agentregistryv1alpha1.Repository{
				URL:    repo.URL,
				Source: repo.Host,
			},
			Packages: packages,
			Remotes:  remotes,
		},
	}

	if err := h.client.Create(ctx, catalog); err != nil {
		return "", err
	}

	// Set status to pending review (not published)
	catalog.Status.Published = false
	catalog.Status.Status = "pending_review"
	if err := h.client.Status().Update(ctx, catalog); err != nil {
		return "", err
	}

	return crName, nil
}

func (h *SubmitHandler) createAgentCatalog(ctx context.Context, m *AgentRegistryManifest, repo *repoInfo) (string, error) {
	crName := sanitizeCRName(fmt.Sprintf("submit-%s-%s-%s", repo.Owner, m.Name, m.Version))

	// Extract image from packages
	var image string
	for _, p := range m.Packages {
		if p.Type == "oci" && p.Image != "" {
			image = p.Image
			break
		}
	}

	catalog := &agentregistryv1alpha1.AgentCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: crName,
			Labels: map[string]string{
				"agentregistry.dev/submitted":     "true",
				"agentregistry.dev/review-status": "pending",
			},
			Annotations: map[string]string{
				"agentregistry.dev/submitted-at":   time.Now().UTC().Format(time.RFC3339),
				"agentregistry.dev/repository-url": repo.URL,
			},
		},
		Spec: agentregistryv1alpha1.AgentCatalogSpec{
			Name:        m.Name,
			Version:     m.Version,
			Title:       m.Title,
			Description: m.Description,
			Image:       image,
			Repository: &agentregistryv1alpha1.Repository{
				URL:    repo.URL,
				Source: repo.Host,
			},
		},
	}

	// Add agent-specific fields
	if m.Agent != nil {
		catalog.Spec.Framework = m.Agent.Framework
		catalog.Spec.Language = m.Agent.Language
		catalog.Spec.ModelProvider = m.Agent.ModelProvider
		catalog.Spec.ModelName = m.Agent.ModelName
	}

	if err := h.client.Create(ctx, catalog); err != nil {
		return "", err
	}

	// Set status to pending review (not published)
	catalog.Status.Published = false
	catalog.Status.Status = "pending_review"
	if err := h.client.Status().Update(ctx, catalog); err != nil {
		return "", err
	}

	return crName, nil
}

func (h *SubmitHandler) createSkillCatalog(ctx context.Context, m *AgentRegistryManifest, repo *repoInfo) (string, error) {
	crName := sanitizeCRName(fmt.Sprintf("submit-%s-%s-%s", repo.Owner, m.Name, m.Version))

	// Convert packages
	var packages []agentregistryv1alpha1.SkillPackage
	for _, p := range m.Packages {
		pkg := agentregistryv1alpha1.SkillPackage{
			RegistryType: p.Type,
		}
		if p.Image != "" {
			pkg.Identifier = p.Image
		} else {
			pkg.Identifier = p.Identifier
		}
		packages = append(packages, pkg)
	}

	catalog := &agentregistryv1alpha1.SkillCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: crName,
			Labels: map[string]string{
				"agentregistry.dev/submitted":     "true",
				"agentregistry.dev/review-status": "pending",
			},
			Annotations: map[string]string{
				"agentregistry.dev/submitted-at":   time.Now().UTC().Format(time.RFC3339),
				"agentregistry.dev/repository-url": repo.URL,
			},
		},
		Spec: agentregistryv1alpha1.SkillCatalogSpec{
			Name:        m.Name,
			Version:     m.Version,
			Title:       m.Title,
			Description: m.Description,
			Repository: &agentregistryv1alpha1.SkillRepository{
				URL:    repo.URL,
				Source: repo.Host,
			},
			Packages: packages,
		},
	}

	// Add skill-specific fields
	if m.Skill != nil {
		catalog.Spec.Category = m.Skill.Category
	}

	if err := h.client.Create(ctx, catalog); err != nil {
		return "", err
	}

	// Set status to pending review (not published)
	catalog.Status.Published = false
	catalog.Status.Status = "pending_review"
	if err := h.client.Status().Update(ctx, catalog); err != nil {
		return "", err
	}

	return crName, nil
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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"message": message,
	})
}
