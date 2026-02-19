package config

import (
	"os"
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

// IsAuthEnabled returns whether authentication is enabled.
// Auth is disabled by default. Set AGENTREGISTRY_AUTH_ENABLED=true to enable.
func IsAuthEnabled() bool {
	return os.Getenv("AGENTREGISTRY_AUTH_ENABLED") == "true"
}

