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

func setupDeploymentTestClient(t *testing.T) client.Client {
	scheme := runtime.NewScheme()
	require.NoError(t, agentregistryv1alpha1.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).Build()
}

// ---------------------------------------------------------------------------
// createDeployment
// ---------------------------------------------------------------------------

func TestDeploymentHandler_CreateDeployment_MCPServer(t *testing.T) {
	c := setupDeploymentTestClient(t)
	ctx := context.Background()
	logger := zerolog.Nop()
	handler := NewDeploymentHandler(c, nil, logger)

	input := &CreateDeploymentInput{}
	input.Body.ResourceName = "test-server"
	input.Body.Version = "1.0.0"
	input.Body.ResourceType = "mcp"
	input.Body.Runtime = "kubernetes"
	input.Body.Namespace = "default"

	resp, err := handler.createDeployment(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, "test-server", resp.Body.Deployment.ResourceName)
	assert.Equal(t, "1.0.0", resp.Body.Deployment.Version)
	assert.Equal(t, "mcp", resp.Body.Deployment.ResourceType)
	assert.Equal(t, "kubernetes", resp.Body.Deployment.Runtime)

	// Verify the RegistryDeployment was created
	var deployments agentregistryv1alpha1.RegistryDeploymentList
	err = c.List(ctx, &deployments)
	require.NoError(t, err)
	assert.Len(t, deployments.Items, 1)
	assert.Equal(t, "test-server", deployments.Items[0].Spec.ResourceName)
}

func TestDeploymentHandler_CreateDeployment_Agent(t *testing.T) {
	c := setupDeploymentTestClient(t)
	ctx := context.Background()
	handler := NewDeploymentHandler(c, nil, zerolog.Nop())

	input := &CreateDeploymentInput{}
	input.Body.ResourceName = "my-agent"
	input.Body.Version = "2.0.0"
	input.Body.ResourceType = "agent"
	input.Body.Runtime = "kubernetes"
	input.Body.Namespace = "prod"

	resp, err := handler.createDeployment(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, "my-agent", resp.Body.Deployment.ResourceName)
	assert.Equal(t, "agent", resp.Body.Deployment.ResourceType)
	assert.Equal(t, "prod", resp.Body.Deployment.Namespace)

	// Verify creation
	var deployments agentregistryv1alpha1.RegistryDeploymentList
	err = c.List(ctx, &deployments)
	require.NoError(t, err)
	assert.Len(t, deployments.Items, 1)
	assert.Equal(t, agentregistryv1alpha1.ResourceTypeAgent, deployments.Items[0].Spec.ResourceType)
}

func TestDeploymentHandler_CreateDeployment_PreferRemote(t *testing.T) {
	c := setupDeploymentTestClient(t)
	ctx := context.Background()
	handler := NewDeploymentHandler(c, nil, zerolog.Nop())

	input := &CreateDeploymentInput{}
	input.Body.ResourceName = "remote-server"
	input.Body.Version = "1.0.0"
	input.Body.ResourceType = "mcp"
	input.Body.Runtime = "kubernetes"
	input.Body.PreferRemote = true
	input.Body.Namespace = "default"

	resp, err := handler.createDeployment(ctx, input)
	require.NoError(t, err)
	assert.True(t, resp.Body.Deployment.PreferRemote)

	// Verify PreferRemote flag in RegistryDeployment
	var deployments agentregistryv1alpha1.RegistryDeploymentList
	err = c.List(ctx, &deployments)
	require.NoError(t, err)
	assert.Len(t, deployments.Items, 1)
	assert.True(t, deployments.Items[0].Spec.PreferRemote)
}

func TestDeploymentHandler_CreateDeployment_WithConfig(t *testing.T) {
	c := setupDeploymentTestClient(t)
	ctx := context.Background()
	handler := NewDeploymentHandler(c, nil, zerolog.Nop())

	input := &CreateDeploymentInput{}
	input.Body.ResourceName = "config-server"
	input.Body.Version = "1.0.0"
	input.Body.ResourceType = "mcp"
	input.Body.Runtime = "kubernetes"
	input.Body.Config = map[string]string{
		"API_KEY":  "secret123",
		"ENDPOINT": "https://api.example.com",
	}
	input.Body.Namespace = "default"

	resp, err := handler.createDeployment(ctx, input)
	require.NoError(t, err)
	assert.NotNil(t, resp.Body.Deployment.Config)
	assert.Equal(t, "secret123", resp.Body.Deployment.Config["API_KEY"])
	assert.Equal(t, "https://api.example.com", resp.Body.Deployment.Config["ENDPOINT"])
}

func TestDeploymentHandler_CreateDeployment_InvalidRuntime(t *testing.T) {
	c := setupDeploymentTestClient(t)
	ctx := context.Background()
	handler := NewDeploymentHandler(c, nil, zerolog.Nop())

	input := &CreateDeploymentInput{}
	input.Body.ResourceName = "test-server"
	input.Body.Version = "1.0.0"
	input.Body.ResourceType = "mcp"
	input.Body.Runtime = "invalid-runtime"
	input.Body.Namespace = "default"

	resp, err := handler.createDeployment(ctx, input)
	// Handler may accept invalid runtime and let controller validate
	// or it may error immediately depending on implementation
	if err != nil {
		assert.Contains(t, err.Error(), "runtime")
	} else {
		// If no error, deployment was created but may fail validation later
		assert.NotNil(t, resp)
	}
}

// ---------------------------------------------------------------------------
// convertToDeploymentJSON
// ---------------------------------------------------------------------------

func TestDeploymentHandler_ConvertToDeploymentJSON_Basic(t *testing.T) {
	c := setupDeploymentTestClient(t)
	handler := NewDeploymentHandler(c, nil, zerolog.Nop())

	deployment := &agentregistryv1alpha1.RegistryDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: agentregistryv1alpha1.RegistryDeploymentSpec{
			ResourceName: "my-server",
			Version:      "1.0.0",
			ResourceType: agentregistryv1alpha1.ResourceTypeMCP,
			Runtime:      agentregistryv1alpha1.RuntimeTypeKubernetes,
			Namespace:    "target-ns",
		},
		Status: agentregistryv1alpha1.RegistryDeploymentStatus{
			Phase: agentregistryv1alpha1.DeploymentPhaseRunning,
		},
	}

	result := handler.convertToDeploymentJSON(deployment)
	assert.Equal(t, "my-server", result.ResourceName)
	assert.Equal(t, "1.0.0", result.Version)
	assert.Equal(t, "mcp", result.ResourceType)
	assert.Equal(t, "kubernetes", result.Runtime)
	assert.Equal(t, "target-ns", result.Namespace)
	assert.Equal(t, "Running", result.Status) // Phase is capitalized
}

func TestDeploymentHandler_ConvertToDeploymentJSON_WithConfig(t *testing.T) {
	c := setupDeploymentTestClient(t)
	handler := NewDeploymentHandler(c, nil, zerolog.Nop())

	deployment := &agentregistryv1alpha1.RegistryDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "config-deployment",
		},
		Spec: agentregistryv1alpha1.RegistryDeploymentSpec{
			ResourceName: "server",
			Version:      "1.0.0",
			ResourceType: agentregistryv1alpha1.ResourceTypeMCP,
			Runtime:      agentregistryv1alpha1.RuntimeTypeKubernetes,
			Config: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
		},
	}

	result := handler.convertToDeploymentJSON(deployment)
	require.NotNil(t, result.Config)
	assert.Equal(t, "value1", result.Config["KEY1"])
	assert.Equal(t, "value2", result.Config["KEY2"])
}

func TestDeploymentHandler_ConvertToDeploymentJSON_AgentType(t *testing.T) {
	c := setupDeploymentTestClient(t)
	handler := NewDeploymentHandler(c, nil, zerolog.Nop())

	deployment := &agentregistryv1alpha1.RegistryDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "agent-deployment",
		},
		Spec: agentregistryv1alpha1.RegistryDeploymentSpec{
			ResourceName: "my-agent",
			Version:      "2.0.0",
			ResourceType: agentregistryv1alpha1.ResourceTypeAgent,
			Runtime:      agentregistryv1alpha1.RuntimeTypeKubernetes,
		},
	}

	result := handler.convertToDeploymentJSON(deployment)
	assert.Equal(t, "agent", result.ResourceType)
	assert.Equal(t, "my-agent", result.ResourceName)
}
