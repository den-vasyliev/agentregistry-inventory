package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

// WebhookHandler handles CI/CD webhooks from monorepo
type WebhookHandler struct {
	client client.Client
	logger zerolog.Logger
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(c client.Client, logger zerolog.Logger) *WebhookHandler {
	return &WebhookHandler{
		client: c,
		logger: logger.With().Str("handler", "webhook").Logger(),
	}
}

// GitHubWebhookPayload represents GitHub webhook payload for push events
type GitHubWebhookPayload struct {
	Ref        string `json:"ref"`
	Repository struct {
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
	} `json:"repository"`
	HeadCommit struct {
		ID       string   `json:"id"`
		Message  string   `json:"message"`
		Added    []string `json:"added"`
		Modified []string `json:"modified"`
		Removed  []string `json:"removed"`
	} `json:"head_commit"`
	Pusher struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"pusher"`
}

// WebhookResponse represents the response from webhook processing
type WebhookResponse struct {
	Success   bool                `json:"success"`
	Message   string              `json:"message"`
	Processed []ProcessedResource `json:"processed,omitempty"`
	Errors    []string            `json:"errors,omitempty"`
}

// ProcessedResource represents a resource that was processed
type ProcessedResource struct {
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Action   string `json:"action"` // created, updated, deleted
	Status   string `json:"status"` // success, error
	ErrorMsg string `json:"error,omitempty"`
}

// HandleGitHubWebhook handles GitHub push webhooks from monorepo
func (h *WebhookHandler) HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse webhook payload
	var payload GitHubWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logger.Error().Err(err).Msg("failed to parse webhook payload")
		h.respondError(w, http.StatusBadRequest, "Invalid webhook payload")
		return
	}

	// Only process pushes to main/master branch
	if !strings.HasSuffix(payload.Ref, "/main") && !strings.HasSuffix(payload.Ref, "/master") {
		h.logger.Debug().Str("ref", payload.Ref).Msg("ignoring push to non-main branch")
		h.respondSuccess(w, "Ignoring push to non-main branch", nil, nil)
		return
	}

	h.logger.Info().
		Str("repo", payload.Repository.FullName).
		Str("commit", func() string {
			if len(payload.HeadCommit.ID) >= 8 {
				return payload.HeadCommit.ID[:8]
			}
			return payload.HeadCommit.ID
		}()).
		Str("pusher", payload.Pusher.Name).
		Msg("processing webhook")

	var processed []ProcessedResource
	var errors []string

	// Process added and modified files
	allFiles := append(payload.HeadCommit.Added, payload.HeadCommit.Modified...)
	for _, filePath := range allFiles {
		if h.isAgentRegistryFile(filePath) {
			result, err := h.processResourceFile(ctx, &payload, filePath, "upsert")
			if err != nil {
				h.logger.Error().Err(err).Str("file", filePath).Msg("failed to process file")
				errors = append(errors, fmt.Sprintf("%s: %v", filePath, err))
				result.Status = "error"
				result.ErrorMsg = err.Error()
			} else {
				result.Status = "success"
			}
			processed = append(processed, result)
		}
	}

	// Process removed files
	for _, filePath := range payload.HeadCommit.Removed {
		if h.isAgentRegistryFile(filePath) {
			result, err := h.processResourceFile(ctx, &payload, filePath, "delete")
			if err != nil {
				h.logger.Error().Err(err).Str("file", filePath).Msg("failed to delete resource")
				errors = append(errors, fmt.Sprintf("%s: %v", filePath, err))
				result.Status = "error"
				result.ErrorMsg = err.Error()
			} else {
				result.Status = "success"
			}
			processed = append(processed, result)
		}
	}

	if len(processed) == 0 {
		h.respondSuccess(w, "No Agent Registry resources to process", nil, nil)
		return
	}

	h.logger.Info().
		Int("processed", len(processed)).
		Int("errors", len(errors)).
		Msg("webhook processing completed")

	h.respondSuccess(w, fmt.Sprintf("Processed %d resources", len(processed)), processed, errors)
}

// isAgentRegistryFile checks if a file path is an Agent Registry resource file
func (h *WebhookHandler) isAgentRegistryFile(filePath string) bool {
	// Check if file is in resources/{kind}/ directory and ends with .yaml or .yml
	if !strings.HasPrefix(filePath, "resources/") {
		return false
	}

	ext := filepath.Ext(filePath)
	if ext != ".yaml" && ext != ".yml" {
		return false
	}

	// Check if it's in a valid kind directory
	parts := strings.Split(filePath, "/")
	if len(parts) < 3 {
		return false
	}

	kind := parts[1]
	validKinds := []string{"mcp-server", "agent", "skill", "model"}
	for _, validKind := range validKinds {
		if kind == validKind {
			return true
		}
	}

	return false
}

// processResourceFile processes a single resource file
func (h *WebhookHandler) processResourceFile(ctx context.Context, payload *GitHubWebhookPayload, filePath, action string) (ProcessedResource, error) {
	result := ProcessedResource{
		Path:   filePath,
		Action: action,
	}

	if action == "delete" {
		// For deletions, parse the file path to get resource info
		parts := strings.Split(filePath, "/")
		if len(parts) < 4 {
			return result, fmt.Errorf("invalid file path format")
		}

		result.Kind = parts[1]
		resourceName := parts[2]
		result.Name = resourceName

		// Extract version from filename if possible
		filename := filepath.Base(filePath)
		name := strings.TrimSuffix(filename, filepath.Ext(filename))
		if strings.HasPrefix(name, resourceName+"-") {
			result.Version = strings.TrimPrefix(name, resourceName+"-")
		}

		return h.deleteResource(ctx, &result)
	}

	// For create/update, fetch and parse the manifest
	manifest, err := h.fetchManifestFromRepo(ctx, payload.Repository.FullName, filePath, payload.HeadCommit.ID)
	if err != nil {
		return result, fmt.Errorf("failed to fetch manifest: %w", err)
	}

	result.Kind = manifest.Kind
	result.Name = manifest.Name
	result.Version = manifest.Version

	// Create or update the resource
	switch manifest.Kind {
	case "mcp-server":
		return h.createOrUpdateMCPServer(ctx, manifest, payload, &result)
	case "agent":
		return h.createOrUpdateAgent(ctx, manifest, payload, &result)
	case "skill":
		return h.createOrUpdateSkill(ctx, manifest, payload, &result)
	case "model":
		return h.createOrUpdateModel(ctx, manifest, payload, &result)
	default:
		return result, fmt.Errorf("unsupported resource kind: %s", manifest.Kind)
	}
}

// fetchManifestFromRepo fetches a manifest file from GitHub repository
func (h *WebhookHandler) fetchManifestFromRepo(ctx context.Context, repoFullName, filePath, commitSHA string) (*AgentRegistryManifest, error) {
	// Use GitHub API to fetch file content
	// For now, using raw content URL - in production, you'd want to use GitHub API with proper auth
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", repoFullName, commitSHA, filePath)

	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch file: HTTP %d", resp.StatusCode)
	}

	var manifest AgentRegistryManifest
	if err := yaml.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// createOrUpdateMCPServer creates or updates an MCP server catalog entry
func (h *WebhookHandler) createOrUpdateMCPServer(ctx context.Context, manifest *AgentRegistryManifest, payload *GitHubWebhookPayload, result *ProcessedResource) (ProcessedResource, error) {
	// Implementation similar to existing submit handler but with webhook-specific metadata
	crName := generateCRName(manifest.Name, manifest.Version)
	
	mcpServer := &agentregistryv1alpha1.MCPServerCatalog{}
	mcpServer.Name = crName
	mcpServer.Namespace = "agentregistry"
	
	// Set labels for webhook source
	mcpServer.Labels = map[string]string{
		"agentregistry.dev/source":    "webhook",
		"agentregistry.dev/repo":      payload.Repository.FullName,
		"agentregistry.dev/commit":    payload.HeadCommit.ID,
		"agentregistry.dev/pusher":    payload.Pusher.Name,
	}

	mcpServer.Spec.Name = manifest.Name
	mcpServer.Spec.Version = manifest.Version
	mcpServer.Spec.Title = manifest.Title
	mcpServer.Spec.Description = manifest.Description

	// Add packages if specified
	if len(manifest.Packages) > 0 {
		for _, pkg := range manifest.Packages {
			mcpServer.Spec.Packages = append(mcpServer.Spec.Packages, agentregistryv1alpha1.Package{
				RegistryType: pkg.Type,
				Identifier:   pkg.Identifier,
				Transport: agentregistryv1alpha1.Transport{
					Type: pkg.Transport,
				},
			})
		}
	}

	// Add repository information
	mcpServer.Spec.Repository = &agentregistryv1alpha1.Repository{
		URL:    payload.Repository.HTMLURL,
		Source: "github",
	}

	// Create or update the resource
	existing := &agentregistryv1alpha1.MCPServerCatalog{}
	err := h.client.Get(ctx, client.ObjectKey{
		Name:      crName,
		Namespace: "agentregistry",
	}, existing)

	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return *result, err
		}
		// Create new resource
		if err := h.client.Create(ctx, mcpServer); err != nil {
			return *result, err
		}
		result.Action = "created"
	} else {
		// Update existing resource
		existing.Spec = mcpServer.Spec
		existing.Labels = mcpServer.Labels
		if err := h.client.Update(ctx, existing); err != nil {
			return *result, err
		}
		result.Action = "updated"
	}

	return *result, nil
}

// createOrUpdateAgent creates or updates an agent catalog entry
func (h *WebhookHandler) createOrUpdateAgent(ctx context.Context, manifest *AgentRegistryManifest, payload *GitHubWebhookPayload, result *ProcessedResource) (ProcessedResource, error) {
	// Similar to MCP server but for Agent catalog
	// Implementation would follow similar pattern...
	return *result, fmt.Errorf("agent processing not yet implemented")
}

// createOrUpdateSkill creates or updates a skill catalog entry  
func (h *WebhookHandler) createOrUpdateSkill(ctx context.Context, manifest *AgentRegistryManifest, payload *GitHubWebhookPayload, result *ProcessedResource) (ProcessedResource, error) {
	// Similar implementation for skills...
	return *result, fmt.Errorf("skill processing not yet implemented")
}

// createOrUpdateModel creates or updates a model catalog entry
func (h *WebhookHandler) createOrUpdateModel(ctx context.Context, manifest *AgentRegistryManifest, payload *GitHubWebhookPayload, result *ProcessedResource) (ProcessedResource, error) {
	// Similar implementation for models...
	return *result, fmt.Errorf("model processing not yet implemented")
}

// deleteResource deletes a resource based on the result info
func (h *WebhookHandler) deleteResource(ctx context.Context, result *ProcessedResource) (ProcessedResource, error) {
	if result.Version == "" {
		return *result, fmt.Errorf("cannot delete resource without version information")
	}

	crName := generateCRName(result.Name, result.Version)

	switch result.Kind {
	case "mcp-server":
		mcpServer := &agentregistryv1alpha1.MCPServerCatalog{}
		mcpServer.Name = crName
		mcpServer.Namespace = "agentregistry"
		if err := h.client.Delete(ctx, mcpServer); err != nil {
			return *result, client.IgnoreNotFound(err)
		}
	case "agent":
		agent := &agentregistryv1alpha1.AgentCatalog{}
		agent.Name = crName
		agent.Namespace = "agentregistry"
		if err := h.client.Delete(ctx, agent); err != nil {
			return *result, client.IgnoreNotFound(err)
		}
	case "skill":
		skill := &agentregistryv1alpha1.SkillCatalog{}
		skill.Name = crName
		skill.Namespace = "agentregistry"
		if err := h.client.Delete(ctx, skill); err != nil {
			return *result, client.IgnoreNotFound(err)
		}
	case "model":
		model := &agentregistryv1alpha1.ModelCatalog{}
		model.Name = crName
		model.Namespace = "agentregistry"
		if err := h.client.Delete(ctx, model); err != nil {
			return *result, client.IgnoreNotFound(err)
		}
	default:
		return *result, fmt.Errorf("unsupported resource kind for deletion: %s", result.Kind)
	}

	return *result, nil
}

// generateCRName generates a consistent CR name from resource name and version
func generateCRName(name, version string) string {
	sanitizedName := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	sanitizedVersion := strings.ReplaceAll(version, ".", "-")
	return fmt.Sprintf("%s-%s", sanitizedName, sanitizedVersion)
}

// respondSuccess sends a successful response
func (h *WebhookHandler) respondSuccess(w http.ResponseWriter, message string, processed []ProcessedResource, errors []string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(WebhookResponse{
		Success:   true,
		Message:   message,
		Processed: processed,
		Errors:    errors,
	})
}

// respondError sends an error response
func (h *WebhookHandler) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(WebhookResponse{
		Success: false,
		Message: message,
	})
}