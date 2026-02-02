package controller

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	kagentv1alpha2 "github.com/kagent-dev/kagent/go/api/v1alpha2"
	kmcpv1alpha1 "github.com/kagent-dev/kmcp/api/v1alpha1"
)

// DiscoveryConfigReconciler reconciles DiscoveryConfig and sets up informers per cluster/namespace
type DiscoveryConfigReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Logger  zerolog.Logger
	Manager manager.Manager

	// informers tracks active informers per environment/resourceType
	informersMu sync.RWMutex
	informers   map[string]cache.SharedIndexInformer
	stopChans   map[string]chan struct{}
}

// RemoteClientFactory creates clients for remote clusters (injectable for testing)
var RemoteClientFactory func(env *agentregistryv1alpha1.Environment, scheme *runtime.Scheme) (client.WithWatch, error)

// +kubebuilder:rbac:groups=agentregistry.dev,resources=discoveryconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentregistry.dev,resources=discoveryconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agentregistry.dev,resources=mcpservercatalogs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentregistry.dev,resources=agentcatalogs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentregistry.dev,resources=modelcatalogs,verbs=get;list;watch;create;update;patch;delete

// Reconcile sets up informers for each environment in the DiscoveryConfig
func (r *DiscoveryConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.With().Str("discoveryconfig", req.Name).Logger()

	// Initialize maps
	if r.informers == nil {
		r.informersMu.Lock()
		r.informers = make(map[string]cache.SharedIndexInformer)
		r.stopChans = make(map[string]chan struct{})
		r.informersMu.Unlock()
	}

	// Fetch DiscoveryConfig
	var config agentregistryv1alpha1.DiscoveryConfig
	if err := r.Get(ctx, req.NamespacedName, &config); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info().Msg("DiscoveryConfig deleted, stopping all informers")
			r.stopAllInformers()
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger.Info().Int("environments", len(config.Spec.Environments)).Msg("reconciling DiscoveryConfig")

	// Set up informers for each environment/namespace/resourceType
	for _, env := range config.Spec.Environments {
		resourceTypes := env.ResourceTypes
		if len(resourceTypes) == 0 {
			// Default to all types
			resourceTypes = []string{"MCPServer", "Agent", "ModelConfig"}
		}

		for _, ns := range env.Namespaces {
			for _, resourceType := range resourceTypes {
				envKey := fmt.Sprintf("%s/%s/%s/%s", config.Name, env.Name, ns, resourceType)

				r.informersMu.RLock()
				_, exists := r.informers[envKey]
				r.informersMu.RUnlock()

				if exists {
					logger.Debug().Str("key", envKey).Msg("informer already running")
					continue
				}

				if err := r.setupInformerForResource(ctx, &env, ns, resourceType, envKey, logger); err != nil {
					logger.Error().Err(err).Str("key", envKey).Msg("failed to setup informer")
					continue
				}
				logger.Info().Str("key", envKey).Msg("informer started")
			}
		}
	}

	// Update status
	now := metav1.Now()
	config.Status.LastSyncTime = &now
	config.Status.Conditions = []metav1.Condition{{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: config.Generation,
		LastTransitionTime: now,
		Reason:             "InformersStarted",
		Message:            fmt.Sprintf("Watching %d environments", len(config.Spec.Environments)),
	}}

	if err := r.Status().Update(ctx, &config); err != nil {
		if apierrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// setupInformerForResource creates a SharedIndexInformer for a specific resource type
func (r *DiscoveryConfigReconciler) setupInformerForResource(
	ctx context.Context,
	env *agentregistryv1alpha1.Environment,
	namespace string,
	resourceType string,
	envKey string,
	logger zerolog.Logger,
) error {
	logger = logger.With().Str("namespace", namespace).Str("cluster", env.Cluster.Name).Str("resourceType", resourceType).Logger()

	// Get client for remote cluster
	remoteClient, err := r.getRemoteClient(env)
	if err != nil {
		return fmt.Errorf("failed to create remote client: %w", err)
	}

	var informer cache.SharedIndexInformer

	switch resourceType {
	case "MCPServer":
		informer = r.createMCPServerInformer(ctx, remoteClient, namespace, env, logger)
	case "Agent":
		informer = r.createAgentInformer(ctx, remoteClient, namespace, env, logger)
	case "ModelConfig":
		informer = r.createModelConfigInformer(ctx, remoteClient, namespace, env, logger)
	default:
		return fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	// Store informer and stop channel
	stopCh := make(chan struct{})
	r.informersMu.Lock()
	r.informers[envKey] = informer
	r.stopChans[envKey] = stopCh
	r.informersMu.Unlock()

	// Run informer as manager runnable
	r.Manager.Add(manager.RunnableFunc(func(ctx context.Context) error {
		go informer.Run(stopCh)
		if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
			return fmt.Errorf("failed to sync informer for %s", envKey)
		}
		return nil
	}))

	return nil
}

// createMCPServerInformer creates an informer for MCPServer resources
func (r *DiscoveryConfigReconciler) createMCPServerInformer(
	ctx context.Context,
	remoteClient client.WithWatch,
	namespace string,
	env *agentregistryv1alpha1.Environment,
	logger zerolog.Logger,
) cache.SharedIndexInformer {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				list := &kmcpv1alpha1.MCPServerList{}
				err := remoteClient.List(context.Background(), list, client.InNamespace(namespace))
				return list, err
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return remoteClient.Watch(context.Background(), &kmcpv1alpha1.MCPServerList{}, client.InNamespace(namespace))
			},
		},
		&kmcpv1alpha1.MCPServer{},
		0,
		cache.Indexers{},
	)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			mcpServer := obj.(*kmcpv1alpha1.MCPServer)
			logger.Info().Str("mcpserver", mcpServer.Name).Msg("MCPServer added")
			if err := r.handleMCPServerAdd(ctx, mcpServer, env); err != nil {
				logger.Error().Err(err).Str("mcpserver", mcpServer.Name).Msg("failed to handle add")
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			mcpServer := newObj.(*kmcpv1alpha1.MCPServer)
			logger.Debug().Str("mcpserver", mcpServer.Name).Msg("MCPServer updated")
			if err := r.handleMCPServerAdd(ctx, mcpServer, env); err != nil {
				logger.Error().Err(err).Str("mcpserver", mcpServer.Name).Msg("failed to handle update")
			}
		},
		DeleteFunc: func(obj interface{}) {
			mcpServer := obj.(*kmcpv1alpha1.MCPServer)
			logger.Info().Str("mcpserver", mcpServer.Name).Msg("MCPServer deleted")
		},
	})

	return informer
}

// createAgentInformer creates an informer for Agent resources
func (r *DiscoveryConfigReconciler) createAgentInformer(
	ctx context.Context,
	remoteClient client.WithWatch,
	namespace string,
	env *agentregistryv1alpha1.Environment,
	logger zerolog.Logger,
) cache.SharedIndexInformer {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				list := &kagentv1alpha2.AgentList{}
				err := remoteClient.List(context.Background(), list, client.InNamespace(namespace))
				return list, err
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return remoteClient.Watch(context.Background(), &kagentv1alpha2.AgentList{}, client.InNamespace(namespace))
			},
		},
		&kagentv1alpha2.Agent{},
		0,
		cache.Indexers{},
	)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			agent := obj.(*kagentv1alpha2.Agent)
			logger.Info().Str("agent", agent.Name).Msg("Agent added")
			if err := r.handleAgentAdd(ctx, agent, env); err != nil {
				logger.Error().Err(err).Str("agent", agent.Name).Msg("failed to handle add")
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			agent := newObj.(*kagentv1alpha2.Agent)
			logger.Debug().Str("agent", agent.Name).Msg("Agent updated")
			if err := r.handleAgentAdd(ctx, agent, env); err != nil {
				logger.Error().Err(err).Str("agent", agent.Name).Msg("failed to handle update")
			}
		},
		DeleteFunc: func(obj interface{}) {
			agent := obj.(*kagentv1alpha2.Agent)
			logger.Info().Str("agent", agent.Name).Msg("Agent deleted")
		},
	})

	return informer
}

// createModelConfigInformer creates an informer for ModelConfig resources
func (r *DiscoveryConfigReconciler) createModelConfigInformer(
	ctx context.Context,
	remoteClient client.WithWatch,
	namespace string,
	env *agentregistryv1alpha1.Environment,
	logger zerolog.Logger,
) cache.SharedIndexInformer {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				list := &kagentv1alpha2.ModelConfigList{}
				err := remoteClient.List(context.Background(), list, client.InNamespace(namespace))
				return list, err
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return remoteClient.Watch(context.Background(), &kagentv1alpha2.ModelConfigList{}, client.InNamespace(namespace))
			},
		},
		&kagentv1alpha2.ModelConfig{},
		0,
		cache.Indexers{},
	)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			model := obj.(*kagentv1alpha2.ModelConfig)
			logger.Info().Str("modelconfig", model.Name).Msg("ModelConfig added")
			if err := r.handleModelConfigAdd(ctx, model, env); err != nil {
				logger.Error().Err(err).Str("modelconfig", model.Name).Msg("failed to handle add")
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			model := newObj.(*kagentv1alpha2.ModelConfig)
			logger.Debug().Str("modelconfig", model.Name).Msg("ModelConfig updated")
			if err := r.handleModelConfigAdd(ctx, model, env); err != nil {
				logger.Error().Err(err).Str("modelconfig", model.Name).Msg("failed to handle update")
			}
		},
		DeleteFunc: func(obj interface{}) {
			model := obj.(*kagentv1alpha2.ModelConfig)
			logger.Info().Str("modelconfig", model.Name).Msg("ModelConfig deleted")
		},
	})

	return informer
}

// handleMCPServerAdd creates/updates catalog entry for discovered MCPServer
func (r *DiscoveryConfigReconciler) handleMCPServerAdd(
	ctx context.Context,
	mcpServer *kmcpv1alpha1.MCPServer,
	env *agentregistryv1alpha1.Environment,
) error {
	// Catalog name: namespace-name (environment/cluster info in labels)
	catalogName := generateCatalogName(mcpServer.Namespace, mcpServer.Name)

	// Extract version
	version := "latest"
	if v, ok := mcpServer.Labels["kmcp.dev/version"]; ok {
		version = v
	}
	if v, ok := mcpServer.Labels["app.kubernetes.io/version"]; ok {
		version = v
	}

	// Extract metadata
	title := mcpServer.Name
	description := ""
	if t, ok := mcpServer.Annotations["kmcp.dev/project-name"]; ok {
		title = t
	}
	if d, ok := mcpServer.Annotations["kmcp.dev/description"]; ok {
		description = d
	}

	// Build transport
	transportType := "stdio"
	if mcpServer.Spec.TransportType == "http" {
		transportType = "streamable-http"
	}

	// Build labels
	labels := make(map[string]string)
	for k, v := range env.Labels {
		labels[k] = v
	}
	labels[discoveryLabel] = "true"
	labels[sourceKindLabel] = "MCPServer"
	labels[sourceNameLabel] = mcpServer.Name
	labels[sourceNSLabel] = mcpServer.Namespace
	labels["agentregistry.dev/environment"] = env.Name
	labels["agentregistry.dev/cluster"] = env.Cluster.Name

	catalog := agentregistryv1alpha1.MCPServerCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      catalogName,
			Namespace: "default",
			Labels:    labels,
		},
		Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
			Name:        fmt.Sprintf("%s/%s", mcpServer.Namespace, mcpServer.Name),
			Version:     version,
			Title:       title,
			Description: description,
			SourceRef: &agentregistryv1alpha1.SourceReference{
				Kind:      "MCPServer",
				Name:      mcpServer.Name,
				Namespace: mcpServer.Namespace,
			},
			Packages: []agentregistryv1alpha1.Package{{
				RegistryType: "oci",
				Identifier:   mcpServer.Spec.Deployment.Image,
				Transport:    agentregistryv1alpha1.Transport{Type: transportType},
			}},
		},
	}

	// Create or update
	existing := &agentregistryv1alpha1.MCPServerCatalog{}
	err := r.Get(ctx, client.ObjectKey{Name: catalogName, Namespace: "default"}, existing)

	if apierrors.IsNotFound(err) {
		return r.Create(ctx, &catalog)
	} else if err != nil {
		return err
	}

	existing.Spec = catalog.Spec
	existing.Labels = labels
	return r.Update(ctx, existing)
}

// handleAgentAdd creates/updates catalog entry for discovered Agent
func (r *DiscoveryConfigReconciler) handleAgentAdd(
	ctx context.Context,
	agent *kagentv1alpha2.Agent,
	env *agentregistryv1alpha1.Environment,
) error {
	// Catalog name: namespace-name (environment/cluster info in labels)
	catalogName := generateAgentCatalogName(agent.Namespace, agent.Name)

	// Extract version
	version := "latest"
	if v, ok := agent.Labels["app.kubernetes.io/version"]; ok {
		version = v
	}

	// Extract metadata
	title := agent.Name
	description := ""
	if agent.Spec.Description != "" {
		description = agent.Spec.Description
	}

	// Build labels
	labels := make(map[string]string)
	for k, v := range env.Labels {
		labels[k] = v
	}
	labels[discoveryLabel] = "true"
	labels[sourceKindLabel] = "Agent"
	labels[sourceNameLabel] = agent.Name
	labels[sourceNSLabel] = agent.Namespace
	labels["agentregistry.dev/environment"] = env.Name
	labels["agentregistry.dev/cluster"] = env.Cluster.Name

	catalog := agentregistryv1alpha1.AgentCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      catalogName,
			Namespace: "default",
			Labels:    labels,
		},
		Spec: agentregistryv1alpha1.AgentCatalogSpec{
			Name:        fmt.Sprintf("%s/%s", agent.Namespace, agent.Name),
			Version:     version,
			Title:       title,
			Description: description,
			Image:       "", // TODO: Extract from Agent spec if available
		},
	}

	// Create or update
	existing := &agentregistryv1alpha1.AgentCatalog{}
	err := r.Get(ctx, client.ObjectKey{Name: catalogName, Namespace: "default"}, existing)

	if apierrors.IsNotFound(err) {
		return r.Create(ctx, &catalog)
	} else if err != nil {
		return err
	}

	existing.Spec = catalog.Spec
	existing.Labels = labels
	return r.Update(ctx, existing)
}

// handleModelConfigAdd creates/updates catalog entry for discovered ModelConfig
func (r *DiscoveryConfigReconciler) handleModelConfigAdd(
	ctx context.Context,
	model *kagentv1alpha2.ModelConfig,
	env *agentregistryv1alpha1.Environment,
) error {
	// Catalog name: namespace-name (environment/cluster info in labels)
	catalogName := generateModelCatalogName(model.Namespace, model.Name)

	// Build labels
	labels := make(map[string]string)
	for k, v := range env.Labels {
		labels[k] = v
	}
	labels[discoveryLabel] = "true"
	labels[sourceKindLabel] = "ModelConfig"
	labels[sourceNameLabel] = model.Name
	labels[sourceNSLabel] = model.Namespace
	labels["agentregistry.dev/environment"] = env.Name
	labels["agentregistry.dev/cluster"] = env.Cluster.Name

	catalog := agentregistryv1alpha1.ModelCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      catalogName,
			Namespace: "default",
			Labels:    labels,
		},
		Spec: agentregistryv1alpha1.ModelCatalogSpec{
			Name:     fmt.Sprintf("%s/%s", model.Namespace, model.Name),
			Provider: string(model.Spec.Provider),
			Model:    model.Spec.Model,
			SourceRef: &agentregistryv1alpha1.SourceReference{
				Kind:      "ModelConfig",
				Name:      model.Name,
				Namespace: model.Namespace,
			},
		},
	}

	// Create or update
	existing := &agentregistryv1alpha1.ModelCatalog{}
	err := r.Get(ctx, client.ObjectKey{Name: catalogName, Namespace: "default"}, existing)

	if apierrors.IsNotFound(err) {
		return r.Create(ctx, &catalog)
	} else if err != nil {
		return err
	}

	existing.Spec = catalog.Spec
	existing.Labels = labels
	return r.Update(ctx, existing)
}

// getRemoteClient gets or creates a client for remote cluster
func (r *DiscoveryConfigReconciler) getRemoteClient(env *agentregistryv1alpha1.Environment) (client.WithWatch, error) {
	// Use factory if provided (for testing)
	if RemoteClientFactory != nil {
		return RemoteClientFactory(env, r.Scheme)
	}
	return nil, fmt.Errorf("remote client factory not configured")
}

// stopAllInformers stops all running informers
func (r *DiscoveryConfigReconciler) stopAllInformers() {
	r.informersMu.Lock()
	defer r.informersMu.Unlock()

	for key, stopCh := range r.stopChans {
		close(stopCh)
		r.Logger.Info().Str("key", key).Msg("stopped informer")
	}
	r.informers = make(map[string]cache.SharedIndexInformer)
	r.stopChans = make(map[string]chan struct{})
}

// SetupWithManager sets up the controller
func (r *DiscoveryConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Manager = mgr
	return ctrl.NewControllerManagedBy(mgr).
		For(&agentregistryv1alpha1.DiscoveryConfig{}).
		Complete(r)
}
