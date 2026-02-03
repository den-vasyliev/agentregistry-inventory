package cluster

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

func TestIsLocalCluster(t *testing.T) {
	logger := zerolog.Nop()
	factory := NewFactory(nil, logger)

	tests := []struct {
		name     string
		env      *agentregistryv1alpha1.Environment
		expected bool
	}{
		{
			name: "explicit local cluster",
			env: &agentregistryv1alpha1.Environment{
				Name: "local-env",
				Cluster: agentregistryv1alpha1.ClusterConfig{
					Name: "local",
				},
			},
			expected: true,
		},
		{
			name: "empty cluster name",
			env: &agentregistryv1alpha1.Environment{
				Name: "unnamed-env",
				Cluster: agentregistryv1alpha1.ClusterConfig{
					Name: "",
				},
			},
			expected: true,
		},
		{
			name: "no endpoint and no workload identity",
			env: &agentregistryv1alpha1.Environment{
				Name: "simple-env",
				Cluster: agentregistryv1alpha1.ClusterConfig{
					Name:                "some-cluster",
					UseWorkloadIdentity: false,
				},
			},
			expected: true,
		},
		{
			name: "GCP workload identity configured",
			env: &agentregistryv1alpha1.Environment{
				Name:     "gke-env",
				Provider: "gcp",
				Cluster: agentregistryv1alpha1.ClusterConfig{
					Name:                "gke-cluster",
					ProjectID:           "my-project",
					Zone:                "us-central1-a",
					UseWorkloadIdentity: true,
				},
			},
			expected: false,
		},
		{
			name: "static endpoint configured",
			env: &agentregistryv1alpha1.Environment{
				Name: "static-env",
				Cluster: agentregistryv1alpha1.ClusterConfig{
					Name:     "remote-cluster",
					Endpoint: "https://10.0.0.1:6443",
					CAData:   "base64-ca-data",
				},
			},
			expected: false,
		},
		{
			name: "workload identity without GCP provider info returns false (requires remote connection attempt)",
			env: &agentregistryv1alpha1.Environment{
				Name: "wi-no-project",
				Cluster: agentregistryv1alpha1.ClusterConfig{
					Name:                "some-cluster",
					UseWorkloadIdentity: true,
				},
			},
			expected: false,
		},
		{
			name: "GCP provider with project ID",
			env: &agentregistryv1alpha1.Environment{
				Name:     "gke-with-project",
				Provider: "gcp",
				Cluster: agentregistryv1alpha1.ClusterConfig{
					Name:                "gke-cluster",
					ProjectID:           "my-project",
					UseWorkloadIdentity: true,
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := factory.isLocalCluster(tt.env)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestComputeConfigHash(t *testing.T) {
	logger := zerolog.Nop()
	factory := NewFactory(nil, logger)

	env1 := &agentregistryv1alpha1.Environment{
		Cluster: agentregistryv1alpha1.ClusterConfig{
			Name:      "cluster-1",
			ProjectID: "project-1",
			Zone:      "us-central1-a",
		},
	}

	env2 := &agentregistryv1alpha1.Environment{
		Cluster: agentregistryv1alpha1.ClusterConfig{
			Name:      "cluster-1",
			ProjectID: "project-1",
			Zone:      "us-central1-a",
		},
	}

	env3 := &agentregistryv1alpha1.Environment{
		Cluster: agentregistryv1alpha1.ClusterConfig{
			Name:      "cluster-2",
			ProjectID: "project-1",
			Zone:      "us-central1-a",
		},
	}

	// Same config should produce same hash
	hash1 := factory.computeConfigHash(env1)
	hash2 := factory.computeConfigHash(env2)
	assert.Equal(t, hash1, hash2, "identical configs should have identical hashes")

	// Different config should produce different hash
	hash3 := factory.computeConfigHash(env3)
	assert.NotEqual(t, hash1, hash3, "different configs should have different hashes")

	// Hash should be deterministic
	hash1Again := factory.computeConfigHash(env1)
	assert.Equal(t, hash1, hash1Again, "hash should be deterministic")
}

func TestInvalidateClient(t *testing.T) {
	logger := zerolog.Nop()
	factory := NewFactory(nil, logger)

	// Add a cached entry
	factory.cache["test-env"] = &cachedClient{
		client:     nil,
		configHash: "abc123",
	}

	// Verify it exists
	assert.Contains(t, factory.cache, "test-env")

	// Invalidate it
	factory.InvalidateClient("test-env")

	// Verify it's removed
	assert.NotContains(t, factory.cache, "test-env")

	// Invalidating non-existent entry should not panic
	factory.InvalidateClient("non-existent")
}

func TestNewFactory(t *testing.T) {
	logger := zerolog.Nop()
	factory := NewFactory(nil, logger)

	assert.NotNil(t, factory)
	assert.NotNil(t, factory.cache)
	assert.Equal(t, DefaultCacheTTL, factory.cacheTTL)
}
