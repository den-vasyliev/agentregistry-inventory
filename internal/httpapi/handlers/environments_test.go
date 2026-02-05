package handlers

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

func TestEnvironmentHandler_ListEnvironments_Empty(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, agentregistryv1alpha1.AddToScheme(scheme))
	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	handler := NewEnvironmentHandler(c, nil, zerolog.Nop())

	resp, err := handler.listEnvironments(context.Background())
	require.NoError(t, err)
	assert.Empty(t, resp.Body.Environments)
	assert.Equal(t, 0, resp.Body.Metadata.Count)
}

func TestEnvironmentHandler_ListEnvironments_WithDiscoveryConfig(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, agentregistryv1alpha1.AddToScheme(scheme))

	dc := &agentregistryv1alpha1.DiscoveryConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-discovery",
			Namespace: "agentregistry",
		},
		Spec: agentregistryv1alpha1.DiscoveryConfigSpec{
			Environments: []agentregistryv1alpha1.Environment{
				{
					Name: "production",
					Cluster: agentregistryv1alpha1.ClusterConfig{
						Namespace: "prod-ns",
					},
					Labels: map[string]string{"env": "prod"},
				},
				{
					Name: "staging",
					Cluster: agentregistryv1alpha1.ClusterConfig{
						Namespace: "staging-ns",
					},
				},
			},
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(dc).
		Build()

	handler := NewEnvironmentHandler(c, nil, zerolog.Nop())

	resp, err := handler.listEnvironments(context.Background())
	require.NoError(t, err)
	assert.Len(t, resp.Body.Environments, 2)
	assert.Equal(t, 2, resp.Body.Metadata.Count)

	assert.Equal(t, "production", resp.Body.Environments[0].Name)
	assert.Equal(t, "prod-ns", resp.Body.Environments[0].Namespace)
	assert.Equal(t, "prod", resp.Body.Environments[0].Labels["env"])

	assert.Equal(t, "staging", resp.Body.Environments[1].Name)
	assert.Equal(t, "staging-ns", resp.Body.Environments[1].Namespace)
}

func TestEnvironmentHandler_ListEnvironments_FallbackToNamespaces(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, agentregistryv1alpha1.AddToScheme(scheme))

	dc := &agentregistryv1alpha1.DiscoveryConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-discovery",
			Namespace: "agentregistry",
		},
		Spec: agentregistryv1alpha1.DiscoveryConfigSpec{
			Environments: []agentregistryv1alpha1.Environment{
				{
					Name:    "dev",
					Cluster: agentregistryv1alpha1.ClusterConfig{
						// No Namespace set
					},
					Namespaces: []string{"dev-ns", "dev-ns-2"},
				},
			},
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(dc).
		Build()

	handler := NewEnvironmentHandler(c, nil, zerolog.Nop())

	resp, err := handler.listEnvironments(context.Background())
	require.NoError(t, err)
	require.Len(t, resp.Body.Environments, 1)
	// Should fall back to first namespace
	assert.Equal(t, "dev-ns", resp.Body.Environments[0].Namespace)
}
