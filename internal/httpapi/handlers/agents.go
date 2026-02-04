package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	"github.com/agentregistry-dev/agentregistry/internal/controller"
)

// AgentHandler handles agent catalog operations
type AgentHandler struct {
	client client.Client
	cache  cache.Cache
	logger zerolog.Logger
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(c client.Client, cache cache.Cache, logger zerolog.Logger) *AgentHandler {
	return &AgentHandler{
		client: c,
		cache:  cache,
		logger: logger.With().Str("handler", "agents").Logger(),
	}
}

// Agent response types
type AgentJSON struct {
	Name              string                `json:"name"`
	Version           string                `json:"version"`
	Title             string                `json:"title,omitempty"`
	Description       string                `json:"description,omitempty"`
	Image             string                `json:"image"`
	Language          string                `json:"language,omitempty"`
	Framework         string                `json:"framework,omitempty"`
	ModelProvider     string                `json:"modelProvider,omitempty"`
	ModelName         string                `json:"modelName,omitempty"`
	TelemetryEndpoint string                `json:"telemetryEndpoint,omitempty"`
	WebsiteURL        string                `json:"websiteUrl,omitempty"`
	Repository        *RepositoryJSON       `json:"repository,omitempty"`
	Packages          []AgentPackageJSON    `json:"packages,omitempty"`
	Remotes           []TransportJSON       `json:"remotes,omitempty"`
	McpServers        []McpServerConfigJSON `json:"mcpServers,omitempty"`
}

type AgentPackageJSON struct {
	RegistryType string                     `json:"registryType"`
	Identifier   string                     `json:"identifier"`
	Version      string                     `json:"version,omitempty"`
	Transport    *AgentPackageTransportJSON `json:"transport,omitempty"`
}

type AgentPackageTransportJSON struct {
	Type string `json:"type"`
}

type McpServerConfigJSON struct {
	Type                       string            `json:"type"`
	Name                       string            `json:"name"`
	Image                      string            `json:"image,omitempty"`
	Build                      string            `json:"build,omitempty"`
	Command                    string            `json:"command,omitempty"`
	Args                       []string          `json:"args,omitempty"`
	Env                        []string          `json:"env,omitempty"`
	URL                        string            `json:"url,omitempty"`
	Headers                    map[string]string `json:"headers,omitempty"`
	RegistryURL                string            `json:"registryURL,omitempty"`
	RegistryServerName         string            `json:"registryServerName,omitempty"`
	RegistryServerVersion      string            `json:"registryServerVersion,omitempty"`
	RegistryServerPreferRemote bool              `json:"registryServerPreferRemote,omitempty"`
}

type AgentMeta struct {
	Official          *OfficialMeta          `json:"io.modelcontextprotocol.registry/official,omitempty"`
	PublisherProvided map[string]interface{} `json:"io.modelcontextprotocol.registry/publisher-provided,omitempty"`
	Deployment        *DeploymentInfo        `json:"deployment,omitempty"`
	Source            string                 `json:"source,omitempty"` // discovery, manual, deployment
	IsDiscovered      bool                   `json:"isDiscovered,omitempty"`
}

type AgentResponse struct {
	Agent AgentJSON `json:"agent"`
	Meta  AgentMeta `json:"_meta"`
}

type AgentListResponse struct {
	Agents   []AgentResponse `json:"agents"`
	Metadata ListMetadata    `json:"metadata"`
}

// Input types
type ListAgentsInput struct {
	Cursor  string `query:"cursor" json:"cursor,omitempty"`
	Limit   int    `query:"limit" json:"limit,omitempty" default:"30" minimum:"1" maximum:"100"`
	Search  string `query:"search" json:"search,omitempty"`
	Version string `query:"version" json:"version,omitempty"`
}

type AgentDetailInput struct {
	AgentName string `path:"agentName" json:"agentName"`
}

type AgentVersionDetailInput struct {
	AgentName string `path:"agentName" json:"agentName"`
	Version   string `path:"version" json:"version"`
}

type CreateAgentInput struct {
	Body AgentJSON
}

// RegisterRoutes registers agent endpoints
func (h *AgentHandler) RegisterRoutes(api huma.API, pathPrefix string, isAdmin bool) {
	tags := []string{"agents"}
	if isAdmin {
		tags = append(tags, "admin")
	}

	// List agents
	huma.Register(api, huma.Operation{
		OperationID: "list-agents" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodGet,
		Path:        pathPrefix + "/agents",
		Summary:     "List agents",
		Tags:        tags,
	}, func(ctx context.Context, input *ListAgentsInput) (*Response[AgentListResponse], error) {
		return h.listAgents(ctx, input, isAdmin)
	})

	// Get agent by name
	huma.Register(api, huma.Operation{
		OperationID: "get-agent" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodGet,
		Path:        pathPrefix + "/agents/{agentName}",
		Summary:     "Get agent details",
		Tags:        tags,
	}, func(ctx context.Context, input *AgentDetailInput) (*Response[AgentResponse], error) {
		return h.getAgent(ctx, input, isAdmin)
	})

	// Get specific version
	huma.Register(api, huma.Operation{
		OperationID: "get-agent-version" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodGet,
		Path:        pathPrefix + "/agents/{agentName}/versions/{version}",
		Summary:     "Get specific agent version",
		Tags:        tags,
	}, func(ctx context.Context, input *AgentVersionDetailInput) (*Response[AgentResponse], error) {
		return h.getAgentVersion(ctx, input, isAdmin)
	})

	// Create agent
	huma.Register(api, huma.Operation{
		OperationID: "push-agent" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodPost,
		Path:        pathPrefix + "/agents/push",
		Summary:     "Push agent",
		Tags:        tags,
	}, func(ctx context.Context, input *CreateAgentInput) (*Response[AgentResponse], error) {
		return h.createAgent(ctx, input)
	})

	// Admin-only endpoints
	if isAdmin {
		// Create agent (POST /admin/v0/agents) - same as push but different path for UI compatibility
		huma.Register(api, huma.Operation{
			OperationID: "create-agent" + strings.ReplaceAll(pathPrefix, "/", "-"),
			Method:      http.MethodPost,
			Path:        pathPrefix + "/agents",
			Summary:     "Create agent",
			Tags:        tags,
		}, func(ctx context.Context, input *CreateAgentInput) (*Response[AgentResponse], error) {
			return h.createAgent(ctx, input)
		})

		// List all versions of an agent
		huma.Register(api, huma.Operation{
			OperationID: "list-agent-versions" + strings.ReplaceAll(pathPrefix, "/", "-"),
			Method:      http.MethodGet,
			Path:        pathPrefix + "/agents/{agentName}/versions",
			Summary:     "List all versions of an agent",
			Tags:        tags,
		}, func(ctx context.Context, input *AgentDetailInput) (*Response[AgentListResponse], error) {
			return h.listAgentVersions(ctx, input)
		})

		// Delete agent version
		huma.Register(api, huma.Operation{
			OperationID: "delete-agent-version" + strings.ReplaceAll(pathPrefix, "/", "-"),
			Method:      http.MethodDelete,
			Path:        pathPrefix + "/agents/{agentName}/versions/{version}",
			Summary:     "Delete agent version",
			Tags:        tags,
		}, func(ctx context.Context, input *AgentVersionDetailInput) (*Response[EmptyResponse], error) {
			return h.deleteAgentVersion(ctx, input)
		})
	}
}

func (h *AgentHandler) listAgents(ctx context.Context, input *ListAgentsInput, isAdmin bool) (*Response[AgentListResponse], error) {
	var agentList agentregistryv1alpha1.AgentCatalogList

	listOpts := []client.ListOption{}

	if input.Version == "latest" {
		listOpts = append(listOpts, client.MatchingFields{
			controller.IndexAgentIsLatest: "true",
		})
	}

	if err := h.cache.List(ctx, &agentList, listOpts...); err != nil {
		return nil, huma.Error500InternalServerError("Failed to list agents", err)
	}

	agents := make([]AgentResponse, 0, len(agentList.Items))
	for _, a := range agentList.Items {
		if input.Search != "" && !strings.Contains(strings.ToLower(a.Spec.Name), strings.ToLower(input.Search)) {
			continue
		}

		if input.Version != "" && input.Version != "latest" && a.Spec.Version != input.Version {
			continue
		}

		agents = append(agents, h.convertToAgentResponse(&a))
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 30
	}
	if len(agents) > limit {
		agents = agents[:limit]
	}

	return &Response[AgentListResponse]{
		Body: AgentListResponse{
			Agents: agents,
			Metadata: ListMetadata{
				Count: len(agents),
			},
		},
	}, nil
}

func (h *AgentHandler) getAgent(ctx context.Context, input *AgentDetailInput, isAdmin bool) (*Response[AgentResponse], error) {
	agentName, err := url.PathUnescape(input.AgentName)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid agent name encoding", err)
	}

	var agentList agentregistryv1alpha1.AgentCatalogList
	listOpts := []client.ListOption{
		client.MatchingFields{
			controller.IndexAgentName:     agentName,
			controller.IndexAgentIsLatest: "true",
		},
	}

	if err := h.cache.List(ctx, &agentList, listOpts...); err != nil {
		return nil, huma.Error500InternalServerError("Failed to get agent", err)
	}

	if len(agentList.Items) == 0 {
		return nil, huma.Error404NotFound("Agent not found")
	}

	return &Response[AgentResponse]{
		Body: h.convertToAgentResponse(&agentList.Items[0]),
	}, nil
}

func (h *AgentHandler) getAgentVersion(ctx context.Context, input *AgentVersionDetailInput, isAdmin bool) (*Response[AgentResponse], error) {
	agentName, err := url.PathUnescape(input.AgentName)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid agent name encoding", err)
	}
	version, err := url.PathUnescape(input.Version)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid version encoding", err)
	}

	var agentList agentregistryv1alpha1.AgentCatalogList
	if err := h.cache.List(ctx, &agentList, client.MatchingFields{
		controller.IndexAgentName: agentName,
	}); err != nil {
		return nil, huma.Error500InternalServerError("Failed to get agent", err)
	}

	for _, a := range agentList.Items {
		if a.Spec.Version == version {
			return &Response[AgentResponse]{
				Body: h.convertToAgentResponse(&a),
			}, nil
		}
	}

	return nil, huma.Error404NotFound("Agent version not found")
}

func (h *AgentHandler) createAgent(ctx context.Context, input *CreateAgentInput) (*Response[AgentResponse], error) {
	crName := GenerateCRName(input.Body.Name, input.Body.Version)

	agent := &agentregistryv1alpha1.AgentCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: crName,
			Labels: map[string]string{
				"agentregistry.dev/name":    SanitizeK8sName(input.Body.Name),
				"agentregistry.dev/version": SanitizeK8sName(input.Body.Version),
			},
		},
		Spec: agentregistryv1alpha1.AgentCatalogSpec{
			Name:              input.Body.Name,
			Version:           input.Body.Version,
			Title:             input.Body.Title,
			Description:       input.Body.Description,
			Image:             input.Body.Image,
			Language:          input.Body.Language,
			Framework:         input.Body.Framework,
			ModelProvider:     input.Body.ModelProvider,
			ModelName:         input.Body.ModelName,
			TelemetryEndpoint: input.Body.TelemetryEndpoint,
			WebsiteURL:        input.Body.WebsiteURL,
		},
	}

	if input.Body.Repository != nil {
		agent.Spec.Repository = &agentregistryv1alpha1.Repository{
			URL:       input.Body.Repository.URL,
			Source:    input.Body.Repository.Source,
			ID:        input.Body.Repository.ID,
			Subfolder: input.Body.Repository.Subfolder,
		}
	}

	for _, p := range input.Body.Packages {
		pkg := agentregistryv1alpha1.AgentPackage{
			RegistryType: p.RegistryType,
			Identifier:   p.Identifier,
			Version:      p.Version,
		}
		if p.Transport != nil {
			pkg.Transport = &agentregistryv1alpha1.AgentPackageTransport{
				Type: p.Transport.Type,
			}
		}
		agent.Spec.Packages = append(agent.Spec.Packages, pkg)
	}

	for _, r := range input.Body.Remotes {
		remote := agentregistryv1alpha1.Transport{
			Type: r.Type,
			URL:  r.URL,
		}
		for _, hdr := range r.Headers {
			remote.Headers = append(remote.Headers, agentregistryv1alpha1.KeyValueInput{
				Name:        hdr.Name,
				Description: hdr.Description,
				Value:       hdr.Value,
				Required:    hdr.Required,
			})
		}
		agent.Spec.Remotes = append(agent.Spec.Remotes, remote)
	}

	for _, m := range input.Body.McpServers {
		mcp := agentregistryv1alpha1.McpServerConfig{
			Type:                       m.Type,
			Name:                       m.Name,
			Image:                      m.Image,
			Build:                      m.Build,
			Command:                    m.Command,
			Args:                       m.Args,
			Env:                        m.Env,
			URL:                        m.URL,
			Headers:                    m.Headers,
			RegistryURL:                m.RegistryURL,
			RegistryServerName:         m.RegistryServerName,
			RegistryServerVersion:      m.RegistryServerVersion,
			RegistryServerPreferRemote: m.RegistryServerPreferRemote,
		}
		agent.Spec.McpServers = append(agent.Spec.McpServers, mcp)
	}

	if err := h.client.Create(ctx, agent); err != nil {
		return nil, huma.Error500InternalServerError("Failed to create agent", err)
	}

	return &Response[AgentResponse]{
		Body: h.convertToAgentResponse(agent),
	}, nil
}

func (h *AgentHandler) listAgentVersions(ctx context.Context, input *AgentDetailInput) (*Response[AgentListResponse], error) {
	agentName, err := url.PathUnescape(input.AgentName)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid agent name encoding", err)
	}

	var agentList agentregistryv1alpha1.AgentCatalogList
	if err := h.cache.List(ctx, &agentList, client.MatchingFields{
		controller.IndexAgentName: agentName,
	}); err != nil {
		return nil, huma.Error500InternalServerError("Failed to list agent versions", err)
	}

	agents := make([]AgentResponse, 0, len(agentList.Items))
	for _, a := range agentList.Items {
		agents = append(agents, h.convertToAgentResponse(&a))
	}

	return &Response[AgentListResponse]{
		Body: AgentListResponse{
			Agents: agents,
			Metadata: ListMetadata{
				Count: len(agents),
			},
		},
	}, nil
}

func (h *AgentHandler) deleteAgentVersion(ctx context.Context, input *AgentVersionDetailInput) (*Response[EmptyResponse], error) {
	agentName, err := url.PathUnescape(input.AgentName)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid agent name encoding", err)
	}
	version, err := url.PathUnescape(input.Version)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid version encoding", err)
	}

	crName := GenerateCRName(agentName, version)
	agent := &agentregistryv1alpha1.AgentCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: crName,
		},
	}

	if err := h.client.Delete(ctx, agent); err != nil {
		return nil, huma.Error500InternalServerError("Failed to delete agent", err)
	}

	return &Response[EmptyResponse]{
		Body: EmptyResponse{Message: "Agent deleted successfully"},
	}, nil
}

func (h *AgentHandler) convertToAgentResponse(a *agentregistryv1alpha1.AgentCatalog) AgentResponse {
	agent := AgentJSON{
		Name:              a.Spec.Name,
		Version:           a.Spec.Version,
		Title:             a.Spec.Title,
		Description:       a.Spec.Description,
		Image:             a.Spec.Image,
		Language:          a.Spec.Language,
		Framework:         a.Spec.Framework,
		ModelProvider:     a.Spec.ModelProvider,
		ModelName:         a.Spec.ModelName,
		TelemetryEndpoint: a.Spec.TelemetryEndpoint,
		WebsiteURL:        a.Spec.WebsiteURL,
	}

	if a.Spec.Repository != nil {
		agent.Repository = &RepositoryJSON{
			URL:       a.Spec.Repository.URL,
			Source:    a.Spec.Repository.Source,
			ID:        a.Spec.Repository.ID,
			Subfolder: a.Spec.Repository.Subfolder,
		}
	}

	for _, p := range a.Spec.Packages {
		pkg := AgentPackageJSON{
			RegistryType: p.RegistryType,
			Identifier:   p.Identifier,
			Version:      p.Version,
		}
		if p.Transport != nil {
			pkg.Transport = &AgentPackageTransportJSON{
				Type: p.Transport.Type,
			}
		}
		agent.Packages = append(agent.Packages, pkg)
	}

	for _, r := range a.Spec.Remotes {
		remote := TransportJSON{
			Type: r.Type,
			URL:  r.URL,
		}
		for _, hdr := range r.Headers {
			remote.Headers = append(remote.Headers, KeyValueJSON{
				Name:        hdr.Name,
				Description: hdr.Description,
				Value:       hdr.Value,
				Required:    hdr.Required,
			})
		}
		agent.Remotes = append(agent.Remotes, remote)
	}

	for _, m := range a.Spec.McpServers {
		agent.McpServers = append(agent.McpServers, McpServerConfigJSON{
			Type:                       m.Type,
			Name:                       m.Name,
			Image:                      m.Image,
			Build:                      m.Build,
			Command:                    m.Command,
			Args:                       m.Args,
			Env:                        m.Env,
			URL:                        m.URL,
			Headers:                    m.Headers,
			RegistryURL:                m.RegistryURL,
			RegistryServerName:         m.RegistryServerName,
			RegistryServerVersion:      m.RegistryServerVersion,
			RegistryServerPreferRemote: m.RegistryServerPreferRemote,
		})
	}

	var publishedAt *time.Time
	if a.Status.PublishedAt != nil {
		t := a.Status.PublishedAt.Time
		publishedAt = &t
	}

	resp := AgentResponse{
		Agent: agent,
		Meta: AgentMeta{
			Official: &OfficialMeta{
				Status:      string(a.Status.Status),
				PublishedAt: publishedAt,
				UpdatedAt:   a.CreationTimestamp.Time,
				IsLatest:    a.Status.IsLatest,
				Published:   true,
			},
		},
	}

	// Check for discovery labels to determine source
	if a.Labels != nil {
		if a.Labels["agentregistry.dev/discovered"] == "true" {
			resp.Meta.IsDiscovered = true
			resp.Meta.Source = "discovery"
		} else if source := a.Labels["agentregistry.dev/resource-source"]; source != "" {
			resp.Meta.Source = source
		}
	}

	// Include publisher-provided metadata if available
	if a.Spec.Metadata != nil && len(a.Spec.Metadata.Raw) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(a.Spec.Metadata.Raw, &metadata); err == nil {
			if publisherProvided, ok := metadata["io.modelcontextprotocol.registry/publisher-provided"].(map[string]interface{}); ok {
				resp.Meta.PublisherProvided = publisherProvided
			}
		}
	}

	// Include deployment info if available from source resource
	if a.Status.Deployment != nil {
		var lastChecked *time.Time
		if a.Status.Deployment.LastChecked != nil {
			t := a.Status.Deployment.LastChecked.Time
			lastChecked = &t
		}
		resp.Meta.Deployment = &DeploymentInfo{
			Namespace:   a.Status.Deployment.Namespace,
			ServiceName: a.Status.Deployment.ServiceName,
			URL:         a.Status.Deployment.URL,
			Ready:       a.Status.Deployment.Ready,
			Message:     a.Status.Deployment.Message,
			LastChecked: lastChecked,
		}
	}

	return resp
}
