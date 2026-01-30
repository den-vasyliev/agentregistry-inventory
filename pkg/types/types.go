package types

import (
	"context"
	"net/http"

	"github.com/agentregistry-dev/agentregistry/pkg/registry/auth"
	"github.com/danielgtaylor/huma/v2"
)

// AppOptions contains configuration for the registry app.
// All fields are optional and allow external developers to extend functionality.
type AppOptions struct {
	// HTTPServerFactory is an optional function to create a server that adds new API routes.
	HTTPServerFactory HTTPServerFactory

	// OnHTTPServerCreated is an optional callback that receives the created server
	// (potentially extended via HTTPServerFactory).
	OnHTTPServerCreated func(Server)

	// UIHandler is an optional HTTP handler for serving a custom UI at the root path ("/").
	// If provided, this handler will be used instead of the default redirect to docs.
	// API routes will still take precedence over the UI handler.
	UIHandler http.Handler

	// AuthnProvider is an optional authentication provider.
	AuthnProvider auth.AuthnProvider

	// AuthzProvider is an optional authorization provider.
	AuthzProvider auth.AuthzProvider
}

// Server represents the HTTP server and provides access to the Huma API
// and HTTP mux for registering new routes and handlers.
//
// This interface allows external packages to extend the server functionality
// by adding new endpoints without accessing internal implementation details.
type Server interface {
	// HumaAPI returns the Huma API instance, allowing registration of new routes
	// that will appear in the OpenAPI documentation.
	HumaAPI() huma.API

	// Mux returns the HTTP ServeMux, allowing registration of custom HTTP handlers
	Mux() *http.ServeMux

	// Start begins listening for incoming HTTP requests
	Start() error

	// Shutdown gracefully shuts down the server
	Shutdown(ctx context.Context) error
}

// CLIAuthnProvider provides authentication for CLI commands.
// External libraries can implement this to support different auth mechanisms
type CLIAuthnProvider interface {
	// Authenticate returns credentials for API calls.
	Authenticate(ctx context.Context) (token string, err error)
}

// HTTPServerFactory is a function type that creates a server implementation that
// adds new API routes and handlers.
//
// The factory receives a Server interface and should return a Server after
// registering new routes using base.HumaAPI() or base.Mux().
type HTTPServerFactory func(base Server) Server
