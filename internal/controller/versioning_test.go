package controller

import (
	"testing"
	"time"
)

func TestIsSemanticVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		{"valid semver", "1.0.0", true},
		{"valid semver with v prefix", "v1.0.0", true},
		{"valid semver with prerelease", "1.0.0-alpha.1", true},
		{"valid semver with build metadata", "1.0.0+build.123", true},
		{"invalid - missing patch", "1.0", false},
		{"invalid - only major", "1", false},
		{"invalid - non-numeric", "abc", false},
		{"invalid - empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSemanticVersion(tt.version)
			if result != tt.expected {
				t.Errorf("isSemanticVersion(%q) = %v, expected %v", tt.version, result, tt.expected)
			}
		})
	}
}

func TestCompareSemanticVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{"equal versions", "1.0.0", "1.0.0", 0},
		{"v1 greater (major)", "2.0.0", "1.0.0", 1},
		{"v1 less (major)", "1.0.0", "2.0.0", -1},
		{"v1 greater (minor)", "1.2.0", "1.1.0", 1},
		{"v1 less (patch)", "1.0.0", "1.0.1", -1},
		{"prerelease less than release", "1.0.0-alpha", "1.0.0", -1},
		{"with v prefix", "v1.0.0", "v1.0.1", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareSemanticVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("compareSemanticVersions(%q, %q) = %v, expected %v", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	later := now.Add(1 * time.Hour)

	tests := []struct {
		name       string
		v1         string
		v2         string
		timestamp1 time.Time
		timestamp2 time.Time
		expected   int
	}{
		{"both semver - v1 greater", "2.0.0", "1.0.0", now, now, 1},
		{"both semver - equal", "1.0.0", "1.0.0", now, now, 0},
		{"neither semver - later timestamp wins", "latest", "main", later, earlier, 1},
		{"neither semver - earlier timestamp", "latest", "main", earlier, later, -1},
		{"semver beats non-semver", "1.0.0", "latest", earlier, later, 1},
		{"non-semver loses to semver", "main", "1.0.0", later, earlier, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareVersions(tt.v1, tt.v2, tt.timestamp1, tt.timestamp2)
			if result != tt.expected {
				t.Errorf("compareVersions(%q, %q, _, _) = %v, expected %v",
					tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestEnsureVPrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.0.0", "v1.0.0"},
		{"v1.0.0", "v1.0.0"},
		{"", "v"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ensureVPrefix(tt.input)
			if result != tt.expected {
				t.Errorf("ensureVPrefix(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
