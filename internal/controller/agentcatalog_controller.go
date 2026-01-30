package controller

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

// AgentCatalogReconciler reconciles an AgentCatalog object
type AgentCatalogReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger zerolog.Logger
}

// +kubebuilder:rbac:groups=agentregistry.dev,resources=agentcatalogs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentregistry.dev,resources=agentcatalogs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agentregistry.dev,resources=agentcatalogs/finalizers,verbs=update

// Reconcile handles AgentCatalog reconciliation
func (r *AgentCatalogReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.With().Str("name", req.Name).Str("namespace", req.Namespace).Logger()

	// Fetch the AgentCatalog
	var agent agentregistryv1alpha1.AgentCatalog
	if err := r.Get(ctx, req.NamespacedName, &agent); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Debug().
		Str("specName", agent.Spec.Name).
		Str("version", agent.Spec.Version).
		Msg("reconciling AgentCatalog")

	// Update isLatest status for all versions of this agent
	if err := r.updateLatestVersion(ctx, &agent); err != nil {
		logger.Error().Err(err).Msg("failed to update latest version")
		return ctrl.Result{}, err
	}

	// Update observed generation
	if agent.Status.ObservedGeneration != agent.Generation {
		agent.Status.ObservedGeneration = agent.Generation
		if err := r.Status().Update(ctx, &agent); err != nil {
			if apierrors.IsConflict(err) {
				// Conflict means resource was modified, requeue to retry with latest version
				logger.Debug().Msg("conflict updating status, will retry")
				return ctrl.Result{Requeue: true}, nil
			}
			logger.Error().Err(err).Msg("failed to update status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// updateLatestVersion determines and updates the latest version flag for all versions of an agent
func (r *AgentCatalogReconciler) updateLatestVersion(ctx context.Context, agent *agentregistryv1alpha1.AgentCatalog) error {
	// Get all versions of this agent
	var agentList agentregistryv1alpha1.AgentCatalogList
	if err := r.List(ctx, &agentList, client.MatchingFields{
		IndexAgentName: agent.Spec.Name,
	}); err != nil {
		return err
	}

	// Find the latest version among published agents
	var latestAgent *agentregistryv1alpha1.AgentCatalog
	var latestTimestamp time.Time

	for i := range agentList.Items {
		a := &agentList.Items[i]
		if !a.Status.Published {
			continue
		}

		if latestAgent == nil {
			latestAgent = a
			if a.Status.PublishedAt != nil {
				latestTimestamp = a.Status.PublishedAt.Time
			}
			continue
		}

		var aTimestamp time.Time
		if a.Status.PublishedAt != nil {
			aTimestamp = a.Status.PublishedAt.Time
		}

		cmp := compareVersions(a.Spec.Version, latestAgent.Spec.Version, aTimestamp, latestTimestamp)
		if cmp > 0 {
			latestAgent = a
			latestTimestamp = aTimestamp
		}
	}

	// Update isLatest flag for all versions
	for i := range agentList.Items {
		a := &agentList.Items[i]
		shouldBeLatest := latestAgent != nil && a.Name == latestAgent.Name && a.Status.Published

		if a.Status.IsLatest != shouldBeLatest {
			a.Status.IsLatest = shouldBeLatest
			if err := r.Status().Update(ctx, a); err != nil {
				return err
			}
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentCatalogReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&agentregistryv1alpha1.AgentCatalog{}).
		Complete(r)
}
