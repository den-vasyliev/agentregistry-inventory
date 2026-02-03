// Package cluster provides a factory for creating Kubernetes clients for remote clusters.
package cluster

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

const (
	// DefaultCacheTTL is the default time-to-live for cached clients.
	DefaultCacheTTL = 30 * time.Minute
)

// ClientFactory creates Kubernetes clients for remote clusters.
type ClientFactory interface {
	// GetClient returns a client for the specified environment.
	// Clients are cached and reused when possible.
	GetClient(ctx context.Context, env *agentregistryv1alpha1.Environment, scheme *runtime.Scheme) (client.WithWatch, error)

	// InvalidateClient removes a cached client for the specified environment.
	InvalidateClient(envName string)
}

// cachedClient holds a client and its metadata for cache management.
type cachedClient struct {
	client     client.WithWatch
	configHash string
	createdAt  time.Time
}

// Factory implements ClientFactory with caching and support for multiple auth methods.
type Factory struct {
	localClient client.Client
	logger      zerolog.Logger
	cacheTTL    time.Duration

	mu     sync.RWMutex
	cache  map[string]*cachedClient
	scheme *runtime.Scheme
}

// NewFactory creates a new Factory with the given local client.
func NewFactory(localClient client.Client, logger zerolog.Logger) *Factory {
	return &Factory{
		localClient: localClient,
		logger:      logger.With().Str("component", "cluster-factory").Logger(),
		cacheTTL:    DefaultCacheTTL,
		cache:       make(map[string]*cachedClient),
	}
}

// GetClient returns a client for the specified environment.
func (f *Factory) GetClient(ctx context.Context, env *agentregistryv1alpha1.Environment, scheme *runtime.Scheme) (client.WithWatch, error) {
	f.scheme = scheme

	// Check if this is a local cluster request
	if f.isLocalCluster(env) {
		f.logger.Debug().
			Str("environment", env.Name).
			Msg("using local cluster client")
		return f.wrapLocalClient()
	}

	// Calculate config hash for cache invalidation
	configHash := f.computeConfigHash(env)

	// Check cache
	f.mu.RLock()
	cached, exists := f.cache[env.Name]
	f.mu.RUnlock()

	if exists {
		// Check if cache is still valid
		if cached.configHash == configHash && time.Since(cached.createdAt) < f.cacheTTL {
			f.logger.Debug().
				Str("environment", env.Name).
				Msg("using cached client")
			return cached.client, nil
		}
		// Cache is stale, invalidate it
		f.InvalidateClient(env.Name)
	}

	// Create new client
	remoteClient, err := f.createClient(ctx, env)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for environment %s: %w", env.Name, err)
	}

	// Cache the client
	f.mu.Lock()
	f.cache[env.Name] = &cachedClient{
		client:     remoteClient,
		configHash: configHash,
		createdAt:  time.Now(),
	}
	f.mu.Unlock()

	f.logger.Info().
		Str("environment", env.Name).
		Str("cluster", env.Cluster.Name).
		Msg("created and cached new client")

	return remoteClient, nil
}

// InvalidateClient removes a cached client for the specified environment.
func (f *Factory) InvalidateClient(envName string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.cache[envName]; exists {
		delete(f.cache, envName)
		f.logger.Debug().
			Str("environment", envName).
			Msg("invalidated cached client")
	}
}

// isLocalCluster determines if the environment refers to the local cluster.
func (f *Factory) isLocalCluster(env *agentregistryv1alpha1.Environment) bool {
	// Explicit local cluster
	if env.Cluster.Name == "local" || env.Cluster.Name == "" {
		return true
	}

	// No endpoint and no workload identity configured means local
	if env.Cluster.Endpoint == "" && !env.Cluster.UseWorkloadIdentity {
		// Check if provider-specific config is present
		if env.Provider == "" || (env.Provider == "gcp" && env.Cluster.ProjectID == "") {
			return true
		}
	}

	return false
}

// wrapLocalClient wraps the local client to implement client.WithWatch.
func (f *Factory) wrapLocalClient() (client.WithWatch, error) {
	// The local client from manager already implements WithWatch
	if withWatch, ok := f.localClient.(client.WithWatch); ok {
		return withWatch, nil
	}

	// This shouldn't happen with controller-runtime's client
	return nil, fmt.Errorf("local client does not implement client.WithWatch")
}

// createClient creates a new client for the environment based on its configuration.
func (f *Factory) createClient(ctx context.Context, env *agentregistryv1alpha1.Environment) (client.WithWatch, error) {
	var config *rest.Config
	var err error

	// Determine how to get the cluster config
	if env.Cluster.Endpoint != "" && env.Cluster.CAData != "" {
		// Static credentials provided
		config, err = f.createStaticConfig(env)
	} else if env.Provider == "gcp" && env.Cluster.UseWorkloadIdentity {
		// GCP workload identity
		config, err = f.createGKEConfig(ctx, env)
	} else if env.Cluster.UseWorkloadIdentity {
		// Generic workload identity - try to auto-detect provider
		config, err = f.createWorkloadIdentityConfig(ctx, env)
	} else {
		return nil, fmt.Errorf("no valid authentication method for environment %s: need endpoint+caData or workload identity", env.Name)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	// Create the client
	remoteClient, err := client.NewWithWatch(config, client.Options{
		Scheme: f.scheme,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client from config: %w", err)
	}

	return remoteClient, nil
}

// createWorkloadIdentityConfig attempts to create a config using workload identity,
// auto-detecting the provider if not specified.
func (f *Factory) createWorkloadIdentityConfig(ctx context.Context, env *agentregistryv1alpha1.Environment) (*rest.Config, error) {
	// If we have GCP project info, use GKE
	if env.Cluster.ProjectID != "" {
		return f.createGKEConfig(ctx, env)
	}

	// Add support for other providers (AWS, Azure) here in the future
	return nil, fmt.Errorf("unable to determine provider for workload identity; set provider field or provide projectId for GCP")
}

// computeConfigHash computes a hash of the environment config for cache invalidation.
func (f *Factory) computeConfigHash(env *agentregistryv1alpha1.Environment) string {
	// Hash relevant config fields that affect the client
	data := fmt.Sprintf("%s:%s:%s:%s:%s:%s:%t",
		env.Cluster.Name,
		env.Cluster.Endpoint,
		env.Cluster.CAData,
		env.Cluster.ProjectID,
		env.Cluster.Zone,
		env.Cluster.Region,
		env.Cluster.UseWorkloadIdentity,
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes for shorter hash
}

// CreateClientFunc returns a function suitable for use as controller.RemoteClientFactory.
func (f *Factory) CreateClientFunc() func(*agentregistryv1alpha1.Environment, *runtime.Scheme) (client.WithWatch, error) {
	return func(env *agentregistryv1alpha1.Environment, scheme *runtime.Scheme) (client.WithWatch, error) {
		return f.GetClient(context.Background(), env, scheme)
	}
}
