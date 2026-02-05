package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindAvailablePort(t *testing.T) {
	// Test happy path - should find a valid port
	port, err := FindAvailablePort()
	require.NoError(t, err)
	assert.Greater(t, port, uint16(0), "Port should be greater than 0")
	assert.Less(t, int(port), 65536, "Port should be less than 65536")
}

func TestFindAvailablePort_MultipleCalls(t *testing.T) {
	// Multiple calls should return different ports (or at least succeed)
	port1, err := FindAvailablePort()
	require.NoError(t, err)

	port2, err := FindAvailablePort()
	require.NoError(t, err)

	// Both should be valid
	assert.Greater(t, port1, uint16(0))
	assert.Greater(t, port2, uint16(0))
}
