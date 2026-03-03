package mcp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/agentregistry-dev/agentregistry/internal/config"
	"github.com/agentregistry-dev/agentregistry/internal/version"
)

// MCPServer wraps the mcp-go server with registry-specific tools, resources, and prompts.
type MCPServer struct {
	client        client.Client
	cache         cache.Cache
	logger        zerolog.Logger
	authEnabled   bool
	allowedTokens map[string]bool
	mcpServer     *server.MCPServer
	httpServer    *server.StreamableHTTPServer
}

// ServerOption is a functional option for configuring the MCP server
type ServerOption func(*MCPServer)

// NewMCPServer creates a new MCP server with all registry tools, resources, and prompts registered.
func NewMCPServer(c client.Client, cache cache.Cache, logger zerolog.Logger, authEnabled bool, opts ...ServerOption) *MCPServer {
	s := &MCPServer{
		client:        c,
		cache:         cache,
		logger:        logger.With().Str("component", "mcp").Logger(),
		authEnabled:   authEnabled,
		allowedTokens: make(map[string]bool),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	mcpServer := server.NewMCPServer(
		"agentregistry",
		version.Version,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithInstructions("Agent Registry MCP Server - browse and manage MCP servers, agents, skills, models, and deployments in your Kubernetes cluster."),
	)
	mcpServer.EnableSampling()

	s.mcpServer = mcpServer

	s.registerTools()
	s.registerResources()
	s.registerPrompts()

	s.httpServer = server.NewStreamableHTTPServer(mcpServer)

	return s
}

// Handler returns an http.Handler for the MCP server, wrapped with auth middleware.
func (s *MCPServer) Handler() http.Handler {
	return s.authMiddleware(s.httpServer)
}

// Runnable returns a manager.Runnable that starts the MCP server on its own port.
func (s *MCPServer) Runnable(addr string) manager.Runnable {
	return manager.RunnableFunc(func(ctx context.Context) error {
		// Load tokens now that the cache is started
		s.loadTokensFromSecret()

		srv := &http.Server{
			Addr:              addr,
			Handler:           s.authMiddleware(s.httpServer),
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      0, // Disabled: streaming responses and sampling callbacks need long-lived connections
			ReadHeaderTimeout: 10 * time.Second,
		}

		listener, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}

		s.logger.Info().Str("addr", addr).Msg("starting MCP server")

		errCh := make(chan error, 1)
		go func() {
			if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		}()

		select {
		case <-ctx.Done():
			s.logger.Info().Msg("shutting down MCP server")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			return srv.Shutdown(shutdownCtx)
		case err := <-errCh:
			return err
		}
	})
}

// loadTokensFromSecret loads API tokens from the Kubernetes Secret "agentregistry-api-tokens".
// Each key in the secret data is treated as a valid token. This is the same secret used by the HTTP API.
func (s *MCPServer) loadTokensFromSecret() {
	namespace := config.GetNamespace()

	secret := &corev1.Secret{}
	err := s.client.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      "agentregistry-api-tokens",
	}, secret)

	if err != nil {
		if apierrors.IsNotFound(err) {
			s.logger.Warn().Msg("Secret 'agentregistry-api-tokens' not found - MCP server will reject all authenticated requests")
		} else {
			s.logger.Error().Err(err).Msg("failed to read API tokens secret for MCP server")
		}
		return
	}

	for name, tokenBytes := range secret.Data {
		token := strings.TrimSpace(string(tokenBytes))
		if token != "" {
			s.allowedTokens[token] = true
			s.logger.Debug().Str("name", name).Msg("loaded MCP API token")
		}
	}

	s.logger.Info().Int("count", len(s.allowedTokens)).Msg("loaded MCP API tokens from secret")
}

// authMiddleware wraps an http.Handler with Bearer token authentication.
// When auth is disabled, requests pass through unchanged.
func (s *MCPServer) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.authEnabled {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Missing Authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Invalid Authorization header format"}`, http.StatusUnauthorized)
			return
		}

		if !s.allowedTokens[parts[1]] {
			s.logger.Warn().Str("token_prefix", parts[1][:min(8, len(parts[1]))]).Msg("invalid MCP token")
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Invalid token"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requestSampling is a helper to call sampling on the MCP server from tool handlers.
// Returns an error if the client does not support sampling.
func (s *MCPServer) requestSampling(ctx context.Context, systemPrompt string, userMessage string) (string, error) {
	// NOTE: We intentionally do NOT check GetClientCapabilities().Sampling here.
	//
	// mcp-go has a race condition: a concurrent GET request can register a new session
	// object before the initialize POST stores the one that received SetClientCapabilities,
	// so the session found during tools/call may show Sampling==nil even for clients that
	// correctly declared sampling during initialize. Checking here would incorrectly reject
	// those clients.
	//
	// Instead we rely on the detached context timeout below as the safety net: if no GET
	// stream is open to deliver the sampling request, RequestSampling will fail fast or
	// time out after 5 minutes rather than hanging indefinitely.
	session := server.ClientSessionFromContext(ctx)
	if session == nil {
		return "", fmt.Errorf("no active session")
	}

	// Use a detached context with a generous timeout for the sampling call.
	// The tools/call POST connection may be cancelled or time out on the client side
	// before the sampling round-trip (LLM inference) completes. Using the request ctx
	// directly would cause the pending sampling request to be cleaned up, resulting in
	// a 500 when the client POSTs the sampling response back. A detached context keeps
	// the pending request alive independently of the calling HTTP request's lifecycle.
	samplingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := s.mcpServer.RequestSampling(samplingCtx, mcp.CreateMessageRequest{
		CreateMessageParams: mcp.CreateMessageParams{
			Messages: []mcp.SamplingMessage{
				{
					Role:    mcp.RoleUser,
					Content: mcp.TextContent{Type: "text", Text: userMessage},
				},
			},
			SystemPrompt: systemPrompt,
			MaxTokens:    4096,
		},
	})
	if err != nil {
		return "", err
	}

	if textContent, ok := result.Content.(mcp.TextContent); ok {
		return textContent.Text, nil
	}
	return "Sampling returned non-text content", nil
}
