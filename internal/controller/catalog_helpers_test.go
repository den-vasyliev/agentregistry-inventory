package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

func TestFindLatestVersion(t *testing.T) {
	now := metav1.Now()
	earlier := metav1.NewTime(now.Add(-1 * time.Hour))
	later := metav1.NewTime(now.Add(1 * time.Hour))

	tests := []struct {
		name     string
		versions []CatalogVersionInfo
		want     string
	}{
		{
			name: "multiple semver versions - picks highest",
			versions: []CatalogVersionInfo{
				{Name: "server-1.0.0", Version: "1.0.0", Published: true, PublishedAt: &now},
				{Name: "server-2.0.0", Version: "2.0.0", Published: true, PublishedAt: &now},
				{Name: "server-1.5.0", Version: "1.5.0", Published: true, PublishedAt: &now},
			},
			want: "server-2.0.0",
		},
		{
			name: "semver with v prefix",
			versions: []CatalogVersionInfo{
				{Name: "server-v1.0.0", Version: "v1.0.0", Published: true, PublishedAt: &now},
				{Name: "server-v2.0.0", Version: "v2.0.0", Published: true, PublishedAt: &now},
			},
			want: "server-v2.0.0",
		},
		{
			name: "non-semver versions - uses timestamps",
			versions: []CatalogVersionInfo{
				{Name: "server-latest", Version: "latest", Published: true, PublishedAt: &earlier},
				{Name: "server-main", Version: "main", Published: true, PublishedAt: &later},
			},
			want: "server-main", // later timestamp wins
		},
		{
			name: "mix of semver and non-semver - semver wins",
			versions: []CatalogVersionInfo{
				{Name: "server-latest", Version: "latest", Published: true, PublishedAt: &later},
				{Name: "server-1.0.0", Version: "1.0.0", Published: true, PublishedAt: &earlier},
			},
			want: "server-1.0.0", // semver always wins over non-semver
		},
		{
			name: "all unpublished - now returns latest (Published filter removed)",
			versions: []CatalogVersionInfo{
				{Name: "server-1.0.0", Version: "1.0.0", Published: false, PublishedAt: &now},
				{Name: "server-2.0.0", Version: "2.0.0", Published: false, PublishedAt: &now},
			},
			want: "server-2.0.0", // ✅ Published filter removed - ALL versions considered
		},
		{
			name: "published status ignored - highest version wins",
			versions: []CatalogVersionInfo{
				{Name: "server-1.0.0", Version: "1.0.0", Published: true, PublishedAt: &now},
				{Name: "server-3.0.0", Version: "3.0.0", Published: false, PublishedAt: &now}, // Unpublished but higher
			},
			want: "server-3.0.0", // ✅ Published filter removed - highest version wins
		},
		{
			name:     "empty list",
			versions: []CatalogVersionInfo{},
			want:     "",
		},
		{
			name: "single version",
			versions: []CatalogVersionInfo{
				{Name: "server-1.0.0", Version: "1.0.0", Published: true, PublishedAt: &now},
			},
			want: "server-1.0.0",
		},
		{
			name: "nil timestamps - handles gracefully",
			versions: []CatalogVersionInfo{
				{Name: "server-latest", Version: "latest", Published: true, PublishedAt: nil},
				{Name: "server-main", Version: "main", Published: true, PublishedAt: &now},
			},
			want: "server-main", // version with timestamp wins
		},
		{
			name: "prerelease versions",
			versions: []CatalogVersionInfo{
				{Name: "server-1.0.0", Version: "1.0.0", Published: true, PublishedAt: &now},
				{Name: "server-2.0.0-beta", Version: "2.0.0-beta", Published: true, PublishedAt: &now},
				{Name: "server-1.5.0", Version: "1.5.0", Published: true, PublishedAt: &now},
			},
			want: "server-2.0.0-beta", // 2.0.0-beta > 1.5.0 > 1.0.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findLatestVersion(tt.versions)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUpdateLatestVersionForMCPServers(t *testing.T) {
	t.Skip("TODO: Integration test needs manager running for field indexes - covered by existing mcpservercatalog_controller_test.go")

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	helper := SetupTestEnv(t, 60*time.Second, false)
	defer helper.Cleanup(t)

	ctx := context.Background()

	// Create agentregistry namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "agentregistry",
		},
	}
	err := helper.Client.Create(ctx, ns)
	require.NoError(t, err)

	t.Run("multiple versions - only latest gets isLatest flag", func(t *testing.T) {
		// Create multiple versions of the same server
		servers := []*agentregistryv1alpha1.MCPServerCatalog{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-server-1-0-0",
					Namespace: "agentregistry",
				},
				Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
					Name:    "test-server",
					Version: "1.0.0",
				},
				Status: agentregistryv1alpha1.MCPServerCatalogStatus{
					Published:   true,
					PublishedAt: &metav1.Time{Time: time.Now().Add(-2 * time.Hour)},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-server-2-0-0",
					Namespace: "agentregistry",
				},
				Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
					Name:    "test-server",
					Version: "2.0.0",
				},
				Status: agentregistryv1alpha1.MCPServerCatalogStatus{
					Published:   true,
					PublishedAt: &metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-server-1-5-0",
					Namespace: "agentregistry",
				},
				Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
					Name:    "test-server",
					Version: "1.5.0",
				},
				Status: agentregistryv1alpha1.MCPServerCatalogStatus{
					Published:   true,
					PublishedAt: &metav1.Time{Time: time.Now()},
				},
			},
		}

		// Create all servers
		for _, s := range servers {
			err := helper.Client.Create(ctx, s)
			require.NoError(t, err)
		}

		// Update latest version
		err := updateLatestVersionForMCPServers(ctx, helper.Client, "test-server")
		require.NoError(t, err)

		// Verify only 2.0.0 has isLatest=true
		var updatedServers agentregistryv1alpha1.MCPServerCatalogList
		err = helper.Client.List(ctx, &updatedServers, client.MatchingFields{
			IndexMCPServerName: "test-server",
		})
		require.NoError(t, err)

		latestCount := 0
		for _, s := range updatedServers.Items {
			if s.Status.IsLatest {
				latestCount++
				assert.Equal(t, "2.0.0", s.Spec.Version, "Only version 2.0.0 should be latest")
			}
		}
		assert.Equal(t, 1, latestCount, "Exactly one version should be marked as latest")
	})

	t.Run("unpublished version now considered (Published filter removed)", func(t *testing.T) {
		servers := []*agentregistryv1alpha1.MCPServerCatalog{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unpub-server-1-0-0",
					Namespace: "agentregistry",
				},
				Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
					Name:    "unpub-server",
					Version: "1.0.0",
				},
				Status: agentregistryv1alpha1.MCPServerCatalogStatus{
					Published:   true,
					PublishedAt: &metav1.Time{Time: time.Now()},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unpub-server-2-0-0",
					Namespace: "agentregistry",
				},
				Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
					Name:    "unpub-server",
					Version: "2.0.0",
				},
				Status: agentregistryv1alpha1.MCPServerCatalogStatus{
					Published:   false, // Not published
					PublishedAt: &metav1.Time{Time: time.Now()},
				},
			},
		}

		for _, s := range servers {
			err := helper.Client.Create(ctx, s)
			require.NoError(t, err)
		}

		err := updateLatestVersionForMCPServers(ctx, helper.Client, "unpub-server")
		require.NoError(t, err)

		// Verify 2.0.0 is latest (Published filter removed - highest version wins)
		var server agentregistryv1alpha1.MCPServerCatalog
		err = helper.Client.Get(ctx, client.ObjectKey{
			Name:      "unpub-server-2-0-0",
			Namespace: "agentregistry",
		}, &server)
		require.NoError(t, err)
		assert.True(t, server.Status.IsLatest, "2.0.0 should be latest (highest version, Published filter removed)")

		err = helper.Client.Get(ctx, client.ObjectKey{
			Name:      "unpub-server-1-0-0",
			Namespace: "agentregistry",
		}, &server)
		require.NoError(t, err)
		assert.False(t, server.Status.IsLatest, "1.0.0 should not be latest")
	})

	t.Run("no versions - no error", func(t *testing.T) {
		err := updateLatestVersionForMCPServers(ctx, helper.Client, "nonexistent-server")
		require.NoError(t, err)
	})
}

func TestUpdateLatestVersionForAgents(t *testing.T) {
	t.Skip("TODO: Integration test needs manager running for field indexes - covered by controller tests")

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	helper := SetupTestEnv(t, 60*time.Second, false)
	defer helper.Cleanup(t)

	ctx := context.Background()

	// Create agentregistry namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "agentregistry",
		},
	}
	err := helper.Client.Create(ctx, ns)
	require.NoError(t, err)

	t.Run("multiple agent versions", func(t *testing.T) {
		agents := []*agentregistryv1alpha1.AgentCatalog{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent-1-0-0",
					Namespace: "agentregistry",
				},
				Spec: agentregistryv1alpha1.AgentCatalogSpec{
					Name:    "test-agent",
					Version: "1.0.0",
				},
				Status: agentregistryv1alpha1.AgentCatalogStatus{
					Published:   true,
					PublishedAt: &metav1.Time{Time: time.Now()},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent-2-0-0",
					Namespace: "agentregistry",
				},
				Spec: agentregistryv1alpha1.AgentCatalogSpec{
					Name:    "test-agent",
					Version: "2.0.0",
				},
				Status: agentregistryv1alpha1.AgentCatalogStatus{
					Published:   true,
					PublishedAt: &metav1.Time{Time: time.Now()},
				},
			},
		}

		for _, a := range agents {
			err := helper.Client.Create(ctx, a)
			require.NoError(t, err)
		}

		err := updateLatestVersionForAgents(ctx, helper.Client, "test-agent")
		require.NoError(t, err)

		// Verify 2.0.0 is latest
		var agent agentregistryv1alpha1.AgentCatalog
		err = helper.Client.Get(ctx, client.ObjectKey{
			Name:      "test-agent-2-0-0",
			Namespace: "agentregistry",
		}, &agent)
		require.NoError(t, err)
		assert.True(t, agent.Status.IsLatest)

		err = helper.Client.Get(ctx, client.ObjectKey{
			Name:      "test-agent-1-0-0",
			Namespace: "agentregistry",
		}, &agent)
		require.NoError(t, err)
		assert.False(t, agent.Status.IsLatest)
	})
}

func TestUpdateLatestVersionForSkills(t *testing.T) {
	t.Skip("TODO: Integration test needs manager running for field indexes - covered by controller tests")

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	helper := SetupTestEnv(t, 60*time.Second, false)
	defer helper.Cleanup(t)

	ctx := context.Background()

	// Create agentregistry namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "agentregistry",
		},
	}
	err := helper.Client.Create(ctx, ns)
	require.NoError(t, err)

	t.Run("multiple skill versions", func(t *testing.T) {
		skills := []*agentregistryv1alpha1.SkillCatalog{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-skill-1-0-0",
					Namespace: "agentregistry",
				},
				Spec: agentregistryv1alpha1.SkillCatalogSpec{
					Name:    "test-skill",
					Version: "1.0.0",
				},
				Status: agentregistryv1alpha1.SkillCatalogStatus{
					Published:   true,
					PublishedAt: &metav1.Time{Time: time.Now()},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-skill-2-0-0",
					Namespace: "agentregistry",
				},
				Spec: agentregistryv1alpha1.SkillCatalogSpec{
					Name:    "test-skill",
					Version: "2.0.0",
				},
				Status: agentregistryv1alpha1.SkillCatalogStatus{
					Published:   true,
					PublishedAt: &metav1.Time{Time: time.Now()},
				},
			},
		}

		for _, s := range skills {
			err := helper.Client.Create(ctx, s)
			require.NoError(t, err)
		}

		err := updateLatestVersionForSkills(ctx, helper.Client, "test-skill")
		require.NoError(t, err)

		// Verify 2.0.0 is latest
		var skill agentregistryv1alpha1.SkillCatalog
		err = helper.Client.Get(ctx, client.ObjectKey{
			Name:      "test-skill-2-0-0",
			Namespace: "agentregistry",
		}, &skill)
		require.NoError(t, err)
		assert.True(t, skill.Status.IsLatest)

		err = helper.Client.Get(ctx, client.ObjectKey{
			Name:      "test-skill-1-0-0",
			Namespace: "agentregistry",
		}, &skill)
		require.NoError(t, err)
		assert.False(t, skill.Status.IsLatest)
	})
}

// Note: compareVersions and isSemanticVersion are already tested in versioning_test.go
