package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"strconv"
	"strings"

	kagentv1alpha2 "github.com/kagent-dev/kagent/go/api/v1alpha2"
	kmcpv1alpha1 "github.com/kagent-dev/kmcp/api/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	sigyaml "sigs.k8s.io/yaml"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	"github.com/agentregistry-dev/agentregistry/internal/runtime/translation/api"
	"github.com/agentregistry-dev/agentregistry/internal/runtime/translation/kagent"
)

// RegistryDeploymentReconciler reconciles a RegistryDeployment object
type RegistryDeploymentReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	Logger              zerolog.Logger
	RemoteClientFactory func(env *agentregistryv1alpha1.Environment, scheme *runtime.Scheme) (client.WithWatch, error)
}

const (
	finalizerName       = "agentregistry.dev/finalizer"
	defaultNamespace    = "kagent"
	managedByLabel      = "agentregistry.dev/managed-by"
	deploymentNameLabel = "agentregistry.dev/deployment-name"
	deploymentNSLabel   = "agentregistry.dev/deployment-namespace"
)

// +kubebuilder:rbac:groups=agentregistry.dev,resources=registrydeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentregistry.dev,resources=registrydeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agentregistry.dev,resources=registrydeployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=kagent.dev,resources=agents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kagent.dev,resources=remotemcpservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kmcp.io,resources=mcpservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile handles RegistryDeployment reconciliation
func (r *RegistryDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.With().Str("name", req.Name).Str("namespace", req.Namespace).Logger()

	// Skip resources with empty namespace (invalid legacy cluster-scoped resources)
	if req.Namespace == "" {
		logger.Warn().Msg("skipping RegistryDeployment with empty namespace (invalid resource)")
		return ctrl.Result{}, nil
	}

	// Fetch the RegistryDeployment
	var deployment agentregistryv1alpha1.RegistryDeployment
	if err := r.Get(ctx, req.NamespacedName, &deployment); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Trace().
		Str("resourceName", deployment.Spec.ResourceName).
		Str("version", deployment.Spec.Version).
		Str("resourceType", string(deployment.Spec.ResourceType)).
		Str("runtime", string(deployment.Spec.Runtime)).
		Msg("reconciling RegistryDeployment")

	// Handle deletion
	if !deployment.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &deployment)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&deployment, finalizerName) {
		controllerutil.AddFinalizer(&deployment, finalizerName)
		if err := r.Update(ctx, &deployment); err != nil {
			return ctrl.Result{}, err
		}
		// Re-fetch after the finalizer write so the rest of this loop
		// works against the latest resourceVersion.
		if err := r.Get(ctx, req.NamespacedName, &deployment); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	// Reconcile based on resource type
	var err error
	switch deployment.Spec.ResourceType {
	case agentregistryv1alpha1.ResourceTypeMCP:
		err = r.reconcileMCPDeployment(ctx, &deployment)
	case agentregistryv1alpha1.ResourceTypeAgent:
		err = r.reconcileAgentDeployment(ctx, &deployment)
	default:
		err = fmt.Errorf("unknown resource type: %s", deployment.Spec.ResourceType)
	}

	if err != nil {
		logger.Error().Err(err).Msg("failed to reconcile deployment")
		deployment.Status.Phase = agentregistryv1alpha1.DeploymentPhaseFailed
		deployment.Status.Message = err.Error()
	} else {
		// Check if managed resources are actually ready
		ready, message := r.checkManagedResourcesReady(ctx, &deployment)
		if ready {
			deployment.Status.Phase = agentregistryv1alpha1.DeploymentPhaseRunning
			deployment.Status.Message = ""
		} else {
			deployment.Status.Phase = agentregistryv1alpha1.DeploymentPhasePending
			deployment.Status.Message = message
		}
	}

	// Update status
	now := metav1.Now()
	deployment.Status.UpdatedAt = &now
	if deployment.Status.DeployedAt == nil {
		deployment.Status.DeployedAt = &now
	}
	deployment.Status.ObservedGeneration = deployment.Generation

	if err := r.Status().Update(ctx, &deployment); err != nil {
		logger.Error().Err(err).Msg("failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, err
}

// reconcileMCPDeployment reconciles an MCP server deployment
func (r *RegistryDeploymentReconciler) reconcileMCPDeployment(ctx context.Context, deployment *agentregistryv1alpha1.RegistryDeployment) error {
	// Look up the MCPServerCatalog
	var serverList agentregistryv1alpha1.MCPServerCatalogList
	if err := r.List(ctx, &serverList, client.MatchingFields{
		IndexMCPServerName: deployment.Spec.ResourceName,
	}); err != nil {
		return fmt.Errorf("failed to list MCP servers: %w", err)
	}

	// Find the specific version
	var catalogEntry *agentregistryv1alpha1.MCPServerCatalog
	for i := range serverList.Items {
		s := &serverList.Items[i]
		if s.Spec.Version == deployment.Spec.Version {
			catalogEntry = s
			break
		}
	}

	if catalogEntry == nil {
		return fmt.Errorf("MCP server %s version %s not found", deployment.Spec.ResourceName, deployment.Spec.Version)
	}

	// Validate publisher identity before deploying
	if err := validatePublisherIdentity(catalogEntry.Spec.Metadata); err != nil {
		return fmt.Errorf("deployment blocked for %s %s: %w", deployment.Spec.ResourceName, deployment.Spec.Version, err)
	}

	// Mark as managed if not already set
	if catalogEntry.Status.ManagementType != agentregistryv1alpha1.ManagementTypeManaged {
		catalogEntry.Status.ManagementType = agentregistryv1alpha1.ManagementTypeManaged
		if err := r.Status().Update(ctx, catalogEntry); err != nil {
			return fmt.Errorf("failed to update catalog management type: %w", err)
		}
	}

	// Resolve the target client and environment
	env, targetClient, clusterName, err := r.getTargetClientAndEnv(ctx, deployment)
	if err != nil {
		return fmt.Errorf("failed to resolve target: %w", err)
	}
	mcpURL := ""
	if env != nil {
		mcpURL = env.MCPToolServerURL
	}

	// Convert catalog to runtime format
	mcpServer, err := r.convertCatalogToMCPServer(catalogEntry, deployment)
	if err != nil {
		return fmt.Errorf("failed to convert catalog to MCP server: %w", err)
	}

	// Use KAgent translator to create Kubernetes resources
	translator := kagent.NewTranslator()
	desiredState := &api.DesiredState{
		MCPServers: []*api.MCPServer{mcpServer},
	}

	runtimeConfig, err := translator.TranslateRuntimeConfig(ctx, desiredState)
	if err != nil {
		return fmt.Errorf("failed to translate runtime config: %w", err)
	}

	// Apply Kubernetes resources
	managedResources := []agentregistryv1alpha1.ManagedResource{}

	// Apply MCPServers (local)
	for _, mcpServer := range runtimeConfig.Kubernetes.MCPServers {
		r.setOwnerLabels(mcpServer, deployment)
		if err := r.applyObj(ctx, mcpURL, targetClient, mcpServer); err != nil {
			return fmt.Errorf("failed to apply MCPServer: %w", err)
		}
		managedResources = append(managedResources, agentregistryv1alpha1.ManagedResource{
			APIVersion: mcpServer.APIVersion,
			Kind:       mcpServer.Kind,
			Name:       mcpServer.Name,
			Namespace:  mcpServer.Namespace,
			Cluster:    clusterName,
		})
	}

	// Apply RemoteMCPServers
	for _, remoteMCP := range runtimeConfig.Kubernetes.RemoteMCPServers {
		r.setOwnerLabels(remoteMCP, deployment)
		if err := r.applyObj(ctx, mcpURL, targetClient, remoteMCP); err != nil {
			return fmt.Errorf("failed to apply RemoteMCPServer: %w", err)
		}
		managedResources = append(managedResources, agentregistryv1alpha1.ManagedResource{
			APIVersion: remoteMCP.APIVersion,
			Kind:       remoteMCP.Kind,
			Name:       remoteMCP.Name,
			Namespace:  remoteMCP.Namespace,
			Cluster:    clusterName,
		})
	}

	deployment.Status.ManagedResources = managedResources
	return nil
}

// reconcileAgentDeployment reconciles an Agent deployment
func (r *RegistryDeploymentReconciler) reconcileAgentDeployment(ctx context.Context, deployment *agentregistryv1alpha1.RegistryDeployment) error {
	// Look up the AgentCatalog
	var agentList agentregistryv1alpha1.AgentCatalogList
	if err := r.List(ctx, &agentList, client.MatchingFields{
		IndexAgentName: deployment.Spec.ResourceName,
	}); err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}

	// Find the specific version
	var catalogEntry *agentregistryv1alpha1.AgentCatalog
	for i := range agentList.Items {
		a := &agentList.Items[i]
		if a.Spec.Version == deployment.Spec.Version {
			catalogEntry = a
			break
		}
	}

	if catalogEntry == nil {
		return fmt.Errorf("agent %s version %s not found", deployment.Spec.ResourceName, deployment.Spec.Version)
	}

	// Validate publisher identity before deploying
	if err := validatePublisherIdentity(catalogEntry.Spec.Metadata); err != nil {
		return fmt.Errorf("deployment blocked for %s %s: %w", deployment.Spec.ResourceName, deployment.Spec.Version, err)
	}

	// Mark as managed if not already set
	if catalogEntry.Status.ManagementType != agentregistryv1alpha1.ManagementTypeManaged {
		catalogEntry.Status.ManagementType = agentregistryv1alpha1.ManagementTypeManaged
		if err := r.Status().Update(ctx, catalogEntry); err != nil {
			return fmt.Errorf("failed to update catalog management type: %w", err)
		}
	}

	// Resolve the target client and environment
	env, targetClient, clusterName, err := r.getTargetClientAndEnv(ctx, deployment)
	if err != nil {
		return fmt.Errorf("failed to resolve target: %w", err)
	}
	mcpURL := ""
	if env != nil {
		mcpURL = env.MCPToolServerURL
	}

	// Convert catalog to runtime format
	agent, err := r.convertCatalogToAgent(catalogEntry, deployment)
	if err != nil {
		return fmt.Errorf("failed to convert catalog to agent: %w", err)
	}

	// Use KAgent translator to create Kubernetes resources
	translator := kagent.NewTranslator()
	desiredState := &api.DesiredState{
		Agents: []*api.Agent{agent},
	}

	runtimeConfig, err := translator.TranslateRuntimeConfig(ctx, desiredState)
	if err != nil {
		return fmt.Errorf("failed to translate runtime config: %w", err)
	}

	// Apply Kubernetes resources
	managedResources := []agentregistryv1alpha1.ManagedResource{}

	// Apply ConfigMaps
	for _, cm := range runtimeConfig.Kubernetes.ConfigMaps {
		r.setOwnerLabels(cm, deployment)
		if err := r.applyObj(ctx, mcpURL, targetClient, cm); err != nil {
			return fmt.Errorf("failed to apply ConfigMap: %w", err)
		}
		managedResources = append(managedResources, agentregistryv1alpha1.ManagedResource{
			APIVersion: "v1",
			Kind:       "ConfigMap",
			Name:       cm.Name,
			Namespace:  cm.Namespace,
			Cluster:    clusterName,
		})
	}

	// Apply Agents
	for _, agent := range runtimeConfig.Kubernetes.Agents {
		r.setOwnerLabels(agent, deployment)
		if err := r.applyObj(ctx, mcpURL, targetClient, agent); err != nil {
			return fmt.Errorf("failed to apply Agent: %w", err)
		}
		managedResources = append(managedResources, agentregistryv1alpha1.ManagedResource{
			APIVersion: agent.APIVersion,
			Kind:       agent.Kind,
			Name:       agent.Name,
			Namespace:  agent.Namespace,
			Cluster:    clusterName,
		})
	}

	deployment.Status.ManagedResources = managedResources
	return nil
}

// convertCatalogToMCPServer converts an MCPServerCatalog to the runtime API format
func (r *RegistryDeploymentReconciler) convertCatalogToMCPServer(catalog *agentregistryv1alpha1.MCPServerCatalog, deployment *agentregistryv1alpha1.RegistryDeployment) (*api.MCPServer, error) {
	// Determine if we should use remote or local
	useRemote := len(catalog.Spec.Remotes) > 0 && (deployment.Spec.PreferRemote || len(catalog.Spec.Packages) == 0)

	targetNamespace := deployment.Spec.Namespace
	if targetNamespace == "" {
		targetNamespace = defaultNamespace
	}

	if useRemote {
		// Use remote transport
		remote := catalog.Spec.Remotes[0]
		headers := make([]api.HeaderValue, 0, len(remote.Headers))
		for _, h := range remote.Headers {
			value := h.Value
			// Substitute from config
			if v, ok := deployment.Spec.Config[h.Name]; ok {
				value = v
			}
			headers = append(headers, api.HeaderValue{
				Name:  h.Name,
				Value: value,
			})
		}

		host, port, path := parseURLComponents(remote.URL)
		return &api.MCPServer{
			Name:          generateInternalName(catalog.Spec.Name),
			MCPServerType: api.MCPServerTypeRemote,
			Namespace:     targetNamespace,
			Remote: &api.RemoteMCPServer{
				Host:    host,
				Port:    port,
				Path:    path,
				Headers: headers,
			},
		}, nil
	}

	// Use local package deployment
	if len(catalog.Spec.Packages) == 0 {
		return nil, fmt.Errorf("no packages available for server %s", catalog.Spec.Name)
	}

	pkg := catalog.Spec.Packages[0]

	// Build environment variables from package spec and deployment config
	env := make(map[string]string)
	for _, envVar := range pkg.EnvironmentVariables {
		if v, ok := deployment.Spec.Config[envVar.Name]; ok {
			env[envVar.Name] = v
		} else if envVar.Value != "" {
			env[envVar.Name] = envVar.Value
		}
	}

	// Build arguments
	var args []string
	for _, arg := range pkg.RuntimeArguments {
		if v, ok := deployment.Spec.Config[arg.Name]; ok {
			args = append(args, v)
		} else if arg.Value != "" {
			args = append(args, arg.Value)
		}
	}
	// Add package identifier based on registry type
	args = append(args, pkg.Identifier)
	for _, arg := range pkg.PackageArguments {
		if v, ok := deployment.Spec.Config[arg.Name]; ok {
			args = append(args, v)
		} else if arg.Value != "" {
			args = append(args, arg.Value)
		}
	}

	// Determine image and command based on registry type
	image, cmd := getImageAndCommand(pkg.RegistryType, pkg.RuntimeHint)

	// For OCI registry, use identifier as the image
	if pkg.RegistryType == "oci" {
		image = pkg.Identifier
		cmd = ""   // OCI images have their own entrypoint
		args = nil // OCI images use their own CMD/ARGS
	}

	var transportType api.TransportType
	var httpTransport *api.HTTPTransport

	switch pkg.Transport.Type {
	case "http", "streamable-http":
		transportType = api.TransportTypeHTTP
		// HTTP transport requires port/path config
		port := uint32(8080)
		path := "/"
		if pkg.Transport.URL != "" {
			_, port, path = parseURLComponents(pkg.Transport.URL)
		}
		httpTransport = &api.HTTPTransport{
			Port: port,
			Path: path,
		}
	default:
		// Default to stdio for local packages (npm, pypi)
		transportType = api.TransportTypeStdio
	}

	return &api.MCPServer{
		Name:          generateInternalName(catalog.Spec.Name),
		MCPServerType: api.MCPServerTypeLocal,
		Namespace:     targetNamespace,
		Local: &api.LocalMCPServer{
			Deployment: api.MCPServerDeployment{
				Image: image,
				Cmd:   cmd,
				Args:  args,
				Env:   env,
			},
			TransportType: transportType,
			HTTP:          httpTransport,
		},
	}, nil
}

// convertCatalogToAgent converts an AgentCatalog to the runtime API format
func (r *RegistryDeploymentReconciler) convertCatalogToAgent(catalog *agentregistryv1alpha1.AgentCatalog, deployment *agentregistryv1alpha1.RegistryDeployment) (*api.Agent, error) {
	targetNamespace := deployment.Spec.Namespace
	if targetNamespace == "" {
		targetNamespace = defaultNamespace
	}

	// Build environment variables
	env := make(map[string]string)
	if deployment.Spec.Config != nil {
		env = maps.Clone(deployment.Spec.Config)
	}

	// Set standard agent environment variables
	env["KAGENT_URL"] = "http://localhost"
	env["KAGENT_NAME"] = catalog.Spec.Name
	env["KAGENT_NAMESPACE"] = targetNamespace
	env["AGENT_NAME"] = catalog.Spec.Name
	if catalog.Spec.ModelProvider != "" {
		env["MODEL_PROVIDER"] = catalog.Spec.ModelProvider
	}
	if catalog.Spec.ModelName != "" {
		env["MODEL_NAME"] = catalog.Spec.ModelName
	}

	return &api.Agent{
		Name:    catalog.Spec.Name,
		Version: catalog.Spec.Version,
		Deployment: api.AgentDeployment{
			Image: catalog.Spec.Image,
			Env:   env,
		},
	}, nil
}

// handleDeletion handles the deletion of a RegistryDeployment
func (r *RegistryDeploymentReconciler) handleDeletion(ctx context.Context, deployment *agentregistryv1alpha1.RegistryDeployment) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(deployment, finalizerName) {
		return ctrl.Result{}, nil
	}

	// Resolve the target client and environment for deletion
	env, targetClient, _, err := r.getTargetClientAndEnv(ctx, deployment)
	if err != nil {
		r.Logger.Error().Err(err).Msg("failed to resolve target for deletion, falling back to local client")
		targetClient = r.Client
	}
	mcpURL := ""
	if env != nil {
		mcpURL = env.MCPToolServerURL
	}

	// Delete managed resources
	for _, res := range deployment.Status.ManagedResources {
		if err := r.deleteObj(ctx, mcpURL, targetClient, res); err != nil {
			r.Logger.Error().Err(err).
				Str("kind", res.Kind).
				Str("name", res.Name).
				Str("namespace", res.Namespace).
				Msg("failed to delete managed resource")
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(deployment, finalizerName)
	if err := r.Update(ctx, deployment); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// getTargetClientAndEnv resolves the target client and environment for a deployment.
// If the deployment specifies an environment, it looks up the environment from DiscoveryConfig
// resources and returns a remote client along with the environment. Otherwise, it returns
// the local client and nil environment.
func (r *RegistryDeploymentReconciler) getTargetClientAndEnv(ctx context.Context, deployment *agentregistryv1alpha1.RegistryDeployment) (*agentregistryv1alpha1.Environment, client.Client, string, error) {
	envName := deployment.Spec.Environment
	if envName == "" {
		return nil, r.Client, "", nil
	}

	// List DiscoveryConfigs in the same namespace to find the environment
	var dcList agentregistryv1alpha1.DiscoveryConfigList
	if err := r.List(ctx, &dcList, client.InNamespace(deployment.Namespace)); err != nil {
		return nil, nil, "", fmt.Errorf("failed to list DiscoveryConfigs: %w", err)
	}

	for i := range dcList.Items {
		for j := range dcList.Items[i].Spec.Environments {
			env := &dcList.Items[i].Spec.Environments[j]
			if env.Name == envName {
				if !env.DeployEnabled {
					return nil, nil, "", fmt.Errorf("deployment to environment %q is not allowed (deployEnabled is false)", envName)
				}

				// If MCP tool server is available, we don't need a K8s client
				if env.MCPToolServerURL != "" {
					return env, nil, env.Cluster.Name, nil
				}

				factory := r.RemoteClientFactory
				if factory == nil {
					factory = RemoteClientFactory
				}
				if factory == nil {
					return nil, nil, "", fmt.Errorf("remote client factory not configured, cannot deploy to environment %q", envName)
				}
				remoteClient, err := factory(env, r.Scheme)
				if err != nil {
					return nil, nil, "", fmt.Errorf("failed to create remote client for environment %q: %w", envName, err)
				}
				return env, remoteClient, env.Cluster.Name, nil
			}
		}
	}

	return nil, nil, "", fmt.Errorf("environment %q not found in any DiscoveryConfig in namespace %q", envName, deployment.Namespace)
}

// applyObj dispatches to MCP or direct K8s apply based on the mcpURL.
func (r *RegistryDeploymentReconciler) applyObj(ctx context.Context, mcpURL string, targetClient client.Client, obj client.Object) error {
	if mcpURL != "" {
		return r.applyViaMCP(ctx, mcpURL, obj)
	}
	return r.applyResource(ctx, targetClient, obj)
}

// deleteObj dispatches to MCP or direct K8s delete based on the mcpURL.
func (r *RegistryDeploymentReconciler) deleteObj(ctx context.Context, mcpURL string, targetClient client.Client, res agentregistryv1alpha1.ManagedResource) error {
	if mcpURL != "" {
		return r.deleteViaMCP(ctx, mcpURL, res)
	}
	return r.deleteResource(ctx, targetClient, res)
}

// applyViaMCP applies a Kubernetes resource via the MCP tool server's k8s_apply_manifest tool.
func (r *RegistryDeploymentReconciler) applyViaMCP(ctx context.Context, mcpURL string, obj client.Object) error {
	yamlBytes, err := sigyaml.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal object to YAML: %w", err)
	}

	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "agentregistry",
		Version: "1.0.0",
	}, nil)

	session, err := mcpClient.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: mcpURL,
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to MCP tool server at %s: %w", mcpURL, err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "k8s_apply_manifest",
		Arguments: map[string]any{
			"manifest": string(yamlBytes),
		},
	})
	if err != nil {
		return fmt.Errorf("MCP tool call failed: %w", err)
	}
	if result.IsError {
		return fmt.Errorf("k8s_apply_manifest error: %v", result.Content)
	}

	r.Logger.Debug().
		Str("kind", obj.GetObjectKind().GroupVersionKind().Kind).
		Str("name", obj.GetName()).
		Str("namespace", obj.GetNamespace()).
		Str("mcpURL", mcpURL).
		Msg("applied resource via MCP tool server")

	return nil
}

// deleteViaMCP deletes a Kubernetes resource via the MCP tool server's k8s_delete_resource tool.
func (r *RegistryDeploymentReconciler) deleteViaMCP(ctx context.Context, mcpURL string, res agentregistryv1alpha1.ManagedResource) error {
	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "agentregistry",
		Version: "1.0.0",
	}, nil)

	session, err := mcpClient.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: mcpURL,
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to MCP tool server at %s: %w", mcpURL, err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "k8s_delete_resource",
		Arguments: map[string]any{
			"apiVersion": res.APIVersion,
			"kind":       res.Kind,
			"name":       res.Name,
			"namespace":  res.Namespace,
		},
	})
	if err != nil {
		return fmt.Errorf("MCP tool call failed: %w", err)
	}
	if result.IsError {
		return fmt.Errorf("k8s_delete_resource error: %v", result.Content)
	}

	r.Logger.Debug().
		Str("kind", res.Kind).
		Str("name", res.Name).
		Str("namespace", res.Namespace).
		Str("mcpURL", mcpURL).
		Msg("deleted resource via MCP tool server")

	return nil
}

// applyResource applies a Kubernetes resource using server-side apply
func (r *RegistryDeploymentReconciler) applyResource(ctx context.Context, targetClient client.Client, obj client.Object) error {
	return targetClient.Patch(ctx, obj, client.Apply, client.FieldOwner("agentregistry"), client.ForceOwnership)
}

// deleteResource deletes a managed resource using the provided client
func (r *RegistryDeploymentReconciler) deleteResource(ctx context.Context, targetClient client.Client, res agentregistryv1alpha1.ManagedResource) error {
	var obj client.Object
	switch res.Kind {
	case "Agent":
		obj = &kagentv1alpha2.Agent{}
	case "RemoteMCPServer":
		obj = &kagentv1alpha2.RemoteMCPServer{}
	case "MCPServer":
		obj = &kmcpv1alpha1.MCPServer{}
	case "ConfigMap":
		obj = &corev1.ConfigMap{}
	default:
		return fmt.Errorf("unknown resource kind: %s", res.Kind)
	}

	obj.SetName(res.Name)
	obj.SetNamespace(res.Namespace)

	if err := targetClient.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

// setOwnerLabels sets labels to track ownership
func (r *RegistryDeploymentReconciler) setOwnerLabels(obj client.Object, deployment *agentregistryv1alpha1.RegistryDeployment) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[managedByLabel] = "agentregistry"
	labels[deploymentNameLabel] = deployment.Name
	labels[deploymentNSLabel] = deployment.Namespace
	obj.SetLabels(labels)
}

// SetupWithManager sets up the controller with the Manager.
func (r *RegistryDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Map function to enqueue RegistryDeployment from managed resources
	enqueueFromManagedResource := func(ctx context.Context, obj client.Object) []reconcile.Request {
		labels := obj.GetLabels()
		if labels == nil {
			return nil
		}
		// Check if this resource is managed by agentregistry
		if labels[managedByLabel] != "agentregistry" {
			return nil
		}
		// Get the deployment name and namespace from labels
		depName := labels[deploymentNameLabel]
		depNS := labels[deploymentNSLabel]
		if depName == "" {
			return nil
		}
		return []reconcile.Request{{
			NamespacedName: types.NamespacedName{
				Name:      depName,
				Namespace: depNS,
			},
		}}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&agentregistryv1alpha1.RegistryDeployment{}).
		// Watch Agents managed by this controller
		Watches(
			&kagentv1alpha2.Agent{},
			handler.EnqueueRequestsFromMapFunc(enqueueFromManagedResource),
		).
		// Watch MCP Servers managed by this controller
		Watches(
			&kmcpv1alpha1.MCPServer{},
			handler.EnqueueRequestsFromMapFunc(enqueueFromManagedResource),
		).
		// Watch RemoteMCP Servers managed by this controller
		Watches(
			&kagentv1alpha2.RemoteMCPServer{},
			handler.EnqueueRequestsFromMapFunc(enqueueFromManagedResource),
		).
		// Watch ConfigMaps managed by this controller
		Watches(
			&corev1.ConfigMap{},
			handler.EnqueueRequestsFromMapFunc(enqueueFromManagedResource),
		).
		Complete(r)
}

// Helper functions

func generateInternalName(name string) string {
	return kagent.MCPServerResourceName(name)
}

func parseURLComponents(rawURL string) (host string, port uint32, path string) {
	if rawURL == "" {
		return "", 0, "/"
	}

	// Ensure URL has a scheme so net/url.Parse works correctly
	urlStr := rawURL
	if !strings.Contains(urlStr, "://") {
		urlStr = "http://" + urlStr
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return "", 0, "/"
	}

	host = u.Hostname()
	path = u.Path
	if path == "" {
		path = "/"
	}

	if portStr := u.Port(); portStr != "" {
		p, err := strconv.ParseUint(portStr, 10, 32)
		if err == nil {
			port = uint32(p)
		}
	} else {
		// Default ports by scheme
		switch u.Scheme {
		case "https":
			port = 443
		default:
			port = 80
		}
	}

	return host, port, path
}

func getImageAndCommand(registryType, runtimeHint string) (image, cmd string) {
	switch registryType {
	case "npm":
		if runtimeHint == "npx" {
			return "node:20-alpine", "npx"
		}
		return "node:20-alpine", "npm"
	case "pypi":
		if runtimeHint == "uvx" {
			return "ghcr.io/astral-sh/uv:latest", "uvx"
		}
		return "python:3.12-slim", "pip"
	case "oci":
		return "", "" // Image is specified in the identifier
	default:
		return "", ""
	}
}

// validatePublisherIdentity checks that the catalog entry has both verified organization
// and verified publisher identity. Deployments are blocked if either validation is missing.
func validatePublisherIdentity(metadata *apiextensionsv1.JSON) error {
	if metadata == nil || len(metadata.Raw) == 0 {
		return fmt.Errorf("missing publisher metadata: both org_is_verified and publisher_identity_verified_by_jwt are required")
	}

	var meta map[string]json.RawMessage
	if err := json.Unmarshal(metadata.Raw, &meta); err != nil {
		return fmt.Errorf("invalid metadata: %w", err)
	}

	publisherRaw, ok := meta["io.modelcontextprotocol.registry/publisher-provided"]
	if !ok {
		return fmt.Errorf("missing publisher-provided metadata: both org_is_verified and publisher_identity_verified_by_jwt are required")
	}

	var publisherProvided map[string]json.RawMessage
	if err := json.Unmarshal(publisherRaw, &publisherProvided); err != nil {
		return fmt.Errorf("invalid publisher-provided metadata: %w", err)
	}

	aregistryRaw, ok := publisherProvided["aregistry.ai/metadata"]
	if !ok {
		return fmt.Errorf("missing aregistry.ai/metadata: both org_is_verified and publisher_identity_verified_by_jwt are required")
	}

	var aregistryMeta struct {
		Identity struct {
			OrgIsVerified                  bool `json:"org_is_verified"`
			PublisherIdentityVerifiedByJWT bool `json:"publisher_identity_verified_by_jwt"`
		} `json:"identity"`
	}
	if err := json.Unmarshal(aregistryRaw, &aregistryMeta); err != nil {
		return fmt.Errorf("invalid aregistry.ai/metadata: %w", err)
	}

	if !aregistryMeta.Identity.OrgIsVerified && !aregistryMeta.Identity.PublisherIdentityVerifiedByJWT {
		return fmt.Errorf("organization is not verified and publisher identity is not verified")
	}
	if !aregistryMeta.Identity.OrgIsVerified {
		return fmt.Errorf("organization is not verified (org_is_verified=false)")
	}
	if !aregistryMeta.Identity.PublisherIdentityVerifiedByJWT {
		return fmt.Errorf("publisher identity is not verified (publisher_identity_verified_by_jwt=false)")
	}

	return nil
}

// checkManagedResourcesReady checks managed resources status from their conditions
func (r *RegistryDeploymentReconciler) checkManagedResourcesReady(ctx context.Context, deployment *agentregistryv1alpha1.RegistryDeployment) (bool, string) {
	// If no managed resources yet, pending
	if len(deployment.Status.ManagedResources) == 0 {
		return false, "Pending"
	}

	// Resolve the target client and environment for status checks
	env, targetClient, _, err := r.getTargetClientAndEnv(ctx, deployment)
	if err != nil {
		return false, fmt.Sprintf("Failed to resolve target: %v", err)
	}

	// When MCP tool server is used, we can't query resource status directly.
	// If apply succeeded, consider resources ready.
	if env != nil && env.MCPToolServerURL != "" {
		return true, ""
	}

	// Check each managed resource status
	for _, res := range deployment.Status.ManagedResources {
		switch res.Kind {
		case "MCPServer":
			var mcp kmcpv1alpha1.MCPServer
			key := client.ObjectKey{Namespace: res.Namespace, Name: res.Name}
			if err := targetClient.Get(ctx, key, &mcp); err != nil {
				if apierrors.IsNotFound(err) {
					return false, fmt.Sprintf("Managed %s %s/%s not found - will recreate", res.Kind, res.Namespace, res.Name)
				}
				return false, fmt.Sprintf("Error checking %s %s/%s: %v", res.Kind, res.Namespace, res.Name, err)
			}

			// Check Ready condition - if exists and True, running
			ready := false
			for _, cond := range mcp.Status.Conditions {
				if cond.Type == "Ready" {
					if cond.Status == metav1.ConditionTrue {
						ready = true
						break // This resource is ready, check next resource
					}
					return false, cond.Message // Not ready, use condition message
				}
			}
			// If no Ready condition found or Ready=False
			if !ready {
				return false, "Pending"
			}

		case "RemoteMCPServer":
			var remoteMCP kagentv1alpha2.RemoteMCPServer
			key := client.ObjectKey{Namespace: res.Namespace, Name: res.Name}
			if err := targetClient.Get(ctx, key, &remoteMCP); err != nil {
				if apierrors.IsNotFound(err) {
					return false, fmt.Sprintf("Managed %s %s/%s not found - will recreate", res.Kind, res.Namespace, res.Name)
				}
				return false, fmt.Sprintf("Error checking %s %s/%s: %v", res.Kind, res.Namespace, res.Name, err)
			}

			// Check Ready condition - if exists and True, running
			ready := false
			for _, cond := range remoteMCP.Status.Conditions {
				if cond.Type == "Ready" {
					if cond.Status == metav1.ConditionTrue {
						ready = true
						break // This resource is ready, check next resource
					}
					return false, cond.Message // Not ready, use condition message
				}
			}
			// If no Ready condition found or Ready=False
			if !ready {
				return false, "Pending"
			}

		case "Agent":
			var agent kagentv1alpha2.Agent
			key := client.ObjectKey{Namespace: res.Namespace, Name: res.Name}
			if err := targetClient.Get(ctx, key, &agent); err != nil {
				if apierrors.IsNotFound(err) {
					return false, fmt.Sprintf("Managed %s %s/%s not found - will recreate", res.Kind, res.Namespace, res.Name)
				}
				return false, fmt.Sprintf("Error checking %s %s/%s: %v", res.Kind, res.Namespace, res.Name, err)
			}

			// Check Ready condition - if exists and True, running
			ready := false
			for _, cond := range agent.Status.Conditions {
				if cond.Type == "Ready" {
					if cond.Status == metav1.ConditionTrue {
						ready = true
						break // This resource is ready, check next resource
					}
					return false, cond.Message // Not ready, use condition message
				}
			}
			// If no Ready condition found or Ready=False
			if !ready {
				return false, "Pending"
			}

		case "ConfigMap":
			var cm corev1.ConfigMap
			key := client.ObjectKey{Namespace: res.Namespace, Name: res.Name}
			if err := targetClient.Get(ctx, key, &cm); err != nil {
				if apierrors.IsNotFound(err) {
					return false, fmt.Sprintf("Managed %s %s/%s not found - will recreate", res.Kind, res.Namespace, res.Name)
				}
				return false, fmt.Sprintf("Error checking %s %s/%s: %v", res.Kind, res.Namespace, res.Name, err)
			}
			// ConfigMaps don't have conditions, just existence check
			continue
		}
	}

	// All resources have Ready=True
	return true, ""
}
