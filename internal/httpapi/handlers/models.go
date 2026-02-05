package handlers

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	"github.com/agentregistry-dev/agentregistry/internal/controller"
)

// ModelHandler handles model catalog operations
type ModelHandler struct {
	client client.Client
	cache  cache.Cache
	logger zerolog.Logger
}

// NewModelHandler creates a new model handler
func NewModelHandler(c client.Client, cache cache.Cache, logger zerolog.Logger) *ModelHandler {
	return &ModelHandler{
		client: c,
		cache:  cache,
		logger: logger.With().Str("handler", "models").Logger(),
	}
}

// Model response types
type ModelJSON struct {
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	BaseURL     string `json:"baseUrl,omitempty"`
	Description string `json:"description,omitempty"`
}

type ModelUsageRefJSON struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Kind      string `json:"kind,omitempty"`
}

type ModelMeta struct {
	Official *OfficialMeta       `json:"io.modelcontextprotocol.registry/official,omitempty"`
	UsedBy   []ModelUsageRefJSON `json:"usedBy,omitempty"`
	Ready    bool                `json:"ready"`
	Message  string              `json:"message,omitempty"`
}

type ModelResponse struct {
	Model ModelJSON `json:"model"`
	Meta  ModelMeta `json:"_meta"`
}

type ModelListResponse struct {
	Models   []ModelResponse `json:"models"`
	Metadata ListMetadata    `json:"metadata"`
}

// Input types
type ListModelsInput struct {
	Cursor   string `query:"cursor" json:"cursor,omitempty"`
	Limit    int    `query:"limit" json:"limit,omitempty" default:"30" minimum:"1" maximum:"100"`
	Search   string `query:"search" json:"search,omitempty"`
	Provider string `query:"provider" json:"provider,omitempty"`
}

type ModelDetailInput struct {
	ModelName string `path:"modelName" json:"modelName"`
}

type CreateModelInput struct {
	Body ModelJSON
}

// RegisterRoutes registers model endpoints
func (h *ModelHandler) RegisterRoutes(api huma.API, pathPrefix string, isAdmin bool) {
	tags := []string{"models"}
	if isAdmin {
		tags = append(tags, "admin")
	}

	// List models
	huma.Register(api, huma.Operation{
		OperationID: "list-models" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodGet,
		Path:        pathPrefix + "/models",
		Summary:     "List models",
		Tags:        tags,
	}, func(ctx context.Context, input *ListModelsInput) (*Response[ModelListResponse], error) {
		return h.listModels(ctx, input, isAdmin)
	})

	// Get model by name
	huma.Register(api, huma.Operation{
		OperationID: "get-model" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodGet,
		Path:        pathPrefix + "/models/{modelName}",
		Summary:     "Get model details",
		Tags:        tags,
	}, func(ctx context.Context, input *ModelDetailInput) (*Response[ModelResponse], error) {
		return h.getModel(ctx, input, isAdmin)
	})

	// Admin-only endpoints
	if isAdmin {
		// Create model
		huma.Register(api, huma.Operation{
			OperationID: "create-model" + strings.ReplaceAll(pathPrefix, "/", "-"),
			Method:      http.MethodPost,
			Path:        pathPrefix + "/models",
			Summary:     "Create model",
			Tags:        tags,
		}, func(ctx context.Context, input *CreateModelInput) (*Response[ModelResponse], error) {
			return h.createModel(ctx, input)
		})

		// Delete model
		huma.Register(api, huma.Operation{
			OperationID: "delete-model" + strings.ReplaceAll(pathPrefix, "/", "-"),
			Method:      http.MethodDelete,
			Path:        pathPrefix + "/models/{modelName}",
			Summary:     "Delete model",
			Tags:        tags,
		}, func(ctx context.Context, input *ModelDetailInput) (*Response[EmptyResponse], error) {
			return h.deleteModel(ctx, input)
		})
	}
}

func (h *ModelHandler) listModels(ctx context.Context, input *ListModelsInput, isAdmin bool) (*Response[ModelListResponse], error) {
	var modelList agentregistryv1alpha1.ModelCatalogList

	listOpts := []client.ListOption{}

	if err := h.cache.List(ctx, &modelList, listOpts...); err != nil {
		return nil, huma.Error500InternalServerError("Failed to list models", err)
	}

	models := make([]ModelResponse, 0, len(modelList.Items))
	for _, m := range modelList.Items {
		if input.Search != "" && !strings.Contains(strings.ToLower(m.Spec.Name), strings.ToLower(input.Search)) {
			continue
		}

		if input.Provider != "" && !strings.EqualFold(m.Spec.Provider, input.Provider) {
			continue
		}

		models = append(models, h.convertToModelResponse(&m))
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 30
	}
	if len(models) > limit {
		models = models[:limit]
	}

	return &Response[ModelListResponse]{
		Body: ModelListResponse{
			Models: models,
			Metadata: ListMetadata{
				Count: len(models),
			},
		},
	}, nil
}

func (h *ModelHandler) getModel(ctx context.Context, input *ModelDetailInput, isAdmin bool) (*Response[ModelResponse], error) {
	modelName, err := url.PathUnescape(input.ModelName)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid model name encoding", err)
	}

	var modelList agentregistryv1alpha1.ModelCatalogList
	listOpts := []client.ListOption{
		client.MatchingFields{
			controller.IndexModelName: modelName,
		},
	}

	if err := h.cache.List(ctx, &modelList, listOpts...); err != nil {
		return nil, huma.Error500InternalServerError("Failed to get model", err)
	}

	if len(modelList.Items) == 0 {
		return nil, huma.Error404NotFound("Model not found")
	}

	return &Response[ModelResponse]{
		Body: h.convertToModelResponse(&modelList.Items[0]),
	}, nil
}

func (h *ModelHandler) createModel(ctx context.Context, input *CreateModelInput) (*Response[ModelResponse], error) {
	crName := SanitizeK8sName(input.Body.Name)

	model := &agentregistryv1alpha1.ModelCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: crName,
			Labels: map[string]string{
				"agentregistry.dev/name": SanitizeK8sName(input.Body.Name),
			},
		},
		Spec: agentregistryv1alpha1.ModelCatalogSpec{
			Name:        input.Body.Name,
			Provider:    input.Body.Provider,
			Model:       input.Body.Model,
			BaseURL:     input.Body.BaseURL,
			Description: input.Body.Description,
		},
	}

	if err := h.client.Create(ctx, model); err != nil {
		return nil, huma.Error500InternalServerError("Failed to create model", err)
	}

	return &Response[ModelResponse]{
		Body: h.convertToModelResponse(model),
	}, nil
}

func (h *ModelHandler) deleteModel(ctx context.Context, input *ModelDetailInput) (*Response[EmptyResponse], error) {
	modelName, err := url.PathUnescape(input.ModelName)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid model name encoding", err)
	}

	crName := SanitizeK8sName(modelName)
	model := &agentregistryv1alpha1.ModelCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: crName,
		},
	}

	if err := h.client.Delete(ctx, model); err != nil {
		return nil, huma.Error500InternalServerError("Failed to delete model", err)
	}

	return &Response[EmptyResponse]{
		Body: EmptyResponse{Message: "Model deleted successfully"},
	}, nil
}

func (h *ModelHandler) convertToModelResponse(m *agentregistryv1alpha1.ModelCatalog) ModelResponse {
	model := ModelJSON{
		Name:        m.Spec.Name,
		Provider:    m.Spec.Provider,
		Model:       m.Spec.Model,
		BaseURL:     m.Spec.BaseURL,
		Description: m.Spec.Description,
	}

	var publishedAt *time.Time
	if m.Status.PublishedAt != nil {
		t := m.Status.PublishedAt.Time
		publishedAt = &t
	}

	// Convert usedBy references
	var usedBy []ModelUsageRefJSON
	for _, ref := range m.Status.UsedBy {
		usedBy = append(usedBy, ModelUsageRefJSON{
			Namespace: ref.Namespace,
			Name:      ref.Name,
			Kind:      ref.Kind,
		})
	}

	return ModelResponse{
		Model: model,
		Meta: ModelMeta{
			Official: &OfficialMeta{
				Status:      string(m.Status.Status),
				PublishedAt: publishedAt,
				UpdatedAt:   m.CreationTimestamp.Time,
				IsLatest:    true, // Models don't have versions currently
				Published:   true,
			},
			UsedBy:  usedBy,
			Ready:   m.Status.Ready,
			Message: m.Status.Message,
		},
	}
}
