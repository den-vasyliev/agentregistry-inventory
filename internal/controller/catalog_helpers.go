package controller

import (
	"context"
	"strings"
	"time"

	"golang.org/x/mod/semver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

// CatalogVersionInfo represents version metadata for a catalog item
type CatalogVersionInfo struct {
	Name        string
	Version     string
	PublishedAt *metav1.Time
	IsLatest    bool
	Published   bool
}

// updateLatestVersionForMCPServers updates isLatest flag for all versions of an MCP server
func updateLatestVersionForMCPServers(ctx context.Context, c client.Client, serverName string) error {
	var serverList agentregistryv1alpha1.MCPServerCatalogList
	if err := c.List(ctx, &serverList, client.MatchingFields{
		IndexMCPServerName: serverName,
	}); err != nil {
		return err
	}

	// Extract version info
	versions := make([]CatalogVersionInfo, len(serverList.Items))
	for i := range serverList.Items {
		s := &serverList.Items[i]
		versions[i] = CatalogVersionInfo{
			Name:        s.Name,
			Version:     s.Spec.Version,
			PublishedAt: s.Status.PublishedAt,
			IsLatest:    s.Status.IsLatest,
			Published:   s.Status.Published,
		}
	}

	// Find latest version (currently filters by published, will be removed)
	latestName := findLatestVersion(versions)

	// Update isLatest flag for all versions
	for i := range serverList.Items {
		s := &serverList.Items[i]
		shouldBeLatest := (latestName != "" && s.Name == latestName)

		if s.Status.IsLatest != shouldBeLatest {
			s.Status.IsLatest = shouldBeLatest
			if err := c.Status().Update(ctx, s); err != nil {
				return err
			}
		}
	}

	return nil
}

// updateLatestVersionForAgents updates isLatest flag for all versions of an agent
func updateLatestVersionForAgents(ctx context.Context, c client.Client, agentName string) error {
	var agentList agentregistryv1alpha1.AgentCatalogList
	if err := c.List(ctx, &agentList, client.MatchingFields{
		IndexAgentName: agentName,
	}); err != nil {
		return err
	}

	// Extract version info
	versions := make([]CatalogVersionInfo, len(agentList.Items))
	for i := range agentList.Items {
		a := &agentList.Items[i]
		versions[i] = CatalogVersionInfo{
			Name:        a.Name,
			Version:     a.Spec.Version,
			PublishedAt: a.Status.PublishedAt,
			IsLatest:    a.Status.IsLatest,
			Published:   a.Status.Published,
		}
	}

	// Find latest version (currently filters by published, will be removed)
	latestName := findLatestVersion(versions)

	// Update isLatest flag for all versions
	for i := range agentList.Items {
		a := &agentList.Items[i]
		shouldBeLatest := (latestName != "" && a.Name == latestName)

		if a.Status.IsLatest != shouldBeLatest {
			a.Status.IsLatest = shouldBeLatest
			if err := c.Status().Update(ctx, a); err != nil {
				return err
			}
		}
	}

	return nil
}

// updateLatestVersionForSkills updates isLatest flag for all versions of a skill
func updateLatestVersionForSkills(ctx context.Context, c client.Client, skillName string) error {
	var skillList agentregistryv1alpha1.SkillCatalogList
	if err := c.List(ctx, &skillList, client.MatchingFields{
		IndexSkillName: skillName,
	}); err != nil {
		return err
	}

	// Extract version info
	versions := make([]CatalogVersionInfo, len(skillList.Items))
	for i := range skillList.Items {
		s := &skillList.Items[i]
		versions[i] = CatalogVersionInfo{
			Name:        s.Name,
			Version:     s.Spec.Version,
			PublishedAt: s.Status.PublishedAt,
			IsLatest:    s.Status.IsLatest,
			Published:   s.Status.Published,
		}
	}

	// Find latest version (currently filters by published, will be removed)
	latestName := findLatestVersion(versions)

	// Update isLatest flag for all versions
	for i := range skillList.Items {
		s := &skillList.Items[i]
		shouldBeLatest := (latestName != "" && s.Name == latestName)

		if s.Status.IsLatest != shouldBeLatest {
			s.Status.IsLatest = shouldBeLatest
			if err := c.Status().Update(ctx, s); err != nil {
				return err
			}
		}
	}

	return nil
}

// findLatestVersion finds the latest version from a list of catalog items
// TODO: Remove Published filter when publish/unpublish is removed
func findLatestVersion(versions []CatalogVersionInfo) string {
	var latest *CatalogVersionInfo
	var latestTimestamp time.Time

	for i := range versions {
		v := &versions[i]

		// TODO: Remove this filter when Published status is removed
		if !v.Published {
			continue
		}

		if latest == nil {
			latest = v
			if v.PublishedAt != nil {
				latestTimestamp = v.PublishedAt.Time
			}
			continue
		}

		var vTimestamp time.Time
		if v.PublishedAt != nil {
			vTimestamp = v.PublishedAt.Time
		}

		cmp := compareVersions(v.Version, latest.Version, vTimestamp, latestTimestamp)
		if cmp > 0 {
			latest = v
			latestTimestamp = vTimestamp
		}
	}

	if latest == nil {
		return ""
	}
	return latest.Name
}

// Version comparison utilities

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
