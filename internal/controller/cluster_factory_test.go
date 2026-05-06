package controller

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	"github.com/agentregistry-dev/agentregistry/internal/cluster"
)

// TestClusterFactory_LocalCluster verifies that cluster.Factory correctly returns
// the local WithWatch client when cluster.name is "local", without requiring
// workload identity or a remote endpoint.
func TestClusterFactory_LocalCluster(t *testing.T) {
	helper := SetupTestEnv(t, 30*time.Second, false)
	defer helper.Cleanup(t)

	localClient, err := client.NewWithWatch(helper.Config, client.Options{Scheme: helper.Scheme})
	require.NoError(t, err)

	logger := zerolog.Nop()
	factory := cluster.NewFactory(localClient, logger)

	env := &agentregistryv1alpha1.Environment{
		Name: "local",
		Cluster: agentregistryv1alpha1.ClusterConfig{
			Name:                "local",
			UseWorkloadIdentity: false,
		},
	}

	got, err := factory.GetClient(context.Background(), env, helper.Scheme)
	require.NoError(t, err)
	assert.NotNil(t, got)

	// Verify it works — list DiscoveryConfigs (empty result is fine)
	list := &agentregistryv1alpha1.DiscoveryConfigList{}
	err = got.List(context.Background(), list)
	require.NoError(t, err)
}
