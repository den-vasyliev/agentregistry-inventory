package controller

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

// MCPServerCatalogReconciler reconciles a MCPServerCatalog object
type MCPServerCatalogReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger zerolog.Logger
}

// +kubebuilder:rbac:groups=agentregistry.dev,resources=mcpservercatalogs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentregistry.dev,resources=mcpservercatalogs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agentregistry.dev,resources=mcpservercatalogs/finalizers,verbs=update
// +kubebuilder:rbac:groups=kagent.dev,resources=mcpservers,verbs=get;list;watch

// Reconcile handles MCPServerCatalog reconciliation
func (r *MCPServerCatalogReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.With().Str("name", req.Name).Str("namespace", req.Namespace).Logger()

	// Fetch the MCPServerCatalog
	var server agentregistryv1alpha1.MCPServerCatalog
	if err := r.Get(ctx, req.NamespacedName, &server); err != nil {
		// Object not found, could have been deleted
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Debug().
		Str("specName", server.Spec.Name).
		Str("version", server.Spec.Version).
		Msg("reconciling MCPServerCatalog")

	statusChanged := false

	// Sync from sourceRef only for external resources (discovered)
	// Managed resources get their status from RegistryDeployment
	if server.Spec.SourceRef != nil && server.Status.ManagementType == agentregistryv1alpha1.ManagementTypeExternal {
		if err := r.syncFromSource(ctx, &server, &statusChanged); err != nil {
			if apierrors.IsNotFound(err) {
				// Source was deleted, this is expected
				logger.Debug().Msg("source MCPServer not found, was likely deleted")
			} else {
				logger.Warn().Err(err).Msg("failed to sync from source")
			}
			// Don't fail reconciliation
		}
	}

	// Update isLatest status for all versions of this server
	if err := r.updateLatestVersion(ctx, &server); err != nil {
		logger.Error().Err(err).Msg("failed to update latest version")
		return ctrl.Result{}, err
	}

	// Update observed generation
	if server.Status.ObservedGeneration != server.Generation || statusChanged {
		server.Status.ObservedGeneration = server.Generation
		if err := r.Status().Update(ctx, &server); err != nil {
			if apierrors.IsConflict(err) {
				// Conflict means resource was modified, requeue to retry with latest version
				logger.Debug().Msg("conflict updating status, will retry")
				return ctrl.Result{Requeue: true}, nil
			}
			logger.Error().Err(err).Msg("failed to update status")
			return ctrl.Result{}, err
		}
	}

	// Requeue to periodically sync sourceRef status (only for external resources)
	if server.Spec.SourceRef != nil && server.Status.ManagementType == agentregistryv1alpha1.ManagementTypeExternal {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

// syncFromSource syncs deployment status from the referenced MCPServer
func (r *MCPServerCatalogReconciler) syncFromSource(ctx context.Context, server *agentregistryv1alpha1.MCPServerCatalog, statusChanged *bool) error {
	ref := server.Spec.SourceRef

	// Currently only support MCPServer kind
	if ref.Kind != "MCPServer" {
		return nil
	}

	// Get the MCPServer resource from discovery cache (populated by DiscoveryConfig informers)
	mcpServer, err := GetMCPServer(ctx, ref.Namespace, ref.Name)
	if err != nil {
		// Update status to reflect source not found
		now := metav1.Now()
		newDeployment := &agentregistryv1alpha1.DeploymentRef{
			Namespace:   ref.Namespace,
			Ready:       false,
			Message:     "Source MCPServer not found",
			LastChecked: &now,
		}
		if !deploymentRefEqual(server.Status.Deployment, newDeployment) {
			server.Status.Deployment = newDeployment
			*statusChanged = true
		}
		return err
	}

	// Extract status from MCPServer
	ready := false
	message := ""
	for _, cond := range mcpServer.Status.Conditions {
		if cond.Type == "Ready" {
			ready = cond.Status == "True"
			message = cond.Message
			break
		}
	}

	now := metav1.Now()
	newDeployment := &agentregistryv1alpha1.DeploymentRef{
		Namespace:   ref.Namespace,
		ServiceName: ref.Name,
		Ready:       ready,
		Message:     message,
		LastChecked: &now,
	}

	if !deploymentRefEqual(server.Status.Deployment, newDeployment) {
		server.Status.Deployment = newDeployment
		*statusChanged = true
	}

	return nil
}

// deploymentRefEqual compares two DeploymentRef pointers (ignoring LastChecked)
func deploymentRefEqual(a, b *agentregistryv1alpha1.DeploymentRef) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Namespace == b.Namespace &&
		a.ServiceName == b.ServiceName &&
		a.URL == b.URL &&
		a.Ready == b.Ready &&
		a.Message == b.Message
}

// updateLatestVersion determines and updates the latest version flag for all versions of a server
func (r *MCPServerCatalogReconciler) updateLatestVersion(ctx context.Context, server *agentregistryv1alpha1.MCPServerCatalog) error {
	return updateLatestVersionForMCPServers(ctx, r.Client, server.Spec.Name)
}

// SetupWithManager sets up the controller with the Manager.
func (r *MCPServerCatalogReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&agentregistryv1alpha1.MCPServerCatalog{}).
		Complete(r)
}
