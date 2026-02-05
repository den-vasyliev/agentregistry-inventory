package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
