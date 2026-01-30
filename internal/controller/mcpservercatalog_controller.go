package controller

import (
	"context"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/mod/semver"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	kmcpv1alpha1 "github.com/kagent-dev/kmcp/api/v1alpha1"
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

	// Sync from sourceRef if present
	if server.Spec.SourceRef != nil {
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

	// Requeue to periodically sync sourceRef status
	if server.Spec.SourceRef != nil {
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

	// Get the MCPServer resource
	var mcpServer kmcpv1alpha1.MCPServer
	if err := r.Get(ctx, types.NamespacedName{
		Name:      ref.Name,
		Namespace: ref.Namespace,
	}, &mcpServer); err != nil {
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
	// Get all versions of this server
	var serverList agentregistryv1alpha1.MCPServerCatalogList
	if err := r.List(ctx, &serverList, client.MatchingFields{
		IndexMCPServerName: server.Spec.Name,
	}); err != nil {
		return err
	}

	// Find the latest version among published servers
	var latestServer *agentregistryv1alpha1.MCPServerCatalog
	var latestTimestamp time.Time

	for i := range serverList.Items {
		s := &serverList.Items[i]
		if !s.Status.Published {
			continue
		}

		if latestServer == nil {
			latestServer = s
			if s.Status.PublishedAt != nil {
				latestTimestamp = s.Status.PublishedAt.Time
			}
			continue
		}

		var sTimestamp time.Time
		if s.Status.PublishedAt != nil {
			sTimestamp = s.Status.PublishedAt.Time
		}

		cmp := compareVersions(s.Spec.Version, latestServer.Spec.Version, sTimestamp, latestTimestamp)
		if cmp > 0 {
			latestServer = s
			latestTimestamp = sTimestamp
		}
	}

	// Update isLatest flag for all versions
	for i := range serverList.Items {
		s := &serverList.Items[i]
		shouldBeLatest := latestServer != nil && s.Name == latestServer.Name && s.Status.Published

		if s.Status.IsLatest != shouldBeLatest {
			s.Status.IsLatest = shouldBeLatest
			if err := r.Status().Update(ctx, s); err != nil {
				return err
			}
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MCPServerCatalogReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&agentregistryv1alpha1.MCPServerCatalog{}).
		Complete(r)
}

// Version comparison utilities (copied from internal/registry/service/versioning.go)

// isSemanticVersion checks if a version string follows semantic versioning format
func isSemanticVersion(version string) bool {
	versionWithV := ensureVPrefix(version)
	if !semver.IsValid(versionWithV) {
		return false
	}

	versionCore := strings.TrimPrefix(versionWithV, "v")
	if idx := strings.Index(versionCore, "-"); idx != -1 {
		versionCore = versionCore[:idx]
	}
	if idx := strings.Index(versionCore, "+"); idx != -1 {
		versionCore = versionCore[:idx]
	}

	parts := strings.Split(versionCore, ".")
	return len(parts) == 3
}

// ensureVPrefix adds a "v" prefix if not present
func ensureVPrefix(version string) string {
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

// compareSemanticVersions compares two semantic version strings
func compareSemanticVersions(version1 string, version2 string) int {
	v1 := ensureVPrefix(version1)
	v2 := ensureVPrefix(version2)
	return semver.Compare(v1, v2)
}

// compareVersions implements the versioning strategy:
// 1. If both versions are valid semver, use semantic version comparison
// 2. If neither are valid semver, use publication timestamp
// 3. If one is semver and one is not, the semver version is always considered higher
func compareVersions(version1 string, version2 string, timestamp1 time.Time, timestamp2 time.Time) int {
	isSemver1 := isSemanticVersion(version1)
	isSemver2 := isSemanticVersion(version2)

	if isSemver1 && isSemver2 {
		return compareSemanticVersions(version1, version2)
	}

	if !isSemver1 && !isSemver2 {
		if timestamp1.Before(timestamp2) {
			return -1
		} else if timestamp1.After(timestamp2) {
			return 1
		}
		return 0
	}

	if isSemver1 && !isSemver2 {
		return 1
	}
	return -1
}
