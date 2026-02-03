package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/zerolog"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	// Return static list of deployment target namespaces
	// These are the namespaces where resources can be deployed
	// NOT from DiscoveryConfig (which is for discovering already-deployed resources)
	environments := []EnvironmentJSON{
		{Name: "dev", Namespace: "dev"},
		{Name: "staging", Namespace: "staging"},
		{Name: "prod", Namespace: "prod"},
		{Name: "agentregistry", Namespace: "agentregistry"},
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
