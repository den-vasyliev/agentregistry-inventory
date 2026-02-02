package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	"github.com/agentregistry-dev/agentregistry/internal/httpapi/handlers"
)

// Server is the HTTP API server that reads from the informer cache
type Server struct {
	client        client.Client
	cache         cache.Cache
	logger        zerolog.Logger
	mux           *http.ServeMux
	api           huma.API
	authEnabled   bool
	allowedTokens map[string]bool // Simple token allowlist for now
}

// NewServer creates a new HTTP API server
func NewServer(c client.Client, cache cache.Cache, logger zerolog.Logger) *Server {
	mux := http.NewServeMux()

	config := huma.DefaultConfig("Agent Registry API", "1.0.0")
	config.Info.Description = "Kubernetes-native agent and MCP server registry"

	api := humago.New(mux, config)

	s := &Server{
		client:        c,
		cache:         cache,
		logger:        logger,
		mux:           mux,
		api:           api,
		authEnabled:   false, // Auth disabled by default - can be enabled via env var
		allowedTokens: make(map[string]bool),
	}

	// TODO: Load allowed tokens from config/secret
	// For now, auth is disabled

	s.registerRoutes()

	return s
}

// authMiddleware checks for valid Bearer token on admin endpoints
func (s *Server) authMiddleware(ctx huma.Context, next func(huma.Context)) {
	// Skip auth if disabled
	if !s.authEnabled {
		next(ctx)
		return
	}

	// Check for Authorization header
	authHeader := ctx.Header("Authorization")
	if authHeader == "" {
		ctx.SetStatus(http.StatusUnauthorized)
		huma.WriteErr(s.api, ctx, http.StatusUnauthorized, "Missing Authorization header")
		return
	}

	// Extract Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		ctx.SetStatus(http.StatusUnauthorized)
		huma.WriteErr(s.api, ctx, http.StatusUnauthorized, "Invalid Authorization header format")
		return
	}

	token := parts[1]

	// Validate token
	if !s.allowedTokens[token] {
		s.logger.Warn().Str("token_prefix", token[:min(8, len(token))]).Msg("invalid token")
		ctx.SetStatus(http.StatusUnauthorized)
		huma.WriteErr(s.api, ctx, http.StatusUnauthorized, "Invalid token")
		return
	}

	// Token valid, continue
	next(ctx)
}

// registerRoutes registers all HTTP routes
func (s *Server) registerRoutes() {
	// Create handlers with cache access
	serverHandler := handlers.NewServerHandler(s.client, s.cache, s.logger)
	agentHandler := handlers.NewAgentHandler(s.client, s.cache, s.logger)
	skillHandler := handlers.NewSkillHandler(s.client, s.cache, s.logger)
	modelHandler := handlers.NewModelHandler(s.client, s.cache, s.logger)
	deploymentHandler := handlers.NewDeploymentHandler(s.client, s.cache, s.logger)

	// Register public API endpoints (v0)
	serverHandler.RegisterRoutes(s.api, "/v0", false)
	agentHandler.RegisterRoutes(s.api, "/v0", false)
	skillHandler.RegisterRoutes(s.api, "/v0", false)
	modelHandler.RegisterRoutes(s.api, "/v0", false)
	deploymentHandler.RegisterRoutes(s.api, "/v0", false)

	// Register admin API endpoints with auth middleware
	if s.authEnabled {
		s.api.UseMiddleware(s.authMiddleware)
	}
	serverHandler.RegisterRoutes(s.api, "/admin/v0", true)
	agentHandler.RegisterRoutes(s.api, "/admin/v0", true)
	skillHandler.RegisterRoutes(s.api, "/admin/v0", true)
	modelHandler.RegisterRoutes(s.api, "/admin/v0", true)
	deploymentHandler.RegisterRoutes(s.api, "/admin/v0", true)

	// Register admin utility endpoints
	s.registerAdminUtilityRoutes()

	// Register submit endpoint
	submitHandler := handlers.NewSubmitHandler(s.client, s.logger)
	s.mux.HandleFunc("/admin/v0/submit", submitHandler.Submit)

	// Ping endpoint for CLI compatibility
	s.mux.HandleFunc("/v0/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Health endpoint
	s.mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Ready endpoint
	s.mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}

// Stats response types
type StatsResponse struct {
	Body ServerStats
}

type ServerStats struct {
	TotalServers      int `json:"total_servers"`
	TotalServerNames  int `json:"total_server_names"`
	ActiveServers     int `json:"active_servers"`
	DeprecatedServers int `json:"deprecated_servers"`
	DeletedServers    int `json:"deleted_servers"`
	TotalAgents       int `json:"total_agents"`
	TotalSkills       int `json:"total_skills"`
}

type HealthResponse struct {
	Body HealthStatus
}

type HealthStatus struct {
	Status string `json:"status"`
}

type ImportRequest struct {
	Source         string            `json:"source"`
	Headers        map[string]string `json:"headers,omitempty"`
	Update         bool              `json:"update,omitempty"`
	SkipValidation bool              `json:"skip_validation,omitempty"`
}

type ImportInput struct {
	Body ImportRequest
}

type ImportResponse struct {
	Body ImportResult
}

type ImportResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// registerAdminUtilityRoutes registers admin utility endpoints
func (s *Server) registerAdminUtilityRoutes() {
	tags := []string{"admin", "utility"}

	// Health check
	huma.Register(s.api, huma.Operation{
		OperationID: "admin-health",
		Method:      http.MethodGet,
		Path:        "/admin/v0/health",
		Summary:     "Health check",
		Tags:        tags,
	}, func(ctx context.Context, input *struct{}) (*HealthResponse, error) {
		return &HealthResponse{
			Body: HealthStatus{Status: "healthy"},
		}, nil
	})

	// Stats
	huma.Register(s.api, huma.Operation{
		OperationID: "admin-stats",
		Method:      http.MethodGet,
		Path:        "/admin/v0/stats",
		Summary:     "Get registry statistics",
		Tags:        tags,
	}, func(ctx context.Context, input *struct{}) (*StatsResponse, error) {
		return s.getStats(ctx)
	})

	// Import
	huma.Register(s.api, huma.Operation{
		OperationID: "admin-import",
		Method:      http.MethodPost,
		Path:        "/admin/v0/import",
		Summary:     "Import from external registry",
		Tags:        tags,
	}, func(ctx context.Context, input *ImportInput) (*ImportResponse, error) {
		return s.importFromSource(ctx, input)
	})
}

func (s *Server) getStats(ctx context.Context) (*StatsResponse, error) {
	var serverList agentregistryv1alpha1.MCPServerCatalogList
	if err := s.cache.List(ctx, &serverList); err != nil {
		return nil, err
	}

	var agentList agentregistryv1alpha1.AgentCatalogList
	if err := s.cache.List(ctx, &agentList); err != nil {
		return nil, err
	}

	var skillList agentregistryv1alpha1.SkillCatalogList
	if err := s.cache.List(ctx, &skillList); err != nil {
		return nil, err
	}

	// Count stats
	stats := ServerStats{
		TotalServers: len(serverList.Items),
		TotalAgents:  len(agentList.Items),
		TotalSkills:  len(skillList.Items),
	}

	// Count unique server names and statuses
	serverNames := make(map[string]bool)
	for _, server := range serverList.Items {
		serverNames[server.Spec.Name] = true
		switch server.Status.Status {
		case agentregistryv1alpha1.CatalogStatusActive:
			stats.ActiveServers++
		case agentregistryv1alpha1.CatalogStatusDeprecated:
			stats.DeprecatedServers++
		case agentregistryv1alpha1.CatalogStatusDeleted:
			stats.DeletedServers++
		}
	}
	stats.TotalServerNames = len(serverNames)

	return &StatsResponse{Body: stats}, nil
}

func (s *Server) importFromSource(ctx context.Context, input *ImportInput) (*ImportResponse, error) {
	s.logger.Info().
		Str("source", input.Body.Source).
		Bool("update", input.Body.Update).
		Msg("import requested")

	// Fetch data from source
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, input.Body.Source, nil)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid source URL", err)
	}

	// Add custom headers if provided
	for k, v := range input.Body.Headers {
		req.Header.Set(k, v)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to fetch from source", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, huma.Error500InternalServerError(
			fmt.Sprintf("Source returned status %d: %s", resp.StatusCode, string(body)),
			nil,
		)
	}

	// Parse response - support both array of servers and object with servers field
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to read response body", err)
	}

	var servers []ExternalServerJSON
	// Try parsing as array first
	if err := json.Unmarshal(body, &servers); err != nil {
		// Try parsing as object with servers field
		var wrapper struct {
			Servers []ExternalServerJSON `json:"servers"`
		}
		if err := json.Unmarshal(body, &wrapper); err != nil {
			return nil, huma.Error400BadRequest("Failed to parse server data", err)
		}
		servers = wrapper.Servers
	}

	if len(servers) == 0 {
		return &ImportResponse{
			Body: ImportResult{
				Success: true,
				Message: "No servers found to import",
			},
		}, nil
	}

	// Import each server
	imported := 0
	updated := 0
	skipped := 0
	var errors []string

	for _, extServer := range servers {
		if extServer.Name == "" || extServer.Version == "" {
			skipped++
			continue
		}

		crName := handlers.GenerateCRName(extServer.Name, extServer.Version)

		// Check if server already exists
		existing := &agentregistryv1alpha1.MCPServerCatalog{}
		err := s.client.Get(ctx, client.ObjectKey{Name: crName}, existing)
		if err == nil {
			// Server exists
			if !input.Body.Update {
				skipped++
				continue
			}
			// Update existing server
			existing.Spec = s.convertExternalToSpec(extServer)
			if err := s.client.Update(ctx, existing); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", extServer.Name, err))
				continue
			}
			updated++
			continue
		}

		if !apierrors.IsNotFound(err) {
			errors = append(errors, fmt.Sprintf("%s: %v", extServer.Name, err))
			continue
		}

		// Create new server
		server := &agentregistryv1alpha1.MCPServerCatalog{
			ObjectMeta: metav1.ObjectMeta{
				Name: crName,
				Labels: map[string]string{
					"agentregistry.dev/name":    handlers.SanitizeK8sName(extServer.Name),
					"agentregistry.dev/version": handlers.SanitizeK8sName(extServer.Version),
				},
			},
			Spec: s.convertExternalToSpec(extServer),
		}

		if err := s.client.Create(ctx, server); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", extServer.Name, err))
			continue
		}
		imported++
	}

	// Build result message
	message := fmt.Sprintf("Imported %d, updated %d, skipped %d servers", imported, updated, skipped)
	if len(errors) > 0 {
		message += fmt.Sprintf(", %d errors", len(errors))
	}

	return &ImportResponse{
		Body: ImportResult{
			Success: len(errors) == 0,
			Message: message,
		},
	}, nil
}

// ExternalServerJSON represents a server from external registries (MCP Registry format)
type ExternalServerJSON struct {
	Name        string                  `json:"name"`
	Version     string                  `json:"version"`
	Title       string                  `json:"title,omitempty"`
	Description string                  `json:"description,omitempty"`
	WebsiteURL  string                  `json:"websiteUrl,omitempty"`
	Repository  *ExternalRepositoryJSON `json:"repository,omitempty"`
	Packages    []ExternalPackageJSON   `json:"packages,omitempty"`
	Remotes     []ExternalTransportJSON `json:"remotes,omitempty"`
}

type ExternalRepositoryJSON struct {
	URL       string `json:"url,omitempty"`
	Source    string `json:"source,omitempty"`
	ID        string `json:"id,omitempty"`
	Subfolder string `json:"subfolder,omitempty"`
}

type ExternalTransportJSON struct {
	Type    string                 `json:"type"`
	URL     string                 `json:"url,omitempty"`
	Headers []ExternalKeyValueJSON `json:"headers,omitempty"`
}

type ExternalKeyValueJSON struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Value       string `json:"value,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type ExternalPackageJSON struct {
	RegistryType         string                 `json:"registryType"`
	RegistryBaseURL      string                 `json:"registryBaseUrl,omitempty"`
	Identifier           string                 `json:"identifier"`
	Version              string                 `json:"version,omitempty"`
	FileSHA256           string                 `json:"fileSha256,omitempty"`
	RuntimeHint          string                 `json:"runtimeHint,omitempty"`
	Transport            ExternalTransportJSON  `json:"transport"`
	RuntimeArguments     []ExternalArgumentJSON `json:"runtimeArguments,omitempty"`
	PackageArguments     []ExternalArgumentJSON `json:"packageArguments,omitempty"`
	EnvironmentVariables []ExternalKeyValueJSON `json:"environmentVariables,omitempty"`
}

type ExternalArgumentJSON struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Value       string `json:"value,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Multiple    bool   `json:"multiple,omitempty"`
}

func (s *Server) convertExternalToSpec(ext ExternalServerJSON) agentregistryv1alpha1.MCPServerCatalogSpec {
	spec := agentregistryv1alpha1.MCPServerCatalogSpec{
		Name:        ext.Name,
		Version:     ext.Version,
		Title:       ext.Title,
		Description: ext.Description,
		WebsiteURL:  ext.WebsiteURL,
	}

	if ext.Repository != nil {
		spec.Repository = &agentregistryv1alpha1.Repository{
			URL:       ext.Repository.URL,
			Source:    ext.Repository.Source,
			ID:        ext.Repository.ID,
			Subfolder: ext.Repository.Subfolder,
		}
	}

	for _, p := range ext.Packages {
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
		spec.Packages = append(spec.Packages, pkg)
	}

	for _, r := range ext.Remotes {
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
		spec.Remotes = append(spec.Remotes, remote)
	}

	return spec
}

// Runnable returns a manager.Runnable that starts the HTTP server
func (s *Server) Runnable(addr string) manager.Runnable {
	return &serverRunnable{
		server: s,
		addr:   addr,
	}
}

type serverRunnable struct {
	server *Server
	addr   string
}

func (r *serverRunnable) Start(ctx context.Context) error {
	// Set up CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	httpServer := &http.Server{
		Addr:         r.addr,
		Handler:      c.Handler(r.server.mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start listening
	listener, err := net.Listen("tcp", r.addr)
	if err != nil {
		return err
	}

	r.server.logger.Info().Str("addr", r.addr).Msg("starting HTTP API server")

	// Run server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		r.server.logger.Info().Msg("shutting down HTTP API server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
