package registry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mcpregistry "github.com/agentregistry-dev/agentregistry/internal/mcp/registryserver"
	"github.com/agentregistry-dev/agentregistry/internal/registry/api"
	v0 "github.com/agentregistry-dev/agentregistry/internal/registry/api/handlers/v0"
	"github.com/agentregistry-dev/agentregistry/internal/registry/config"
	"github.com/agentregistry-dev/agentregistry/internal/registry/telemetry"
	"github.com/agentregistry-dev/agentregistry/internal/version"
	"github.com/agentregistry-dev/agentregistry/pkg/registry/auth"
	"github.com/agentregistry-dev/agentregistry/pkg/types"
)

func App(_ context.Context, opts ...types.AppOptions) error {
	var options types.AppOptions
	if len(opts) > 0 {
		options = opts[0]
	}
	cfg := config.NewConfig()
	if err := config.Validate(cfg); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Build auth providers from options
	var jwtManager *auth.JWTManager
	if cfg.JWTPrivateKey != "" {
		jwtManager = auth.NewJWTManager(cfg)
	}

	// Resolve authn provider: use provided, or default to JWT-based if configured
	authnProvider := options.AuthnProvider
	if authnProvider == nil && jwtManager != nil {
		authnProvider = jwtManager
	}

	// Resolve authz provider: use provided, or default to public authz
	authzProvider := options.AuthzProvider
	if authzProvider == nil {
		log.Println("Using public authz provider")
		authzProvider = auth.NewPublicAuthzProvider(jwtManager)
	}

	log.Printf("Starting agentregistry %s (commit: %s)", version.Version, version.GitCommit)

	// Prepare version information
	versionInfo := &v0.VersionBody{
		Version:   version.Version,
		GitCommit: version.GitCommit,
		BuildTime: version.BuildDate,
	}

	shutdownTelemetry, metrics, err := telemetry.InitMetrics(cfg.Version)
	if err != nil {
		return fmt.Errorf("failed to initialize metrics: %v", err)
	}

	defer func() {
		if err := shutdownTelemetry(context.Background()); err != nil {
			log.Printf("Failed to shutdown telemetry: %v", err)
		}
	}()

	// TODO: Initialize Kubernetes client and use CRDs instead of database
	// For now, create a minimal server without registry service
	log.Println("Note: Using Kubernetes CRD-based architecture (no database)")

	// Initialize HTTP server
	baseServer := api.NewServer(cfg, nil, metrics, versionInfo, options.UIHandler, authnProvider)

	var server types.Server
	if options.HTTPServerFactory != nil {
		server = options.HTTPServerFactory(baseServer)
	} else {
		server = baseServer
	}

	if options.OnHTTPServerCreated != nil {
		options.OnHTTPServerCreated(server)
	}

	var mcpHTTPServer *http.Server
	if cfg.MCPPort > 0 {
		// TODO: Update MCP server to use Kubernetes CRDs
		mcpServer := mcpregistry.NewServer(nil)

		var handler http.Handler = mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
			return mcpServer
		}, &mcp.StreamableHTTPOptions{})

		// Set up authentication middleware if one is configured
		if authnProvider != nil {
			handler = mcpAuthnMiddleware(authnProvider)(handler)
		}

		addr := ":" + strconv.Itoa(int(cfg.MCPPort))
		mcpHTTPServer = &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
		}

		go func() {
			log.Printf("MCP HTTP server starting on %s", addr)
			if err := mcpHTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("Failed to start MCP server: %v", err)
				os.Exit(1)
			}
		}()
	}

	// Start server in a goroutine so it doesn't block signal handling
	go func() {
		if err := server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Failed to start server: %v", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create context with timeout for shutdown
	sctx, scancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer scancel()

	// Gracefully shutdown the server
	if err := server.Shutdown(sctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}
	if mcpHTTPServer != nil {
		if err := mcpHTTPServer.Shutdown(sctx); err != nil {
			log.Printf("MCP server forced to shutdown: %v", err)
		}
	}

	log.Println("Server exiting")
	return nil
}

// mcpAuthnMiddleware creates a middleware that uses the AuthnProvider to authenticate requests and add to session context.
func mcpAuthnMiddleware(authn auth.AuthnProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// authenticate using the configured provider
			session, err := authn.Authenticate(ctx, r.Header.Get, r.URL.Query())
			if err == nil && session != nil {
				ctx = auth.AuthSessionTo(ctx, session)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}
