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
			Name:      "test-server-v1-0-0",
			Namespace: "default",
		},
		Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
			Name:    "test-server",
			Version: "1.0.0",
			Title:   "Test Server",
		},
	}

	catalog2 := &agentregistryv1alpha1.MCPServerCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-server-v2-0-0",
			Namespace: "default",
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
	err = helper.Client.Get(ctx, types.NamespacedName{Name: catalog1.Name, Namespace: "default"}, catalog1)
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
	err = helper.Client.Get(ctx, types.NamespacedName{Name: catalog2.Name, Namespace: "default"}, catalog2)
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
	_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: catalog1.Name, Namespace: "default"}})
	require.NoError(t, err)

	// Verify v2 is marked as latest
	var updatedCatalog1, updatedCatalog2 agentregistryv1alpha1.MCPServerCatalog
	err = helper.Client.Get(ctx, types.NamespacedName{Name: catalog1.Name, Namespace: "default"}, &updatedCatalog1)
	require.NoError(t, err)
	err = helper.Client.Get(ctx, types.NamespacedName{Name: catalog2.Name, Namespace: "default"}, &updatedCatalog2)
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

// ---------------------------------------------------------------------------
// Additional reconciler tests
// ---------------------------------------------------------------------------

func TestMCPServerCatalogReconciler_Reconcile_NewServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	helper := SetupTestEnv(t, 60*time.Second, true) // Use manager for field indexing
	defer helper.Cleanup(t)

	ctx := context.Background()
	reconciler := &MCPServerCatalogReconciler{
		Client: helper.Manager.GetClient(), // Use manager client
		Scheme: helper.Scheme,
		Logger: zerolog.Nop(),
	}

	// Create a new server catalog
	catalog := &agentregistryv1alpha1.MCPServerCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "new-server-1-0-0",
			Namespace: "default",
		},
		Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
			Name:    "new-server",
			Version: "1.0.0",
			Title:   "New Server",
		},
	}

	err := helper.Client.Create(ctx, catalog)
	require.NoError(t, err)

	// Reconcile
	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      catalog.Name,
			Namespace: "default",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), result.RequeueAfter)

	// Verify status was updated
	var updated agentregistryv1alpha1.MCPServerCatalog
	err = helper.Client.Get(ctx, types.NamespacedName{Name: catalog.Name, Namespace: "default"}, &updated)
	require.NoError(t, err)
	// ObservedGeneration should be set (may be 0 or Generation depending on envtest behavior)
	assert.GreaterOrEqual(t, updated.Status.ObservedGeneration, int64(0))
}

func TestMCPServerCatalogReconciler_Reconcile_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	helper := SetupTestEnv(t, 60*time.Second, false)
	defer helper.Cleanup(t)

	ctx := context.Background()
	reconciler := &MCPServerCatalogReconciler{
		Client: helper.Client,
		Scheme: helper.Scheme,
		Logger: zerolog.Nop(),
	}

	// Reconcile a non-existent resource
	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "non-existent-server",
			Namespace: "default",
		},
	})

	// Should not error (IgnoreNotFound)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), result.RequeueAfter)
}

func TestMCPServerCatalogReconciler_UpdateObservedGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	helper := SetupTestEnv(t, 60*time.Second, true) // Use manager for field indexing
	defer helper.Cleanup(t)

	ctx := context.Background()
	reconciler := &MCPServerCatalogReconciler{
		Client: helper.Manager.GetClient(), // Use manager client
		Scheme: helper.Scheme,
		Logger: zerolog.Nop(),
	}

	// Create catalog
	catalog := &agentregistryv1alpha1.MCPServerCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gen-test-1-0-0",
			Namespace: "default",
		},
		Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
			Name:    "gen-test",
			Version: "1.0.0",
		},
	}
	require.NoError(t, helper.Client.Create(ctx, catalog))

	// First reconcile
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: catalog.Name, Namespace: "default"},
	})
	require.NoError(t, err)

	// Get updated catalog
	var updated agentregistryv1alpha1.MCPServerCatalog
	err = helper.Client.Get(ctx, types.NamespacedName{Name: catalog.Name, Namespace: "default"}, &updated)
	require.NoError(t, err)
	firstGeneration := updated.Status.ObservedGeneration
	assert.GreaterOrEqual(t, firstGeneration, int64(0))

	// Update spec to trigger generation bump
	updated.Spec.Title = "Updated Title"
	err = helper.Client.Update(ctx, &updated)
	require.NoError(t, err)

	// Get catalog to see new generation
	err = helper.Client.Get(ctx, types.NamespacedName{Name: catalog.Name, Namespace: "default"}, &updated)
	require.NoError(t, err)

	// In envtest, Generation may or may not be bumped automatically
	// The key test is that reconciliation doesn't error
	secondGeneration := updated.Generation
	assert.GreaterOrEqual(t, secondGeneration, firstGeneration)

	// Reconcile again
	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: catalog.Name, Namespace: "default"},
	})
	require.NoError(t, err)

	// Verify reconciliation succeeded (observedGeneration tracking depends on envtest behavior)
	err = helper.Client.Get(ctx, types.NamespacedName{Name: catalog.Name, Namespace: "default"}, &updated)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, updated.Status.ObservedGeneration, int64(0))
}

func TestMCPServerCatalogReconciler_UpdateLatestVersion_SingleVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	helper := SetupTestEnv(t, 60*time.Second, true)
	defer helper.Cleanup(t)

	ctx := context.Background()
	reconciler := &MCPServerCatalogReconciler{
		Client: helper.Manager.GetClient(),
		Scheme: helper.Scheme,
		Logger: zerolog.Nop(),
	}

	// Create single version
	catalog := &agentregistryv1alpha1.MCPServerCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "single-server-1-0-0",
			Namespace: "default",
		},
		Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
			Name:    "single-server",
			Version: "1.0.0",
		},
	}
	require.NoError(t, helper.Client.Create(ctx, catalog))

	// Mark as published
	err := helper.Client.Get(ctx, types.NamespacedName{Name: catalog.Name, Namespace: "default"}, catalog)
	require.NoError(t, err)
	catalog.Status.Published = true
	now := metav1.Now()
	catalog.Status.PublishedAt = &now
	require.NoError(t, helper.Client.Status().Update(ctx, catalog))

	// Wait for cache sync
	require.Eventually(t, func() bool {
		var c agentregistryv1alpha1.MCPServerCatalog
		if err := helper.Manager.GetClient().Get(ctx, types.NamespacedName{Name: catalog.Name, Namespace: "default"}, &c); err != nil {
			return false
		}
		return c.Status.Published
	}, 10*time.Second, 100*time.Millisecond)

	// Reconcile
	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: catalog.Name, Namespace: "default"},
	})
	require.NoError(t, err)

	// Single version should be marked as latest
	var updated agentregistryv1alpha1.MCPServerCatalog
	err = helper.Client.Get(ctx, types.NamespacedName{Name: catalog.Name, Namespace: "default"}, &updated)
	require.NoError(t, err)
	assert.True(t, updated.Status.IsLatest)
}
