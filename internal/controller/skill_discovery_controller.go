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

// SkillDiscoveryReconciler watches KAgent Agent resources and creates skill catalog entries
type SkillDiscoveryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger zerolog.Logger
}

const (
	skillDiscoveryLabel  = "agentregistry.dev/skill-discovered"
	skillSourceLabel     = "agentregistry.dev/skill-source"
)

// +kubebuilder:rbac:groups=kagent.dev,resources=agents,verbs=get;list;watch

// Reconcile watches Agents and creates/updates SkillCatalog entries from their skill refs
func (r *SkillDiscoveryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.With().
		Str("agent", req.Name).
		Str("namespace", req.Namespace).
		Logger()

	// Fetch the Agent
	var agent kagentv1alpha2.Agent
	if err := r.Get(ctx, req.NamespacedName, &agent); err != nil {
		if apierrors.IsNotFound(err) {
			// Agent deleted - update skill usage tracking
			logger.Info().Msg("Agent deleted, updating skill usage")
			return r.removeAgentFromSkillUsage(ctx, req.Namespace, req.Name)
		}
		return ctrl.Result{}, err
	}

	// Check if agent has skills
	if agent.Spec.Skills == nil || len(agent.Spec.Skills.Refs) == 0 {
		return ctrl.Result{}, nil
	}

	logger.Info().Int("skillCount", len(agent.Spec.Skills.Refs)).Msg("discovered Agent with skills, syncing to catalog")

	// Process each skill ref
	for _, skillRef := range agent.Spec.Skills.Refs {
		if err := r.ensureSkillCatalog(ctx, skillRef, &agent); err != nil {
			logger.Error().Err(err).Str("skill", skillRef).Msg("failed to ensure skill catalog")
			// Continue with other skills
		}
	}

	return ctrl.Result{}, nil
}

// ensureSkillCatalog creates or updates a SkillCatalog entry for a skill ref
func (r *SkillDiscoveryReconciler) ensureSkillCatalog(ctx context.Context, skillRef string, agent *kagentv1alpha2.Agent) error {
	// Parse skill ref (OCI image reference like "ghcr.io/user/skill:v1.0.0")
	skillName, version := parseSkillRef(skillRef)
	catalogName := generateSkillCatalogName(skillName, version)

	logger := r.Logger.With().
		Str("skill", skillRef).
		Str("catalogName", catalogName).
		Logger()

	// Check if catalog entry exists
	var catalog agentregistryv1alpha1.SkillCatalog
	err := r.Get(ctx, types.NamespacedName{Name: catalogName}, &catalog)

	if apierrors.IsNotFound(err) {
		// Create new catalog entry
		catalog = r.buildCatalogFromSkillRef(skillRef, skillName, version, catalogName)
		if err := r.Create(ctx, &catalog); err != nil {
			return fmt.Errorf("failed to create skill catalog: %w", err)
		}
		logger.Info().Msg("created skill catalog entry")
	} else if err != nil {
		return err
	}

	// Update usage tracking
	return r.updateSkillUsage(ctx, &catalog, agent)
}

// buildCatalogFromSkillRef creates a SkillCatalog from an OCI skill reference
func (r *SkillDiscoveryReconciler) buildCatalogFromSkillRef(skillRef, skillName, version, catalogName string) agentregistryv1alpha1.SkillCatalog {
	// Extract title from skill name (last part of path)
	title := skillName
	if idx := strings.LastIndex(skillName, "/"); idx != -1 {
		title = skillName[idx+1:]
	}

	return agentregistryv1alpha1.SkillCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: catalogName,
			Labels: map[string]string{
				skillDiscoveryLabel: "true",
				skillSourceLabel:    "agent",
			},
		},
		Spec: agentregistryv1alpha1.SkillCatalogSpec{
			Name:    skillName,
			Version: version,
			Title:   title,
			Packages: []agentregistryv1alpha1.SkillPackage{
				{
					RegistryType: "oci",
					Identifier:   skillRef,
					Version:      version,
				},
			},
		},
	}
}

// updateSkillUsage adds the agent to the skill's UsedBy list
func (r *SkillDiscoveryReconciler) updateSkillUsage(ctx context.Context, catalog *agentregistryv1alpha1.SkillCatalog, agent *kagentv1alpha2.Agent) error {
	// Check if agent is already in UsedBy list
	agentRef := agentregistryv1alpha1.SkillUsageRef{
		Namespace: agent.Namespace,
		Name:      agent.Name,
		Kind:      "Agent",
	}

	for _, ref := range catalog.Status.UsedBy {
		if ref.Namespace == agentRef.Namespace && ref.Name == agentRef.Name {
			// Already tracked
			return nil
		}
	}

	// Add to UsedBy list
	catalog.Status.UsedBy = append(catalog.Status.UsedBy, agentRef)

	// Auto-publish discovered skills
	if !catalog.Status.Published {
		now := metav1.Now()
		catalog.Status.Published = true
		catalog.Status.PublishedAt = &now
		catalog.Status.Status = agentregistryv1alpha1.CatalogStatusActive
	}

	return r.Status().Update(ctx, catalog)
}

// removeAgentFromSkillUsage removes an agent from all skill UsedBy lists
func (r *SkillDiscoveryReconciler) removeAgentFromSkillUsage(ctx context.Context, namespace, name string) (ctrl.Result, error) {
	// List all skill catalogs with discovery label
	var skillList agentregistryv1alpha1.SkillCatalogList
	if err := r.List(ctx, &skillList, client.MatchingLabels{
		skillDiscoveryLabel: "true",
	}); err != nil {
		return ctrl.Result{}, err
	}

	for i := range skillList.Items {
		skill := &skillList.Items[i]
		updated := false
		newUsedBy := make([]agentregistryv1alpha1.SkillUsageRef, 0, len(skill.Status.UsedBy))

		for _, ref := range skill.Status.UsedBy {
			if ref.Namespace == namespace && ref.Name == name {
				updated = true
				continue
			}
			newUsedBy = append(newUsedBy, ref)
		}

		if updated {
			skill.Status.UsedBy = newUsedBy
			if err := r.Status().Update(ctx, skill); err != nil {
				r.Logger.Error().Err(err).
					Str("skill", skill.Name).
					Msg("failed to update skill usage after agent deletion")
			}
		}
	}

	return ctrl.Result{}, nil
}

// parseSkillRef parses an OCI image reference into name and version
// e.g., "ghcr.io/antonbabenko/terraform-skill:v1.0.0" -> "ghcr.io/antonbabenko/terraform-skill", "v1.0.0"
func parseSkillRef(ref string) (name, version string) {
	// Handle digest references
	if idx := strings.LastIndex(ref, "@"); idx != -1 {
		return ref[:idx], ref[idx+1:]
	}

	// Handle tag references
	if idx := strings.LastIndex(ref, ":"); idx != -1 {
		// Make sure we're not splitting on a port number
		possibleTag := ref[idx+1:]
		if !strings.Contains(possibleTag, "/") {
			return ref[:idx], possibleTag
		}
	}

	return ref, "latest"
}

// generateSkillCatalogName creates a valid K8s name from skill name and version
func generateSkillCatalogName(name, version string) string {
	// Remove registry prefix for shorter names
	shortName := name
	if idx := strings.Index(name, "/"); idx != -1 {
		// Keep only the last two parts (org/skill)
		parts := strings.Split(name, "/")
		if len(parts) >= 2 {
			shortName = strings.Join(parts[len(parts)-2:], "-")
		}
	}

	combined := fmt.Sprintf("%s-%s", shortName, version)
	combined = strings.ReplaceAll(combined, "/", "-")
	combined = strings.ReplaceAll(combined, "_", "-")
	combined = strings.ReplaceAll(combined, ".", "-")
	combined = strings.ToLower(combined)

	if len(combined) > 63 {
		combined = combined[:63]
	}

	// Trim trailing dashes
	combined = strings.TrimRight(combined, "-")

	return combined
}

// SetupWithManager sets up the controller with the Manager.
func (r *SkillDiscoveryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("skill-discovery").
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
