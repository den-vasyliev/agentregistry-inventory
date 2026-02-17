package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestOCIHandler_CreateOCIArtifact(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, agentregistryv1alpha1.AddToScheme(scheme))

	tests := []struct {
		name           string
		request        OCIArtifactRequest
		expectedStatus int
		expectedError  bool
		setupClient    func() client.Client
	}{
		{
			name: "valid mcp-server artifact creation",
			request: OCIArtifactRequest{
				ResourceName: "filesystem-tools",
				Version:      "1.0.0",
				ResourceType: "mcp-server",
				Environment:  "production",
				Config: map[string]string{
					"ALLOWED_PATHS": "/tmp,/home",
				},
				RepositoryURL: "https://github.com/test-org/filesystem-tools",
				CommitSHA:     "abc123",
				Pusher:        "testuser",
			},
			expectedStatus: http.StatusCreated,
			expectedError:  false,
			setupClient: func() client.Client {
				mcpServer := &agentregistryv1alpha1.MCPServerCatalog{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "filesystem-tools-1-0-0",
						Namespace: "agentregistry",
					},
					Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
						Name:        "filesystem-tools",
						Version:     "1.0.0",
						Title:       "Filesystem Tools",
						Description: "MCP server for file operations",
					},
				}
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(mcpServer).Build()
			},
		},
		{
			name: "missing required fields",
			request: OCIArtifactRequest{
				ResourceName: "filesystem-tools",
				// Missing version and resourceType
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
			setupClient: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
		},
		{
			name: "resource not found in catalog",
			request: OCIArtifactRequest{
				ResourceName: "nonexistent-resource",
				Version:      "1.0.0",
				ResourceType: "mcp-server",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
			setupClient: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
		},
		{
			name: "valid agent artifact creation",
			request: OCIArtifactRequest{
				ResourceName: "code-reviewer",
				Version:      "2.1.0",
				ResourceType: "agent",
				Environment:  "staging",
			},
			expectedStatus: http.StatusCreated,
			expectedError:  false,
			setupClient: func() client.Client {
				agent := &agentregistryv1alpha1.AgentCatalog{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "code-reviewer-2-1-0",
						Namespace: "agentregistry",
					},
					Spec: agentregistryv1alpha1.AgentCatalogSpec{
						Name:        "code-reviewer",
						Version:     "2.1.0",
						Title:       "Code Reviewer Agent",
						Description: "AI agent for code reviews",
					},
				}
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			logger := zerolog.New(zerolog.NewTestWriter(t))
			handler := NewOCIHandler(client, logger, "registry.test.com", "test-auth")

			// Create request
			requestBytes, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/admin/v0/oci/artifacts", bytes.NewReader(requestBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Call handler
			handler.CreateOCIArtifact(w, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			var response OCIArtifactResponse
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.expectedError {
				assert.False(t, response.Success)
			} else {
				assert.True(t, response.Success)
				assert.NotEmpty(t, response.ArtifactRef)
				assert.NotEmpty(t, response.Digest)
			}
		})
	}
}

func TestOCIHandler_fetchCatalogResource(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, agentregistryv1alpha1.AddToScheme(scheme))

	tests := []struct {
		name         string
		resourceName string
		version      string
		resourceType string
		expectError  bool
		setupClient  func() client.Client
	}{
		{
			name:         "fetch existing mcp-server",
			resourceName: "filesystem-tools",
			version:      "1.0.0",
			resourceType: "mcp-server",
			expectError:  false,
			setupClient: func() client.Client {
				mcpServer := &agentregistryv1alpha1.MCPServerCatalog{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "filesystem-tools-1-0-0",
						Namespace: "agentregistry",
					},
					Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
						Name:    "filesystem-tools",
						Version: "1.0.0",
					},
				}
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(mcpServer).Build()
			},
		},
		{
			name:         "fetch non-existing resource",
			resourceName: "nonexistent",
			version:      "1.0.0",
			resourceType: "mcp-server",
			expectError:  true,
			setupClient: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
		},
		{
			name:         "unsupported resource type",
			resourceName: "test",
			version:      "1.0.0",
			resourceType: "invalid-type",
			expectError:  true,
			setupClient: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
		},
		{
			name:         "fetch existing agent",
			resourceName: "code-reviewer",
			version:      "2.1.0",
			resourceType: "agent",
			expectError:  false,
			setupClient: func() client.Client {
				agent := &agentregistryv1alpha1.AgentCatalog{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "code-reviewer-2-1-0",
						Namespace: "agentregistry",
					},
					Spec: agentregistryv1alpha1.AgentCatalogSpec{
						Name:    "code-reviewer",
						Version: "2.1.0",
					},
				}
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()
			},
		},
		{
			name:         "fetch existing skill",
			resourceName: "terraform-deploy",
			version:      "1.5.0",
			resourceType: "skill",
			expectError:  false,
			setupClient: func() client.Client {
				skill := &agentregistryv1alpha1.SkillCatalog{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "terraform-deploy-1-5-0",
						Namespace: "agentregistry",
					},
					Spec: agentregistryv1alpha1.SkillCatalogSpec{
						Name:    "terraform-deploy",
						Version: "1.5.0",
					},
				}
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(skill).Build()
			},
		},
		{
			name:         "fetch existing model",
			resourceName: "claude-sonnet",
			version:      "3.5",
			resourceType: "model",
			expectError:  false,
			setupClient: func() client.Client {
				model := &agentregistryv1alpha1.ModelCatalog{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "claude-sonnet-3-5",
						Namespace: "agentregistry",
					},
					Spec: agentregistryv1alpha1.ModelCatalogSpec{
						Name:     "claude-sonnet",
						Provider: "Anthropic",
						Model:    "claude-3-5-sonnet-20241022",
					},
				}
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(model).Build()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			logger := zerolog.New(zerolog.NewTestWriter(t))
			handler := NewOCIHandler(client, logger, "registry.test.com", "test-auth")

			resource, err := handler.fetchCatalogResource(context.Background(), tt.resourceName, tt.version, tt.resourceType)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resource)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resource)
			}
		})
	}
}

func TestOCIHandler_pushOCIArtifact(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	handler := NewOCIHandler(nil, logger, "ghcr.io/test-org/registry", "test-auth")

	artifactConfig := &OCIArtifactConfig{
		Architecture: "any",
		OS:           "any",
		ResourceName: "test-resource",
		Version:      "1.0.0",
		ResourceType: "mcp-server",
		Environment:  "production",
	}

	artifactRef, digest, size, err := handler.pushOCIArtifact(context.Background(), artifactConfig)

	// Since this is a mock implementation, it should not error
	assert.NoError(t, err)
	assert.Contains(t, artifactRef, "test-resource:1.0.0")
	assert.NotEmpty(t, digest)
	assert.Greater(t, size, int64(0))
}