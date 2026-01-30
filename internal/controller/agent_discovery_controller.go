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

// AgentDiscoveryReconciler watches KAgent Agent resources and creates catalog entries
type AgentDiscoveryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger zerolog.Logger
}

// +kubebuilder:rbac:groups=kagent.dev,resources=agents,verbs=get;list;watch

// Reconcile watches Agent and creates/updates AgentCatalog
func (r *AgentDiscoveryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.With().
		Str("agent", req.Name).
		Str("namespace", req.Namespace).
		Logger()

	// Fetch the Agent
	var agent kagentv1alpha2.Agent
	if err := r.Get(ctx, req.NamespacedName, &agent); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info().Msg("Agent deleted, catalog entry preserved")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger.Debug().Msg("discovered Agent, syncing to catalog")

	// Generate catalog entry name
	catalogName := generateAgentCatalogName(agent.Namespace, agent.Name)

	// Check if catalog entry exists
	var catalog agentregistryv1alpha1.AgentCatalog
	err := r.Get(ctx, types.NamespacedName{Name: catalogName}, &catalog)

	if apierrors.IsNotFound(err) {
		// Create new catalog entry
		catalog = r.buildCatalogFromAgent(&agent, catalogName)
		if err := r.Create(ctx, &catalog); err != nil {
			logger.Error().Err(err).Msg("failed to create catalog entry")
			return ctrl.Result{}, err
		}
		logger.Info().Str("catalog", catalogName).Msg("created catalog entry from Agent")
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// Update status from Agent
	r.syncStatusFromAgent(&catalog, &agent)

	if err := r.Status().Update(ctx, &catalog); err != nil {
		if apierrors.IsConflict(err) {
			// Conflict means resource was modified, requeue to retry with latest version
			logger.Debug().Msg("conflict updating catalog status, will retry")
			return ctrl.Result{Requeue: true}, nil
		}
		logger.Error().Err(err).Msg("failed to update catalog status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// buildCatalogFromAgent creates an AgentCatalog from KAgent Agent
func (r *AgentDiscoveryReconciler) buildCatalogFromAgent(agent *kagentv1alpha2.Agent, catalogName string) agentregistryv1alpha1.AgentCatalog {
	// Extract version from labels or use "latest"
	version := "latest"
	if v, ok := agent.Labels["app.kubernetes.io/version"]; ok {
		version = v
	}

	// Get description from spec or annotations
	description := agent.Spec.Description
	if description == "" {
		if d, ok := agent.Annotations["kagent.dev/description"]; ok {
			description = d
		}
	}

	// Get image from agent spec based on type
	image := ""
	modelConfig := ""
	if agent.Spec.Type == kagentv1alpha2.AgentType_BYO && agent.Spec.BYO != nil && agent.Spec.BYO.Deployment != nil {
		image = agent.Spec.BYO.Deployment.Image
	} else if agent.Spec.Type == kagentv1alpha2.AgentType_Declarative && agent.Spec.Declarative != nil {
		modelConfig = agent.Spec.Declarative.ModelConfig
	}

	return agentregistryv1alpha1.AgentCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: catalogName,
			Labels: map[string]string{
				discoveryLabel:  "true",
				sourceKindLabel: "Agent",
				sourceNameLabel: agent.Name,
				sourceNSLabel:   agent.Namespace,
			},
		},
		Spec: agentregistryv1alpha1.AgentCatalogSpec{
			Name:        fmt.Sprintf("%s/%s", agent.Namespace, agent.Name),
			Version:     version,
			Title:       agent.Name,
			Description: description,
			Image:       image,
			Framework:   string(agent.Spec.Type),
			ModelName:   modelConfig, // Store model config name
		},
	}
}

// syncStatusFromAgent updates catalog status from Agent status
func (r *AgentDiscoveryReconciler) syncStatusFromAgent(catalog *agentregistryv1alpha1.AgentCatalog, agent *kagentv1alpha2.Agent) {
	ready := false
	message := ""

	for _, cond := range agent.Status.Conditions {
		if cond.Type == "Ready" {
			ready = cond.Status == metav1.ConditionTrue
			message = cond.Message
			break
		}
	}

	now := metav1.Now()
	catalog.Status.Deployment = &agentregistryv1alpha1.DeploymentRef{
		Namespace:   agent.Namespace,
		ServiceName: agent.Name,
		Ready:       ready,
		Message:     message,
		LastChecked: &now,
	}

	// Auto-publish discovered agents
	if !catalog.Status.Published {
		catalog.Status.Published = true
		catalog.Status.PublishedAt = &now
		catalog.Status.Status = agentregistryv1alpha1.CatalogStatusActive
	}
}

// generateAgentCatalogName creates a valid K8s name from namespace and Agent name
func generateAgentCatalogName(namespace, name string) string {
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
func (r *AgentDiscoveryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kagentv1alpha2.Agent{}).
		Watches(
			&kagentv1alpha2.Agent{},
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
