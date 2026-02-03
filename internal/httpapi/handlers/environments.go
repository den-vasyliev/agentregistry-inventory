package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/zerolog"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

// EnvironmentHandler handles environment/namespace operations
type EnvironmentHandler struct {
	client client.Client
	cache  cache.Cache
	logger zerolog.Logger
}

// NewEnvironmentHandler creates a new environment handler
func NewEnvironmentHandler(c client.Client, cache cache.Cache, logger zerolog.Logger) *EnvironmentHandler {
	return &EnvironmentHandler{
		client: c,
		cache:  cache,
		logger: logger.With().Str("handler", "environments").Logger(),
	}
}

// Environment response types
type EnvironmentJSON struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Labels    map[string]string `json:"labels,omitempty"`
}

type EnvironmentListResponse struct {
	Environments []EnvironmentJSON `json:"environments"`
	Metadata     ListMetadata      `json:"metadata"`
}

// RegisterRoutes registers environment endpoints
func (h *EnvironmentHandler) RegisterRoutes(api huma.API, pathPrefix string, isAdmin bool) {
	tags := []string{"environments"}
	if isAdmin {
		tags = append(tags, "admin")
	}

	// List environments from DiscoveryConfig
	huma.Register(api, huma.Operation{
		OperationID: "list-environments",
		Method:      http.MethodGet,
		Path:        pathPrefix + "/environments",
		Summary:     "List available environments from DiscoveryConfig",
		Tags:        tags,
	}, func(ctx context.Context, input *struct{}) (*Response[EnvironmentListResponse], error) {
		return h.listEnvironments(ctx)
	})
}

func (h *EnvironmentHandler) listEnvironments(ctx context.Context) (*Response[EnvironmentListResponse], error) {
	var discoveryConfigList agentregistryv1alpha1.DiscoveryConfigList

	// List DiscoveryConfigs from agentregistry namespace (where they are created)
	listOpts := &client.ListOptions{
		Namespace: "agentregistry",
	}
	if err := h.client.List(ctx, &discoveryConfigList, listOpts); err != nil {
		h.logger.Error().Err(err).Msg("Failed to list DiscoveryConfigs")
		return nil, huma.Error500InternalServerError("Failed to list DiscoveryConfigs", err)
	}

	h.logger.Info().Int("count", len(discoveryConfigList.Items)).Msg("Found DiscoveryConfigs")

	environments := make([]EnvironmentJSON, 0)

	// Collect all environments from all DiscoveryConfigs
	for _, dc := range discoveryConfigList.Items {
		h.logger.Info().
			Str("name", dc.Name).
			Str("namespace", dc.Namespace).
			Int("envCount", len(dc.Spec.Environments)).
			Msg("Processing DiscoveryConfig")

		for _, env := range dc.Spec.Environments {
			// Use the primary namespace for the environment
			namespace := env.Cluster.Namespace
			if namespace == "" && len(env.Namespaces) > 0 {
				namespace = env.Namespaces[0]
			}
			if namespace == "" {
				namespace = "agentregistry"
			}

			h.logger.Info().
				Str("envName", env.Name).
				Str("clusterNamespace", env.Cluster.Namespace).
				Strs("namespaces", env.Namespaces).
				Str("selectedNamespace", namespace).
				Msg("Adding environment")

			environments = append(environments, EnvironmentJSON{
				Name:      env.Name,
				Namespace: namespace,
				Labels:    env.Labels,
			})
		}
	}

	// If no DiscoveryConfigs found, return agentregistry namespace as default
	if len(environments) == 0 {
		environments = []EnvironmentJSON{
			{Name: "agentregistry", Namespace: "agentregistry"},
		}
	}

	return &Response[EnvironmentListResponse]{
		Body: EnvironmentListResponse{
			Environments: environments,
			Metadata: ListMetadata{
				Count: len(environments),
			},
		},
	}, nil
}
