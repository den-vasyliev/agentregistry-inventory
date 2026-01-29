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

// DeploymentHandler handles deployment operations
type DeploymentHandler struct {
	client client.Client
	cache  cache.Cache
	logger zerolog.Logger
}

// NewDeploymentHandler creates a new deployment handler
func NewDeploymentHandler(c client.Client, cache cache.Cache, logger zerolog.Logger) *DeploymentHandler {
	return &DeploymentHandler{
		client: c,
		cache:  cache,
		logger: logger.With().Str("handler", "deployments").Logger(),
	}
}

// Deployment response types
type DeploymentJSON struct {
	ResourceName string            `json:"resourceName"`
	Version      string            `json:"version"`
	ResourceType string            `json:"resourceType"`
	Runtime      string            `json:"runtime"`
	PreferRemote bool              `json:"preferRemote,omitempty"`
	Config       map[string]string `json:"config,omitempty"`
	Namespace    string            `json:"namespace,omitempty"`
	Status       string            `json:"status,omitempty"`
	DeployedAt   *time.Time        `json:"deployedAt,omitempty"`
	UpdatedAt    *time.Time        `json:"updatedAt,omitempty"`
	Message      string            `json:"message,omitempty"`
}

type DeploymentResponse struct {
	Deployment DeploymentJSON `json:"deployment"`
}

type DeploymentListResponse struct {
	Deployments []DeploymentJSON `json:"deployments"`
	Metadata    ListMetadata     `json:"metadata"`
}

// Input types
type ListDeploymentsInput struct {
	Cursor       string `query:"cursor" json:"cursor,omitempty"`
	Limit        int    `query:"limit" json:"limit,omitempty" default:"30" minimum:"1" maximum:"100"`
	ResourceType string `query:"resourceType" json:"resourceType,omitempty"`
	Runtime      string `query:"runtime" json:"runtime,omitempty"`
}

type DeploymentDetailInput struct {
	DeploymentName string `path:"deploymentName" json:"deploymentName"`
}

type CreateDeploymentInput struct {
	Body struct {
		ResourceName string            `json:"resourceName"`
		Version      string            `json:"version"`
		ResourceType string            `json:"resourceType"`
		Runtime      string            `json:"runtime"`
		PreferRemote bool              `json:"preferRemote,omitempty"`
		Config       map[string]string `json:"config,omitempty"`
		Namespace    string            `json:"namespace,omitempty"`
	}
}

type UpdateDeploymentConfigInput struct {
	DeploymentName string `path:"deploymentName" json:"deploymentName"`
	Body           struct {
		Config map[string]string `json:"config"`
	}
}

type DeleteDeploymentVersionInput struct {
	ServerName   string `path:"serverName" json:"serverName"`
	Version      string `path:"version" json:"version"`
	ResourceType string `query:"resourceType" json:"resourceType,omitempty"`
}

// RegisterRoutes registers deployment endpoints
func (h *DeploymentHandler) RegisterRoutes(api huma.API, pathPrefix string, isAdmin bool) {
	tags := []string{"deployments"}
	if isAdmin {
		tags = append(tags, "admin")
	}

	// List deployments
	huma.Register(api, huma.Operation{
		OperationID: "list-deployments" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodGet,
		Path:        pathPrefix + "/deployments",
		Summary:     "List deployments",
		Tags:        tags,
	}, func(ctx context.Context, input *ListDeploymentsInput) (*Response[DeploymentListResponse], error) {
		return h.listDeployments(ctx, input)
	})

	// Get deployment by name
	huma.Register(api, huma.Operation{
		OperationID: "get-deployment" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodGet,
		Path:        pathPrefix + "/deployments/{deploymentName}",
		Summary:     "Get deployment details",
		Tags:        tags,
	}, func(ctx context.Context, input *DeploymentDetailInput) (*Response[DeploymentResponse], error) {
		return h.getDeployment(ctx, input)
	})

	// Create deployment
	huma.Register(api, huma.Operation{
		OperationID: "create-deployment" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodPost,
		Path:        pathPrefix + "/deployments",
		Summary:     "Create deployment",
		Tags:        tags,
	}, func(ctx context.Context, input *CreateDeploymentInput) (*Response[DeploymentResponse], error) {
		return h.createDeployment(ctx, input)
	})

	// Update deployment config
	huma.Register(api, huma.Operation{
		OperationID: "update-deployment-config" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodPatch,
		Path:        pathPrefix + "/deployments/{deploymentName}/config",
		Summary:     "Update deployment configuration",
		Tags:        tags,
	}, func(ctx context.Context, input *UpdateDeploymentConfigInput) (*Response[DeploymentResponse], error) {
		return h.updateDeploymentConfig(ctx, input)
	})

	// Delete deployment by name
	huma.Register(api, huma.Operation{
		OperationID: "delete-deployment" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodDelete,
		Path:        pathPrefix + "/deployments/{deploymentName}",
		Summary:     "Delete deployment",
		Tags:        tags,
	}, func(ctx context.Context, input *DeploymentDetailInput) (*Response[EmptyResponse], error) {
		return h.deleteDeployment(ctx, input)
	})

	// Delete deployment by server name and version (UI compatibility)
	huma.Register(api, huma.Operation{
		OperationID: "delete-deployment-version" + strings.ReplaceAll(pathPrefix, "/", "-"),
		Method:      http.MethodDelete,
		Path:        pathPrefix + "/deployments/{serverName}/versions/{version}",
		Summary:     "Delete deployment by server name and version",
		Tags:        tags,
	}, func(ctx context.Context, input *DeleteDeploymentVersionInput) (*Response[EmptyResponse], error) {
		return h.deleteDeploymentVersion(ctx, input)
	})
}

func (h *DeploymentHandler) listDeployments(ctx context.Context, input *ListDeploymentsInput) (*Response[DeploymentListResponse], error) {
	var deploymentList agentregistryv1alpha1.RegistryDeploymentList

	listOpts := []client.ListOption{}

	if input.ResourceType != "" {
		listOpts = append(listOpts, client.MatchingFields{
			controller.IndexDeploymentResourceType: input.ResourceType,
		})
	}

	if input.Runtime != "" {
		listOpts = append(listOpts, client.MatchingFields{
			controller.IndexDeploymentRuntime: input.Runtime,
		})
	}

	if err := h.cache.List(ctx, &deploymentList, listOpts...); err != nil {
		return nil, huma.Error500InternalServerError("Failed to list deployments", err)
	}

	deployments := make([]DeploymentJSON, 0, len(deploymentList.Items))
	for _, d := range deploymentList.Items {
		deployments = append(deployments, h.convertToDeploymentJSON(&d))
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 30
	}
	if len(deployments) > limit {
		deployments = deployments[:limit]
	}

	return &Response[DeploymentListResponse]{
		Body: DeploymentListResponse{
			Deployments: deployments,
			Metadata: ListMetadata{
				Count: len(deployments),
			},
		},
	}, nil
}

func (h *DeploymentHandler) getDeployment(ctx context.Context, input *DeploymentDetailInput) (*Response[DeploymentResponse], error) {
	deploymentName, err := url.PathUnescape(input.DeploymentName)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid deployment name encoding", err)
	}

	var deployment agentregistryv1alpha1.RegistryDeployment
	if err := h.cache.Get(ctx, client.ObjectKey{Name: deploymentName}, &deployment); err != nil {
		return nil, huma.Error404NotFound("Deployment not found")
	}

	return &Response[DeploymentResponse]{
		Body: DeploymentResponse{
			Deployment: h.convertToDeploymentJSON(&deployment),
		},
	}, nil
}

func (h *DeploymentHandler) createDeployment(ctx context.Context, input *CreateDeploymentInput) (*Response[DeploymentResponse], error) {
	crName := GenerateCRName(input.Body.ResourceName, input.Body.Version)

	// Always use kubernetes runtime
	runtime := agentregistryv1alpha1.RuntimeTypeKubernetes

	deployment := &agentregistryv1alpha1.RegistryDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: crName,
			Labels: map[string]string{
				"agentregistry.dev/resource-name": SanitizeK8sName(input.Body.ResourceName),
				"agentregistry.dev/version":       SanitizeK8sName(input.Body.Version),
				"agentregistry.dev/resource-type": input.Body.ResourceType,
				"agentregistry.dev/runtime":       string(runtime),
			},
		},
		Spec: agentregistryv1alpha1.RegistryDeploymentSpec{
			ResourceName: input.Body.ResourceName,
			Version:      input.Body.Version,
			ResourceType: agentregistryv1alpha1.ResourceType(input.Body.ResourceType),
			Runtime:      runtime,
			PreferRemote: input.Body.PreferRemote,
			Config:       input.Body.Config,
			Namespace:    input.Body.Namespace,
		},
	}

	if err := h.client.Create(ctx, deployment); err != nil {
		return nil, huma.Error500InternalServerError("Failed to create deployment", err)
	}

	// Set initial status
	now := metav1.Now()
	deployment.Status.Phase = agentregistryv1alpha1.DeploymentPhasePending
	deployment.Status.DeployedAt = &now
	if err := h.client.Status().Update(ctx, deployment); err != nil {
		h.logger.Error().Err(err).Msg("failed to update deployment status")
	}

	return &Response[DeploymentResponse]{
		Body: DeploymentResponse{
			Deployment: h.convertToDeploymentJSON(deployment),
		},
	}, nil
}

func (h *DeploymentHandler) updateDeploymentConfig(ctx context.Context, input *UpdateDeploymentConfigInput) (*Response[DeploymentResponse], error) {
	deploymentName, err := url.PathUnescape(input.DeploymentName)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid deployment name encoding", err)
	}

	var deployment agentregistryv1alpha1.RegistryDeployment
	if err := h.client.Get(ctx, client.ObjectKey{Name: deploymentName}, &deployment); err != nil {
		return nil, huma.Error404NotFound("Deployment not found")
	}

	// Merge config
	if deployment.Spec.Config == nil {
		deployment.Spec.Config = make(map[string]string)
	}
	for k, v := range input.Body.Config {
		deployment.Spec.Config[k] = v
	}

	if err := h.client.Update(ctx, &deployment); err != nil {
		return nil, huma.Error500InternalServerError("Failed to update deployment", err)
	}

	return &Response[DeploymentResponse]{
		Body: DeploymentResponse{
			Deployment: h.convertToDeploymentJSON(&deployment),
		},
	}, nil
}

func (h *DeploymentHandler) deleteDeployment(ctx context.Context, input *DeploymentDetailInput) (*Response[EmptyResponse], error) {
	deploymentName, err := url.PathUnescape(input.DeploymentName)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid deployment name encoding", err)
	}

	deployment := &agentregistryv1alpha1.RegistryDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: deploymentName,
		},
	}

	if err := h.client.Delete(ctx, deployment); err != nil {
		return nil, huma.Error500InternalServerError("Failed to delete deployment", err)
	}

	return &Response[EmptyResponse]{
		Body: EmptyResponse{Message: "Deployment deleted successfully"},
	}, nil
}

func (h *DeploymentHandler) deleteDeploymentVersion(ctx context.Context, input *DeleteDeploymentVersionInput) (*Response[EmptyResponse], error) {
	serverName, err := url.PathUnescape(input.ServerName)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid server name encoding", err)
	}
	version, err := url.PathUnescape(input.Version)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid version encoding", err)
	}

	// Find the deployment by resource name, version, and optionally resource type
	var deploymentList agentregistryv1alpha1.RegistryDeploymentList
	listOpts := []client.ListOption{
		client.MatchingFields{
			controller.IndexDeploymentResourceName: serverName,
		},
	}

	if err := h.cache.List(ctx, &deploymentList, listOpts...); err != nil {
		return nil, huma.Error500InternalServerError("Failed to find deployment", err)
	}

	for _, d := range deploymentList.Items {
		if d.Spec.Version == version {
			// If resource type specified, match it
			if input.ResourceType != "" && string(d.Spec.ResourceType) != input.ResourceType {
				continue
			}

			if err := h.client.Delete(ctx, &d); err != nil {
				return nil, huma.Error500InternalServerError("Failed to delete deployment", err)
			}

			return &Response[EmptyResponse]{
				Body: EmptyResponse{Message: "Deployment deleted successfully"},
			}, nil
		}
	}

	return nil, huma.Error404NotFound("Deployment not found")
}

func (h *DeploymentHandler) convertToDeploymentJSON(d *agentregistryv1alpha1.RegistryDeployment) DeploymentJSON {
	deployment := DeploymentJSON{
		ResourceName: d.Spec.ResourceName,
		Version:      d.Spec.Version,
		ResourceType: string(d.Spec.ResourceType),
		Runtime:      string(d.Spec.Runtime),
		PreferRemote: d.Spec.PreferRemote,
		Config:       d.Spec.Config,
		Namespace:    d.Spec.Namespace,
		Status:       string(d.Status.Phase),
		Message:      d.Status.Message,
	}

	if d.Status.DeployedAt != nil {
		t := d.Status.DeployedAt.Time
		deployment.DeployedAt = &t
	}

	if d.Status.UpdatedAt != nil {
		t := d.Status.UpdatedAt.Time
		deployment.UpdatedAt = &t
	}

	return deployment
}
