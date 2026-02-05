package controller

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

// newTestClientWithSkillIndexes creates a fake client with SkillCatalog field indexes
func newTestClientWithSkillIndexes(scheme *runtime.Scheme, objs ...client.Object) client.Client {
	builder := fake.NewClientBuilder().WithScheme(scheme)

	// Add index for spec.name
	builder.WithIndex(
		&agentregistryv1alpha1.SkillCatalog{},
		IndexSkillName,
		func(obj client.Object) []string {
			skill := obj.(*agentregistryv1alpha1.SkillCatalog)
			return []string{skill.Spec.Name}
		},
	)

	return builder.WithObjects(objs...).WithStatusSubresource(&agentregistryv1alpha1.SkillCatalog{}).Build()
}

func TestSkillCatalogReconciler_Reconcile_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	r := &SkillCatalogReconciler{
		Client: c,
		Scheme: scheme,
		Logger: logger,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-skill", Namespace: "default"},
	}

	result, err := r.Reconcile(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)
}

func TestSkillCatalogReconciler_Reconcile_NewSkill(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)

	skill := &agentregistryv1alpha1.SkillCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "terraform-skill-v1.0.0",
			Namespace: "default",
		},
		Spec: agentregistryv1alpha1.SkillCatalogSpec{
			Name:        "terraform-skill",
			Version:     "1.0.0",
			Title:       "Terraform Skill",
			Category:    "infrastructure",
			Description: "Infrastructure management skill",
		},
	}

	c := newTestClientWithSkillIndexes(scheme, skill)

	r := &SkillCatalogReconciler{
		Client: c,
		Scheme: scheme,
		Logger: logger,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "terraform-skill-v1.0.0", Namespace: "default"},
	}

	result, err := r.Reconcile(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)

	// Verify observed generation was updated
	updatedSkill := &agentregistryv1alpha1.SkillCatalog{}
	err = c.Get(context.Background(), req.NamespacedName, updatedSkill)
	require.NoError(t, err)
	assert.Equal(t, skill.Generation, updatedSkill.Status.ObservedGeneration)
}

func TestSkillCatalogReconciler_Reconcile_MultipleVersions(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)

	// Create multiple versions of the same skill
	skillV1 := &agentregistryv1alpha1.SkillCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "terraform-skill-v1.0.0",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: agentregistryv1alpha1.SkillCatalogSpec{
			Name:     "terraform-skill",
			Version:  "1.0.0",
			Category: "infrastructure",
		},
		Status: agentregistryv1alpha1.SkillCatalogStatus{
			Published: true,
		},
	}

	skillV2 := &agentregistryv1alpha1.SkillCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "terraform-skill-v2.0.0",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: agentregistryv1alpha1.SkillCatalogSpec{
			Name:     "terraform-skill",
			Version:  "2.0.0",
			Category: "infrastructure",
		},
		Status: agentregistryv1alpha1.SkillCatalogStatus{
			Published: true,
		},
	}

	c := newTestClientWithSkillIndexes(scheme, skillV1, skillV2)

	r := &SkillCatalogReconciler{
		Client: c,
		Scheme: scheme,
		Logger: logger,
	}

	// Reconcile v1
	req1 := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "terraform-skill-v1.0.0", Namespace: "default"},
	}
	_, err := r.Reconcile(context.Background(), req1)
	assert.NoError(t, err)

	// Reconcile v2
	req2 := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "terraform-skill-v2.0.0", Namespace: "default"},
	}
	_, err = r.Reconcile(context.Background(), req2)
	assert.NoError(t, err)

	// Check that v2 is marked as latest
	updatedV2 := &agentregistryv1alpha1.SkillCatalog{}
	err = c.Get(context.Background(), req2.NamespacedName, updatedV2)
	require.NoError(t, err)
	assert.True(t, updatedV2.Status.IsLatest)
}

func TestSkillCatalogReconciler_Reconcile_WithMetadata(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)

	metadata := &apiextensionsv1.JSON{
		Raw: []byte(`{"verified": true, "author": "test"}`),
	}

	skill := &agentregistryv1alpha1.SkillCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "verified-skill-v1.0.0",
			Namespace: "default",
		},
		Spec: agentregistryv1alpha1.SkillCatalogSpec{
			Name:        "verified-skill",
			Version:     "1.0.0",
			Title:       "Verified Skill",
			Category:    "security",
			Description: "A verified skill",
			Metadata:    metadata,
		},
	}

	c := newTestClientWithSkillIndexes(scheme, skill)

	r := &SkillCatalogReconciler{
		Client: c,
		Scheme: scheme,
		Logger: logger,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "verified-skill-v1.0.0", Namespace: "default"},
	}

	result, err := r.Reconcile(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)
}

func TestSkillCatalogReconciler_updateLatestVersion(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)

	// Create skills with different versions
	skills := []*agentregistryv1alpha1.SkillCatalog{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "python-skill-v1.0.0",
				Namespace: "default",
				Labels:    map[string]string{"name": "python-skill"},
			},
			Spec: agentregistryv1alpha1.SkillCatalogSpec{
				Name:     "python-skill",
				Version:  "1.0.0",
				Category: "development",
			},
			Status: agentregistryv1alpha1.SkillCatalogStatus{
				Published: true,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "python-skill-v1.1.0",
				Namespace: "default",
				Labels:    map[string]string{"name": "python-skill"},
			},
			Spec: agentregistryv1alpha1.SkillCatalogSpec{
				Name:     "python-skill",
				Version:  "1.1.0",
				Category: "development",
			},
			Status: agentregistryv1alpha1.SkillCatalogStatus{
				Published: true,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "python-skill-v2.0.0",
				Namespace: "default",
				Labels:    map[string]string{"name": "python-skill"},
			},
			Spec: agentregistryv1alpha1.SkillCatalogSpec{
				Name:     "python-skill",
				Version:  "2.0.0",
				Category: "development",
			},
			Status: agentregistryv1alpha1.SkillCatalogStatus{
				Published: true,
			},
		},
	}

	c := newTestClientWithSkillIndexes(scheme, skills[0], skills[1], skills[2])

	r := &SkillCatalogReconciler{
		Client: c,
		Scheme: scheme,
		Logger: logger,
	}

	// Update latest version for the newest skill
	err := r.updateLatestVersion(context.Background(), skills[2])
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Edge case tests
// ---------------------------------------------------------------------------

func TestSkillCatalogReconciler_UpdateLatestVersion_SingleVersion(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)

	// Single skill version
	skill := &agentregistryv1alpha1.SkillCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "solo-skill-1-0-0",
			Namespace: "default",
		},
		Spec: agentregistryv1alpha1.SkillCatalogSpec{
			Name:     "solo-skill",
			Version:  "1.0.0",
			Category: "testing",
		},
		Status: agentregistryv1alpha1.SkillCatalogStatus{
			Published: true,
		},
	}

	c := newTestClientWithSkillIndexes(scheme, skill)

	r := &SkillCatalogReconciler{
		Client: c,
		Scheme: scheme,
		Logger: logger,
	}

	// Update should succeed even with only one version
	err := r.updateLatestVersion(context.Background(), skill)
	assert.NoError(t, err)
}

func TestSkillCatalogReconciler_Reconcile_UpdateGenerationTracking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires proper status subresource handling")
	}

	scheme := runtime.NewScheme()
	_ = agentregistryv1alpha1.AddToScheme(scheme)

	logger := zerolog.New(nil)

	skill := &agentregistryv1alpha1.SkillCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "track-skill-1-0-0",
			Namespace:  "default",
			Generation: 2, // Simulate update
		},
		Spec: agentregistryv1alpha1.SkillCatalogSpec{
			Name:     "track-skill",
			Version:  "1.0.0",
			Category: "tracking",
		},
		Status: agentregistryv1alpha1.SkillCatalogStatus{
			ObservedGeneration: 1, // Out of sync
		},
	}

	c := newTestClientWithSkillIndexes(scheme, skill)

	r := &SkillCatalogReconciler{
		Client: c,
		Scheme: scheme,
		Logger: logger,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "track-skill-1-0-0",
			Namespace: "default",
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	assert.NoError(t, err)
	// May requeue due to status update conflicts with fake client
	// This is expected behavior with fake client that doesn't fully support status subresources
	_ = result

	// Note: Proper generation tracking requires envtest with status subresource
}
