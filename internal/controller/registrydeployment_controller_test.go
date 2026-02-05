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
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	kagentv1alpha2 "github.com/kagent-dev/kagent/go/api/v1alpha2"
	kmcpv1alpha1 "github.com/kagent-dev/kmcp/api/v1alpha1"
)

func TestRegistryDeploymentReconciler_Reconcile_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	r := &RegistryDeploymentReconciler{
		Client: c,
		Scheme: scheme,
		Logger: logger,
	}

	// Request for non-existent resource
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-deployment", Namespace: "default"},
	}

	result, err := r.Reconcile(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)
}

func TestRegistryDeploymentReconciler_Reconcile_Creation(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)
	_ = kagentv1alpha2.AddToScheme(scheme)
	_ = kmcpv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)

	// Create deployment with resourceVersion to simulate real object
	deployment := &agentregistryv1alpha1.RegistryDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-mcp-deployment",
			Namespace:       "default",
			ResourceVersion: "1",
		},
		Spec: agentregistryv1alpha1.RegistryDeploymentSpec{
			ResourceName: "test-server",
			Version:      "1.0.0",
			ResourceType: agentregistryv1alpha1.ResourceTypeMCP,
			Runtime:      agentregistryv1alpha1.RuntimeTypeKubernetes,
			Namespace:    "target-ns",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(deployment).
		WithStatusSubresource(&agentregistryv1alpha1.RegistryDeployment{}).
		Build()

	r := &RegistryDeploymentReconciler{
		Client: c,
		Scheme: scheme,
		Logger: logger,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-mcp-deployment", Namespace: "default"},
	}

	// First reconcile - should add finalizer
	_, err := r.Reconcile(context.Background(), req)
	// May return error due to missing catalog entry, but finalizer should be added first
	// Re-fetch to get updated object
	updatedDeployment := &agentregistryv1alpha1.RegistryDeployment{}
	err = c.Get(context.Background(), req.NamespacedName, updatedDeployment)
	require.NoError(t, err)

	// Verify finalizer was added
	assert.Contains(t, updatedDeployment.Finalizers, finalizerName)
}

func TestRegistryDeploymentReconciler_handleDeletion(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)
	_ = kagentv1alpha2.AddToScheme(scheme)
	_ = kmcpv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)

	now := metav1.Now()
	// Create deployment with deletion timestamp
	deployment := &agentregistryv1alpha1.RegistryDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-deployment",
			Namespace:         "default",
			DeletionTimestamp: &now,
			Finalizers:        []string{finalizerName},
		},
		Spec: agentregistryv1alpha1.RegistryDeploymentSpec{
			ResourceName: "test-server",
			Version:      "1.0.0",
			ResourceType: agentregistryv1alpha1.ResourceTypeMCP,
			Runtime:      agentregistryv1alpha1.RuntimeTypeKubernetes,
			Namespace:    "target-ns",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(deployment).
		Build()

	r := &RegistryDeploymentReconciler{
		Client: c,
		Scheme: scheme,
		Logger: logger,
	}

	result, err := r.handleDeletion(context.Background(), deployment)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)
}

func TestParseURLComponents(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantHost string
		wantPort uint32
		wantPath string
	}{
		{
			name:     "https URL with port",
			url:      "https://api.example.com:8443/v1/path",
			wantHost: "api.example.com",
			wantPort: 8443,
			wantPath: "/v1/path",
		},
		{
			name:     "http URL with port",
			url:      "http://api.example.com:8080/path",
			wantHost: "api.example.com",
			wantPort: 8080,
			wantPath: "/path",
		},
		{
			name:     "URL without port (default)",
			url:      "http://api.example.com/path",
			wantHost: "api.example.com",
			wantPort: 80,
			wantPath: "/path",
		},
		{
			name:     "empty URL",
			url:      "",
			wantHost: "",
			wantPort: 0,
			wantPath: "/",
		},
		{
			name:     "https without port defaults to 443",
			url:      "https://api.example.com/secure",
			wantHost: "api.example.com",
			wantPort: 443,
			wantPath: "/secure",
		},
		{
			name:     "host only no path",
			url:      "http://localhost:3000",
			wantHost: "localhost",
			wantPort: 3000,
			wantPath: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, path := parseURLComponents(tt.url)
			assert.Equal(t, tt.wantHost, host)
			assert.Equal(t, tt.wantPort, port)
			assert.Equal(t, tt.wantPath, path)
		})
	}
}

func TestGetImageAndCommand(t *testing.T) {
	tests := []struct {
		name         string
		registryType string
		runtimeHint  string
		wantImage    string
		wantCmd      string
	}{
		{
			name:         "npm with npx",
			registryType: "npm",
			runtimeHint:  "npx",
			wantImage:    "node:20-alpine",
			wantCmd:      "npx",
		},
		{
			name:         "npm default",
			registryType: "npm",
			runtimeHint:  "",
			wantImage:    "node:20-alpine",
			wantCmd:      "npm",
		},
		{
			name:         "pypi with uvx",
			registryType: "pypi",
			runtimeHint:  "uvx",
			wantImage:    "ghcr.io/astral-sh/uv:latest",
			wantCmd:      "uvx",
		},
		{
			name:         "pypi default",
			registryType: "pypi",
			runtimeHint:  "",
			wantImage:    "python:3.12-slim",
			wantCmd:      "pip",
		},
		{
			name:         "oci",
			registryType: "oci",
			runtimeHint:  "",
			wantImage:    "",
			wantCmd:      "",
		},
		{
			name:         "unknown type",
			registryType: "unknown",
			runtimeHint:  "",
			wantImage:    "",
			wantCmd:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			image, cmd := getImageAndCommand(tt.registryType, tt.runtimeHint)
			assert.Equal(t, tt.wantImage, image)
			assert.Equal(t, tt.wantCmd, cmd)
		})
	}
}

func TestSetOwnerLabels(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	r := &RegistryDeploymentReconciler{
		Client: c,
		Scheme: scheme,
		Logger: logger,
	}

	deployment := &agentregistryv1alpha1.RegistryDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-deployment",
			Namespace: "default",
		},
	}

	// Create a mock object to apply labels to
	obj := &agentregistryv1alpha1.MCPServerCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-server",
			Namespace: "default",
		},
	}

	r.setOwnerLabels(obj, deployment)

	labels := obj.GetLabels()
	assert.Equal(t, "agentregistry", labels[managedByLabel])
	assert.Equal(t, "my-deployment", labels[deploymentNameLabel])
	assert.Equal(t, "default", labels[deploymentNSLabel])
}

// ---------------------------------------------------------------------------
// Conversion method tests
// ---------------------------------------------------------------------------

func TestRegistryDeploymentReconciler_ConvertCatalogToMCPServer_Remote(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	r := &RegistryDeploymentReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		Scheme: scheme,
		Logger: zerolog.New(nil),
	}

	catalog := &agentregistryv1alpha1.MCPServerCatalog{
		Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
			Name:    "test-server",
			Version: "1.0.0",
			Remotes: []agentregistryv1alpha1.Transport{
				{
					Type: "streamable-http",
					URL:  "https://api.example.com/mcp",
					Headers: []agentregistryv1alpha1.KeyValueInput{
						{Name: "Authorization", Value: "Bearer token"},
					},
				},
			},
		},
	}

	deployment := &agentregistryv1alpha1.RegistryDeployment{
		Spec: agentregistryv1alpha1.RegistryDeploymentSpec{
			Namespace:    "target-ns",
			PreferRemote: true,
		},
	}

	server, err := r.convertCatalogToMCPServer(catalog, deployment)
	require.NoError(t, err)
	require.NotNil(t, server)
	assert.Equal(t, "target-ns", server.Namespace)
	require.NotNil(t, server.Remote)
	assert.Equal(t, "api.example.com", server.Remote.Host)
}

func TestRegistryDeploymentReconciler_ConvertCatalogToMCPServer_Local(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	r := &RegistryDeploymentReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		Scheme: scheme,
		Logger: zerolog.New(nil),
	}

	catalog := &agentregistryv1alpha1.MCPServerCatalog{
		Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
			Name:    "npm-server",
			Version: "1.0.0",
			Packages: []agentregistryv1alpha1.Package{
				{
					RegistryType: "npm",
					Identifier:   "@test/package",
					Version:      "1.0.0",
					Transport: agentregistryv1alpha1.Transport{
						Type: "stdio",
					},
				},
			},
		},
	}

	deployment := &agentregistryv1alpha1.RegistryDeployment{
		Spec: agentregistryv1alpha1.RegistryDeploymentSpec{
			Namespace: "default",
		},
	}

	server, err := r.convertCatalogToMCPServer(catalog, deployment)
	require.NoError(t, err)
	require.NotNil(t, server)
	assert.Equal(t, "default", server.Namespace)
	require.NotNil(t, server.Local)
}

func TestRegistryDeploymentReconciler_ConvertCatalogToMCPServer_NoPackagesOrRemotes(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	r := &RegistryDeploymentReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		Scheme: scheme,
		Logger: zerolog.New(nil),
	}

	catalog := &agentregistryv1alpha1.MCPServerCatalog{
		Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
			Name:    "empty-server",
			Version: "1.0.0",
		},
	}

	deployment := &agentregistryv1alpha1.RegistryDeployment{
		Spec: agentregistryv1alpha1.RegistryDeploymentSpec{
			Namespace: "default",
		},
	}

	server, err := r.convertCatalogToMCPServer(catalog, deployment)
	assert.Error(t, err)
	assert.Nil(t, server)
	assert.Contains(t, err.Error(), "no packages available")
}

func TestRegistryDeploymentReconciler_ConvertCatalogToAgent_Basic(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	r := &RegistryDeploymentReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		Scheme: scheme,
		Logger: zerolog.New(nil),
	}

	catalog := &agentregistryv1alpha1.AgentCatalog{
		Spec: agentregistryv1alpha1.AgentCatalogSpec{
			Name:    "test-agent",
			Version: "1.0.0",
			Image:   "registry.io/agent:1.0.0",
		},
	}

	deployment := &agentregistryv1alpha1.RegistryDeployment{
		Spec: agentregistryv1alpha1.RegistryDeploymentSpec{
			Namespace: "prod",
		},
	}

	agent, err := r.convertCatalogToAgent(catalog, deployment)
	require.NoError(t, err)
	require.NotNil(t, agent)
	assert.Equal(t, "registry.io/agent:1.0.0", agent.Deployment.Image)
}

func TestRegistryDeploymentReconciler_ConvertCatalogToAgent_WithPackages(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	r := &RegistryDeploymentReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		Scheme: scheme,
		Logger: zerolog.New(nil),
	}

	catalog := &agentregistryv1alpha1.AgentCatalog{
		Spec: agentregistryv1alpha1.AgentCatalogSpec{
			Name:    "pkg-agent",
			Version: "1.0.0",
			Image:   "agent:latest",
			Packages: []agentregistryv1alpha1.AgentPackage{
				{
					RegistryType: "oci",
					Identifier:   "registry.io/pkg",
					Transport: &agentregistryv1alpha1.AgentPackageTransport{
						Type: "stdio",
					},
				},
			},
		},
	}

	deployment := &agentregistryv1alpha1.RegistryDeployment{
		Spec: agentregistryv1alpha1.RegistryDeploymentSpec{
			Namespace: "default",
		},
	}

	agent, err := r.convertCatalogToAgent(catalog, deployment)
	require.NoError(t, err)
	require.NotNil(t, agent)
	// Agent structure doesn't expose packages directly in the test
	// It gets translated to k8s resources through the runtime layer
	assert.NotEmpty(t, agent.Name)
}
