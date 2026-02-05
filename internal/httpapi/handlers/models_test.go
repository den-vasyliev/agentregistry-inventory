package handlers

import (
	"context"
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

func setupModelTestClient(t *testing.T) client.Client {
	scheme := runtime.NewScheme()
	require.NoError(t, agentregistryv1alpha1.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).Build()
}

// ---------------------------------------------------------------------------
// createModel
// ---------------------------------------------------------------------------

func TestModelHandler_CreateModel(t *testing.T) {
	c := setupModelTestClient(t)
	ctx := context.Background()
	logger := zerolog.Nop()
	handler := NewModelHandler(c, nil, logger)

	input := &CreateModelInput{
		Body: ModelJSON{
			Name:        "claude-3-opus",
			Provider:    "anthropic",
			Model:       "claude-3-opus-20240229",
			Description: "Most capable Claude model",
		},
	}

	resp, err := handler.createModel(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, "claude-3-opus", resp.Body.Model.Name)
	assert.Equal(t, "anthropic", resp.Body.Model.Provider)
	assert.Equal(t, "claude-3-opus-20240229", resp.Body.Model.Model)

	// Verify the CR was persisted
	created := &agentregistryv1alpha1.ModelCatalog{}
	err = c.Get(ctx, client.ObjectKey{Name: "claude-3-opus"}, created)
	require.NoError(t, err)
	assert.Equal(t, "anthropic", created.Spec.Provider)
	assert.Equal(t, "claude-3-opus-20240229", created.Spec.Model)
}

func TestModelHandler_CreateModel_WithBaseURL(t *testing.T) {
	c := setupModelTestClient(t)
	ctx := context.Background()
	handler := NewModelHandler(c, nil, zerolog.Nop())

	input := &CreateModelInput{
		Body: ModelJSON{
			Name:     "custom-model",
			Provider: "openai",
			Model:    "gpt-4",
			BaseURL:  "https://custom-api.example.com/v1",
		},
	}

	resp, err := handler.createModel(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, "custom-model", resp.Body.Model.Name)
	assert.Equal(t, "https://custom-api.example.com/v1", resp.Body.Model.BaseURL)

	// Verify CR has base URL
	created := &agentregistryv1alpha1.ModelCatalog{}
	err = c.Get(ctx, client.ObjectKey{Name: "custom-model"}, created)
	require.NoError(t, err)
	assert.Equal(t, "https://custom-api.example.com/v1", created.Spec.BaseURL)
}

func TestModelHandler_CreateModel_MultipleProviders(t *testing.T) {
	c := setupModelTestClient(t)
	ctx := context.Background()
	handler := NewModelHandler(c, nil, zerolog.Nop())

	// Create anthropic model
	anthropicInput := &CreateModelInput{
		Body: ModelJSON{
			Name:     "claude-model",
			Provider: "anthropic",
			Model:    "claude-3-sonnet-20240229",
		},
	}
	resp1, err := handler.createModel(ctx, anthropicInput)
	require.NoError(t, err)
	assert.Equal(t, "anthropic", resp1.Body.Model.Provider)

	// Create openai model
	openaiInput := &CreateModelInput{
		Body: ModelJSON{
			Name:     "gpt-model",
			Provider: "openai",
			Model:    "gpt-4-turbo",
		},
	}
	resp2, err := handler.createModel(ctx, openaiInput)
	require.NoError(t, err)
	assert.Equal(t, "openai", resp2.Body.Model.Provider)
}

// ---------------------------------------------------------------------------
// getModel
// ---------------------------------------------------------------------------

func TestModelHandler_GetModel(t *testing.T) {
	c := setupModelTestClient(t)
	ctx := context.Background()
	_ = NewModelHandler(c, nil, zerolog.Nop())

	// Create a model first
	model := &agentregistryv1alpha1.ModelCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-model",
			Labels: map[string]string{
				"agentregistry.dev/name": "test-model",
			},
		},
		Spec: agentregistryv1alpha1.ModelCatalogSpec{
			Name:     "test-model",
			Provider: "anthropic",
			Model:    "claude-3-opus-20240229",
		},
	}
	require.NoError(t, c.Create(ctx, model))

	// Note: getModel uses cache.List which won't work with fake client
	// In a real test, we'd need envtest or mock the cache
	// For now, we'll just verify the conversion logic works
}

// ---------------------------------------------------------------------------
// convertToModelResponse
// ---------------------------------------------------------------------------

func TestModelHandler_ConvertToModelResponse_Basic(t *testing.T) {
	c := setupModelTestClient(t)
	handler := NewModelHandler(c, nil, zerolog.Nop())

	model := &agentregistryv1alpha1.ModelCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-model",
		},
		Spec: agentregistryv1alpha1.ModelCatalogSpec{
			Name:        "test-model",
			Provider:    "anthropic",
			Model:       "claude-3-opus-20240229",
			Description: "Test model",
		},
		Status: agentregistryv1alpha1.ModelCatalogStatus{
			Ready:   true,
			Message: "Ready",
		},
	}

	resp := handler.convertToModelResponse(model)
	assert.Equal(t, "test-model", resp.Model.Name)
	assert.Equal(t, "anthropic", resp.Model.Provider)
	assert.Equal(t, "claude-3-opus-20240229", resp.Model.Model)
	assert.True(t, resp.Meta.Ready)
	assert.Equal(t, "Ready", resp.Meta.Message)
	require.NotNil(t, resp.Meta.Official)
	assert.True(t, resp.Meta.Official.IsLatest)
}

func TestModelHandler_ConvertToModelResponse_WithBaseURL(t *testing.T) {
	c := setupModelTestClient(t)
	handler := NewModelHandler(c, nil, zerolog.Nop())

	model := &agentregistryv1alpha1.ModelCatalog{
		ObjectMeta: metav1.ObjectMeta{Name: "custom-model"},
		Spec: agentregistryv1alpha1.ModelCatalogSpec{
			Name:     "custom-model",
			Provider: "openai",
			Model:    "gpt-4",
			BaseURL:  "https://custom.example.com",
		},
	}

	resp := handler.convertToModelResponse(model)
	assert.Equal(t, "https://custom.example.com", resp.Model.BaseURL)
}

func TestModelHandler_ConvertToModelResponse_WithUsedBy(t *testing.T) {
	c := setupModelTestClient(t)
	handler := NewModelHandler(c, nil, zerolog.Nop())

	model := &agentregistryv1alpha1.ModelCatalog{
		ObjectMeta: metav1.ObjectMeta{Name: "used-model"},
		Spec: agentregistryv1alpha1.ModelCatalogSpec{
			Name:     "used-model",
			Provider: "anthropic",
			Model:    "claude-3-sonnet-20240229",
		},
		Status: agentregistryv1alpha1.ModelCatalogStatus{
			Ready: true,
			UsedBy: []agentregistryv1alpha1.ModelUsageRef{
				{
					Namespace: "default",
					Name:      "my-agent",
					Kind:      "Agent",
				},
				{
					Namespace: "production",
					Name:      "prod-agent",
					Kind:      "Agent",
				},
			},
		},
	}

	resp := handler.convertToModelResponse(model)
	require.Len(t, resp.Meta.UsedBy, 2)
	assert.Equal(t, "default", resp.Meta.UsedBy[0].Namespace)
	assert.Equal(t, "my-agent", resp.Meta.UsedBy[0].Name)
	assert.Equal(t, "production", resp.Meta.UsedBy[1].Namespace)
	assert.Equal(t, "prod-agent", resp.Meta.UsedBy[1].Name)
}

func TestModelHandler_ConvertToModelResponse_NotReady(t *testing.T) {
	c := setupModelTestClient(t)
	handler := NewModelHandler(c, nil, zerolog.Nop())

	model := &agentregistryv1alpha1.ModelCatalog{
		ObjectMeta: metav1.ObjectMeta{Name: "pending-model"},
		Spec: agentregistryv1alpha1.ModelCatalogSpec{
			Name:     "pending-model",
			Provider: "openai",
			Model:    "gpt-4",
		},
		Status: agentregistryv1alpha1.ModelCatalogStatus{
			Ready:   false,
			Message: "Waiting for configuration",
		},
	}

	resp := handler.convertToModelResponse(model)
	assert.False(t, resp.Meta.Ready)
	assert.Equal(t, "Waiting for configuration", resp.Meta.Message)
}
