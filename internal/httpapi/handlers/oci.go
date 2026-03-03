package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

// OCIHandler handles OCI artifact creation and management
type OCIHandler struct {
	client       client.Client
	logger       zerolog.Logger
	registryURL  string
	registryAuth string
}

// NewOCIHandler creates a new OCI handler
func NewOCIHandler(c client.Client, logger zerolog.Logger, registryURL, registryAuth string) *OCIHandler {
	return &OCIHandler{
		client:       c,
		logger:       logger.With().Str("handler", "oci").Logger(),
		registryURL:  registryURL,
		registryAuth: registryAuth,
	}
}

// OCIArtifactRequest represents a request to create an OCI artifact
type OCIArtifactRequest struct {
	ResourceName    string            `json:"resourceName"`
	Version         string            `json:"version"`
	ResourceType    string            `json:"resourceType"` // mcp-server, agent, skill, model
	Environment     string            `json:"environment,omitempty"`
	Config          map[string]string `json:"config,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
	RepositoryURL   string            `json:"repositoryUrl,omitempty"`
	CommitSHA       string            `json:"commitSha,omitempty"`
	Pusher          string            `json:"pusher,omitempty"`
}

// OCIArtifactResponse represents the response from OCI artifact creation
type OCIArtifactResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	ArtifactRef string `json:"artifactRef,omitempty"`
	Digest      string `json:"digest,omitempty"`
	Size        int64  `json:"size,omitempty"`
}

// OCIArtifactConfig represents the configuration stored in the OCI artifact
type OCIArtifactConfig struct {
	// Standard OCI config fields
	Architecture string            `json:"architecture"`
	OS           string            `json:"os"`
	Config       OCIContainerConfig `json:"config"`

	// Agent Registry specific fields
	ResourceName string            `json:"resourceName"`
	Version      string            `json:"version"`
	ResourceType string            `json:"resourceType"`
	Environment  string            `json:"environment,omitempty"`
	Catalog      interface{}       `json:"catalog"` // The actual catalog resource
	Deployment   interface{}       `json:"deployment,omitempty"` // Optional deployment config
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// OCIContainerConfig represents the container config in OCI image
type OCIContainerConfig struct {
	Env         []string          `json:"env,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	WorkingDir  string            `json:"workingdir,omitempty"`
	Entrypoint  []string          `json:"entrypoint,omitempty"`
	Cmd         []string          `json:"cmd,omitempty"`
}

// CreateOCIArtifact creates an OCI artifact containing the registry resource config
func (h *OCIHandler) CreateOCIArtifact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req OCIArtifactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.ResourceName == "" || req.Version == "" || req.ResourceType == "" {
		h.respondError(w, http.StatusBadRequest, "resourceName, version, and resourceType are required")
		return
	}

	h.logger.Info().
		Str("resourceName", req.ResourceName).
		Str("version", req.Version).
		Str("resourceType", req.ResourceType).
		Msg("creating OCI artifact")

	// Fetch the catalog resource
	catalog, err := h.fetchCatalogResource(ctx, req.ResourceName, req.Version, req.ResourceType)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to fetch catalog resource")
		h.respondError(w, http.StatusNotFound, fmt.Sprintf("Failed to fetch catalog resource: %v", err))
		return
	}

	// Create OCI artifact config
	artifactConfig := &OCIArtifactConfig{
		Architecture: "any",
		OS:           "any",
		Config: OCIContainerConfig{
			Labels: map[string]string{
				"agentregistry.dev/resource-name":    req.ResourceName,
				"agentregistry.dev/resource-version": req.Version,
				"agentregistry.dev/resource-type":    req.ResourceType,
				"agentregistry.dev/environment":      req.Environment,
			},
		},
		ResourceName: req.ResourceName,
		Version:      req.Version,
		ResourceType: req.ResourceType,
		Environment:  req.Environment,
		Catalog:      catalog,
		Metadata:     req.Annotations,
	}

	// Add repository metadata if provided
	if req.RepositoryURL != "" {
		artifactConfig.Config.Labels["agentregistry.dev/repository"] = req.RepositoryURL
		if req.CommitSHA != "" {
			artifactConfig.Config.Labels["agentregistry.dev/commit"] = req.CommitSHA
		}
		if req.Pusher != "" {
			artifactConfig.Config.Labels["agentregistry.dev/pusher"] = req.Pusher
		}
	}

	// Create deployment config if provided
	if len(req.Config) > 0 {
		deploymentConfig := map[string]interface{}{
			"config": req.Config,
		}
		artifactConfig.Deployment = deploymentConfig
	}

	// Push to OCI registry
	artifactRef, digest, size, err := h.pushOCIArtifact(ctx, artifactConfig)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to push OCI artifact")
		h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to push OCI artifact: %v", err))
		return
	}

	h.logger.Info().
		Str("artifactRef", artifactRef).
		Str("digest", digest).
		Int64("size", size).
		Msg("OCI artifact created successfully")

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(OCIArtifactResponse{
		Success:     true,
		Message:     "OCI artifact created successfully",
		ArtifactRef: artifactRef,
		Digest:      digest,
		Size:        size,
	})
}

// fetchCatalogResource fetches the catalog resource by name, version, and type
func (h *OCIHandler) fetchCatalogResource(ctx context.Context, resourceName, version, resourceType string) (interface{}, error) {
	crName := generateCRName(resourceName, version)

	switch resourceType {
	case "mcp-server":
		mcpServer := &agentregistryv1alpha1.MCPServerCatalog{}
		if err := h.client.Get(ctx, client.ObjectKey{
			Name:      crName,
			Namespace: "agentregistry",
		}, mcpServer); err != nil {
			return nil, err
		}
		return mcpServer, nil

	case "agent":
		agent := &agentregistryv1alpha1.AgentCatalog{}
		if err := h.client.Get(ctx, client.ObjectKey{
			Name:      crName,
			Namespace: "agentregistry",
		}, agent); err != nil {
			return nil, err
		}
		return agent, nil

	case "skill":
		skill := &agentregistryv1alpha1.SkillCatalog{}
		if err := h.client.Get(ctx, client.ObjectKey{
			Name:      crName,
			Namespace: "agentregistry",
		}, skill); err != nil {
			return nil, err
		}
		return skill, nil

	case "model":
		model := &agentregistryv1alpha1.ModelCatalog{}
		if err := h.client.Get(ctx, client.ObjectKey{
			Name:      crName,
			Namespace: "agentregistry",
		}, model); err != nil {
			return nil, err
		}
		return model, nil

	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// pushOCIArtifact pushes the artifact config to the OCI registry
func (h *OCIHandler) pushOCIArtifact(ctx context.Context, artifactConfig *OCIArtifactConfig) (string, string, int64, error) {
	// This is a placeholder implementation
	// In a real implementation, you would:
	// 1. Create an OCI image with the config as a layer
	// 2. Push to the configured OCI registry
	// 3. Return the artifact reference, digest, and size
	
	// For now, we'll simulate the response
	artifactRef := fmt.Sprintf("%s/%s:%s", 
		h.registryURL, 
		artifactConfig.ResourceName, 
		artifactConfig.Version)
	
	// Generate a mock digest
	digest := "sha256:1234567890abcdef1234567890abcdef12345678901234567890abcdef123456"
	
	// Mock size
	var size int64 = 1024
	
	h.logger.Info().
		Str("artifactRef", artifactRef).
		Msg("simulated OCI artifact push (implementation needed)")
	
	return artifactRef, digest, size, nil
}

// ListOCIArtifacts lists available OCI artifacts for a resource
func (h *OCIHandler) ListOCIArtifacts(w http.ResponseWriter, r *http.Request) {
	// Implementation for listing artifacts
	// This would query the OCI registry for available artifacts
	h.respondError(w, http.StatusNotImplemented, "ListOCIArtifacts not yet implemented")
}

// GetOCIArtifact retrieves information about a specific OCI artifact
func (h *OCIHandler) GetOCIArtifact(w http.ResponseWriter, r *http.Request) {
	// Implementation for getting artifact details
	// This would fetch artifact metadata from the OCI registry
	h.respondError(w, http.StatusNotImplemented, "GetOCIArtifact not yet implemented")
}

// DeleteOCIArtifact deletes an OCI artifact
func (h *OCIHandler) DeleteOCIArtifact(w http.ResponseWriter, r *http.Request) {
	// Implementation for deleting artifacts
	// This would remove the artifact from the OCI registry
	h.respondError(w, http.StatusNotImplemented, "DeleteOCIArtifact not yet implemented")
}

// respondError sends an error response
func (h *OCIHandler) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(OCIArtifactResponse{
		Success: false,
		Message: message,
	})
}