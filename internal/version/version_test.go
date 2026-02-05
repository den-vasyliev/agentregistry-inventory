package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionVariables_Defaults(t *testing.T) {
	// Test that version variables have expected default values
	assert.Equal(t, "dev", Version)
	assert.Equal(t, "unknown", GitCommit)
	assert.Equal(t, "unknown", BuildDate)
	assert.Equal(t, "localhost:5001", DockerRegistry)
}

func TestVersionVariables_CanBeSet(t *testing.T) {
	// Store originals
	origVersion := Version
	origCommit := GitCommit
	origDate := BuildDate
	origRegistry := DockerRegistry

	// Modify values (simulating build-time injection)
	Version = "1.0.0"
	GitCommit = "abc123"
	BuildDate = "2024-01-01"
	DockerRegistry = "ghcr.io/agentregistry-dev"

	// Verify changes
	assert.Equal(t, "1.0.0", Version)
	assert.Equal(t, "abc123", GitCommit)
	assert.Equal(t, "2024-01-01", BuildDate)
	assert.Equal(t, "ghcr.io/agentregistry-dev", DockerRegistry)

	// Restore originals
	Version = origVersion
	GitCommit = origCommit
	BuildDate = origDate
	DockerRegistry = origRegistry
}
