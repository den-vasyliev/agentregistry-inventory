package config

import (
	"os"
	"testing"
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


func TestIsAuthEnabled(t *testing.T) {
	// Save original value
	original := os.Getenv("AGENTREGISTRY_AUTH_ENABLED")
	defer os.Setenv("AGENTREGISTRY_AUTH_ENABLED", original)

	tests := []struct {
		name     string
		envValue string
		want     bool
	}{
		{
			name:     "auth disabled by default",
			envValue: "",
			want:     false,
		},
		{
			name:     "auth explicitly enabled",
			envValue: "true",
			want:     true,
		},
		{
			name:     "auth explicitly disabled",
			envValue: "false",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("AGENTREGISTRY_AUTH_ENABLED", tt.envValue)
			} else {
				os.Unsetenv("AGENTREGISTRY_AUTH_ENABLED")
			}

			got := IsAuthEnabled()
			if got != tt.want {
				t.Errorf("IsAuthEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
