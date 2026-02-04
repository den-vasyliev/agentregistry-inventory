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
	"github.com/agentregistry-dev/agentregistry/internal/conversion"
	"github.com/agentregistry-dev/agentregistry/internal/validation"
)

// ServerHandler handles MCP server catalog operations
type ServerHandler struct {
	client client.Client
	cache  cache.Cache
	logger zerolog.Logger
}

// listFromCacheOrClient lists resources from cache if available, otherwise from client
func (h *ServerHandler) listFromCacheOrClient(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	if h.cache != nil {
		return h.cache.List(ctx, list, opts...)
	}
	return h.client.List(ctx, list, opts...)
}

// NewServerHandler creates a new server handler
func NewServerHandler(c client.Client, cache cache.Cache, logger zerolog.Logger) *ServerHandler {
	return &ServerHandler{
		client: c,
		cache:  cache,
		logger: logger.With().Str("handler", "servers").Logger(),
	}
}

// Server response types
type ServerJSON struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	WebsiteURL  string          `json:"websiteUrl,omitempty"`
	Repository  *RepositoryJSON `json:"repository,omitempty"`
	Packages    []PackageJSON   `json:"packages,omitempty"`
	Remotes     []TransportJSON `json:"remotes,omitempty"`
}

type RepositoryJSON struct {
	URL       string `json:"url,omitempty"`
	Source    string `json:"source,omitempty"`
	ID        string `json:"id,omitempty"`
	Subfolder string `json:"subfolder,omitempty"`
}

type TransportJSON struct {
	Type    string         `json:"type"`
	URL     string         `json:"url,omitempty"`
	Headers []KeyValueJSON `json:"headers,omitempty"`
}

type KeyValueJSON struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Value       string `json:"value,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type PackageJSON struct {
	RegistryType         string         `json:"registryType"`
	RegistryBaseURL      string         `json:"registryBaseUrl,omitempty"`
	Identifier           string         `json:"identifier"`
	Version              string         `json:"version,omitempty"`
	FileSHA256           string         `json:"fileSha256,omitempty"`
	RuntimeHint          string         `json:"runtimeHint,omitempty"`
	Transport            TransportJSON  `json:"transport"`
	RuntimeArguments     []ArgumentJSON `json:"runtimeArguments,omitempty"`
	PackageArguments     []ArgumentJSON `json:"packageArguments,omitempty"`
	EnvironmentVariables []KeyValueJSON `json:"environmentVariables,omitempty"`
}

type ArgumentJSON struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Value       string `json:"value,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Multiple    bool   `json:"multiple,omitempty"`
}

type ServerMeta struct {
	Official          *OfficialMeta          `json:"io.modelcontextprotocol.registry/official,omitempty"`
	PublisherProvided map[string]interface{} `json:"io.modelcontextprotocol.registry/publisher-provided,omitempty"`
	Deployment        *DeploymentInfo        `json:"deployment,omitempty"`
	Source            string                 `json:"source,omitempty"` // discovery, manual, deployment
	IsDiscovered      bool                   `json:"isDiscovered,omitempty"`
}

type OfficialMeta struct {
	Status      string     `json:"status"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	IsLatest    bool       `json:"isLatest"`
	Published   bool       `json:"published"`
}

type ServerResponse struct {
	Server ServerJSON `json:"server"`
	Meta   ServerMeta `json:"_meta"`
}

type ServerListResponse struct {
	Servers  []ServerResponse `json:"servers"`
	Metadata ListMetadata     `json:"metadata"`
}

// Input types
type ListServersInput struct {
	Cursor  string `query:"cursor" json:"cursor,omitempty"`
	Limit   int    `query:"limit" json:"limit,omitempty" default:"30" minimum:"1" maximum:"100"`
	Search  string `query:"search" json:"search,omitempty"`
	Version string `query:"version" json:"version,omitempty"`
}

type ServerDetailInput struct {
	ServerName string `path:"serverName" json:"serverName"`
}

type ServerVersionDetailInput struct {
	ServerName string `path:"serverName" json:"serverName"`
	Version    string `path:"version" json:"version"`
}

type CreateServerInput struct {
	Body ServerJSON
}

type PublishServerInput struct {
	ServerName string `path:"serverName" json:"serverName"`
	Version    string `path:"version" json:"version"`
}

// RegisterRoutes registers server endpoints
func (h *ServerHandler) RegisterRoutes(api huma.API, pathPrefix string, isAdmin bool) {
	tags := []string{"servers"}
	if isAdmin {
		tags = append(tags, "admin")
	}

	// List servers
	huma.Register(api, huma.Operation{
		OperationID: "list-servers" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodGet,
		Path:        pathPrefix + "/servers",
		Summary:     "List MCP servers",
		Tags:        tags,
	}, func(ctx context.Context, input *ListServersInput) (*Response[ServerListResponse], error) {
		return h.listServers(ctx, input, isAdmin)
	})

	// Get server by name (latest version)
	huma.Register(api, huma.Operation{
		OperationID: "get-server" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodGet,
		Path:        pathPrefix + "/servers/{serverName}",
		Summary:     "Get MCP server details",
		Tags:        tags,
	}, func(ctx context.Context, input *ServerDetailInput) (*Response[ServerResponse], error) {
		return h.getServer(ctx, input, isAdmin)
	})

	// Get specific version
	huma.Register(api, huma.Operation{
		OperationID: "get-server-version" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodGet,
		Path:        pathPrefix + "/servers/{serverName}/versions/{version}",
		Summary:     "Get specific MCP server version",
		Tags:        tags,
	}, func(ctx context.Context, input *ServerVersionDetailInput) (*Response[ServerResponse], error) {
		return h.getServerVersion(ctx, input, isAdmin)
	})

	// Create server (push)
	huma.Register(api, huma.Operation{
		OperationID: "push-server" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodPost,
		Path:        pathPrefix + "/servers/push",
		Summary:     "Push MCP server",
		Tags:        tags,
	}, func(ctx context.Context, input *CreateServerInput) (*Response[ServerResponse], error) {
		return h.createServer(ctx, input)
	})

	// Admin-only endpoints
	if isAdmin {
		// Create server (POST /admin/v0/servers) - same as push but different path for UI compatibility
		huma.Register(api, huma.Operation{
			OperationID: "create-server" + strings.ReplaceAll(pathPrefix, "/", "-"),
			Method:      http.MethodPost,
			Path:        pathPrefix + "/servers",
			Summary:     "Create MCP server",
			Tags:        tags,
		}, func(ctx context.Context, input *CreateServerInput) (*Response[ServerResponse], error) {
			return h.createServer(ctx, input)
		})

		// List all versions of a server
		huma.Register(api, huma.Operation{
			OperationID: "list-server-versions" + strings.ReplaceAll(pathPrefix, "/", "-"),
			Method:      http.MethodGet,
			Path:        pathPrefix + "/servers/{serverName}/versions",
			Summary:     "List all versions of an MCP server",
			Tags:        tags,
		}, func(ctx context.Context, input *ServerDetailInput) (*Response[ServerListResponse], error) {
			return h.listServerVersions(ctx, input)
		})

		// Publish server
		huma.Register(api, huma.Operation{
			OperationID: "publish-server" + strings.ReplaceAll(pathPrefix, "/", "-"),
			Method:      http.MethodPost,
			Path:        pathPrefix + "/servers/{serverName}/versions/{version}/publish",
			Summary:     "Publish MCP server version",
			Tags:        tags,
		}, func(ctx context.Context, input *PublishServerInput) (*Response[ServerResponse], error) {
			return h.publishServer(ctx, input)
		})

		// Unpublish server
		huma.Register(api, huma.Operation{
			OperationID: "unpublish-server" + strings.ReplaceAll(pathPrefix, "/", "-"),
			Method:      http.MethodPost,
			Path:        pathPrefix + "/servers/{serverName}/versions/{version}/unpublish",
			Summary:     "Unpublish MCP server version",
			Tags:        tags,
		}, func(ctx context.Context, input *PublishServerInput) (*Response[ServerResponse], error) {
			return h.unpublishServer(ctx, input)
		})

		// Delete server version
		huma.Register(api, huma.Operation{
			OperationID: "delete-server-version" + strings.ReplaceAll(pathPrefix, "/", "-"),
			Method:      http.MethodDelete,
			Path:        pathPrefix + "/servers/{serverName}/versions/{version}",
			Summary:     "Delete MCP server version",
			Tags:        tags,
		}, func(ctx context.Context, input *ServerVersionDetailInput) (*Response[EmptyResponse], error) {
			return h.deleteServerVersion(ctx, input)
		})
	}
}

func (h *ServerHandler) listServers(ctx context.Context, input *ListServersInput, isAdmin bool) (*Response[ServerListResponse], error) {
	var serverList agentregistryv1alpha1.MCPServerCatalogList

	listOpts := []client.ListOption{}

	// Filter by published status for non-admin
	if !isAdmin {
		listOpts = append(listOpts, client.MatchingFields{
			controller.IndexMCPServerPublished: "true",
		})
	}

	// Filter by latest version if requested
	if input.Version == "latest" {
		listOpts = append(listOpts, client.MatchingFields{
			controller.IndexMCPServerIsLatest: "true",
		})
	}

	if err := h.listFromCacheOrClient(ctx, &serverList, listOpts...); err != nil {
		return nil, huma.Error500InternalServerError("Failed to list servers", err)
	}

	// Apply additional filters
	servers := make([]ServerResponse, 0, len(serverList.Items))
	for _, s := range serverList.Items {
		// Filter by search term
		if input.Search != "" && !strings.Contains(strings.ToLower(s.Spec.Name), strings.ToLower(input.Search)) {
			continue
		}

		// Filter by specific version
		if input.Version != "" && input.Version != "latest" && s.Spec.Version != input.Version {
			continue
		}

		servers = append(servers, h.convertToServerResponse(&s))
	}

	// Apply pagination
	limit := input.Limit
	if limit <= 0 {
		limit = 30
	}
	if len(servers) > limit {
		servers = servers[:limit]
	}

	return &Response[ServerListResponse]{
		Body: ServerListResponse{
			Servers: servers,
			Metadata: ListMetadata{
				Count: len(servers),
			},
		},
	}, nil
}

func (h *ServerHandler) getServer(ctx context.Context, input *ServerDetailInput, isAdmin bool) (*Response[ServerResponse], error) {
	serverName, err := url.PathUnescape(input.ServerName)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid server name encoding", err)
	}

	var serverList agentregistryv1alpha1.MCPServerCatalogList
	listOpts := []client.ListOption{
		client.MatchingFields{
			controller.IndexMCPServerName:     serverName,
			controller.IndexMCPServerIsLatest: "true",
		},
	}

	if !isAdmin {
		listOpts = append(listOpts, client.MatchingFields{
			controller.IndexMCPServerPublished: "true",
		})
	}

	if err := h.listFromCacheOrClient(ctx, &serverList, listOpts...); err != nil {
		return nil, huma.Error500InternalServerError("Failed to get server", err)
	}

	if len(serverList.Items) == 0 {
		return nil, huma.Error404NotFound("Server not found")
	}

	return &Response[ServerResponse]{
		Body: h.convertToServerResponse(&serverList.Items[0]),
	}, nil
}

func (h *ServerHandler) getServerVersion(ctx context.Context, input *ServerVersionDetailInput, isAdmin bool) (*Response[ServerResponse], error) {
	serverName, err := url.PathUnescape(input.ServerName)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid server name encoding", err)
	}
	version, err := url.PathUnescape(input.Version)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid version encoding", err)
	}

	var serverList agentregistryv1alpha1.MCPServerCatalogList
	if err := h.listFromCacheOrClient(ctx, &serverList, client.MatchingFields{
		controller.IndexMCPServerName: serverName,
	}); err != nil {
		return nil, huma.Error500InternalServerError("Failed to get server", err)
	}

	for _, s := range serverList.Items {
		if s.Spec.Version == version {
			if !isAdmin && !s.Status.Published {
				return nil, huma.Error404NotFound("Server not found")
			}
			return &Response[ServerResponse]{
				Body: h.convertToServerResponse(&s),
			}, nil
		}
	}

	return nil, huma.Error404NotFound("Server version not found")
}

func (h *ServerHandler) createServer(ctx context.Context, input *CreateServerInput) (*Response[ServerResponse], error) {
	// Validate server name
	if err := validation.ValidateServerName(input.Body.Name); err != nil {
		return nil, huma.Error400BadRequest("Invalid server name", err)
	}

	// Validate version
	if err := validation.ValidateSemanticVersion(input.Body.Version); err != nil {
		return nil, huma.Error400BadRequest("Invalid version", err)
	}

	// Validate repository URL if provided
	if input.Body.Repository != nil && input.Body.Repository.URL != "" {
		if err := validation.ValidateRepositoryURL(input.Body.Repository.URL); err != nil {
			return nil, huma.Error400BadRequest("Invalid repository URL", err)
		}
	}

	crName := GenerateCRName(input.Body.Name, input.Body.Version)

	server := &agentregistryv1alpha1.MCPServerCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: crName,
			Labels: map[string]string{
				"agentregistry.dev/name":    SanitizeK8sName(input.Body.Name),
				"agentregistry.dev/version": SanitizeK8sName(input.Body.Version),
			},
		},
		Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
			Name:        input.Body.Name,
			Version:     input.Body.Version,
			Title:       input.Body.Title,
			Description: input.Body.Description,
			WebsiteURL:  input.Body.WebsiteURL,
		},
	}

	// Convert repository
	if input.Body.Repository != nil {
		server.Spec.Repository = &agentregistryv1alpha1.Repository{
			URL:       input.Body.Repository.URL,
			Source:    input.Body.Repository.Source,
			ID:        input.Body.Repository.ID,
			Subfolder: input.Body.Repository.Subfolder,
		}
	}

	// Convert packages
	for _, p := range input.Body.Packages {
		pkg := agentregistryv1alpha1.Package{
			RegistryType:    p.RegistryType,
			RegistryBaseURL: p.RegistryBaseURL,
			Identifier:      p.Identifier,
			Version:         p.Version,
			FileSHA256:      p.FileSHA256,
			RuntimeHint:     p.RuntimeHint,
			Transport: agentregistryv1alpha1.Transport{
				Type: p.Transport.Type,
				URL:  p.Transport.URL,
			},
		}
		for _, h := range p.Transport.Headers {
			pkg.Transport.Headers = append(pkg.Transport.Headers, agentregistryv1alpha1.KeyValueInput{
				Name:        h.Name,
				Description: h.Description,
				Value:       h.Value,
				Required:    h.Required,
			})
		}
		for _, a := range p.RuntimeArguments {
			pkg.RuntimeArguments = append(pkg.RuntimeArguments, agentregistryv1alpha1.Argument{
				Name:        a.Name,
				Type:        a.Type,
				Description: a.Description,
				Value:       a.Value,
				Required:    a.Required,
				Multiple:    a.Multiple,
			})
		}
		for _, a := range p.PackageArguments {
			pkg.PackageArguments = append(pkg.PackageArguments, agentregistryv1alpha1.Argument{
				Name:        a.Name,
				Type:        a.Type,
				Description: a.Description,
				Value:       a.Value,
				Required:    a.Required,
				Multiple:    a.Multiple,
			})
		}
		for _, e := range p.EnvironmentVariables {
			pkg.EnvironmentVariables = append(pkg.EnvironmentVariables, agentregistryv1alpha1.KeyValueInput{
				Name:        e.Name,
				Description: e.Description,
				Value:       e.Value,
				Required:    e.Required,
			})
		}
		server.Spec.Packages = append(server.Spec.Packages, pkg)
	}

	// Convert remotes
	for _, r := range input.Body.Remotes {
		remote := agentregistryv1alpha1.Transport{
			Type: r.Type,
			URL:  r.URL,
		}
		for _, h := range r.Headers {
			remote.Headers = append(remote.Headers, agentregistryv1alpha1.KeyValueInput{
				Name:        h.Name,
				Description: h.Description,
				Value:       h.Value,
				Required:    h.Required,
			})
		}
		server.Spec.Remotes = append(server.Spec.Remotes, remote)
	}

	// Create the CR
	if err := h.client.Create(ctx, server); err != nil {
		return nil, huma.Error500InternalServerError("Failed to create server", err)
	}

	return &Response[ServerResponse]{
		Body: h.convertToServerResponse(server),
	}, nil
}

func (h *ServerHandler) listServerVersions(ctx context.Context, input *ServerDetailInput) (*Response[ServerListResponse], error) {
	serverName, err := url.PathUnescape(input.ServerName)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid server name encoding", err)
	}

	var serverList agentregistryv1alpha1.MCPServerCatalogList
	if err := h.listFromCacheOrClient(ctx, &serverList, client.MatchingFields{
		controller.IndexMCPServerName: serverName,
	}); err != nil {
		return nil, huma.Error500InternalServerError("Failed to list server versions", err)
	}

	servers := make([]ServerResponse, 0, len(serverList.Items))
	for _, s := range serverList.Items {
		servers = append(servers, h.convertToServerResponse(&s))
	}

	return &Response[ServerListResponse]{
		Body: ServerListResponse{
			Servers: servers,
			Metadata: ListMetadata{
				Count: len(servers),
			},
		},
	}, nil
}

func (h *ServerHandler) publishServer(ctx context.Context, input *PublishServerInput) (*Response[ServerResponse], error) {
	serverName, err := url.PathUnescape(input.ServerName)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid server name encoding", err)
	}
	version, err := url.PathUnescape(input.Version)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid version encoding", err)
	}

	var serverList agentregistryv1alpha1.MCPServerCatalogList
	if err := h.listFromCacheOrClient(ctx, &serverList, client.MatchingFields{
		controller.IndexMCPServerName: serverName,
	}); err != nil {
		return nil, huma.Error500InternalServerError("Failed to find server", err)
	}

	for i := range serverList.Items {
		s := &serverList.Items[i]
		if s.Spec.Version == version {
			// Update status to published
			now := metav1.Now()
			s.Status.Published = true
			s.Status.PublishedAt = &now
			s.Status.Status = agentregistryv1alpha1.CatalogStatusActive
			s.Status.Conditions = SetCatalogCondition(s.Status.Conditions,
				agentregistryv1alpha1.CatalogConditionPublished,
				metav1.ConditionTrue, "Published", "Server version published")

			if err := h.client.Status().Update(ctx, s); err != nil {
				return nil, huma.Error500InternalServerError("Failed to publish server", err)
			}

			return &Response[ServerResponse]{
				Body: h.convertToServerResponse(s),
			}, nil
		}
	}

	return nil, huma.Error404NotFound("Server version not found")
}

func (h *ServerHandler) unpublishServer(ctx context.Context, input *PublishServerInput) (*Response[ServerResponse], error) {
	serverName, err := url.PathUnescape(input.ServerName)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid server name encoding", err)
	}
	version, err := url.PathUnescape(input.Version)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid version encoding", err)
	}

	var serverList agentregistryv1alpha1.MCPServerCatalogList
	if err := h.listFromCacheOrClient(ctx, &serverList, client.MatchingFields{
		controller.IndexMCPServerName: serverName,
	}); err != nil {
		return nil, huma.Error500InternalServerError("Failed to find server", err)
	}

	for i := range serverList.Items {
		s := &serverList.Items[i]
		if s.Spec.Version == version {
			// Update status to unpublished
			s.Status.Published = false
			s.Status.Status = agentregistryv1alpha1.CatalogStatusDeprecated
			s.Status.Conditions = SetCatalogCondition(s.Status.Conditions,
				agentregistryv1alpha1.CatalogConditionPublished,
				metav1.ConditionFalse, "Unpublished", "Server version unpublished")

			if err := h.client.Status().Update(ctx, s); err != nil {
				return nil, huma.Error500InternalServerError("Failed to unpublish server", err)
			}

			return &Response[ServerResponse]{
				Body: h.convertToServerResponse(s),
			}, nil
		}
	}

	return nil, huma.Error404NotFound("Server version not found")
}

func (h *ServerHandler) deleteServerVersion(ctx context.Context, input *ServerVersionDetailInput) (*Response[EmptyResponse], error) {
	serverName, err := url.PathUnescape(input.ServerName)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid server name encoding", err)
	}
	version, err := url.PathUnescape(input.Version)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid version encoding", err)
	}

	crName := GenerateCRName(serverName, version)
	server := &agentregistryv1alpha1.MCPServerCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: crName,
		},
	}

	if err := h.client.Delete(ctx, server); err != nil {
		return nil, huma.Error500InternalServerError("Failed to delete server", err)
	}

	return &Response[EmptyResponse]{
		Body: EmptyResponse{Message: "Server deleted successfully"},
	}, nil
}

func (h *ServerHandler) convertToServerResponse(s *agentregistryv1alpha1.MCPServerCatalog) ServerResponse {
	// Convert repository using conversion package
	var repoJSON *RepositoryJSON
	if repo := conversion.RepositoryFromCRD(s.Spec.Repository); repo != nil {
		repoJSON = &RepositoryJSON{
			URL:       repo.URL,
			Source:    repo.Source,
			ID:        repo.ID,
			Subfolder: repo.Subfolder,
		}
	}

	// Convert packages using conversion package
	packages := make([]PackageJSON, len(s.Spec.Packages))
	for i, p := range s.Spec.Packages {
		pkg := conversion.PackageFromCRD(p)
		packages[i] = PackageJSON{
			RegistryType:         pkg.RegistryType,
			RegistryBaseURL:      pkg.RegistryBaseURL,
			Identifier:           pkg.Identifier,
			Version:              pkg.Version,
			FileSHA256:           pkg.FileSHA256,
			RuntimeHint:          pkg.RuntimeHint,
			Transport:            convertTransport(pkg.Transport),
			RuntimeArguments:     convertArguments(pkg.RuntimeArguments),
			PackageArguments:     convertArguments(pkg.PackageArguments),
			EnvironmentVariables: convertKeyValues(pkg.EnvironmentVariables),
		}
	}

	// Convert remotes using conversion package
	remotes := make([]TransportJSON, len(s.Spec.Remotes))
	for i, r := range s.Spec.Remotes {
		transport := conversion.TransportFromCRD(r)
		remotes[i] = convertTransport(transport)
	}

	server := ServerJSON{
		Name:        s.Spec.Name,
		Version:     s.Spec.Version,
		Title:       s.Spec.Title,
		Description: s.Spec.Description,
		WebsiteURL:  s.Spec.WebsiteURL,
		Repository:  repoJSON,
		Packages:    packages,
		Remotes:     remotes,
	}

	var publishedAt *time.Time
	if s.Status.PublishedAt != nil {
		t := s.Status.PublishedAt.Time
		publishedAt = &t
	}

	resp := ServerResponse{
		Server: server,
		Meta: ServerMeta{
			Official: &OfficialMeta{
				Status:      string(s.Status.Status),
				PublishedAt: publishedAt,
				UpdatedAt:   s.CreationTimestamp.Time,
				IsLatest:    s.Status.IsLatest,
				Published:   s.Status.Published,
			},
		},
	}

	// Check for discovery labels to determine source
	if s.Labels != nil {
		if s.Labels["agentregistry.dev/discovered"] == "true" {
			resp.Meta.IsDiscovered = true
			resp.Meta.Source = "discovery"
		} else if source := s.Labels["agentregistry.dev/resource-source"]; source != "" {
			resp.Meta.Source = source
		}
	}

	// Include publisher-provided metadata if available
	if s.Spec.Metadata != nil && len(s.Spec.Metadata.Raw) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(s.Spec.Metadata.Raw, &metadata); err == nil {
			if publisherProvided, ok := metadata["io.modelcontextprotocol.registry/publisher-provided"].(map[string]interface{}); ok {
				resp.Meta.PublisherProvided = publisherProvided
			}
		}
	}

	// Note: deployment status is NOT included here.
	// UI fetches actual status from /deployments endpoint which reads from RegistryDeployment/MCPServer/Agent resources.

	return resp
}

// convertArguments converts conversion.ArgumentJSON slice to handlers.ArgumentJSON slice
func convertArguments(args []conversion.ArgumentJSON) []ArgumentJSON {
	result := make([]ArgumentJSON, len(args))
	for i, a := range args {
		result[i] = ArgumentJSON{
			Name:        a.Name,
			Type:        a.Type,
			Description: a.Description,
			Value:       a.Value,
			Required:    a.Required,
			Multiple:    a.Multiple,
		}
	}
	return result
}

// convertKeyValues converts conversion.KeyValueJSON slice to handlers.KeyValueJSON slice
func convertKeyValues(kvs []conversion.KeyValueJSON) []KeyValueJSON {
	result := make([]KeyValueJSON, len(kvs))
	for i, kv := range kvs {
		result[i] = KeyValueJSON{
			Name:        kv.Name,
			Description: kv.Description,
			Value:       kv.Value,
			Required:    kv.Required,
		}
	}
	return result
}

// convertTransport converts conversion.TransportJSON to handlers.TransportJSON
func convertTransport(t conversion.TransportJSON) TransportJSON {
	return TransportJSON{
		Type:    t.Type,
		URL:     t.URL,
		Headers: convertKeyValues(t.Headers),
	}
}
