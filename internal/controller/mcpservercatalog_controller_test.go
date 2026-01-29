package controller

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

func TestMCPServerCatalogReconciler_LatestVersionTracking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Start manager to enable field indexes via the cache
	helper := SetupTestEnv(t, 60*time.Second, true)
	defer helper.Cleanup(t)

	ctx := context.Background()
	logger := zerolog.Nop()

	// Setup reconciler - must use manager's client for field index support
	// Note: Don't call SetupWithManager since we're manually calling Reconcile.
	// Setting up with manager would auto-trigger reconciliation on object changes.
	reconciler := &MCPServerCatalogReconciler{
		Client: helper.Manager.GetClient(),
		Scheme: helper.Scheme,
		Logger: logger,
	}

	var err error

	// Create test catalogs with different versions
	catalog1 := &agentregistryv1alpha1.MCPServerCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-server-v1-0-0",
		},
		Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
			Name:    "test-server",
			Version: "1.0.0",
			Title:   "Test Server",
		},
	}

	catalog2 := &agentregistryv1alpha1.MCPServerCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-server-v2-0-0",
		},
		Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
			Name:    "test-server",
			Version: "2.0.0",
			Title:   "Test Server v2",
		},
	}

	// Create first catalog
	err = helper.Client.Create(ctx, catalog1)
	require.NoError(t, err)

	// Re-fetch to get current ResourceVersion before status update
	err = helper.Client.Get(ctx, types.NamespacedName{Name: catalog1.Name}, catalog1)
	require.NoError(t, err)

	// Publish it
	catalog1.Status.Published = true
	now := metav1.Now()
	catalog1.Status.PublishedAt = &now
	err = helper.Client.Status().Update(ctx, catalog1)
	require.NoError(t, err)

	// Create second catalog (higher version)
	err = helper.Client.Create(ctx, catalog2)
	require.NoError(t, err)

	// Re-fetch to get current ResourceVersion before status update
	err = helper.Client.Get(ctx, types.NamespacedName{Name: catalog2.Name}, catalog2)
	require.NoError(t, err)

	// Publish it
	catalog2.Status.Published = true
	catalog2.Status.PublishedAt = &now
	err = helper.Client.Status().Update(ctx, catalog2)
	require.NoError(t, err)

	// Wait for cache to sync both catalogs with their Published status
	require.Eventually(t, func() bool {
		var list agentregistryv1alpha1.MCPServerCatalogList
		if err := helper.Manager.GetClient().List(ctx, &list); err != nil {
			return false
		}
		publishedCount := 0
		for _, c := range list.Items {
			if c.Status.Published {
				publishedCount++
			}
		}
		return publishedCount == 2
	}, 10*time.Second, 100*time.Millisecond, "cache should see both published catalogs")

	// Reconcile to update latest version flags
	_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: catalog1.Name}})
	require.NoError(t, err)

	// Verify v2 is marked as latest
	var updatedCatalog1, updatedCatalog2 agentregistryv1alpha1.MCPServerCatalog
	err = helper.Client.Get(ctx, types.NamespacedName{Name: catalog1.Name}, &updatedCatalog1)
	require.NoError(t, err)
	err = helper.Client.Get(ctx, types.NamespacedName{Name: catalog2.Name}, &updatedCatalog2)
	require.NoError(t, err)

	assert.False(t, updatedCatalog1.Status.IsLatest, "v1.0.0 should not be latest")
	assert.True(t, updatedCatalog2.Status.IsLatest, "v2.0.0 should be latest")
}

func TestDeploymentRefEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        *agentregistryv1alpha1.DeploymentRef
		b        *agentregistryv1alpha1.DeploymentRef
		expected bool
	}{
		{
			name:     "both nil",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "a nil",
			a:        nil,
			b:        &agentregistryv1alpha1.DeploymentRef{Namespace: "test"},
			expected: false,
		},
		{
			name:     "b nil",
			a:        &agentregistryv1alpha1.DeploymentRef{Namespace: "test"},
			b:        nil,
			expected: false,
		},
		{
			name: "equal",
			a: &agentregistryv1alpha1.DeploymentRef{
				Namespace:   "ns",
				ServiceName: "svc",
				Ready:       true,
				Message:     "ok",
			},
			b: &agentregistryv1alpha1.DeploymentRef{
				Namespace:   "ns",
				ServiceName: "svc",
				Ready:       true,
				Message:     "ok",
			},
			expected: true,
		},
		{
			name: "different ready",
			a: &agentregistryv1alpha1.DeploymentRef{
				Namespace:   "ns",
				ServiceName: "svc",
				Ready:       true,
			},
			b: &agentregistryv1alpha1.DeploymentRef{
				Namespace:   "ns",
				ServiceName: "svc",
				Ready:       false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deploymentRefEqual(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}
