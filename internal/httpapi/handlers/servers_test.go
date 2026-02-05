package handlers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

func setupTestClient(t *testing.T) client.Client {
	// Create a scheme that includes our CRDs
	scheme := runtime.NewScheme()
	require.NoError(t, agentregistryv1alpha1.AddToScheme(scheme))
	require.NoError(t, apiextensionsv1.AddToScheme(scheme))

	// Create a fake client
	return fake.NewClientBuilder().WithScheme(scheme).Build()
}

func TestServerHandler_CreateServer(t *testing.T) {
	c := setupTestClient(t)
	ctx := context.Background()
	logger := zerolog.Nop()

	handler := NewServerHandler(c, nil, logger)

	// Test creating a server
	input := &CreateServerInput{
		Body: ServerJSON{
			Name:        "my-test-server",
			Version:     "1.0.0",
			Title:       "My Test Server",
			Description: "A test server for unit tests",
			WebsiteURL:  "https://example.com",
		},
	}

	resp, err := handler.createServer(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, "my-test-server", resp.Body.Server.Name)
	assert.Equal(t, "1.0.0", resp.Body.Server.Version)

	// Verify the server was created in the fake client
	created := &agentregistryv1alpha1.MCPServerCatalog{}
	err = c.Get(ctx, client.ObjectKey{Name: "my-test-server-1-0-0"}, created)
	require.NoError(t, err)
	assert.Equal(t, "My Test Server", created.Spec.Title)
}

func TestServerHandler_CreateServer_InvalidVersion(t *testing.T) {
	c := setupTestClient(t)
	ctx := context.Background()
	logger := zerolog.Nop()

	handler := NewServerHandler(c, nil, logger)

	// Test creating a server with invalid version
	input := &CreateServerInput{
		Body: ServerJSON{
			Name:        "my-test-server",
			Version:     "invalid-version",
			Title:       "My Test Server",
			Description: "A test server for unit tests",
		},
	}

	_, err := handler.createServer(ctx, input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid version")
}

func TestServerHandler_CreateServer_InvalidName(t *testing.T) {
	c := setupTestClient(t)
	ctx := context.Background()
	logger := zerolog.Nop()

	handler := NewServerHandler(c, nil, logger)

	// Test creating a server with empty name
	input := &CreateServerInput{
		Body: ServerJSON{
			Name:        "",
			Version:     "1.0.0",
			Title:       "My Test Server",
			Description: "A test server for unit tests",
		},
	}

	_, err := handler.createServer(ctx, input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid server name")
}

func TestSetCatalogCondition(t *testing.T) {
	condType := agentregistryv1alpha1.CatalogConditionType("Ready")

	t.Run("append when condition does not exist", func(t *testing.T) {
		conditions := SetCatalogCondition(nil, condType, metav1.ConditionTrue, "Available", "all good")
		require.Len(t, conditions, 1)
		assert.Equal(t, condType, conditions[0].Type)
		assert.Equal(t, metav1.ConditionTrue, conditions[0].Status)
		assert.Equal(t, "Available", conditions[0].Reason)
		assert.Equal(t, "all good", conditions[0].Message)
		assert.False(t, conditions[0].LastTransitionTime.IsZero())
	})

	t.Run("update status and bump transition time", func(t *testing.T) {
		existing := []agentregistryv1alpha1.CatalogCondition{
			{
				Type:               condType,
				Status:             metav1.ConditionFalse,
				Reason:             "Pending",
				Message:            "waiting",
				LastTransitionTime: metav1.Now(),
			},
		}
		original := existing[0].LastTransitionTime

		updated := SetCatalogCondition(existing, condType, metav1.ConditionTrue, "Available", "ready now")
		require.Len(t, updated, 1)
		assert.Equal(t, metav1.ConditionTrue, updated[0].Status)
		assert.Equal(t, "Available", updated[0].Reason)
		assert.Equal(t, "ready now", updated[0].Message)
		// Status changed → LastTransitionTime must be updated
		assert.False(t, updated[0].LastTransitionTime.Equal(&original))
	})

	t.Run("same status does not bump transition time", func(t *testing.T) {
		existing := []agentregistryv1alpha1.CatalogCondition{
			{
				Type:               condType,
				Status:             metav1.ConditionTrue,
				Reason:             "Available",
				Message:            "old message",
				LastTransitionTime: metav1.Now(),
			},
		}
		original := existing[0].LastTransitionTime

		updated := SetCatalogCondition(existing, condType, metav1.ConditionTrue, "Available", "new message")
		require.Len(t, updated, 1)
		assert.Equal(t, "new message", updated[0].Message)
		// Status unchanged → LastTransitionTime must NOT change
		assert.True(t, updated[0].LastTransitionTime.Equal(&original))
	})
}

func TestGenerateCRName(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{"test-server", "1.0.0", "test-server-1-0-0"},
		{"my/server", "2.1.0", "my-server-2-1-0"},
		{"ServerName", "v1.0.0", "servername-v1-0-0"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"-"+tt.version, func(t *testing.T) {
			got := GenerateCRName(tt.name, tt.version)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestServerResponseSerialization(t *testing.T) {
	publishedAt := time.Now()
	resp := ServerResponse{
		Server: ServerJSON{
			Name:        "test",
			Version:     "1.0.0",
			Title:       "Test",
			Description: "Test description",
		},
		Meta: ServerMeta{
			Official: &OfficialMeta{
				Status:      "active",
				PublishedAt: &publishedAt,
				UpdatedAt:   time.Now(),
				IsLatest:    true,
				Published:   true,
			},
		},
	}

	// Test JSON serialization
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded ServerResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.Server.Name, decoded.Server.Name)
	assert.Equal(t, resp.Server.Version, decoded.Server.Version)
	assert.Equal(t, resp.Meta.Official.Published, decoded.Meta.Official.Published)
	assert.Equal(t, resp.Meta.Official.IsLatest, decoded.Meta.Official.IsLatest)
}

func TestConvertToServerResponse(t *testing.T) {
	c := setupTestClient(t)
	logger := zerolog.Nop()
	handler := NewServerHandler(c, nil, logger)

	publishedAt := metav1.Now()
	server := &agentregistryv1alpha1.MCPServerCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "convert-test-1-0-0",
			Namespace: "default",
		},
		Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
			Name:        "convert-test",
			Version:     "1.0.0",
			Title:       "Convert Test",
			Description: "Test conversion",
			WebsiteURL:  "https://example.com",
			Repository: &agentregistryv1alpha1.Repository{
				URL:    "https://github.com/test/repo",
				Source: "github",
			},
			Packages: []agentregistryv1alpha1.Package{
				{
					RegistryType: "npm",
					Identifier:   "@test/package",
					Version:      "1.0.0",
					Transport: agentregistryv1alpha1.Transport{
						Type: "stdio",
						Headers: []agentregistryv1alpha1.KeyValueInput{
							{Name: "Authorization", Value: "token"},
						},
					},
					RuntimeArguments: []agentregistryv1alpha1.Argument{
						{Name: "arg1", Type: "string", Required: true},
					},
				},
			},
		},
		Status: agentregistryv1alpha1.MCPServerCatalogStatus{
			Published:   true,
			IsLatest:    true,
			PublishedAt: &publishedAt,
			Status:      agentregistryv1alpha1.CatalogStatusActive,
		},
	}

	resp := handler.convertToServerResponse(server, nil)

	assert.Equal(t, "convert-test", resp.Server.Name)
	assert.Equal(t, "1.0.0", resp.Server.Version)
	assert.Equal(t, "Test conversion", resp.Server.Description)
	assert.NotNil(t, resp.Server.Repository)
	assert.Equal(t, "github", resp.Server.Repository.Source)
	assert.Len(t, resp.Server.Packages, 1)
	assert.Equal(t, "npm", resp.Server.Packages[0].RegistryType)
	assert.Equal(t, "stdio", resp.Server.Packages[0].Transport.Type)
	assert.True(t, resp.Meta.Official.Published)
	assert.True(t, resp.Meta.Official.IsLatest)
}

// ---------------------------------------------------------------------------
// GET/LIST operations
// ---------------------------------------------------------------------------

func TestServerHandler_ConvertToServerResponse_Discovered(t *testing.T) {
	c := setupTestClient(t)
	handler := NewServerHandler(c, nil, zerolog.Nop())

	server := &agentregistryv1alpha1.MCPServerCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: "discovered-server-1-0-0",
			Labels: map[string]string{
				"agentregistry.dev/discovered": "true",
			},
		},
		Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
			Name:    "discovered-server",
			Version: "1.0.0",
			Title:   "Discovered",
		},
	}

	resp := handler.convertToServerResponse(server, nil)
	assert.True(t, resp.Meta.IsDiscovered)
	assert.Equal(t, "discovery", resp.Meta.Source)
}

func TestServerHandler_ConvertToServerResponse_WithDeployment(t *testing.T) {
	c := setupTestClient(t)
	handler := NewServerHandler(c, nil, zerolog.Nop())

	now := metav1.Now()
	server := &agentregistryv1alpha1.MCPServerCatalog{
		ObjectMeta: metav1.ObjectMeta{Name: "deployed-server-1-0-0"},
		Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
			Name:    "deployed-server",
			Version: "1.0.0",
		},
		Status: agentregistryv1alpha1.MCPServerCatalogStatus{
			Deployment: &agentregistryv1alpha1.DeploymentRef{
				Namespace:   "production",
				ServiceName: "deployed-server-svc",
				Ready:       true,
				Message:     "running",
				LastChecked: &now,
			},
		},
	}

	resp := handler.convertToServerResponse(server, nil)
	require.NotNil(t, resp.Meta.Deployment)
	assert.Equal(t, "production", resp.Meta.Deployment.Namespace)
	assert.Equal(t, "deployed-server-svc", resp.Meta.Deployment.ServiceName)
	assert.True(t, resp.Meta.Deployment.Ready)
	assert.Equal(t, "running", resp.Meta.Deployment.Message)
}
