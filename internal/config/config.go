package config

import (
	"os"
	"strings"
)

const (
	// DefaultNamespace is the default namespace for Agent Registry resources
	DefaultNamespace = "agentregistry"

	// DefaultHTTPPort is the default port for the HTTP API
	DefaultHTTPPort = ":8080"

	// DefaultMetricsPort is the default port for metrics
	DefaultMetricsPort = ":8081"

	// DefaultHealthPort is the default port for health probes
	DefaultHealthPort = ":8082"
)

// GetNamespace returns the namespace to use for Agent Registry resources.
// It checks the POD_NAMESPACE environment variable first, then falls back to DefaultNamespace.
func GetNamespace() string {
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}
	return DefaultNamespace
}

// GetEnv returns the value of an environment variable or a default value.
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// AllowedDeploymentNamespaces returns the set of namespaces that catalog items
// may be deployed into, as a lookup map.
//
// By default only the controller namespace (GetNamespace) is allowed, so a
// caller cannot schedule workloads into arbitrary namespaces using the
// controller's cluster-wide RBAC. Operators can widen this with a
// comma-separated AGENTREGISTRY_ALLOWED_DEPLOY_NAMESPACES env var.
func AllowedDeploymentNamespaces() map[string]bool {
	allowed := map[string]bool{GetNamespace(): true}
	if raw := os.Getenv("AGENTREGISTRY_ALLOWED_DEPLOY_NAMESPACES"); raw != "" {
		for ns := range strings.SplitSeq(raw, ",") {
			if ns = strings.TrimSpace(ns); ns != "" {
				allowed[ns] = true
			}
		}
	}
	return allowed
}

// IsDeploymentNamespaceAllowed reports whether ns is permitted as a deployment
// target. An empty ns is treated as the default (controller) namespace and is
// always allowed.
func IsDeploymentNamespaceAllowed(ns string) bool {
	if ns == "" {
		return true
	}
	return AllowedDeploymentNamespaces()[ns]
}

// IsAuthEnabled returns whether the optional Bearer-token auth is enabled for
// the MCP server and reflected in the UI auth-config flag.
//
// NOTE: This does NOT gate the HTTP /admin/* API. Admin routes are always
// authenticated and fail-closed (see internal/httpapi authMiddleware),
// independent of this flag. Set AGENTREGISTRY_AUTH_ENABLED=true to also require
// Bearer tokens on the MCP endpoints. The Helm chart sets this by default
// (disableAuth: false).
func IsAuthEnabled() bool {
	return os.Getenv("AGENTREGISTRY_AUTH_ENABLED") == "true"
}
