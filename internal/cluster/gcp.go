package cluster

import (
	"context"
	"encoding/base64"
	"fmt"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/option"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

// createGKEConfig creates a rest.Config for a GKE cluster using workload identity.
func (f *Factory) createGKEConfig(ctx context.Context, env *agentregistryv1alpha1.Environment) (*rest.Config, error) {
	// Get default credentials with GKE scope
	// In GKE with workload identity, this will use the pod's service account
	creds, err := google.FindDefaultCredentials(ctx, container.CloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("failed to get default credentials: %w", err)
	}

	// Create GKE service client
	gkeService, err := container.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to create GKE service: %w", err)
	}

	// Build the cluster API path
	location := env.Cluster.Zone
	if location == "" {
		location = env.Cluster.Region
	}
	if location == "" {
		return nil, fmt.Errorf("cluster zone or region is required for GKE")
	}

	clusterPath := fmt.Sprintf("projects/%s/locations/%s/clusters/%s",
		env.Cluster.ProjectID,
		location,
		env.Cluster.Name,
	)

	f.logger.Debug().
		Str("environment", env.Name).
		Str("cluster_path", clusterPath).
		Msg("fetching GKE cluster info")

	// Fetch cluster info from GKE API
	cluster, err := gkeService.Projects.Locations.Clusters.Get(clusterPath).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get GKE cluster %s: %w", clusterPath, err)
	}

	// Get a fresh token
	token, err := creds.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Decode the CA certificate
	caData, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return nil, fmt.Errorf("failed to decode CA certificate: %w", err)
	}

	// Build the rest.Config
	endpoint := cluster.Endpoint
	if endpoint != "" && endpoint[0:4] != "http" {
		endpoint = "https://" + endpoint
	}

	config, err := clientcmd.BuildConfigFromFlags(endpoint, "")
	if err != nil {
		return nil, fmt.Errorf("failed to build config from endpoint: %w", err)
	}

	config.TLSClientConfig = rest.TLSClientConfig{
		CAData: caData,
	}
	config.BearerToken = token.AccessToken

	// Set reasonable defaults
	config.QPS = 50
	config.Burst = 100

	f.logger.Info().
		Str("environment", env.Name).
		Str("cluster", env.Cluster.Name).
		Str("endpoint", endpoint).
		Msg("successfully created GKE cluster config")

	return config, nil
}
