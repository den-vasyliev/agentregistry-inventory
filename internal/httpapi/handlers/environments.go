package handlers

import (
	"context"
	"net/http"
	"time"

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
	Name          string            `json:"name"`
	Cluster       string            `json:"cluster"`
	Provider      string            `json:"provider,omitempty"`
	Region        string            `json:"region,omitempty"`
	Namespace     string            `json:"namespace"`
	DeployEnabled bool              `json:"deployEnabled"`
	Labels        map[string]string `json:"labels,omitempty"`
}

type EnvironmentListResponse struct {
	Environments []EnvironmentJSON `json:"environments"`
	Metadata     ListMetadata      `json:"metadata"`
}

// DiscoveryMap response types for the topology visualization

// DiscoveryMapCluster represents a cluster in the discovery map
type DiscoveryMapCluster struct {
	Name     string `json:"name"`
	Provider string `json:"provider,omitempty"`
	Zone     string `json:"zone,omitempty"`
	Region   string `json:"region,omitempty"`
}

// DiscoveryMapEnvironment represents an environment with its discovery status
type DiscoveryMapEnvironment struct {
	Name                string                     `json:"name"`
	Cluster             DiscoveryMapCluster        `json:"cluster"`
	Namespaces          []string                   `json:"namespaces,omitempty"`
	ResourceTypes       []string                   `json:"resourceTypes,omitempty"`
	DiscoveryEnabled    bool                       `json:"discoveryEnabled"`
	Connected           bool                       `json:"connected"`
	DiscoveredResources DiscoveryMapResourceCounts `json:"discoveredResources"`
	LastSyncTime        *time.Time                 `json:"lastSyncTime,omitempty"`
	Error               string                     `json:"error,omitempty"`
	Labels              map[string]string          `json:"labels,omitempty"`
}

// DiscoveryMapResourceCounts represents resource counts per environment
type DiscoveryMapResourceCounts struct {
	MCPServers int `json:"mcpServers"`
	Agents     int `json:"agents"`
	Skills     int `json:"skills"`
	Models     int `json:"models"`
}

// DiscoveryMapResponse is the full discovery map for visualization
type DiscoveryMapResponse struct {
	Configs []DiscoveryMapConfig `json:"configs"`
}

// DiscoveryMapConfig represents a single DiscoveryConfig with its environments
type DiscoveryMapConfig struct {
	Name         string                    `json:"name"`
	Environments []DiscoveryMapEnvironment `json:"environments"`
	LastSyncTime *time.Time                `json:"lastSyncTime,omitempty"`
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

	// Discovery map for topology visualization
	huma.Register(api, huma.Operation{
		OperationID: "get-discovery-map",
		Method:      http.MethodGet,
		Path:        pathPrefix + "/discovery/map",
		Summary:     "Get discovery topology map for visualization",
		Tags:        tags,
	}, func(ctx context.Context, input *struct{}) (*Response[DiscoveryMapResponse], error) {
		return h.getDiscoveryMap(ctx)
	})
}

func (h *EnvironmentHandler) listEnvironments(ctx context.Context) (*Response[EnvironmentListResponse], error) {
	var list agentregistryv1alpha1.DiscoveryConfigList
	if err := h.client.List(ctx, &list, client.InNamespace("agentregistry")); err != nil {
		return nil, huma.Error500InternalServerError("failed to list DiscoveryConfigs", err)
	}

	environments := make([]EnvironmentJSON, 0)
	for _, dc := range list.Items {
		for _, env := range dc.Spec.Environments {
			ns := env.Cluster.Namespace
			if ns == "" && len(env.Namespaces) > 0 {
				ns = env.Namespaces[0]
			}
			region := env.Cluster.Region
			if region == "" {
				region = env.Cluster.Zone
			}
			environments = append(environments, EnvironmentJSON{
				Name:          env.Name,
				Cluster:       env.Cluster.Name,
				Provider:      env.Provider,
				Region:        region,
				Namespace:     ns,
				DeployEnabled: env.DeployEnabled,
				Labels:        env.Labels,
			})
		}
	}

	return &Response[EnvironmentListResponse]{
		Body: EnvironmentListResponse{
			Environments: environments,
			Metadata:     ListMetadata{Count: len(environments)},
		},
	}, nil
}

func (h *EnvironmentHandler) getDiscoveryMap(ctx context.Context) (*Response[DiscoveryMapResponse], error) {
	var list agentregistryv1alpha1.DiscoveryConfigList
	if err := h.client.List(ctx, &list, client.InNamespace("agentregistry")); err != nil {
		return nil, huma.Error500InternalServerError("failed to list DiscoveryConfigs", err)
	}

	configs := make([]DiscoveryMapConfig, 0, len(list.Items))
	for _, dc := range list.Items {
		// Build status lookup by environment name
		statusByEnv := make(map[string]agentregistryv1alpha1.EnvironmentStatus)
		for _, es := range dc.Status.Environments {
			statusByEnv[es.Name] = es
		}

		envs := make([]DiscoveryMapEnvironment, 0, len(dc.Spec.Environments))
		for _, env := range dc.Spec.Environments {
			mapEnv := DiscoveryMapEnvironment{
				Name: env.Name,
				Cluster: DiscoveryMapCluster{
					Name:     env.Cluster.Name,
					Provider: env.Provider,
					Zone:     env.Cluster.Zone,
					Region:   env.Cluster.Region,
				},
				Namespaces:       env.Namespaces,
				ResourceTypes:    env.ResourceTypes,
				DiscoveryEnabled: env.DiscoveryEnabled,
				Labels:           env.Labels,
			}

			// Merge status if available
			if es, ok := statusByEnv[env.Name]; ok {
				mapEnv.Connected = es.Connected
				mapEnv.Error = es.Error
				if es.LastSyncTime != nil {
					t := es.LastSyncTime.Time
					mapEnv.LastSyncTime = &t
				}
				mapEnv.DiscoveredResources = DiscoveryMapResourceCounts{
					MCPServers: es.DiscoveredResources.MCPServers,
					Agents:     es.DiscoveredResources.Agents,
					Skills:     es.DiscoveredResources.Skills,
					Models:     es.DiscoveredResources.Models,
				}
			}

			envs = append(envs, mapEnv)
		}

		var lastSync *time.Time
		if dc.Status.LastSyncTime != nil {
			t := dc.Status.LastSyncTime.Time
			lastSync = &t
		}

		configs = append(configs, DiscoveryMapConfig{
			Name:         dc.Name,
			Environments: envs,
			LastSyncTime: lastSync,
		})
	}

	return &Response[DiscoveryMapResponse]{
		Body: DiscoveryMapResponse{
			Configs: configs,
		},
	}, nil
}
