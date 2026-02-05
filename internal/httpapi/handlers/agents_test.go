package handlers

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

func setupAgentTestClient(t *testing.T) client.Client {
	scheme := runtime.NewScheme()
	require.NoError(t, agentregistryv1alpha1.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).Build()
}

// ---------------------------------------------------------------------------
// createAgent
// ---------------------------------------------------------------------------

func TestAgentHandler_CreateAgent(t *testing.T) {
	c := setupAgentTestClient(t)
	ctx := context.Background()
	logger := zerolog.Nop()
	handler := NewAgentHandler(c, nil, logger)

	input := &CreateAgentInput{
		Body: AgentJSON{
			Name:        "my-agent",
			Version:     "1.0.0",
			Title:       "My Agent",
			Description: "A test agent",
			Image:       "registry.io/myagent:latest",
		},
	}

	resp, err := handler.createAgent(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, "my-agent", resp.Body.Agent.Name)
	assert.Equal(t, "1.0.0", resp.Body.Agent.Version)
	assert.Equal(t, "registry.io/myagent:latest", resp.Body.Agent.Image)

	// Verify the CR was persisted
	created := &agentregistryv1alpha1.AgentCatalog{}
	err = c.Get(ctx, client.ObjectKey{Name: "my-agent-1-0-0"}, created)
	require.NoError(t, err)
	assert.Equal(t, "My Agent", created.Spec.Title)
	assert.Equal(t, "registry.io/myagent:latest", created.Spec.Image)
}

func TestAgentHandler_CreateAgent_WithRepository(t *testing.T) {
	c := setupAgentTestClient(t)
	ctx := context.Background()
	handler := NewAgentHandler(c, nil, zerolog.Nop())

	input := &CreateAgentInput{
		Body: AgentJSON{
			Name:    "repo-agent",
			Version: "2.0.0",
			Image:   "img:latest",
			Repository: &RepositoryJSON{
				URL:    "https://github.com/org/repo",
				Source: "github",
			},
		},
	}

	resp, err := handler.createAgent(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp.Body.Agent.Repository)
	assert.Equal(t, "https://github.com/org/repo", resp.Body.Agent.Repository.URL)
	assert.Equal(t, "github", resp.Body.Agent.Repository.Source)
}

func TestAgentHandler_CreateAgent_WithPackagesAndRemotes(t *testing.T) {
	c := setupAgentTestClient(t)
	ctx := context.Background()
	handler := NewAgentHandler(c, nil, zerolog.Nop())

	input := &CreateAgentInput{
		Body: AgentJSON{
			Name:    "pkg-agent",
			Version: "1.0.0",
			Image:   "img:latest",
			Packages: []AgentPackageJSON{
				{
					RegistryType: "oci",
					Identifier:   "registry.io/pkg",
					Version:      "1.0.0",
					Transport:    &AgentPackageTransportJSON{Type: "stdio"},
				},
			},
			Remotes: []TransportJSON{
				{
					Type: "streamable-http",
					URL:  "https://api.example.com/mcp",
					Headers: []KeyValueJSON{
						{Name: "Authorization", Value: "Bearer x", Required: true},
					},
				},
			},
			McpServers: []McpServerConfigJSON{
				{
					Type:    "stdio",
					Name:    "my-mcp",
					Command: "node",
					Args:    []string{"server.js"},
					Env:     []string{"PORT=3000"},
				},
			},
		},
	}

	resp, err := handler.createAgent(ctx, input)
	require.NoError(t, err)

	// Verify packages
	require.Len(t, resp.Body.Agent.Packages, 1)
	assert.Equal(t, "oci", resp.Body.Agent.Packages[0].RegistryType)
	require.NotNil(t, resp.Body.Agent.Packages[0].Transport)
	assert.Equal(t, "stdio", resp.Body.Agent.Packages[0].Transport.Type)

	// Verify remotes
	require.Len(t, resp.Body.Agent.Remotes, 1)
	assert.Equal(t, "streamable-http", resp.Body.Agent.Remotes[0].Type)
	require.Len(t, resp.Body.Agent.Remotes[0].Headers, 1)
	assert.Equal(t, "Authorization", resp.Body.Agent.Remotes[0].Headers[0].Name)

	// Verify McpServers
	require.Len(t, resp.Body.Agent.McpServers, 1)
	assert.Equal(t, "my-mcp", resp.Body.Agent.McpServers[0].Name)
	assert.Equal(t, []string{"server.js"}, resp.Body.Agent.McpServers[0].Args)
}

// ---------------------------------------------------------------------------
// convertToAgentResponse
// ---------------------------------------------------------------------------

func TestAgentHandler_ConvertToAgentResponse_DiscoveredLabel(t *testing.T) {
	c := setupAgentTestClient(t)
	handler := NewAgentHandler(c, nil, zerolog.Nop())

	agent := &agentregistryv1alpha1.AgentCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: "discovered-agent-1-0-0",
			Labels: map[string]string{
				"agentregistry.dev/discovered": "true",
			},
		},
		Spec: agentregistryv1alpha1.AgentCatalogSpec{
			Name:    "discovered-agent",
			Version: "1.0.0",
			Title:   "Discovered",
		},
	}

	resp := handler.convertToAgentResponse(agent, nil)
	assert.True(t, resp.Meta.IsDiscovered)
	assert.Equal(t, "discovery", resp.Meta.Source)
}

func TestAgentHandler_ConvertToAgentResponse_ResourceSourceLabel(t *testing.T) {
	c := setupAgentTestClient(t)
	handler := NewAgentHandler(c, nil, zerolog.Nop())

	agent := &agentregistryv1alpha1.AgentCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: "manual-agent-1-0-0",
			Labels: map[string]string{
				"agentregistry.dev/resource-source": "import",
			},
		},
		Spec: agentregistryv1alpha1.AgentCatalogSpec{
			Name:    "manual-agent",
			Version: "1.0.0",
		},
	}

	resp := handler.convertToAgentResponse(agent, nil)
	assert.False(t, resp.Meta.IsDiscovered)
	assert.Equal(t, "import", resp.Meta.Source)
}

func TestAgentHandler_ConvertToAgentResponse_WithDeploymentStatus(t *testing.T) {
	c := setupAgentTestClient(t)
	handler := NewAgentHandler(c, nil, zerolog.Nop())

	now := metav1.Now()
	agent := &agentregistryv1alpha1.AgentCatalog{
		ObjectMeta: metav1.ObjectMeta{Name: "dep-agent-1-0-0"},
		Spec: agentregistryv1alpha1.AgentCatalogSpec{
			Name:    "dep-agent",
			Version: "1.0.0",
		},
		Status: agentregistryv1alpha1.AgentCatalogStatus{
			Deployment: &agentregistryv1alpha1.DeploymentRef{
				Namespace:   "production",
				ServiceName: "dep-agent-svc",
				Ready:       true,
				Message:     "running",
				LastChecked: &now,
			},
		},
	}

	resp := handler.convertToAgentResponse(agent, nil)
	require.NotNil(t, resp.Meta.Deployment)
	assert.Equal(t, "production", resp.Meta.Deployment.Namespace)
	assert.Equal(t, "dep-agent-svc", resp.Meta.Deployment.ServiceName)
	assert.True(t, resp.Meta.Deployment.Ready)
	assert.Equal(t, "running", resp.Meta.Deployment.Message)
}

func TestAgentHandler_ConvertToAgentResponse_RegistryDeploymentOverridesStatus(t *testing.T) {
	c := setupAgentTestClient(t)
	handler := NewAgentHandler(c, nil, zerolog.Nop())

	now := metav1.Now()
	agent := &agentregistryv1alpha1.AgentCatalog{
		ObjectMeta: metav1.ObjectMeta{Name: "override-1-0-0"},
		Spec: agentregistryv1alpha1.AgentCatalogSpec{
			Name:    "override",
			Version: "1.0.0",
		},
		Status: agentregistryv1alpha1.AgentCatalogStatus{
			Deployment: &agentregistryv1alpha1.DeploymentRef{
				Namespace: "old-ns",
				Ready:     false,
			},
		},
	}

	regDeploy := &agentregistryv1alpha1.RegistryDeployment{
		ObjectMeta: metav1.ObjectMeta{Name: "override-deploy"},
		Spec: agentregistryv1alpha1.RegistryDeploymentSpec{
			Namespace: "new-ns",
		},
		Status: agentregistryv1alpha1.RegistryDeploymentStatus{
			Phase:     agentregistryv1alpha1.DeploymentPhaseRunning,
			UpdatedAt: &now,
			ManagedResources: []agentregistryv1alpha1.ManagedResource{
				{Kind: "Service", Name: "new-svc", Namespace: "new-ns"},
			},
		},
	}

	resp := handler.convertToAgentResponse(agent, regDeploy)
	require.NotNil(t, resp.Meta.Deployment)
	// RegistryDeployment takes priority
	assert.Equal(t, "new-ns", resp.Meta.Deployment.Namespace)
	assert.Equal(t, "new-svc", resp.Meta.Deployment.ServiceName)
	assert.True(t, resp.Meta.Deployment.Ready)
}
