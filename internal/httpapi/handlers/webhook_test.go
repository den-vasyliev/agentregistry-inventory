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

func TestWebhookHandler_HandleGitHubWebhook(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, agentregistryv1alpha1.AddToScheme(scheme))

	tests := []struct {
		name           string
		payload        GitHubWebhookPayload
		expectedStatus int
		expectedFiles  int
		setupClient    func() client.Client
	}{
		{
			name: "valid webhook with mcp-server resource",
			payload: GitHubWebhookPayload{
				Ref: "refs/heads/main",
				Repository: struct {
					FullName string `json:"full_name"`
					HTMLURL  string `json:"html_url"`
				}{
					FullName: "test-org/test-repo",
					HTMLURL:  "https://github.com/test-org/test-repo",
				},
				HeadCommit: struct {
					ID       string   `json:"id"`
					Message  string   `json:"message"`
					Added    []string `json:"added"`
					Modified []string `json:"modified"`
					Removed  []string `json:"removed"`
				}{
					ID:      "abc123",
					Message: "Add filesystem MCP server",
					Added:   []string{"resources/mcp-server/filesystem/filesystem-1.0.0.yaml"},
				},
				Pusher: struct {
					Name  string `json:"name"`
					Email string `json:"email"`
				}{
					Name:  "testuser",
					Email: "test@example.com",
				},
			},
			expectedStatus: http.StatusOK,
			expectedFiles:  1,
			setupClient: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
		},
		{
			name: "webhook with non-main branch should be ignored",
			payload: GitHubWebhookPayload{
				Ref: "refs/heads/feature/test",
				Repository: struct {
					FullName string `json:"full_name"`
					HTMLURL  string `json:"html_url"`
				}{
					FullName: "test-org/test-repo",
					HTMLURL:  "https://github.com/test-org/test-repo",
				},
				HeadCommit: struct {
					ID       string   `json:"id"`
					Message  string   `json:"message"`
					Added    []string `json:"added"`
					Modified []string `json:"modified"`
					Removed  []string `json:"removed"`
				}{
					Added: []string{"resources/mcp-server/test/test-1.0.0.yaml"},
				},
			},
			expectedStatus: http.StatusOK,
			expectedFiles:  0,
			setupClient: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
		},
		{
			name: "webhook with non-registry files should be ignored",
			payload: GitHubWebhookPayload{
				Ref: "refs/heads/main",
				Repository: struct {
					FullName string `json:"full_name"`
					HTMLURL  string `json:"html_url"`
				}{
					FullName: "test-org/test-repo",
					HTMLURL:  "https://github.com/test-org/test-repo",
				},
				HeadCommit: struct {
					ID       string   `json:"id"`
					Message  string   `json:"message"`
					Added    []string `json:"added"`
					Modified []string `json:"modified"`
					Removed  []string `json:"removed"`
				}{
					Added: []string{"src/main.go", "README.md", "docs/api.md"},
				},
			},
			expectedStatus: http.StatusOK,
			expectedFiles:  0,
			setupClient: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
		},
		{
			name: "webhook with deleted resources",
			payload: GitHubWebhookPayload{
				Ref: "refs/heads/main",
				Repository: struct {
					FullName string `json:"full_name"`
					HTMLURL  string `json:"html_url"`
				}{
					FullName: "test-org/test-repo",
					HTMLURL:  "https://github.com/test-org/test-repo",
				},
				HeadCommit: struct {
					ID       string   `json:"id"`
					Message  string   `json:"message"`
					Added    []string `json:"added"`
					Modified []string `json:"modified"`
					Removed  []string `json:"removed"`
				}{
					Removed: []string{"resources/agent/old-agent/old-agent-1.0.0.yaml"},
				},
			},
			expectedStatus: http.StatusOK,
			expectedFiles:  1,
			setupClient: func() client.Client {
				// Pre-populate with existing resource
				existingAgent := &agentregistryv1alpha1.AgentCatalog{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "old-agent-1-0-0",
						Namespace: "agentregistry",
					},
					Spec: agentregistryv1alpha1.AgentCatalogSpec{
						Name:    "old-agent",
						Version: "1.0.0",
					},
				}
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingAgent).Build()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			logger := zerolog.New(zerolog.NewTestWriter(t))
			handler := NewWebhookHandler(client, logger)

			// Create request
			payloadBytes, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader(payloadBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Call handler
			handler.HandleGitHubWebhook(w, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			var response WebhookResponse
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.True(t, response.Success)
			assert.Len(t, response.Processed, tt.expectedFiles)
		})
	}
}

func TestWebhookHandler_isAgentRegistryFile(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	handler := NewWebhookHandler(nil, logger)

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "valid mcp-server resource file",
			filePath: "resources/mcp-server/filesystem/filesystem-1.0.0.yaml",
			expected: true,
		},
		{
			name:     "valid agent resource file",
			filePath: "resources/agent/code-reviewer/code-reviewer-2.1.0.yml",
			expected: true,
		},
		{
			name:     "valid skill resource file",
			filePath: "resources/skill/terraform/terraform-1.0.0.yaml",
			expected: true,
		},
		{
			name:     "valid model resource file",
			filePath: "resources/model/claude/claude-3.5.yaml",
			expected: true,
		},
		{
			name:     "non-resources directory",
			filePath: "src/main.go",
			expected: false,
		},
		{
			name:     "resources directory but not yaml",
			filePath: "resources/mcp-server/test/README.md",
			expected: false,
		},
		{
			name:     "resources directory but invalid kind",
			filePath: "resources/invalid-kind/test/test-1.0.0.yaml",
			expected: false,
		},
		{
			name:     "resources directory but too shallow",
			filePath: "resources/test.yaml",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.isAgentRegistryFile(tt.filePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWebhookHandler_processResourceFile(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, agentregistryv1alpha1.AddToScheme(scheme))

	tests := []struct {
		name           string
		filePath       string
		action         string
		payload        *GitHubWebhookPayload
		expectedError  bool
		expectedAction string
		setupClient    func() client.Client
	}{
		{
			name:     "delete action with valid path",
			filePath: "resources/mcp-server/filesystem/filesystem-1.0.0.yaml",
			action:   "delete",
			payload: &GitHubWebhookPayload{
				Repository: struct {
					FullName string `json:"full_name"`
					HTMLURL  string `json:"html_url"`
				}{
					FullName: "test-org/test-repo",
				},
			},
			expectedError:  false,
			expectedAction: "delete",
			setupClient: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
		},
		{
			name:     "invalid file path format",
			filePath: "resources/invalid",
			action:   "delete",
			payload: &GitHubWebhookPayload{
				Repository: struct {
					FullName string `json:"full_name"`
					HTMLURL  string `json:"html_url"`
				}{
					FullName: "test-org/test-repo",
				},
			},
			expectedError: true,
			setupClient: func() client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			logger := zerolog.New(zerolog.NewTestWriter(t))
			handler := NewWebhookHandler(client, logger)

			result, err := handler.processResourceFile(context.Background(), tt.payload, tt.filePath, tt.action)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedAction, result.Action)
			}
		})
	}
}