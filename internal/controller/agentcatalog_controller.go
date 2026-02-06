package controller

import (
	"context"

	"github.com/rs/zerolog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

const usedByCleanupFinalizer = "agentregistry.dev/usedby-cleanup"

// AgentCatalogReconciler reconciles an AgentCatalog object
type AgentCatalogReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger zerolog.Logger
}

// +kubebuilder:rbac:groups=agentregistry.dev,resources=agentcatalogs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentregistry.dev,resources=agentcatalogs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agentregistry.dev,resources=agentcatalogs/finalizers,verbs=update
// +kubebuilder:rbac:groups=agentregistry.dev,resources=mcpservercatalogs,verbs=get;list;watch
// +kubebuilder:rbac:groups=agentregistry.dev,resources=mcpservercatalogs/status,verbs=get;update;patch

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

	// Handle deletion: clean up UsedBy refs and remove finalizer
	if !agent.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&agent, usedByCleanupFinalizer) {
			if err := r.handleAgentDeletion(ctx, &agent, logger); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Ensure finalizer is present
	if !controllerutil.ContainsFinalizer(&agent, usedByCleanupFinalizer) {
		controllerutil.AddFinalizer(&agent, usedByCleanupFinalizer)
		if err := r.Update(ctx, &agent); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, err
		}
	}

	// Update isLatest status for all versions of this agent
	if err := r.updateLatestVersion(ctx, &agent); err != nil {
		logger.Error().Err(err).Msg("failed to update latest version")
		return ctrl.Result{}, err
	}

	// Update MCP server UsedBy references
	referencedServers := extractReferencedMCPServerCatalogNames(&agent)
	if err := r.updateMCPServerUsedBy(ctx, &agent, referencedServers, logger); err != nil {
		logger.Error().Err(err).Msg("failed to update MCP server UsedBy")
		return ctrl.Result{}, err
	}
	if err := r.cleanupStaleUsedByRefs(ctx, &agent, referencedServers, logger); err != nil {
		logger.Error().Err(err).Msg("failed to cleanup stale UsedBy refs")
		return ctrl.Result{}, err
	}

	// Update observed generation
	if agent.Status.ObservedGeneration != agent.Generation {
		agent.Status.ObservedGeneration = agent.Generation
		if err := r.Status().Update(ctx, &agent); err != nil {
			if apierrors.IsConflict(err) {
				logger.Debug().Msg("conflict updating status, will retry")
				return ctrl.Result{Requeue: true}, nil
			}
			logger.Error().Err(err).Msg("failed to update status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// extractReferencedMCPServerCatalogNames returns the set of registry server names
// referenced by an AgentCatalog's McpServers with Type=="registry".
func extractReferencedMCPServerCatalogNames(agent *agentregistryv1alpha1.AgentCatalog) map[string]struct{} {
	refs := make(map[string]struct{})
	for _, mcp := range agent.Spec.McpServers {
		if mcp.Type == "registry" && mcp.RegistryServerName != "" {
			refs[mcp.RegistryServerName] = struct{}{}
		}
	}
	return refs
}

// updateMCPServerUsedBy ensures this agent is listed in UsedBy for each referenced MCPServerCatalog.
func (r *AgentCatalogReconciler) updateMCPServerUsedBy(
	ctx context.Context,
	agent *agentregistryv1alpha1.AgentCatalog,
	referencedServers map[string]struct{},
	logger zerolog.Logger,
) error {
	ref := agentregistryv1alpha1.MCPServerUsageRef{
		Namespace: agent.Namespace,
		Name:      agent.Name,
		Kind:      "AgentCatalog",
	}

	for serverName := range referencedServers {
		var serverList agentregistryv1alpha1.MCPServerCatalogList
		if err := r.List(ctx, &serverList, client.MatchingFields{
			IndexMCPServerName: serverName,
		}); err != nil {
			return err
		}

		for i := range serverList.Items {
			server := &serverList.Items[i]
			if containsUsageRef(server.Status.UsedBy, ref) {
				continue
			}
			server.Status.UsedBy = append(server.Status.UsedBy, ref)
			if err := r.Status().Update(ctx, server); err != nil {
				if apierrors.IsConflict(err) {
					logger.Debug().Str("server", server.Name).Msg("conflict updating MCPServerCatalog UsedBy, will retry")
					return err
				}
				return err
			}
			logger.Debug().Str("server", server.Name).Msg("added agent to MCPServerCatalog UsedBy")
		}
	}
	return nil
}

// cleanupStaleUsedByRefs removes this agent from MCPServerCatalogs that are no longer referenced.
func (r *AgentCatalogReconciler) cleanupStaleUsedByRefs(
	ctx context.Context,
	agent *agentregistryv1alpha1.AgentCatalog,
	currentRefs map[string]struct{},
	logger zerolog.Logger,
) error {
	ref := agentregistryv1alpha1.MCPServerUsageRef{
		Namespace: agent.Namespace,
		Name:      agent.Name,
		Kind:      "AgentCatalog",
	}

	var allServers agentregistryv1alpha1.MCPServerCatalogList
	if err := r.List(ctx, &allServers); err != nil {
		return err
	}

	for i := range allServers.Items {
		server := &allServers.Items[i]
		if !containsUsageRef(server.Status.UsedBy, ref) {
			continue
		}
		// If this server's spec.name is still referenced, keep the ref
		if _, ok := currentRefs[server.Spec.Name]; ok {
			continue
		}
		// Remove the stale ref
		server.Status.UsedBy = removeUsageRef(server.Status.UsedBy, ref)
		if err := r.Status().Update(ctx, server); err != nil {
			if apierrors.IsConflict(err) {
				logger.Debug().Str("server", server.Name).Msg("conflict cleaning up stale UsedBy, will retry")
				return err
			}
			return err
		}
		logger.Debug().Str("server", server.Name).Msg("removed stale agent ref from MCPServerCatalog UsedBy")
	}
	return nil
}

// handleAgentDeletion removes this agent from all MCPServerCatalog UsedBy lists and removes the finalizer.
func (r *AgentCatalogReconciler) handleAgentDeletion(
	ctx context.Context,
	agent *agentregistryv1alpha1.AgentCatalog,
	logger zerolog.Logger,
) error {
	// Clean up all UsedBy refs (pass empty set so everything is stale)
	if err := r.cleanupStaleUsedByRefs(ctx, agent, map[string]struct{}{}, logger); err != nil {
		return err
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(agent, usedByCleanupFinalizer)
	if err := r.Update(ctx, agent); err != nil {
		return err
	}

	// Re-fetch to get latest version after the update
	if err := r.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, agent); err != nil {
		return client.IgnoreNotFound(err)
	}

	return nil
}

// containsUsageRef checks if a ref is present in the slice (matches by Namespace and Name).
func containsUsageRef(refs []agentregistryv1alpha1.MCPServerUsageRef, ref agentregistryv1alpha1.MCPServerUsageRef) bool {
	for _, r := range refs {
		if r.Namespace == ref.Namespace && r.Name == ref.Name {
			return true
		}
	}
	return false
}

// removeUsageRef returns a new slice without the matching ref.
func removeUsageRef(refs []agentregistryv1alpha1.MCPServerUsageRef, ref agentregistryv1alpha1.MCPServerUsageRef) []agentregistryv1alpha1.MCPServerUsageRef {
	result := make([]agentregistryv1alpha1.MCPServerUsageRef, 0, len(refs))
	for _, r := range refs {
		if r.Namespace == ref.Namespace && r.Name == ref.Name {
			continue
		}
		result = append(result, r)
	}
	return result
}

// updateLatestVersion determines and updates the latest version flag for all versions of an agent
func (r *AgentCatalogReconciler) updateLatestVersion(ctx context.Context, agent *agentregistryv1alpha1.AgentCatalog) error {
	return updateLatestVersionForAgents(ctx, r.Client, agent.Spec.Name)
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentCatalogReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&agentregistryv1alpha1.AgentCatalog{}).
		Complete(r)
}
