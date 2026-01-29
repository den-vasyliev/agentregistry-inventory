package httpapi

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	"github.com/agentregistry-dev/agentregistry/internal/httpapi/handlers"
)

// Server is the HTTP API server that reads from the informer cache
type Server struct {
	client client.Client
	cache  cache.Cache
	logger zerolog.Logger
	mux    *http.ServeMux
	api    huma.API
}

// NewServer creates a new HTTP API server
func NewServer(c client.Client, cache cache.Cache, logger zerolog.Logger) *Server {
	mux := http.NewServeMux()

	config := huma.DefaultConfig("Agent Registry API", "1.0.0")
	config.Info.Description = "Kubernetes-native agent and MCP server registry"

	api := humago.New(mux, config)

	s := &Server{
		client: c,
		cache:  cache,
		logger: logger,
		mux:    mux,
		api:    api,
	}

	s.registerRoutes()

	return s
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

	// Register admin API endpoints
	serverHandler.RegisterRoutes(s.api, "/admin/v0", true)
	agentHandler.RegisterRoutes(s.api, "/admin/v0", true)
	skillHandler.RegisterRoutes(s.api, "/admin/v0", true)
	modelHandler.RegisterRoutes(s.api, "/admin/v0", true)
	deploymentHandler.RegisterRoutes(s.api, "/admin/v0", true)

	// Register admin utility endpoints
	s.registerAdminUtilityRoutes()

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
	// TODO: Implement actual import logic
	// For now, return a placeholder response
	s.logger.Info().
		Str("source", input.Body.Source).
		Bool("update", input.Body.Update).
		Msg("import requested")

	return &ImportResponse{
		Body: ImportResult{
			Success: false,
			Message: "Import functionality not yet implemented in CRD-based backend",
		},
	}, nil
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
