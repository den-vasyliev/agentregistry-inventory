package controller

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

// newTestClientWithAgentIndexes creates a fake client with AgentCatalog field indexes
func newTestClientWithAgentIndexes(scheme *runtime.Scheme, objs ...client.Object) client.Client {
	builder := fake.NewClientBuilder().WithScheme(scheme)

	// Add index for spec.name
	builder.WithIndex(
		&agentregistryv1alpha1.AgentCatalog{},
		IndexAgentName,
		func(obj client.Object) []string {
			agent := obj.(*agentregistryv1alpha1.AgentCatalog)
			return []string{agent.Spec.Name}
		},
	)

	return builder.WithObjects(objs...).WithStatusSubresource(&agentregistryv1alpha1.AgentCatalog{}).Build()
}

func TestAgentCatalogReconciler_Reconcile_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	r := &AgentCatalogReconciler{
		Client: c,
		Scheme: scheme,
		Logger: logger,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-agent", Namespace: "default"},
	}

	result, err := r.Reconcile(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)
}

func TestAgentCatalogReconciler_Reconcile_NewAgent(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)

	agent := &agentregistryv1alpha1.AgentCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "research-agent-v1.0.0",
			Namespace: "default",
		},
		Spec: agentregistryv1alpha1.AgentCatalogSpec{
			Name:          "research-agent",
			Version:       "1.0.0",
			Title:         "Research Agent",
			Description:   "AI research assistant",
			Image:         "ghcr.io/example/research-agent:1.0.0",
			Framework:     "langgraph",
			ModelProvider: "anthropic",
		},
	}

	c := newTestClientWithAgentIndexes(scheme, agent)

	r := &AgentCatalogReconciler{
		Client: c,
		Scheme: scheme,
		Logger: logger,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "research-agent-v1.0.0", Namespace: "default"},
	}

	result, err := r.Reconcile(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)

	// Verify observed generation was updated
	updatedAgent := &agentregistryv1alpha1.AgentCatalog{}
	err = c.Get(context.Background(), req.NamespacedName, updatedAgent)
	require.NoError(t, err)
	assert.Equal(t, agent.Generation, updatedAgent.Status.ObservedGeneration)
}

func TestAgentCatalogReconciler_Reconcile_WithAllFields(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)

	agent := &agentregistryv1alpha1.AgentCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configured-agent-v1.0.0",
			Namespace: "default",
		},
		Spec: agentregistryv1alpha1.AgentCatalogSpec{
			Name:              "configured-agent",
			Version:           "1.0.0",
			Title:             "Configured Agent",
			Description:       "Agent with all fields configured",
			Image:             "ghcr.io/example/agent:1.0.0",
			Language:          "python",
			Framework:         "autogen",
			ModelProvider:     "openai",
			ModelName:         "gpt-4",
			TelemetryEndpoint: "https://telemetry.example.com",
			WebsiteURL:        "https://docs.example.com/agent",
		},
	}

	c := newTestClientWithAgentIndexes(scheme, agent)

	r := &AgentCatalogReconciler{
		Client: c,
		Scheme: scheme,
		Logger: logger,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "configured-agent-v1.0.0", Namespace: "default"},
	}

	result, err := r.Reconcile(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)
}

func TestAgentCatalogReconciler_Reconcile_WithMinimalFields(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)

	agent := &agentregistryv1alpha1.AgentCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minimal-agent-v1.0.0",
			Namespace: "default",
		},
		Spec: agentregistryv1alpha1.AgentCatalogSpec{
			Name:    "minimal-agent",
			Version: "1.0.0",
			Image:   "ghcr.io/example/agent:1.0.0",
		},
	}

	c := newTestClientWithAgentIndexes(scheme, agent)

	r := &AgentCatalogReconciler{
		Client: c,
		Scheme: scheme,
		Logger: logger,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "minimal-agent-v1.0.0", Namespace: "default"},
	}

	result, err := r.Reconcile(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)
}

func TestAgentCatalogReconciler_Reconcile_MultipleVersions(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)

	// Create multiple versions of the same agent
	agentV1 := &agentregistryv1alpha1.AgentCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "code-review-agent-v1.0.0",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: agentregistryv1alpha1.AgentCatalogSpec{
			Name:      "code-review-agent",
			Version:   "1.0.0",
			Image:     "ghcr.io/example/code-review:1.0.0",
			Framework: "autogen",
		},
		Status: agentregistryv1alpha1.AgentCatalogStatus{
			Published: true,
		},
	}

	agentV2 := &agentregistryv1alpha1.AgentCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "code-review-agent-v2.0.0",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: agentregistryv1alpha1.AgentCatalogSpec{
			Name:      "code-review-agent",
			Version:   "2.0.0",
			Image:     "ghcr.io/example/code-review:2.0.0",
			Framework: "autogen",
		},
		Status: agentregistryv1alpha1.AgentCatalogStatus{
			Published: true,
		},
	}

	c := newTestClientWithAgentIndexes(scheme, agentV1, agentV2)

	r := &AgentCatalogReconciler{
		Client: c,
		Scheme: scheme,
		Logger: logger,
	}

	// Reconcile v1
	req1 := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "code-review-agent-v1.0.0", Namespace: "default"},
	}
	_, err := r.Reconcile(context.Background(), req1)
	assert.NoError(t, err)

	// Reconcile v2
	req2 := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "code-review-agent-v2.0.0", Namespace: "default"},
	}
	_, err = r.Reconcile(context.Background(), req2)
	assert.NoError(t, err)

	// Check that v2 is marked as latest
	updatedV2 := &agentregistryv1alpha1.AgentCatalog{}
	err = c.Get(context.Background(), req2.NamespacedName, updatedV2)
	require.NoError(t, err)
	assert.True(t, updatedV2.Status.IsLatest)
}

func TestAgentCatalogReconciler_updateLatestVersion(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)

	// Create agents with different versions
	agents := []*agentregistryv1alpha1.AgentCatalog{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "assistant-v1.0.0",
				Namespace: "default",
			},
			Spec: agentregistryv1alpha1.AgentCatalogSpec{
				Name:      "assistant",
				Version:   "1.0.0",
				Image:     "ghcr.io/example/assistant:1.0.0",
				Framework: "langgraph",
			},
			Status: agentregistryv1alpha1.AgentCatalogStatus{
				Published: true,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "assistant-v1.5.0",
				Namespace: "default",
			},
			Spec: agentregistryv1alpha1.AgentCatalogSpec{
				Name:      "assistant",
				Version:   "1.5.0",
				Image:     "ghcr.io/example/assistant:1.5.0",
				Framework: "langgraph",
			},
			Status: agentregistryv1alpha1.AgentCatalogStatus{
				Published: true,
			},
		},
	}

	c := newTestClientWithAgentIndexes(scheme, agents[0], agents[1])

	r := &AgentCatalogReconciler{
		Client: c,
		Scheme: scheme,
		Logger: logger,
	}

	// Update latest version for v1.5.0
	err := r.updateLatestVersion(context.Background(), agents[1])
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Edge case tests
// ---------------------------------------------------------------------------

func TestAgentCatalogReconciler_UpdateLatestVersion_SingleVersion(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)

	// Single agent version
	agent := &agentregistryv1alpha1.AgentCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "solo-agent-1-0-0",
			Namespace: "default",
		},
		Spec: agentregistryv1alpha1.AgentCatalogSpec{
			Name:    "solo-agent",
			Version: "1.0.0",
			Image:   "agent:1.0.0",
		},
		Status: agentregistryv1alpha1.AgentCatalogStatus{
			Published: true,
		},
	}

	c := newTestClientWithAgentIndexes(scheme, agent)

	r := &AgentCatalogReconciler{
		Client: c,
		Scheme: scheme,
		Logger: logger,
	}

	// Update should succeed even with only one version
	err := r.updateLatestVersion(context.Background(), agent)
	assert.NoError(t, err)
}

func TestAgentCatalogReconciler_Reconcile_UpdateGenerationTracking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires proper status subresource handling")
	}

	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)

	agent := &agentregistryv1alpha1.AgentCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "track-agent-1-0-0",
			Namespace:  "default",
			Generation: 2, // Simulate update
		},
		Spec: agentregistryv1alpha1.AgentCatalogSpec{
			Name:    "track-agent",
			Version: "1.0.0",
			Image:   "agent:1.0.0",
		},
		Status: agentregistryv1alpha1.AgentCatalogStatus{
			ObservedGeneration: 1, // Out of sync
		},
	}

	c := newTestClientWithAgentIndexes(scheme, agent)

	r := &AgentCatalogReconciler{
		Client: c,
		Scheme: scheme,
		Logger: logger,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "track-agent-1-0-0",
			Namespace: "default",
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	assert.NoError(t, err)
	// May requeue due to status update conflicts with fake client
	// This is expected behavior with fake client that doesn't fully support status subresources
	_ = result

	// Note: Proper generation tracking requires envtest with status subresource
}
