package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

// mockCache implements cache.Cache for testing
type mockCache struct {
	client client.Reader
}

func (m *mockCache) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return m.client.Get(ctx, key, obj, opts...)
}

func (m *mockCache) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return m.client.List(ctx, list, opts...)
}

func (m *mockCache) GetInformer(ctx context.Context, obj client.Object, opts ...cache.InformerGetOption) (cache.Informer, error) {
	return nil, nil
}

func (m *mockCache) GetInformerForKind(ctx context.Context, gvk schema.GroupVersionKind, opts ...cache.InformerGetOption) (cache.Informer, error) {
	return nil, nil
}

func (m *mockCache) Start(ctx context.Context) error {
	return nil
}

func (m *mockCache) WaitForCacheSync(ctx context.Context) bool {
	return true
}

func (m *mockCache) IndexField(ctx context.Context, obj client.Object, field string, extractValue client.IndexerFunc) error {
	return nil
}

func (m *mockCache) RemoveInformer(ctx context.Context, obj client.Object) error {
	return nil
}

func setupTestServer(t *testing.T) (*Server, client.Client) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create mock cache that uses the client
	mockC := &mockCache{client: c}

	server := NewServer(c, mockC, logger)
	return server, c
}

func setupTestServerWithAuthDisabled(t *testing.T) (*Server, client.Client) {
	// Disable auth for testing
	os.Setenv("AGENTREGISTRY_DISABLE_AUTH", "true")
	defer os.Unsetenv("AGENTREGISTRY_DISABLE_AUTH")

	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create mock cache that uses the client
	mockC := &mockCache{client: c}

	server := NewServer(c, mockC, logger)
	return server, c
}

func TestServer_HealthEndpoints(t *testing.T) {
	server, _ := setupTestServer(t)

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{"healthz", "/healthz", http.StatusOK},
		{"readyz", "/readyz", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			server.mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, "ok", rec.Body.String())
		})
	}
}

func TestServer_PingEndpoint(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/v0/ping", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp["status"])
}

func TestServer_GetServers_Empty(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/v0/servers", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	// Should return 200 with empty list
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestServer_GetServers_WithData(t *testing.T) {
	server, c := setupTestServer(t)
	ctx := context.Background()

	// Create a test server
	serverObj := &agentregistryv1alpha1.MCPServerCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-server-v1.0.0",
			Namespace: "default",
		},
		Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
			Name:        "test-server",
			Version:     "1.0.0",
			Title:       "Test Server",
			Description: "A test MCP server",
		},
		Status: agentregistryv1alpha1.MCPServerCatalogStatus{
			Published: true,
			Status:    agentregistryv1alpha1.CatalogStatusActive,
		},
	}
	err := c.Create(ctx, serverObj)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/v0/servers", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "test-server")
}

func TestServer_GetAgents_Empty(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/v0/agents", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestServer_GetSkills_Empty(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/v0/skills", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestServer_GetModels_Empty(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/v0/models", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestServer_GetDeployments_Empty(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/v0/deployments", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestServer_GetEnvironments(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/v0/environments", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestServer_AdminHealthEndpoint_WithAuthDisabled(t *testing.T) {
	server, _ := setupTestServerWithAuthDisabled(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/v0/health", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	// Should succeed when auth is disabled
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestConvertExternalToSpec(t *testing.T) {
	server, _ := setupTestServer(t)

	external := ExternalServerJSON{
		Name:        "test-convert",
		Version:     "1.0.0",
		Title:       "Test Convert",
		Description: "Test conversion",
		Packages: []ExternalPackageJSON{
			{
				RegistryType: "npm",
				Identifier:   "@test/server",
				Transport: ExternalTransportJSON{
					Type: "stdio",
				},
			},
		},
	}

	spec := server.convertExternalToSpec(external)

	assert.Equal(t, "test-convert", spec.Name)
	assert.Equal(t, "1.0.0", spec.Version)
	assert.Equal(t, "Test Convert", spec.Title)
	assert.Equal(t, 1, len(spec.Packages))
	assert.Equal(t, "npm", spec.Packages[0].RegistryType)
}

func TestServer_ImportEndpoint_InvalidMethod(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/v0/import", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	// Import endpoint only accepts POST
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestServer_InvalidPath(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/invalid/path", nil)
	rec := httptest.NewRecorder()

	server.mux.ServeHTTP(rec, req)

	// Should return 404
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestServer_Runnable(t *testing.T) {
	server, _ := setupTestServer(t)

	runnable := server.Runnable(":0")
	assert.NotNil(t, runnable)
}

func TestServer_loadTokensFromSecret_NotFound(t *testing.T) {
	server, _ := setupTestServer(t)

	// Should not panic when secret doesn't exist
	server.loadTokensFromSecret()

	// No tokens should be loaded
	assert.Equal(t, 0, len(server.allowedTokens))
}

func TestServer_AuthEnabled_ByDefault(t *testing.T) {
	server, _ := setupTestServer(t)

	// Auth should be enabled by default (unless DISABLE_AUTH is set)
	assert.True(t, server.authEnabled)
}
