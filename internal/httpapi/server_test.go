package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
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

func TestIsDeployWriteRequest(t *testing.T) {
	server, _ := setupTestServer(t)

	tests := []struct {
		name   string
		method string
		path   string
		want   bool
	}{
		{"POST deployments", http.MethodPost, "/admin/v0/deployments", true},
		{"PATCH deployments", http.MethodPatch, "/admin/v0/deployments", true},
		{"DELETE deployments", http.MethodDelete, "/admin/v0/deployments", true},
		{"DELETE specific deployment", http.MethodDelete, "/admin/v0/deployments/my-deploy", true},
		{"GET deployments is not write", http.MethodGet, "/admin/v0/deployments", false},
		{"POST servers is not deployments", http.MethodPost, "/admin/v0/servers", false},
		{"POST public deployments path", http.MethodPost, "/v0/deployments", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			ctx := humatest.NewContext(nil, req, httptest.NewRecorder())
			got := server.isDeployWriteRequest(ctx)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAuthMiddleware_Disabled(t *testing.T) {
	server, _ := setupTestServerWithAuthDisabled(t)
	// Force authEnabled off (setupTestServerWithAuthDisabled sets env before NewServer, but
	// NewServer reads it at construction; re-confirm it stuck)
	server.authEnabled = false

	called := false
	req := httptest.NewRequest(http.MethodGet, "/admin/v0/servers", nil)
	ctx := humatest.NewContext(nil, req, httptest.NewRecorder())
	server.authMiddleware(ctx, func(_ huma.Context) { called = true })
	assert.True(t, called)
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	server, _ := setupTestServer(t)
	server.authEnabled = true

	req := httptest.NewRequest(http.MethodGet, "/admin/v0/servers", nil)
	rec := httptest.NewRecorder()
	ctx := humatest.NewContext(nil, req, rec)

	called := false
	server.authMiddleware(ctx, func(_ huma.Context) { called = true })
	assert.False(t, called)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	server, _ := setupTestServer(t)
	server.authEnabled = true
	// allowedTokens is empty — any token is invalid

	req := httptest.NewRequest(http.MethodGet, "/admin/v0/servers", nil)
	req.Header.Set("Authorization", "Bearer bad-token-xyz")
	rec := httptest.NewRecorder()
	ctx := humatest.NewContext(nil, req, rec)

	called := false
	server.authMiddleware(ctx, func(_ huma.Context) { called = true })
	assert.False(t, called)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	server, _ := setupTestServer(t)
	server.authEnabled = true
	server.allowedTokens["super-secret-token"] = true

	req := httptest.NewRequest(http.MethodGet, "/admin/v0/servers", nil)
	req.Header.Set("Authorization", "Bearer super-secret-token")
	rec := httptest.NewRecorder()
	ctx := humatest.NewContext(nil, req, rec)

	called := false
	server.authMiddleware(ctx, func(_ huma.Context) { called = true })
	assert.True(t, called)
}

func TestAuthMiddleware_MalformedHeader(t *testing.T) {
	server, _ := setupTestServer(t)
	server.authEnabled = true

	req := httptest.NewRequest(http.MethodGet, "/admin/v0/servers", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()
	ctx := humatest.NewContext(nil, req, rec)

	called := false
	server.authMiddleware(ctx, func(_ huma.Context) { called = true })
	assert.False(t, called)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestConvertExternalToSpec_FullPayload(t *testing.T) {
	server, _ := setupTestServer(t)

	ext := ExternalServerJSON{
		Name:        "full-server",
		Version:     "2.0.0",
		Title:       "Full Server",
		Description: "All fields populated",
		WebsiteURL:  "https://example.com",
		Repository: &ExternalRepositoryJSON{
			URL:       "https://github.com/org/repo",
			Source:    "github",
			ID:        "repo-123",
			Subfolder: "packages/server",
		},
		Packages: []ExternalPackageJSON{
			{
				RegistryType:    "npm",
				RegistryBaseURL: "https://registry.npmjs.org",
				Identifier:      "@org/server",
				Version:         "2.0.0",
				FileSHA256:      "abc123",
				RuntimeHint:     "node",
				Transport: ExternalTransportJSON{
					Type: "stdio",
					Headers: []ExternalKeyValueJSON{
						{Name: "X-Token", Value: "tok", Required: true},
					},
				},
				RuntimeArguments: []ExternalArgumentJSON{
					{Name: "port", Type: "integer", Required: true},
				},
				PackageArguments: []ExternalArgumentJSON{
					{Name: "config", Type: "string", Value: "default.json"},
				},
				EnvironmentVariables: []ExternalKeyValueJSON{
					{Name: "NODE_ENV", Value: "production"},
				},
			},
		},
		Remotes: []ExternalTransportJSON{
			{
				Type: "streamable-http",
				URL:  "https://api.example.com/mcp",
				Headers: []ExternalKeyValueJSON{
					{Name: "Authorization", Value: "Bearer x"},
				},
			},
		},
	}

	spec := server.convertExternalToSpec(ext)

	// Top-level
	assert.Equal(t, "full-server", spec.Name)
	assert.Equal(t, "2.0.0", spec.Version)
	assert.Equal(t, "https://example.com", spec.WebsiteURL)

	// Repository
	require.NotNil(t, spec.Repository)
	assert.Equal(t, "github", spec.Repository.Source)
	assert.Equal(t, "packages/server", spec.Repository.Subfolder)

	// Package
	require.Len(t, spec.Packages, 1)
	pkg := spec.Packages[0]
	assert.Equal(t, "npm", pkg.RegistryType)
	assert.Equal(t, "https://registry.npmjs.org", pkg.RegistryBaseURL)
	assert.Equal(t, "abc123", pkg.FileSHA256)
	assert.Equal(t, "node", pkg.RuntimeHint)
	assert.Equal(t, "stdio", pkg.Transport.Type)
	require.Len(t, pkg.Transport.Headers, 1)
	assert.Equal(t, "X-Token", pkg.Transport.Headers[0].Name)
	assert.True(t, pkg.Transport.Headers[0].Required)
	require.Len(t, pkg.RuntimeArguments, 1)
	assert.Equal(t, "port", pkg.RuntimeArguments[0].Name)
	assert.True(t, pkg.RuntimeArguments[0].Required)
	require.Len(t, pkg.PackageArguments, 1)
	assert.Equal(t, "default.json", pkg.PackageArguments[0].Value)
	require.Len(t, pkg.EnvironmentVariables, 1)
	assert.Equal(t, "NODE_ENV", pkg.EnvironmentVariables[0].Name)

	// Remotes
	require.Len(t, spec.Remotes, 1)
	assert.Equal(t, "streamable-http", spec.Remotes[0].Type)
	assert.Equal(t, "https://api.example.com/mcp", spec.Remotes[0].URL)
	require.Len(t, spec.Remotes[0].Headers, 1)
	assert.Equal(t, "Authorization", spec.Remotes[0].Headers[0].Name)
}

func TestConvertExternalToSpec_EmptyOptionalFields(t *testing.T) {
	server, _ := setupTestServer(t)

	ext := ExternalServerJSON{
		Name:    "minimal",
		Version: "1.0.0",
	}

	spec := server.convertExternalToSpec(ext)
	assert.Equal(t, "minimal", spec.Name)
	assert.Nil(t, spec.Repository)
	assert.Empty(t, spec.Packages)
	assert.Empty(t, spec.Remotes)
}
