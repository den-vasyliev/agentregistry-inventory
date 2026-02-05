package config

import (
	"os"
	"strconv"
	"time"
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
// Auth is enabled by default. Set AGENTREGISTRY_DISABLE_AUTH=true to disable.
func IsAuthEnabled() bool {
	return os.Getenv("AGENTREGISTRY_DISABLE_AUTH") != "true"
}


// GetOIDCIssuer returns the OIDC issuer URL for JWT validation.
func GetOIDCIssuer() string {
	return os.Getenv("AGENTREGISTRY_OIDC_ISSUER")
}

// GetOIDCAudience returns the expected audience/client ID for JWT validation.
func GetOIDCAudience() string {
	return os.Getenv("AGENTREGISTRY_OIDC_AUDIENCE")
}

// GetOIDCAdminGroup returns the required group for admin deploy actions.
func GetOIDCAdminGroup() string {
	return os.Getenv("AGENTREGISTRY_OIDC_ADMIN_GROUP")
}

// GetOIDCGroupClaim returns the claim name containing groups (default: groups).
func GetOIDCGroupClaim() string {
	if claim := os.Getenv("AGENTREGISTRY_OIDC_GROUP_CLAIM"); claim != "" {
		return claim
	}
	return "groups"
}

// GetOIDCCacheSafetyMargin returns the safety margin before token expiry (default: 5 minutes).
// Cache entries expire this duration before the actual token expires.
func GetOIDCCacheSafetyMargin() time.Duration {
	if val := os.Getenv("AGENTREGISTRY_OIDC_CACHE_MARGIN_SECONDS"); val != "" {
		if seconds, err := strconv.Atoi(val); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return 5 * time.Minute
}
