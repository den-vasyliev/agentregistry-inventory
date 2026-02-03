package cluster

import (
	"encoding/base64"
	"fmt"

	"k8s.io/client-go/rest"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

// createStaticConfig creates a rest.Config using static credentials (endpoint + CA data).
func (f *Factory) createStaticConfig(env *agentregistryv1alpha1.Environment) (*rest.Config, error) {
	endpoint := env.Cluster.Endpoint
	if endpoint == "" {
		return nil, fmt.Errorf("cluster endpoint is required for static configuration")
	}

	// Ensure endpoint has https:// prefix
	if len(endpoint) > 4 && endpoint[0:4] != "http" {
		endpoint = "https://" + endpoint
	}

	// Decode CA data
	caData, err := base64.StdEncoding.DecodeString(env.Cluster.CAData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode CA data: %w", err)
	}

	config := &rest.Config{
		Host: endpoint,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: caData,
		},
	}

	// Set reasonable defaults
	config.QPS = 50
	config.Burst = 100

	f.logger.Info().
		Str("environment", env.Name).
		Str("cluster", env.Cluster.Name).
		Str("endpoint", endpoint).
		Msg("created static cluster config")

	return config, nil
}
