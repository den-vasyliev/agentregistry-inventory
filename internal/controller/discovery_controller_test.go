package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateCatalogName(t *testing.T) {
	tests := []struct {
		namespace string
		name      string
		expected  string
	}{
		{"default", "my-server", "default-my-server"},
		{"kagent", "mcp-server-atlassian", "kagent-mcp-server-atlassian"},
		{"ns_with_underscore", "name_with_underscore", "ns-with-underscore-name-with-underscore"},
		{"NS", "NAME", "ns-name"},
		{"a", "very-long-name-that-exceeds-the-kubernetes-resource-name-limit-of-63-characters", "a-very-long-name-that-exceeds-the-kubernetes-resource-name-limi"},
	}

	for _, tt := range tests {
		t.Run(tt.namespace+"/"+tt.name, func(t *testing.T) {
			result := generateCatalogName(tt.namespace, tt.name)
			assert.Equal(t, tt.expected, result)
			assert.LessOrEqual(t, len(result), 63, "Result should be at most 63 characters")
		})
	}
}

func TestGenerateAgentCatalogName(t *testing.T) {
	tests := []struct {
		namespace string
		name      string
		expected  string
	}{
		{"kagent", "helm-agent", "kagent-helm-agent"},
		{"default", "my-agent", "default-my-agent"},
	}

	for _, tt := range tests {
		t.Run(tt.namespace+"/"+tt.name, func(t *testing.T) {
			result := generateAgentCatalogName(tt.namespace, tt.name)
			assert.Equal(t, tt.expected, result)
			assert.LessOrEqual(t, len(result), 63)
		})
	}
}

func TestGenerateModelCatalogName(t *testing.T) {
	tests := []struct {
		namespace string
		name      string
		expected  string
	}{
		{"kagent", "default-model-config", "kagent-default-model-config"},
		{"default", "openai-config", "default-openai-config"},
	}

	for _, tt := range tests {
		t.Run(tt.namespace+"/"+tt.name, func(t *testing.T) {
			result := generateModelCatalogName(tt.namespace, tt.name)
			assert.Equal(t, tt.expected, result)
			assert.LessOrEqual(t, len(result), 63)
		})
	}
}

func TestParseSkillRef(t *testing.T) {
	tests := []struct {
		ref         string
		wantName    string
		wantVersion string
	}{
		{"ghcr.io/user/skill:v1.0.0", "ghcr.io/user/skill", "v1.0.0"},
		{"ghcr.io/user/skill:latest", "ghcr.io/user/skill", "latest"},
		{"ghcr.io/user/skill", "ghcr.io/user/skill", "latest"},
		{"ghcr.io/user/skill@sha256:abc123", "ghcr.io/user/skill", "sha256:abc123"},
		{"localhost:5000/skill:v1", "localhost:5000/skill", "v1"},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			name, version := parseSkillRef(tt.ref)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantVersion, version)
		})
	}
}

func TestGenerateSkillCatalogName(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{"ghcr.io/antonbabenko/terraform-skill", "v1.0.0", "antonbabenko-terraform-skill-v1-0-0"},
		{"user/skill", "latest", "user-skill-latest"},
		{"simple-skill", "1.0", "simple-skill-1-0"},
	}

	for _, tt := range tests {
		t.Run(tt.name+":"+tt.version, func(t *testing.T) {
			result := generateSkillCatalogName(tt.name, tt.version)
			assert.Equal(t, tt.expected, result)
			assert.LessOrEqual(t, len(result), 63)
		})
	}
}
