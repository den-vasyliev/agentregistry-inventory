package handlers

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

func setupSkillTestClient(t *testing.T) client.Client {
	scheme := runtime.NewScheme()
	require.NoError(t, agentregistryv1alpha1.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).Build()
}

// ---------------------------------------------------------------------------
// createSkill
// ---------------------------------------------------------------------------

func TestSkillHandler_CreateSkill(t *testing.T) {
	c := setupSkillTestClient(t)
	ctx := context.Background()
	logger := zerolog.Nop()
	handler := NewSkillHandler(c, nil, logger)

	input := &CreateSkillInput{
		Body: SkillJSON{
			Name:        "my-skill",
			Version:     "1.0.0",
			Title:       "My Skill",
			Category:    "data-processing",
			Description: "A test skill",
			WebsiteURL:  "https://example.com",
		},
	}

	resp, err := handler.createSkill(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, "my-skill", resp.Body.Skill.Name)
	assert.Equal(t, "1.0.0", resp.Body.Skill.Version)
	assert.Equal(t, "My Skill", resp.Body.Skill.Title)
	assert.Equal(t, "data-processing", resp.Body.Skill.Category)
	assert.Equal(t, "https://example.com", resp.Body.Skill.WebsiteURL)

	// Verify the CR was persisted
	created := &agentregistryv1alpha1.SkillCatalog{}
	err = c.Get(ctx, client.ObjectKey{Name: "my-skill-1-0-0"}, created)
	require.NoError(t, err)
	assert.Equal(t, "My Skill", created.Spec.Title)
	assert.Equal(t, "data-processing", created.Spec.Category)
}

func TestSkillHandler_CreateSkill_WithRepository(t *testing.T) {
	c := setupSkillTestClient(t)
	ctx := context.Background()
	handler := NewSkillHandler(c, nil, zerolog.Nop())

	input := &CreateSkillInput{
		Body: SkillJSON{
			Name:    "repo-skill",
			Version: "2.0.0",
			Title:   "Repo Skill",
			Repository: &SkillRepositoryJSON{
				URL:    "https://github.com/org/repo",
				Source: "github",
			},
		},
	}

	resp, err := handler.createSkill(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp.Body.Skill.Repository)
	assert.Equal(t, "https://github.com/org/repo", resp.Body.Skill.Repository.URL)
	assert.Equal(t, "github", resp.Body.Skill.Repository.Source)

	// Verify CR has repository
	created := &agentregistryv1alpha1.SkillCatalog{}
	err = c.Get(ctx, client.ObjectKey{Name: "repo-skill-2-0-0"}, created)
	require.NoError(t, err)
	require.NotNil(t, created.Spec.Repository)
	assert.Equal(t, "https://github.com/org/repo", created.Spec.Repository.URL)
}

func TestSkillHandler_CreateSkill_WithPackagesAndRemotes(t *testing.T) {
	c := setupSkillTestClient(t)
	ctx := context.Background()
	handler := NewSkillHandler(c, nil, zerolog.Nop())

	input := &CreateSkillInput{
		Body: SkillJSON{
			Name:    "pkg-skill",
			Version: "1.0.0",
			Packages: []SkillPackageJSON{
				{
					RegistryType: "oci",
					Identifier:   "registry.io/pkg",
					Version:      "1.0.0",
					Transport:    &SkillPackageTransportJSON{Type: "stdio"},
				},
			},
			Remotes: []SkillRemoteJSON{
				{URL: "https://api.example.com/skill"},
			},
		},
	}

	resp, err := handler.createSkill(ctx, input)
	require.NoError(t, err)

	// Verify packages
	require.Len(t, resp.Body.Skill.Packages, 1)
	assert.Equal(t, "oci", resp.Body.Skill.Packages[0].RegistryType)
	require.NotNil(t, resp.Body.Skill.Packages[0].Transport)
	assert.Equal(t, "stdio", resp.Body.Skill.Packages[0].Transport.Type)

	// Verify remotes
	require.Len(t, resp.Body.Skill.Remotes, 1)
	assert.Equal(t, "https://api.example.com/skill", resp.Body.Skill.Remotes[0].URL)

	// Verify CR
	created := &agentregistryv1alpha1.SkillCatalog{}
	err = c.Get(ctx, client.ObjectKey{Name: "pkg-skill-1-0-0"}, created)
	require.NoError(t, err)
	assert.Len(t, created.Spec.Packages, 1)
	assert.Len(t, created.Spec.Remotes, 1)
}

// ---------------------------------------------------------------------------
// convertToSkillResponse
// ---------------------------------------------------------------------------

func TestSkillHandler_ConvertToSkillResponse_Basic(t *testing.T) {
	c := setupSkillTestClient(t)
	handler := NewSkillHandler(c, nil, zerolog.Nop())

	skill := &agentregistryv1alpha1.SkillCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-skill-1-0-0",
		},
		Spec: agentregistryv1alpha1.SkillCatalogSpec{
			Name:        "test-skill",
			Version:     "1.0.0",
			Title:       "Test Skill",
			Category:    "testing",
			Description: "A test skill",
		},
		Status: agentregistryv1alpha1.SkillCatalogStatus{
			IsLatest: true,
		},
	}

	resp := handler.convertToSkillResponse(skill)
	assert.Equal(t, "test-skill", resp.Skill.Name)
	assert.Equal(t, "1.0.0", resp.Skill.Version)
	assert.Equal(t, "Test Skill", resp.Skill.Title)
	assert.Equal(t, "testing", resp.Skill.Category)
	require.NotNil(t, resp.Meta.Official)
	assert.True(t, resp.Meta.Official.IsLatest)
}

func TestSkillHandler_ConvertToSkillResponse_WithRepository(t *testing.T) {
	c := setupSkillTestClient(t)
	handler := NewSkillHandler(c, nil, zerolog.Nop())

	skill := &agentregistryv1alpha1.SkillCatalog{
		ObjectMeta: metav1.ObjectMeta{Name: "repo-skill-1-0-0"},
		Spec: agentregistryv1alpha1.SkillCatalogSpec{
			Name:    "repo-skill",
			Version: "1.0.0",
			Repository: &agentregistryv1alpha1.SkillRepository{
				URL:    "https://github.com/org/skill",
				Source: "github",
			},
		},
	}

	resp := handler.convertToSkillResponse(skill)
	require.NotNil(t, resp.Skill.Repository)
	assert.Equal(t, "https://github.com/org/skill", resp.Skill.Repository.URL)
	assert.Equal(t, "github", resp.Skill.Repository.Source)
}

func TestSkillHandler_ConvertToSkillResponse_WithUsedBy(t *testing.T) {
	c := setupSkillTestClient(t)
	handler := NewSkillHandler(c, nil, zerolog.Nop())

	skill := &agentregistryv1alpha1.SkillCatalog{
		ObjectMeta: metav1.ObjectMeta{Name: "used-skill-1-0-0"},
		Spec: agentregistryv1alpha1.SkillCatalogSpec{
			Name:    "used-skill",
			Version: "1.0.0",
		},
		Status: agentregistryv1alpha1.SkillCatalogStatus{
			UsedBy: []agentregistryv1alpha1.SkillUsageRef{
				{
					Namespace: "default",
					Name:      "my-agent",
					Kind:      "Agent",
				},
			},
		},
	}

	resp := handler.convertToSkillResponse(skill)
	require.Len(t, resp.Meta.UsedBy, 1)
	assert.Equal(t, "default", resp.Meta.UsedBy[0].Namespace)
	assert.Equal(t, "my-agent", resp.Meta.UsedBy[0].Name)
	assert.Equal(t, "Agent", resp.Meta.UsedBy[0].Kind)
}
