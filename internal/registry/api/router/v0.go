// Package router contains API routing logic
package router

import (
	"github.com/danielgtaylor/huma/v2"

	v0 "github.com/agentregistry-dev/agentregistry/internal/registry/api/handlers/v0"
	"github.com/agentregistry-dev/agentregistry/internal/registry/config"
	"github.com/agentregistry-dev/agentregistry/internal/registry/telemetry"
)

// RegisterRoutes registers all API routes (public and admin) for all versions
// This is the single entry point for all route registration
// TODO: Re-implement with Kubernetes client to query Application CRDs
func RegisterRoutes(
	api huma.API,
	cfg *config.Config,
	_ interface{}, // registry service - not used, will use K8s client
	metrics *telemetry.Metrics,
	versionInfo *v0.VersionBody,
) {
	// Public API endpoints
	registerPublicRoutes(api, "/v0", cfg, metrics, versionInfo)
	registerPublicRoutes(api, "/v0.1", cfg, metrics, versionInfo)

	// Admin API endpoints
	registerAdminRoutes(api, "/admin/v0", cfg, metrics, versionInfo)
	registerAdminRoutes(api, "/admin/v0.1", cfg, metrics, versionInfo)
}

// registerPublicRoutes registers public API routes for a version
func registerPublicRoutes(
	api huma.API,
	pathPrefix string,
	cfg *config.Config,
	metrics *telemetry.Metrics,
	versionInfo *v0.VersionBody,
) {
	// Only basic endpoints for now
	// TODO: Add endpoints that query Kubernetes Application CRDs
	registerCommonEndpoints(api, pathPrefix, cfg, metrics, versionInfo)
}

// registerAdminRoutes registers admin API routes for a version
func registerAdminRoutes(
	api huma.API,
	pathPrefix string,
	cfg *config.Config,
	metrics *telemetry.Metrics,
	versionInfo *v0.VersionBody,
) {
	// Only basic endpoints for now
	// TODO: Add admin endpoints that query Kubernetes Application CRDs
	registerCommonEndpoints(api, pathPrefix, cfg, metrics, versionInfo)
}

// registerCommonEndpoints registers endpoints that are common to both public and admin routes
func registerCommonEndpoints(
	api huma.API,
	pathPrefix string,
	cfg *config.Config,
	metrics *telemetry.Metrics,
	versionInfo *v0.VersionBody,
) {
	v0.RegisterHealthEndpoint(api, pathPrefix, cfg, metrics)
	v0.RegisterPingEndpoint(api, pathPrefix)
	v0.RegisterVersionEndpoint(api, pathPrefix, versionInfo)
}
