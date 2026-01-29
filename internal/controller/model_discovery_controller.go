package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	kagentv1alpha2 "github.com/kagent-dev/kagent/go/api/v1alpha2"
)

// ModelDiscoveryReconciler watches KAgent ModelConfig resources and creates catalog entries
type ModelDiscoveryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger zerolog.Logger
}

const (
	modelDiscoveryLabel = "agentregistry.dev/model-discovered"
	modelSourceLabel    = "agentregistry.dev/model-source"
)

// +kubebuilder:rbac:groups=kagent.dev,resources=modelconfigs,verbs=get;list;watch

// Reconcile watches ModelConfig and creates/updates ModelCatalog
func (r *ModelDiscoveryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.With().
		Str("modelconfig", req.Name).
		Str("namespace", req.Namespace).
		Logger()

	// Fetch the ModelConfig
	var modelConfig kagentv1alpha2.ModelConfig
	if err := r.Get(ctx, req.NamespacedName, &modelConfig); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info().Msg("ModelConfig deleted, catalog entry preserved")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger.Info().
		Str("provider", string(modelConfig.Spec.Provider)).
		Str("model", modelConfig.Spec.Model).
		Msg("discovered ModelConfig, syncing to catalog")

	// Generate catalog entry name
	catalogName := generateModelCatalogName(modelConfig.Namespace, modelConfig.Name)

	// Check if catalog entry exists
	var catalog agentregistryv1alpha1.ModelCatalog
	err := r.Get(ctx, types.NamespacedName{Name: catalogName}, &catalog)

	if apierrors.IsNotFound(err) {
		// Create new catalog entry
		catalog = r.buildCatalogFromModelConfig(&modelConfig, catalogName)
		if err := r.Create(ctx, &catalog); err != nil {
			logger.Error().Err(err).Msg("failed to create model catalog entry")
			return ctrl.Result{}, err
		}
		logger.Info().Str("catalog", catalogName).Msg("created model catalog entry from ModelConfig")
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// Update status from ModelConfig
	r.syncStatusFromModelConfig(&catalog, &modelConfig)

	if err := r.Status().Update(ctx, &catalog); err != nil {
		logger.Error().Err(err).Msg("failed to update model catalog status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// buildCatalogFromModelConfig creates a ModelCatalog from KAgent ModelConfig
func (r *ModelDiscoveryReconciler) buildCatalogFromModelConfig(mc *kagentv1alpha2.ModelConfig, catalogName string) agentregistryv1alpha1.ModelCatalog {
	// Extract base URL based on provider
	baseURL := ""
	if mc.Spec.OpenAI != nil && mc.Spec.OpenAI.BaseURL != "" {
		baseURL = mc.Spec.OpenAI.BaseURL
	} else if mc.Spec.Anthropic != nil && mc.Spec.Anthropic.BaseURL != "" {
		baseURL = mc.Spec.Anthropic.BaseURL
	} else if mc.Spec.Ollama != nil && mc.Spec.Ollama.Host != "" {
		baseURL = mc.Spec.Ollama.Host
	} else if mc.Spec.AzureOpenAI != nil && mc.Spec.AzureOpenAI.Endpoint != "" {
		baseURL = mc.Spec.AzureOpenAI.Endpoint
	}

	return agentregistryv1alpha1.ModelCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: catalogName,
			Labels: map[string]string{
				modelDiscoveryLabel: "true",
				modelSourceLabel:    "ModelConfig",
				sourceNameLabel:     mc.Name,
				sourceNSLabel:       mc.Namespace,
			},
		},
		Spec: agentregistryv1alpha1.ModelCatalogSpec{
			Name:     fmt.Sprintf("%s/%s", mc.Namespace, mc.Name),
			Provider: string(mc.Spec.Provider),
			Model:    mc.Spec.Model,
			BaseURL:  baseURL,
			SourceRef: &agentregistryv1alpha1.SourceReference{
				Kind:      "ModelConfig",
				Name:      mc.Name,
				Namespace: mc.Namespace,
			},
		},
	}
}

// syncStatusFromModelConfig updates catalog status from ModelConfig status
func (r *ModelDiscoveryReconciler) syncStatusFromModelConfig(catalog *agentregistryv1alpha1.ModelCatalog, mc *kagentv1alpha2.ModelConfig) {
	ready := false
	message := ""

	for _, cond := range mc.Status.Conditions {
		if cond.Type == kagentv1alpha2.ModelConfigConditionTypeAccepted {
			ready = cond.Status == metav1.ConditionTrue
			message = cond.Message
			break
		}
	}

	catalog.Status.Ready = ready
	catalog.Status.Message = message

	// Auto-publish discovered models
	if !catalog.Status.Published {
		now := metav1.Now()
		catalog.Status.Published = true
		catalog.Status.PublishedAt = &now
		catalog.Status.Status = agentregistryv1alpha1.CatalogStatusActive
	}
}

// generateModelCatalogName creates a valid K8s name from namespace and ModelConfig name
func generateModelCatalogName(namespace, name string) string {
	combined := fmt.Sprintf("%s-%s", namespace, name)
	combined = strings.ReplaceAll(combined, "/", "-")
	combined = strings.ReplaceAll(combined, "_", "-")
	combined = strings.ToLower(combined)
	if len(combined) > 63 {
		combined = combined[:63]
	}
	return combined
}

// SetupWithManager sets up the controller with the Manager.
func (r *ModelDiscoveryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("model-discovery").
		For(&kagentv1alpha2.ModelConfig{}).
		Watches(
			&kagentv1alpha2.ModelConfig{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				return []reconcile.Request{{
					NamespacedName: types.NamespacedName{
						Name:      obj.GetName(),
						Namespace: obj.GetNamespace(),
					},
				}}
			}),
		).
		Complete(r)
}
