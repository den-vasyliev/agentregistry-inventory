package controller

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

// SkillCatalogReconciler reconciles a SkillCatalog object
type SkillCatalogReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger zerolog.Logger
}

// +kubebuilder:rbac:groups=agentregistry.dev,resources=skillcatalogs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentregistry.dev,resources=skillcatalogs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agentregistry.dev,resources=skillcatalogs/finalizers,verbs=update

// Reconcile handles SkillCatalog reconciliation
func (r *SkillCatalogReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.With().Str("name", req.Name).Str("namespace", req.Namespace).Logger()

	// Fetch the SkillCatalog
	var skill agentregistryv1alpha1.SkillCatalog
	if err := r.Get(ctx, req.NamespacedName, &skill); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info().
		Str("specName", skill.Spec.Name).
		Str("version", skill.Spec.Version).
		Msg("reconciling SkillCatalog")

	// Update isLatest status for all versions of this skill
	if err := r.updateLatestVersion(ctx, &skill); err != nil {
		logger.Error().Err(err).Msg("failed to update latest version")
		return ctrl.Result{}, err
	}

	// Update observed generation
	if skill.Status.ObservedGeneration != skill.Generation {
		skill.Status.ObservedGeneration = skill.Generation
		if err := r.Status().Update(ctx, &skill); err != nil {
			logger.Error().Err(err).Msg("failed to update status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// updateLatestVersion determines and updates the latest version flag for all versions of a skill
func (r *SkillCatalogReconciler) updateLatestVersion(ctx context.Context, skill *agentregistryv1alpha1.SkillCatalog) error {
	// Get all versions of this skill
	var skillList agentregistryv1alpha1.SkillCatalogList
	if err := r.List(ctx, &skillList, client.MatchingFields{
		IndexSkillName: skill.Spec.Name,
	}); err != nil {
		return err
	}

	// Find the latest version among published skills
	var latestSkill *agentregistryv1alpha1.SkillCatalog
	var latestTimestamp time.Time

	for i := range skillList.Items {
		s := &skillList.Items[i]
		if !s.Status.Published {
			continue
		}

		if latestSkill == nil {
			latestSkill = s
			if s.Status.PublishedAt != nil {
				latestTimestamp = s.Status.PublishedAt.Time
			}
			continue
		}

		var sTimestamp time.Time
		if s.Status.PublishedAt != nil {
			sTimestamp = s.Status.PublishedAt.Time
		}

		cmp := compareVersions(s.Spec.Version, latestSkill.Spec.Version, sTimestamp, latestTimestamp)
		if cmp > 0 {
			latestSkill = s
			latestTimestamp = sTimestamp
		}
	}

	// Update isLatest flag for all versions
	for i := range skillList.Items {
		s := &skillList.Items[i]
		shouldBeLatest := latestSkill != nil && s.Name == latestSkill.Name && s.Status.Published

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
func (r *SkillCatalogReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&agentregistryv1alpha1.SkillCatalog{}).
		Complete(r)
}
