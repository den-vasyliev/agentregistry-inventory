package masteragent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSystemPrompt(t *testing.T) {
	tests := []struct {
		name        string
		override    string
		wantDefault bool
	}{
		{"empty override returns default", "", true},
		{"custom override returned", "Custom prompt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildSystemPrompt(tt.override, nil)
			if tt.wantDefault {
				assert.Contains(t, got, "Master Agent for Agent Registry")
			} else {
				assert.Equal(t, tt.override, got)
			}
		})
	}
}

func TestBuildSystemPromptWithA2AAgents(t *testing.T) {
	agents := []A2AAgentInfo{
		{
			Name:        "default/k8s-agent",
			Description: "Kubernetes management agent",
			Endpoint:    "http://kagent:8083/api/a2a/default/k8s-agent/",
			Environment: "dev",
			Cluster:     "panda",
		},
		{
			Name:        "kagent/sre-agent",
			Description: "",
			Endpoint:    "http://kagent:8083/api/a2a/kagent/sre-agent/",
			Environment: "staging",
			Cluster:     "owl",
		},
	}

	getAgents := func() []A2AAgentInfo { return agents }

	got := BuildSystemPrompt("", getAgents)
	assert.Contains(t, got, "## Available A2A Agents")
	assert.Contains(t, got, "**default/k8s-agent** (dev/panda): Kubernetes management agent")
	assert.Contains(t, got, "**kagent/sre-agent** (staging/owl): No description")
}

func TestBuildSystemPromptNoA2AAgents(t *testing.T) {
	getAgents := func() []A2AAgentInfo { return nil }
	got := BuildSystemPrompt("", getAgents)
	assert.NotContains(t, got, "Available A2A Agents")
}
