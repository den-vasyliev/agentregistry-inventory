package controller

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	kmcpv1alpha1 "github.com/kagent-dev/kmcp/api/v1alpha1"
)

// testClientWithWatch wraps a client.Client to implement client.WithWatch
type testClientWithWatch struct {
	client.Client
}

func (c *testClientWithWatch) Watch(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) (watch.Interface, error) {
	// For testing, return a fake watcher that does nothing
	return newFakeWatcher(), nil
}

// fakeWatcher implements watch.Interface for testing
type fakeWatcher struct {
	resultChan chan watch.Event
	stopOnce   sync.Once
}

func newFakeWatcher() *fakeWatcher {
	return &fakeWatcher{
		resultChan: make(chan watch.Event),
	}
}

func (w *fakeWatcher) Stop() {
	w.stopOnce.Do(func() {
		close(w.resultChan)
	})
}

func (w *fakeWatcher) ResultChan() <-chan watch.Event {
	return w.resultChan
}

func TestDiscoveryConfigReconciler_MultiNamespaceDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment with manager started
	helper := SetupTestEnv(t, 60*time.Second, true)
	defer helper.Cleanup(t)

	ctx := helper.Ctx
	logger := zerolog.Nop()

	// Create namespaces
	ns1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ed210",
		},
	}
	ns2 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
	}

	err := helper.Client.Create(ctx, ns1)
	require.NoError(t, err)
	err = helper.Client.Create(ctx, ns2)
	if err != nil && err.Error() != "namespaces \"default\" already exists" {
		require.NoError(t, err)
	}

	// Create MCPServers in different namespaces
	mcpServer1 := &kmcpv1alpha1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "filesystem-server",
			Namespace: "ed210",
			Labels: map[string]string{
				"app.kubernetes.io/version": "1.0.0",
			},
			Annotations: map[string]string{
				"kmcp.dev/project-name": "Filesystem MCP Server",
				"kmcp.dev/description":  "Provides filesystem access",
			},
		},
		Spec: kmcpv1alpha1.MCPServerSpec{
			TransportType: "stdio",
			Deployment: kmcpv1alpha1.MCPServerDeployment{
				Image: "ghcr.io/modelcontextprotocol/servers/filesystem:latest",
			},
		},
		Status: kmcpv1alpha1.MCPServerStatus{
			Conditions: []metav1.Condition{
				{
					Type:    "Ready",
					Status:  metav1.ConditionTrue,
					Message: "Server is ready",
				},
			},
		},
	}

	mcpServer2 := &kmcpv1alpha1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "github-server",
			Namespace: "default",
			Labels: map[string]string{
				"app.kubernetes.io/version": "2.0.0",
			},
		},
		Spec: kmcpv1alpha1.MCPServerSpec{
			TransportType: "http",
			Deployment: kmcpv1alpha1.MCPServerDeployment{
				Image: "ghcr.io/modelcontextprotocol/servers/github:latest",
			},
		},
	}

	err = helper.Client.Create(ctx, mcpServer1)
	require.NoError(t, err)
	err = helper.Client.Create(ctx, mcpServer2)
	require.NoError(t, err)

	// Create a DiscoveryConfig
	discoveryConfig := &agentregistryv1alpha1.DiscoveryConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "multi-namespace-discovery",
			Namespace: "default",
		},
		Spec: agentregistryv1alpha1.DiscoveryConfigSpec{
			Environments: []agentregistryv1alpha1.Environment{
				{
					Name: "dev",
					Cluster: agentregistryv1alpha1.ClusterConfig{
						Name:      "dev",
						Namespace: "ed210",
					},
					Namespaces:    []string{"ed210", "default"},
					ResourceTypes: []string{"MCPServer"},
					Labels: map[string]string{
						"environment": "dev",
						"managed-by":  "agentregistry",
					},
				},
			},
		},
	}

	err = helper.Client.Create(ctx, discoveryConfig)
	require.NoError(t, err)

	// Setup reconciler with manager
	reconciler := &DiscoveryConfigReconciler{
		Client:  helper.Client,
		Scheme:  helper.Scheme,
		Logger:  logger,
		Manager: helper.Manager,
	}

	// Inject remote client factory for testing
	// Use envtest's client wrapped to implement WithWatch
	oldFactory := RemoteClientFactory
	RemoteClientFactory = func(env *agentregistryv1alpha1.Environment, scheme *runtime.Scheme) (client.WithWatch, error) {
		return &testClientWithWatch{Client: helper.Client}, nil
	}
	defer func() { RemoteClientFactory = oldFactory }()

	// Reconcile to set up informers
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "multi-namespace-discovery",
			Namespace: "default",
		},
	}

	result, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	assert.Empty(t, result.RequeueAfter, "Informer-based reconciler should not requeue")

	// Give informers time to sync and trigger handlers
	time.Sleep(2 * time.Second)

	// Check that catalog entries were created
	var catalogs agentregistryv1alpha1.MCPServerCatalogList
	err = helper.Client.List(ctx, &catalogs, client.InNamespace("default"))
	require.NoError(t, err)

	// Should have 2 catalog entries
	assert.Len(t, catalogs.Items, 2, "Expected 2 catalog entries to be created")

	// Verify catalog entries have correct labels
	for _, catalog := range catalogs.Items {
		assert.Equal(t, "true", catalog.Labels[discoveryLabel])
		assert.Equal(t, "MCPServer", catalog.Labels[sourceKindLabel])
		assert.Equal(t, "dev", catalog.Labels["agentregistry.dev/environment"])
		assert.Equal(t, "dev", catalog.Labels["agentregistry.dev/cluster"])
		assert.Equal(t, "dev", catalog.Labels["environment"])
		assert.Equal(t, "agentregistry", catalog.Labels["managed-by"])
	}

	// Verify specific catalog entries
	foundFilesystem := false
	foundGithub := false

	for _, catalog := range catalogs.Items {
		sourceName := catalog.Labels[sourceNameLabel]
		sourceNS := catalog.Labels[sourceNSLabel]

		if sourceName == "filesystem-server" && sourceNS == "ed210" {
			foundFilesystem = true
			assert.Equal(t, "Filesystem MCP Server", catalog.Spec.Title)
			assert.Equal(t, "Provides filesystem access", catalog.Spec.Description)
			assert.Equal(t, "1.0.0", catalog.Spec.Version)
			assert.Equal(t, "ed210/filesystem-server", catalog.Spec.Name)
			assert.Equal(t, "stdio", catalog.Spec.Packages[0].Transport.Type)
			assert.Contains(t, catalog.Name, "ed210-filesystem-server")
		}

		if sourceName == "github-server" && sourceNS == "default" {
			foundGithub = true
			assert.Equal(t, "2.0.0", catalog.Spec.Version)
			assert.Equal(t, "streamable-http", catalog.Spec.Packages[0].Transport.Type)
			assert.Contains(t, catalog.Name, "default-github-server")
		}
	}

	assert.True(t, foundFilesystem, "Filesystem server catalog entry should exist")
	assert.True(t, foundGithub, "Github server catalog entry should exist")

	// Check DiscoveryConfig status
	var updatedConfig agentregistryv1alpha1.DiscoveryConfig
	err = helper.Client.Get(ctx, req.NamespacedName, &updatedConfig)
	require.NoError(t, err)

	assert.NotNil(t, updatedConfig.Status.LastSyncTime)
	assert.Len(t, updatedConfig.Status.Conditions, 1)
	assert.Equal(t, metav1.ConditionTrue, updatedConfig.Status.Conditions[0].Status)
	assert.Equal(t, "Ready", updatedConfig.Status.Conditions[0].Type)
}

func TestDiscoveryConfigReconciler_MultipleEnvironments(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	helper := SetupTestEnv(t, 60*time.Second, true)
	defer helper.Cleanup(t)

	ctx := helper.Ctx
	logger := zerolog.Nop()

	// Create namespaces for different "environments"
	devNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dev-ns",
		},
	}
	stagingNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "staging-ns",
		},
	}
	prodNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "prod-ns",
		},
	}

	err := helper.Client.Create(ctx, devNS)
	require.NoError(t, err)
	err = helper.Client.Create(ctx, stagingNS)
	require.NoError(t, err)
	err = helper.Client.Create(ctx, prodNS)
	require.NoError(t, err)

	// Create MCPServers for different "environments"
	devServer := &kmcpv1alpha1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dev-server",
			Namespace: "dev-ns",
		},
		Spec: kmcpv1alpha1.MCPServerSpec{
			TransportType: "stdio",
			Deployment: kmcpv1alpha1.MCPServerDeployment{
				Image: "dev-image:latest",
			},
		},
	}

	stagingServer := &kmcpv1alpha1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "staging-server",
			Namespace: "staging-ns",
		},
		Spec: kmcpv1alpha1.MCPServerSpec{
			TransportType: "stdio",
			Deployment: kmcpv1alpha1.MCPServerDeployment{
				Image: "staging-image:latest",
			},
		},
	}

	prodServer := &kmcpv1alpha1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prod-server",
			Namespace: "prod-ns",
		},
		Spec: kmcpv1alpha1.MCPServerSpec{
			TransportType: "stdio",
			Deployment: kmcpv1alpha1.MCPServerDeployment{
				Image: "prod-image:latest",
			},
		},
	}

	err = helper.Client.Create(ctx, devServer)
	require.NoError(t, err)
	err = helper.Client.Create(ctx, stagingServer)
	require.NoError(t, err)
	err = helper.Client.Create(ctx, prodServer)
	require.NoError(t, err)

	// Create a DiscoveryConfig with multiple environments
	discoveryConfig := &agentregistryv1alpha1.DiscoveryConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "multi-env-discovery",
			Namespace: "default",
		},
		Spec: agentregistryv1alpha1.DiscoveryConfigSpec{
			Environments: []agentregistryv1alpha1.Environment{
				{
					Name: "dev",
					Cluster: agentregistryv1alpha1.ClusterConfig{
						Name:      "dev-cluster",
						Namespace: "dev-ns",
					},
					Namespaces:    []string{"dev-ns"},
					ResourceTypes: []string{"MCPServer"},
					Labels: map[string]string{
						"environment": "dev",
					},
				},
				{
					Name: "staging",
					Cluster: agentregistryv1alpha1.ClusterConfig{
						Name:      "staging-cluster",
						Namespace: "staging-ns",
					},
					Namespaces:    []string{"staging-ns"},
					ResourceTypes: []string{"MCPServer"},
					Labels: map[string]string{
						"environment": "staging",
					},
				},
				{
					Name: "prod",
					Cluster: agentregistryv1alpha1.ClusterConfig{
						Name:      "prod-cluster",
						Namespace: "prod-ns",
					},
					Namespaces:    []string{"prod-ns"},
					ResourceTypes: []string{"MCPServer"},
					Labels: map[string]string{
						"environment": "prod",
					},
				},
			},
		},
	}

	err = helper.Client.Create(ctx, discoveryConfig)
	require.NoError(t, err)

	// Setup reconciler
	reconciler := &DiscoveryConfigReconciler{
		Client:  helper.Client,
		Scheme:  helper.Scheme,
		Logger:  logger,
		Manager: helper.Manager,
	}

	// Inject remote client factory for testing
	oldFactory := RemoteClientFactory
	RemoteClientFactory = func(env *agentregistryv1alpha1.Environment, scheme *runtime.Scheme) (client.WithWatch, error) {
		return &testClientWithWatch{Client: helper.Client}, nil
	}
	defer func() { RemoteClientFactory = oldFactory }()

	// Reconcile
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "multi-env-discovery",
			Namespace: "default",
		},
	}

	result, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	assert.Empty(t, result.RequeueAfter)

	// Give informers time to sync and trigger handlers
	time.Sleep(2 * time.Second)

	// Check that catalog entries were created for all environments
	var catalogs agentregistryv1alpha1.MCPServerCatalogList
	err = helper.Client.List(ctx, &catalogs, client.InNamespace("default"))
	require.NoError(t, err)

	// Should have 3 catalog entries
	assert.Len(t, catalogs.Items, 3, "Expected 3 catalog entries for 3 environments")

	// Verify each environment has correct labels
	envLabels := map[string]bool{
		"dev":     false,
		"staging": false,
		"prod":    false,
	}

	for _, catalog := range catalogs.Items {
		env := catalog.Labels["agentregistry.dev/environment"]
		envLabels[env] = true
		assert.Equal(t, env, catalog.Labels["environment"])
	}

	assert.True(t, envLabels["dev"], "Dev catalog should exist")
	assert.True(t, envLabels["staging"], "Staging catalog should exist")
	assert.True(t, envLabels["prod"], "Prod catalog should exist")

	// Check DiscoveryConfig status
	var updatedConfig agentregistryv1alpha1.DiscoveryConfig
	err = helper.Client.Get(ctx, req.NamespacedName, &updatedConfig)
	require.NoError(t, err)

	assert.NotNil(t, updatedConfig.Status.LastSyncTime)
	assert.Len(t, updatedConfig.Status.Conditions, 1)
	assert.Equal(t, metav1.ConditionTrue, updatedConfig.Status.Conditions[0].Status)
}

func TestDiscoveryConfigReconciler_InformerLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	helper := SetupTestEnv(t, 60*time.Second, true)
	defer helper.Cleanup(t)

	ctx := helper.Ctx
	logger := zerolog.Nop()

	// Create namespace
	testNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ns",
		},
	}
	err := helper.Client.Create(ctx, testNS)
	require.NoError(t, err)

	// Create MCPServer
	mcpServer := &kmcpv1alpha1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-server",
			Namespace: "test-ns",
		},
		Spec: kmcpv1alpha1.MCPServerSpec{
			TransportType: "stdio",
			Deployment: kmcpv1alpha1.MCPServerDeployment{
				Image: "test-image:latest",
			},
		},
	}
	err = helper.Client.Create(ctx, mcpServer)
	require.NoError(t, err)

	// Create DiscoveryConfig
	discoveryConfig := &agentregistryv1alpha1.DiscoveryConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lifecycle-test",
			Namespace: "default",
		},
		Spec: agentregistryv1alpha1.DiscoveryConfigSpec{
			Environments: []agentregistryv1alpha1.Environment{
				{
					Name: "test-env",
					Cluster: agentregistryv1alpha1.ClusterConfig{
						Name:      "test-cluster",
						Namespace: "test-ns",
					},
					Namespaces:    []string{"test-ns"},
					ResourceTypes: []string{"MCPServer"},
				},
			},
		},
	}
	err = helper.Client.Create(ctx, discoveryConfig)
	require.NoError(t, err)

	// Setup reconciler
	reconciler := &DiscoveryConfigReconciler{
		Client:  helper.Client,
		Scheme:  helper.Scheme,
		Logger:  logger,
		Manager: helper.Manager,
	}

	// Inject remote client factory
	oldFactory := RemoteClientFactory
	RemoteClientFactory = func(env *agentregistryv1alpha1.Environment, scheme *runtime.Scheme) (client.WithWatch, error) {
		return &testClientWithWatch{Client: helper.Client}, nil
	}
	defer func() { RemoteClientFactory = oldFactory }()

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "lifecycle-test",
			Namespace: "default",
		},
	}

	// First reconcile - should start informer
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Verify informer was created
	reconciler.informersMu.RLock()
	assert.Contains(t, reconciler.informers, "lifecycle-test/test-env/test-ns/MCPServer")
	reconciler.informersMu.RUnlock()

	// Second reconcile - should skip existing informer
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Delete the DiscoveryConfig
	err = helper.Client.Delete(ctx, discoveryConfig)
	require.NoError(t, err)

	// Reconcile after deletion - should stop informers
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Verify informers were stopped
	reconciler.informersMu.RLock()
	assert.Empty(t, reconciler.informers)
	reconciler.informersMu.RUnlock()
}
