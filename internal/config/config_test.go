package config

import (
	"os"
	"testing"
	"time"
)

func TestGetNamespace(t *testing.T) {
	// Save original value
	original := os.Getenv("POD_NAMESPACE")
	defer os.Setenv("POD_NAMESPACE", original)

	tests := []struct {
		name        string
		envValue    string
		wantDefault string
	}{
		{
			name:        "with POD_NAMESPACE set",
			envValue:    "custom-namespace",
			wantDefault: "custom-namespace",
		},
		{
			name:        "with POD_NAMESPACE empty",
			envValue:    "",
			wantDefault: DefaultNamespace,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("POD_NAMESPACE", tt.envValue)
			} else {
				os.Unsetenv("POD_NAMESPACE")
			}

			got := GetNamespace()
			if got != tt.wantDefault {
				t.Errorf("GetNamespace() = %v, want %v", got, tt.wantDefault)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue string
		want         string
	}{
		{
			name:         "env set",
			key:          "TEST_KEY",
			value:        "test-value",
			defaultValue: "default",
			want:         "test-value",
		},
		{
			name:         "env not set",
			key:          "NONEXISTENT_KEY_XYZ",
			value:        "",
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "env empty",
			key:          "EMPTY_KEY",
			value:        "",
			defaultValue: "default",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			got := GetEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetEnv(%q, %q) = %v, want %v", tt.key, tt.defaultValue, got, tt.want)
			}
		})
	}
}

func TestGetOIDCIssuer(t *testing.T) {
	original := os.Getenv("AGENTREGISTRY_OIDC_ISSUER")
	defer os.Setenv("AGENTREGISTRY_OIDC_ISSUER", original)

	os.Setenv("AGENTREGISTRY_OIDC_ISSUER", "https://auth.example.com")
	assert(t, "https://auth.example.com", GetOIDCIssuer())

	os.Unsetenv("AGENTREGISTRY_OIDC_ISSUER")
	assert(t, "", GetOIDCIssuer())
}

func TestGetOIDCAudience(t *testing.T) {
	original := os.Getenv("AGENTREGISTRY_OIDC_AUDIENCE")
	defer os.Setenv("AGENTREGISTRY_OIDC_AUDIENCE", original)

	os.Setenv("AGENTREGISTRY_OIDC_AUDIENCE", "my-client-id")
	assert(t, "my-client-id", GetOIDCAudience())

	os.Unsetenv("AGENTREGISTRY_OIDC_AUDIENCE")
	assert(t, "", GetOIDCAudience())
}

func TestGetOIDCAdminGroup(t *testing.T) {
	original := os.Getenv("AGENTREGISTRY_OIDC_ADMIN_GROUP")
	defer os.Setenv("AGENTREGISTRY_OIDC_ADMIN_GROUP", original)

	os.Setenv("AGENTREGISTRY_OIDC_ADMIN_GROUP", "registry-admins")
	assert(t, "registry-admins", GetOIDCAdminGroup())

	os.Unsetenv("AGENTREGISTRY_OIDC_ADMIN_GROUP")
	assert(t, "", GetOIDCAdminGroup())
}

func TestGetOIDCGroupClaim(t *testing.T) {
	original := os.Getenv("AGENTREGISTRY_OIDC_GROUP_CLAIM")
	defer os.Setenv("AGENTREGISTRY_OIDC_GROUP_CLAIM", original)

	// default when unset
	os.Unsetenv("AGENTREGISTRY_OIDC_GROUP_CLAIM")
	assert(t, "groups", GetOIDCGroupClaim())

	// custom value
	os.Setenv("AGENTREGISTRY_OIDC_GROUP_CLAIM", "cognito:groups")
	assert(t, "cognito:groups", GetOIDCGroupClaim())
}

func TestGetOIDCCacheSafetyMargin(t *testing.T) {
	original := os.Getenv("AGENTREGISTRY_OIDC_CACHE_MARGIN_SECONDS")
	defer os.Setenv("AGENTREGISTRY_OIDC_CACHE_MARGIN_SECONDS", original)

	// default when unset
	os.Unsetenv("AGENTREGISTRY_OIDC_CACHE_MARGIN_SECONDS")
	if got := GetOIDCCacheSafetyMargin(); got != 5*time.Minute {
		t.Errorf("GetOIDCCacheSafetyMargin() default = %v, want 5m0s", got)
	}

	// explicit value
	os.Setenv("AGENTREGISTRY_OIDC_CACHE_MARGIN_SECONDS", "30")
	if got := GetOIDCCacheSafetyMargin(); got != 30*time.Second {
		t.Errorf("GetOIDCCacheSafetyMargin() = %v, want 30s", got)
	}

	// non-numeric falls back to default
	os.Setenv("AGENTREGISTRY_OIDC_CACHE_MARGIN_SECONDS", "not-a-number")
	if got := GetOIDCCacheSafetyMargin(); got != 5*time.Minute {
		t.Errorf("GetOIDCCacheSafetyMargin() non-numeric = %v, want 5m0s", got)
	}
}

// assert is a tiny helper to avoid importing testify in this file.
func assert(t *testing.T, want, got string) {
	t.Helper()
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIsAuthEnabled(t *testing.T) {
	// Save original value
	original := os.Getenv("AGENTREGISTRY_DISABLE_AUTH")
	defer os.Setenv("AGENTREGISTRY_DISABLE_AUTH", original)

	tests := []struct {
		name     string
		envValue string
		want     bool
	}{
		{
			name:     "auth enabled by default",
			envValue: "",
			want:     true,
		},
		{
			name:     "auth disabled",
			envValue: "true",
			want:     false,
		},
		{
			name:     "auth explicitly enabled",
			envValue: "false",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("AGENTREGISTRY_DISABLE_AUTH", tt.envValue)
			} else {
				os.Unsetenv("AGENTREGISTRY_DISABLE_AUTH")
			}

			got := IsAuthEnabled()
			if got != tt.want {
				t.Errorf("IsAuthEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
