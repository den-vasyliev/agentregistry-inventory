package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRepositoryURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		owner   string
		repo    string
		host    string
		wantErr bool
	}{
		{
			name:  "github https",
			input: "https://github.com/myorg/myrepo",
			owner: "myorg",
			repo:  "myrepo",
			host:  "github.com",
		},
		{
			name:  "github https with .git",
			input: "https://github.com/myorg/myrepo.git",
			owner: "myorg",
			repo:  "myrepo",
			host:  "github.com",
		},
		{
			name:  "github https with trailing slash",
			input: "https://github.com/myorg/myrepo/",
			owner: "myorg",
			repo:  "myrepo",
			host:  "github.com",
		},
		{
			name:  "gitlab https",
			input: "https://gitlab.com/myorg/myrepo",
			owner: "myorg",
			repo:  "myrepo",
			host:  "gitlab.com",
		},
		{
			name:  "github ssh format",
			input: "git@github.com:myorg/myrepo.git",
			owner: "myorg",
			repo:  "myrepo",
			host:  "github.com",
		},
		{
			name:    "missing repo",
			input:   "https://github.com/myorg",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parseRepositoryURL(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.owner, info.Owner)
			assert.Equal(t, tt.repo, info.Repo)
			assert.Equal(t, tt.host, info.Host)
			assert.Equal(t, "main", info.Branch)
		})
	}
}

func TestValidateManifest(t *testing.T) {
	tests := []struct {
		name     string
		manifest *AgentRegistryManifest
		wantErr  string
	}{
		{
			name:     "valid mcp-server",
			manifest: &AgentRegistryManifest{Kind: "mcp-server", Name: "test", Version: "1.0.0"},
		},
		{
			name:     "valid agent",
			manifest: &AgentRegistryManifest{Kind: "agent", Name: "test", Version: "1.0.0"},
		},
		{
			name:     "valid skill",
			manifest: &AgentRegistryManifest{Kind: "skill", Name: "test", Version: "1.0.0"},
		},
		{
			name:     "missing kind",
			manifest: &AgentRegistryManifest{Name: "test", Version: "1.0.0"},
			wantErr:  "kind is required",
		},
		{
			name:     "invalid kind",
			manifest: &AgentRegistryManifest{Kind: "unknown", Name: "test", Version: "1.0.0"},
			wantErr:  "kind must be one of",
		},
		{
			name:     "missing name",
			manifest: &AgentRegistryManifest{Kind: "agent", Version: "1.0.0"},
			wantErr:  "name is required",
		},
		{
			name:     "missing version",
			manifest: &AgentRegistryManifest{Kind: "agent", Name: "test"},
			wantErr:  "version is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateManifest(tt.manifest)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSanitizeCRName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple-name", "simple-name"},
		{"Name/With/Slashes", "name-with-slashes"},
		{"name_with_underscores", "name-with-underscores"},
		{"name.with.dots", "name-with-dots"},
		{"MixedCase", "mixedcase"},
		// Truncation to 63 chars
		{
			"a-very-long-name-that-exceeds-the-kubernetes-limit-of-sixty-three-characters-total",
			"a-very-long-name-that-exceeds-the-kubernetes-limit-of-sixty-thr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeCRName(tt.input)
			assert.Equal(t, tt.want, got)
			assert.LessOrEqual(t, len(got), 63)
		})
	}
}
