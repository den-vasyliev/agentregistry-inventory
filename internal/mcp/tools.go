package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	"github.com/agentregistry-dev/agentregistry/internal/config"
	"github.com/agentregistry-dev/agentregistry/internal/controller"
	"github.com/agentregistry-dev/agentregistry/internal/httpapi/handlers"
)

func (s *MCPServer) registerTools() {
	// Catalog read tools
	s.mcpServer.AddTool(mcp.NewTool("list_catalog",
		mcp.WithDescription("List catalog entries by type (servers, agents, skills, or models)"),
		mcp.WithString("type", mcp.Description("Resource type: servers, agents, skills, or models"), mcp.Required()),
		mcp.WithString("search", mcp.Description("Filter by name substring")),
		mcp.WithString("version", mcp.Description("Filter by version or 'latest' (servers/agents/skills)")),
		mcp.WithString("category", mcp.Description("Filter by category (skills only)")),
		mcp.WithString("provider", mcp.Description("Filter by provider (models only)")),
		mcp.WithNumber("limit", mcp.Description("Max results (default 30)")),
	), s.handleListCatalog)

	s.mcpServer.AddTool(mcp.NewTool("get_catalog",
		mcp.WithDescription("Get catalog entry details by type and name"),
		mcp.WithString("type", mcp.Description("Resource type: servers, agents, skills, or models"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Resource name"), mcp.Required()),
		mcp.WithString("version", mcp.Description("Specific version (default: latest)")),
	), s.handleGetCatalog)

	s.mcpServer.AddTool(mcp.NewTool("get_registry_stats",
		mcp.WithDescription("Get registry statistics (counts of servers, agents, skills, models)"),
	), s.handleGetRegistryStats)

	// Deployment tools
	s.mcpServer.AddTool(mcp.NewTool("list_deployments",
		mcp.WithDescription("List deployments"),
		mcp.WithString("resourceType", mcp.Description("Filter by resource type (mcp, agent)")),
		mcp.WithNumber("limit", mcp.Description("Max results (default 30)")),
	), s.handleListDeployments)

	s.mcpServer.AddTool(mcp.NewTool("get_deployment",
		mcp.WithDescription("Get deployment details by name"),
		mcp.WithString("name", mcp.Description("Deployment name"), mcp.Required()),
	), s.handleGetDeployment)

	s.mcpServer.AddTool(mcp.NewTool("deploy_catalog_item",
		mcp.WithDescription("Deploy a catalog item to Kubernetes"),
		mcp.WithString("resourceName", mcp.Description("Name of the catalog resource to deploy"), mcp.Required()),
		mcp.WithString("version", mcp.Description("Version to deploy"), mcp.Required()),
		mcp.WithString("resourceType", mcp.Description("Resource type: mcp or agent"), mcp.Required()),
		mcp.WithString("namespace", mcp.Description("Target namespace (default: agentregistry)")),
		mcp.WithObject("config", mcp.Description("Key-value deployment configuration")),
	), s.handleDeployCatalogItem)

	s.mcpServer.AddTool(mcp.NewTool("delete_deployment",
		mcp.WithDescription("Delete a deployment"),
		mcp.WithString("name", mcp.Description("Deployment name"), mcp.Required()),
	), s.handleDeleteDeployment)

	s.mcpServer.AddTool(mcp.NewTool("update_deployment_config",
		mcp.WithDescription("Update deployment configuration"),
		mcp.WithString("name", mcp.Description("Deployment name"), mcp.Required()),
		mcp.WithObject("config", mcp.Description("Key-value configuration to merge"), mcp.Required()),
	), s.handleUpdateDeploymentConfig)

	// Discovery tools
	s.mcpServer.AddTool(mcp.NewTool("list_environments",
		mcp.WithDescription("List discovered environments from DiscoveryConfig"),
	), s.handleListEnvironments)

	s.mcpServer.AddTool(mcp.NewTool("get_discovery_map",
		mcp.WithDescription("Get the discovery topology map showing clusters, environments, and resource counts"),
	), s.handleGetDiscoveryMap)

	s.mcpServer.AddTool(mcp.NewTool("trigger_discovery",
		mcp.WithDescription("Force re-scan of discovery by adding a trigger annotation to DiscoveryConfig"),
		mcp.WithString("configName", mcp.Description("DiscoveryConfig name (default: discovers all)")),
	), s.handleTriggerDiscovery)

	// Sampling-powered tools
	s.mcpServer.AddTool(mcp.NewTool("recommend_servers",
		mcp.WithDescription("Recommend MCP servers for a use case (uses LLM sampling to analyze the catalog)"),
		mcp.WithString("description", mcp.Description("Describe what you need the MCP server(s) for"), mcp.Required()),
	), s.handleRecommendServers)

	s.mcpServer.AddTool(mcp.NewTool("analyze_agent_dependencies",
		mcp.WithDescription("Analyze an agent's dependency tree including MCP servers, models, and skills (uses LLM sampling)"),
		mcp.WithString("name", mcp.Description("Agent name to analyze"), mcp.Required()),
	), s.handleAnalyzeAgentDependencies)

	s.mcpServer.AddTool(mcp.NewTool("generate_deployment_plan",
		mcp.WithDescription("Generate a deployment plan for catalog resources (uses LLM sampling)"),
		mcp.WithString("resources", mcp.Description("Comma-separated list of resource names to deploy"), mcp.Required()),
		mcp.WithString("namespace", mcp.Description("Target namespace")),
	), s.handleGenerateDeploymentPlan)

	// Catalog management tools (auth-gated)
	s.mcpServer.AddTool(mcp.NewTool("create_catalog",
		mcp.WithDescription("Create a new catalog entry (requires auth disabled or dev mode)"),
		mcp.WithString("type", mcp.Description("Resource type: servers, agents, skills, or models"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Canonical name for the resource"), mcp.Required()),
		mcp.WithString("version", mcp.Description("Semantic version (e.g., 1.0.0)"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Human-readable title")),
		mcp.WithString("description", mcp.Description("Resource description")),
		mcp.WithString("category", mcp.Description("Category (skills only)")),
		mcp.WithString("provider", mcp.Description("Model provider (models only, e.g., OpenAI, Anthropic)")),
		mcp.WithString("model", mcp.Description("Model identifier (models only, e.g., gpt-4)")),
	), s.handleCreateCatalog)

	s.mcpServer.AddTool(mcp.NewTool("delete_catalog",
		mcp.WithDescription("Delete a catalog entry (requires auth disabled or dev mode)"),
		mcp.WithString("type", mcp.Description("Resource type: servers, agents, skills, or models"), mcp.Required()),
		mcp.WithString("name", mcp.Description("Resource name to delete"), mcp.Required()),
	), s.handleDeleteCatalog)
}

// --- Helper functions ---

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: text},
		},
	}
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: msg},
		},
		IsError: true,
	}
}

func jsonResult(v any) *mcp.CallToolResult {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to marshal result: %v", err))
	}
	return textResult(string(data))
}

func getStringArg(args map[string]interface{}, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getIntArg(args map[string]interface{}, key string, defaultVal int) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return defaultVal
}

// --- Catalog Read Handlers ---

func (s *MCPServer) handleListCatalog(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	catalogType := getStringArg(args, "type")
	search := getStringArg(args, "search")
	version := getStringArg(args, "version")
	category := getStringArg(args, "category")
	provider := getStringArg(args, "provider")
	limit := getIntArg(args, "limit", 30)

	switch catalogType {
	case "servers":
		var list agentregistryv1alpha1.MCPServerCatalogList
		listOpts := []client.ListOption{}
		if version == "latest" {
			listOpts = append(listOpts, client.MatchingFields{
				controller.IndexMCPServerIsLatest: "true",
			})
		}
		if err := s.cache.List(ctx, &list, listOpts...); err != nil {
			return errorResult(fmt.Sprintf("Failed to list servers: %v", err)), nil
		}
		type serverSummary struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Title       string `json:"title,omitempty"`
			Description string `json:"description,omitempty"`
			Status      string `json:"status,omitempty"`
		}
		results := make([]serverSummary, 0)
		for _, item := range list.Items {
			if search != "" && !strings.Contains(strings.ToLower(item.Spec.Name), strings.ToLower(search)) {
				continue
			}
			if version != "" && version != "latest" && item.Spec.Version != version {
				continue
			}
			results = append(results, serverSummary{
				Name:        item.Spec.Name,
				Version:     item.Spec.Version,
				Title:       item.Spec.Title,
				Description: item.Spec.Description,
				Status:      string(item.Status.Status),
			})
			if len(results) >= limit {
				break
			}
		}
		return jsonResult(results), nil

	case "agents":
		var list agentregistryv1alpha1.AgentCatalogList
		listOpts := []client.ListOption{}
		if version == "latest" {
			listOpts = append(listOpts, client.MatchingFields{
				controller.IndexAgentIsLatest: "true",
			})
		}
		if err := s.cache.List(ctx, &list, listOpts...); err != nil {
			return errorResult(fmt.Sprintf("Failed to list agents: %v", err)), nil
		}
		type agentSummary struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Title       string `json:"title,omitempty"`
			Description string `json:"description,omitempty"`
			Framework   string `json:"framework,omitempty"`
			AgentType   string `json:"agentType,omitempty"`
		}
		results := make([]agentSummary, 0)
		for _, item := range list.Items {
			if search != "" && !strings.Contains(strings.ToLower(item.Spec.Name), strings.ToLower(search)) {
				continue
			}
			if version != "" && version != "latest" && item.Spec.Version != version {
				continue
			}
			results = append(results, agentSummary{
				Name:        item.Spec.Name,
				Version:     item.Spec.Version,
				Title:       item.Spec.Title,
				Description: item.Spec.Description,
				Framework:   item.Spec.Framework,
				AgentType:   item.Spec.AgentType,
			})
			if len(results) >= limit {
				break
			}
		}
		return jsonResult(results), nil

	case "skills":
		var list agentregistryv1alpha1.SkillCatalogList
		listOpts := []client.ListOption{}
		if err := s.cache.List(ctx, &list, listOpts...); err != nil {
			return errorResult(fmt.Sprintf("Failed to list skills: %v", err)), nil
		}
		type skillSummary struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Title       string `json:"title,omitempty"`
			Category    string `json:"category,omitempty"`
			Description string `json:"description,omitempty"`
		}
		results := make([]skillSummary, 0)
		for _, item := range list.Items {
			if search != "" && !strings.Contains(strings.ToLower(item.Spec.Name), strings.ToLower(search)) {
				continue
			}
			if category != "" && item.Spec.Category != category {
				continue
			}
			results = append(results, skillSummary{
				Name:        item.Spec.Name,
				Version:     item.Spec.Version,
				Title:       item.Spec.Title,
				Category:    item.Spec.Category,
				Description: item.Spec.Description,
			})
			if len(results) >= limit {
				break
			}
		}
		return jsonResult(results), nil

	case "models":
		var list agentregistryv1alpha1.ModelCatalogList
		if err := s.cache.List(ctx, &list); err != nil {
			return errorResult(fmt.Sprintf("Failed to list models: %v", err)), nil
		}
		type modelSummary struct {
			Name        string `json:"name"`
			Provider    string `json:"provider"`
			Model       string `json:"model"`
			Description string `json:"description,omitempty"`
		}
		results := make([]modelSummary, 0)
		for _, item := range list.Items {
			if search != "" && !strings.Contains(strings.ToLower(item.Spec.Name), strings.ToLower(search)) {
				continue
			}
			if provider != "" && !strings.EqualFold(item.Spec.Provider, provider) {
				continue
			}
			results = append(results, modelSummary{
				Name:        item.Spec.Name,
				Provider:    item.Spec.Provider,
				Model:       item.Spec.Model,
				Description: item.Spec.Description,
			})
			if len(results) >= limit {
				break
			}
		}
		return jsonResult(results), nil

	default:
		return errorResult("Invalid type: must be servers, agents, skills, or models"), nil
	}
}

func (s *MCPServer) handleGetCatalog(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	catalogType := getStringArg(args, "type")
	name := getStringArg(args, "name")
	version := getStringArg(args, "version")

	switch catalogType {
	case "servers":
		var list agentregistryv1alpha1.MCPServerCatalogList
		fields := client.MatchingFields{controller.IndexMCPServerName: name}
		if version == "" {
			fields[controller.IndexMCPServerIsLatest] = "true"
		}
		if err := s.cache.List(ctx, &list, fields); err != nil {
			return errorResult(fmt.Sprintf("Failed to get server: %v", err)), nil
		}
		for _, item := range list.Items {
			if version != "" && item.Spec.Version != version {
				continue
			}
			return jsonResult(item.Spec), nil
		}
		return errorResult(fmt.Sprintf("Server '%s' not found", name)), nil

	case "agents":
		var list agentregistryv1alpha1.AgentCatalogList
		fields := client.MatchingFields{controller.IndexAgentName: name}
		if version == "" {
			fields[controller.IndexAgentIsLatest] = "true"
		}
		if err := s.cache.List(ctx, &list, fields); err != nil {
			return errorResult(fmt.Sprintf("Failed to get agent: %v", err)), nil
		}
		for _, item := range list.Items {
			if version != "" && item.Spec.Version != version {
				continue
			}
			return jsonResult(item.Spec), nil
		}
		return errorResult(fmt.Sprintf("Agent '%s' not found", name)), nil

	case "skills":
		var list agentregistryv1alpha1.SkillCatalogList
		fields := client.MatchingFields{controller.IndexSkillName: name}
		if version == "" {
			fields[controller.IndexSkillIsLatest] = "true"
		}
		if err := s.cache.List(ctx, &list, fields); err != nil {
			return errorResult(fmt.Sprintf("Failed to get skill: %v", err)), nil
		}
		for _, item := range list.Items {
			if version != "" && item.Spec.Version != version {
				continue
			}
			return jsonResult(item.Spec), nil
		}
		return errorResult(fmt.Sprintf("Skill '%s' not found", name)), nil

	case "models":
		var list agentregistryv1alpha1.ModelCatalogList
		if err := s.cache.List(ctx, &list, client.MatchingFields{
			controller.IndexModelName: name,
		}); err != nil {
			return errorResult(fmt.Sprintf("Failed to get model: %v", err)), nil
		}
		if len(list.Items) == 0 {
			return errorResult(fmt.Sprintf("Model '%s' not found", name)), nil
		}
		return jsonResult(list.Items[0].Spec), nil

	default:
		return errorResult("Invalid type: must be servers, agents, skills, or models"), nil
	}
}

func (s *MCPServer) handleGetRegistryStats(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stats, err := s.getStats(ctx)
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to get stats: %v", err)), nil
	}
	return jsonResult(stats), nil
}

// --- Deployment Handlers ---

func (s *MCPServer) handleListDeployments(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	resourceType := getStringArg(args, "resourceType")
	limit := getIntArg(args, "limit", 30)

	var list agentregistryv1alpha1.RegistryDeploymentList
	listOpts := []client.ListOption{}
	if resourceType != "" {
		listOpts = append(listOpts, client.MatchingFields{
			controller.IndexDeploymentResourceType: resourceType,
		})
	}
	if err := s.cache.List(ctx, &list, listOpts...); err != nil {
		return errorResult(fmt.Sprintf("Failed to list deployments: %v", err)), nil
	}

	type deploySummary struct {
		Name         string `json:"name"`
		ResourceName string `json:"resourceName"`
		Version      string `json:"version"`
		ResourceType string `json:"resourceType"`
		Namespace    string `json:"namespace"`
		Phase        string `json:"phase"`
		Message      string `json:"message,omitempty"`
	}

	results := make([]deploySummary, 0)
	for _, item := range list.Items {
		results = append(results, deploySummary{
			Name:         item.Name,
			ResourceName: item.Spec.ResourceName,
			Version:      item.Spec.Version,
			ResourceType: string(item.Spec.ResourceType),
			Namespace:    item.Spec.Namespace,
			Phase:        string(item.Status.Phase),
			Message:      item.Status.Message,
		})
		if len(results) >= limit {
			break
		}
	}

	return jsonResult(results), nil
}

func (s *MCPServer) handleGetDeployment(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := getStringArg(request.GetArguments(), "name")

	var deployment agentregistryv1alpha1.RegistryDeployment
	if err := s.cache.Get(ctx, client.ObjectKey{Namespace: "agentregistry", Name: name}, &deployment); err != nil {
		return errorResult(fmt.Sprintf("Deployment '%s' not found", name)), nil
	}

	type deployDetail struct {
		Name             string            `json:"name"`
		ResourceName     string            `json:"resourceName"`
		Version          string            `json:"version"`
		ResourceType     string            `json:"resourceType"`
		Namespace        string            `json:"namespace"`
		Config           map[string]string `json:"config,omitempty"`
		Phase            string            `json:"phase"`
		Message          string            `json:"message,omitempty"`
		ManagedResources []string          `json:"managedResources,omitempty"`
	}

	managed := make([]string, 0)
	for _, r := range deployment.Status.ManagedResources {
		managed = append(managed, fmt.Sprintf("%s/%s (%s)", r.Namespace, r.Name, r.Kind))
	}

	return jsonResult(deployDetail{
		Name:             deployment.Name,
		ResourceName:     deployment.Spec.ResourceName,
		Version:          deployment.Spec.Version,
		ResourceType:     string(deployment.Spec.ResourceType),
		Namespace:        deployment.Spec.Namespace,
		Config:           deployment.Spec.Config,
		Phase:            string(deployment.Status.Phase),
		Message:          deployment.Status.Message,
		ManagedResources: managed,
	}), nil
}

func (s *MCPServer) handleDeployCatalogItem(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireAdmin(); err != nil {
		return err, nil
	}

	args := request.GetArguments()
	resourceName := getStringArg(args, "resourceName")
	version := getStringArg(args, "version")
	resourceType := getStringArg(args, "resourceType")
	namespace := getStringArg(args, "namespace")

	if namespace == "" {
		namespace = "agentregistry"
	}

	if resourceType != "mcp" && resourceType != "agent" {
		return errorResult("resourceType must be 'mcp' or 'agent'"), nil
	}

	crName := sanitizeName(resourceName) + "-" + sanitizeName(version)

	// Extract config if provided
	var config map[string]string
	if cfgRaw, ok := args["config"]; ok && cfgRaw != nil {
		if cfgMap, ok := cfgRaw.(map[string]interface{}); ok {
			config = make(map[string]string)
			for k, v := range cfgMap {
				config[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	deployment := &agentregistryv1alpha1.RegistryDeployment{}
	deployment.Name = crName
	deployment.Namespace = "agentregistry"
	deployment.Labels = map[string]string{
		"agentregistry.dev/resource-name": sanitizeName(resourceName),
		"agentregistry.dev/version":       sanitizeName(version),
		"agentregistry.dev/resource-type": resourceType,
		"agentregistry.dev/runtime":       "kubernetes",
	}
	deployment.Spec = agentregistryv1alpha1.RegistryDeploymentSpec{
		ResourceName: resourceName,
		Version:      version,
		ResourceType: agentregistryv1alpha1.ResourceType(resourceType),
		Runtime:      agentregistryv1alpha1.RuntimeTypeKubernetes,
		Config:       config,
		Namespace:    namespace,
	}

	if err := s.client.Create(ctx, deployment); err != nil {
		return errorResult(fmt.Sprintf("Failed to create deployment: %v", err)), nil
	}

	return textResult(fmt.Sprintf("Deployment '%s' created for %s %s/%s in namespace %s", crName, resourceType, resourceName, version, namespace)), nil
}

func (s *MCPServer) handleDeleteDeployment(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireAdmin(); err != nil {
		return err, nil
	}

	name := getStringArg(request.GetArguments(), "name")

	deployment := &agentregistryv1alpha1.RegistryDeployment{}
	deployment.Name = name
	deployment.Namespace = "agentregistry"

	if err := s.client.Delete(ctx, deployment); err != nil {
		return errorResult(fmt.Sprintf("Failed to delete deployment: %v", err)), nil
	}

	return textResult(fmt.Sprintf("Deployment '%s' deleted", name)), nil
}

func (s *MCPServer) handleUpdateDeploymentConfig(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireAdmin(); err != nil {
		return err, nil
	}

	args := request.GetArguments()
	name := getStringArg(args, "name")

	var deployment agentregistryv1alpha1.RegistryDeployment
	if err := s.client.Get(ctx, client.ObjectKey{Namespace: "agentregistry", Name: name}, &deployment); err != nil {
		return errorResult(fmt.Sprintf("Deployment '%s' not found", name)), nil
	}

	if cfgRaw, ok := args["config"]; ok && cfgRaw != nil {
		if cfgMap, ok := cfgRaw.(map[string]interface{}); ok {
			if deployment.Spec.Config == nil {
				deployment.Spec.Config = make(map[string]string)
			}
			for k, v := range cfgMap {
				deployment.Spec.Config[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	if err := s.client.Update(ctx, &deployment); err != nil {
		return errorResult(fmt.Sprintf("Failed to update deployment: %v", err)), nil
	}

	return textResult(fmt.Sprintf("Deployment '%s' config updated", name)), nil
}

// --- Discovery Handlers ---

func (s *MCPServer) handleListEnvironments(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var list agentregistryv1alpha1.DiscoveryConfigList
	if err := s.client.List(ctx, &list, client.InNamespace("agentregistry")); err != nil {
		return errorResult(fmt.Sprintf("Failed to list DiscoveryConfigs: %v", err)), nil
	}

	type envSummary struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}

	results := make([]envSummary, 0)
	for _, dc := range list.Items {
		for _, env := range dc.Spec.Environments {
			ns := env.Cluster.Namespace
			if ns == "" && len(env.Namespaces) > 0 {
				ns = env.Namespaces[0]
			}
			results = append(results, envSummary{
				Name:      env.Name,
				Namespace: ns,
			})
		}
	}

	return jsonResult(results), nil
}

func (s *MCPServer) handleGetDiscoveryMap(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var list agentregistryv1alpha1.DiscoveryConfigList
	if err := s.client.List(ctx, &list, client.InNamespace("agentregistry")); err != nil {
		return errorResult(fmt.Sprintf("Failed to list DiscoveryConfigs: %v", err)), nil
	}

	type resourceCounts struct {
		MCPServers int `json:"mcpServers"`
		Agents     int `json:"agents"`
		Skills     int `json:"skills"`
		Models     int `json:"models"`
	}
	type envDetail struct {
		Name             string         `json:"name"`
		ClusterName      string         `json:"clusterName"`
		Namespaces       []string       `json:"namespaces,omitempty"`
		Connected        bool           `json:"connected"`
		DiscoveryEnabled bool           `json:"discoveryEnabled"`
		DiscoveredCounts resourceCounts `json:"discoveredResources"`
		Error            string         `json:"error,omitempty"`
	}
	type configDetail struct {
		Name         string      `json:"name"`
		Environments []envDetail `json:"environments"`
	}

	configs := make([]configDetail, 0)
	for _, dc := range list.Items {
		statusByEnv := make(map[string]agentregistryv1alpha1.EnvironmentStatus)
		for _, es := range dc.Status.Environments {
			statusByEnv[es.Name] = es
		}

		envs := make([]envDetail, 0)
		for _, env := range dc.Spec.Environments {
			ed := envDetail{
				Name:             env.Name,
				ClusterName:      env.Cluster.Name,
				Namespaces:       env.Namespaces,
				DiscoveryEnabled: env.DiscoveryEnabled,
			}
			if es, ok := statusByEnv[env.Name]; ok {
				ed.Connected = es.Connected
				ed.Error = es.Error
				ed.DiscoveredCounts = resourceCounts{
					MCPServers: es.DiscoveredResources.MCPServers,
					Agents:     es.DiscoveredResources.Agents,
					Skills:     es.DiscoveredResources.Skills,
					Models:     es.DiscoveredResources.Models,
				}
			}
			envs = append(envs, ed)
		}
		configs = append(configs, configDetail{
			Name:         dc.Name,
			Environments: envs,
		})
	}

	return jsonResult(configs), nil
}

func (s *MCPServer) handleTriggerDiscovery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireAdmin(); err != nil {
		return err, nil
	}

	configName := getStringArg(request.GetArguments(), "configName")

	var list agentregistryv1alpha1.DiscoveryConfigList
	if err := s.client.List(ctx, &list, client.InNamespace("agentregistry")); err != nil {
		return errorResult(fmt.Sprintf("Failed to list DiscoveryConfigs: %v", err)), nil
	}

	triggered := 0
	for _, dc := range list.Items {
		if configName != "" && dc.Name != configName {
			continue
		}
		if dc.Annotations == nil {
			dc.Annotations = make(map[string]string)
		}
		dc.Annotations["agentregistry.dev/trigger-discovery"] = "true"
		if err := s.client.Update(ctx, &dc); err != nil {
			return errorResult(fmt.Sprintf("Failed to trigger discovery on %s: %v", dc.Name, err)), nil
		}
		triggered++
	}

	if triggered == 0 {
		return errorResult("No DiscoveryConfig found to trigger"), nil
	}

	return textResult(fmt.Sprintf("Triggered discovery on %d config(s)", triggered)), nil
}

// --- Sampling-Powered Handlers ---

func (s *MCPServer) handleRecommendServers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	description := getStringArg(request.GetArguments(), "description")

	var list agentregistryv1alpha1.MCPServerCatalogList
	if err := s.cache.List(ctx, &list, client.MatchingFields{
		controller.IndexMCPServerIsLatest: "true",
	}); err != nil {
		return errorResult(fmt.Sprintf("Failed to list servers: %v", err)), nil
	}

	type serverInfo struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description,omitempty"`
		Title       string `json:"title,omitempty"`
	}

	servers := make([]serverInfo, 0, len(list.Items))
	for _, item := range list.Items {
		servers = append(servers, serverInfo{
			Name:        item.Spec.Name,
			Version:     item.Spec.Version,
			Description: item.Spec.Description,
			Title:       item.Spec.Title,
		})
	}

	catalogJSON, _ := json.MarshalIndent(servers, "", "  ")
	userMsg := fmt.Sprintf("User needs: %s\n\nAvailable MCP servers in the registry:\n%s\n\nRecommend the best matching servers and explain why each is relevant. If none match, say so.", description, string(catalogJSON))

	result, err := s.requestSampling(ctx, "You are an Agent Registry advisor. Analyze the MCP server catalog and recommend the best matches for the user's needs. Be concise and specific.", userMsg)
	if err != nil {
		// Graceful degradation: return raw data
		return textResult(fmt.Sprintf("Sampling unavailable (client may not support it). Here are all %d servers:\n%s", len(servers), string(catalogJSON))), nil
	}

	return textResult(result), nil
}

func (s *MCPServer) handleAnalyzeAgentDependencies(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := getStringArg(request.GetArguments(), "name")

	// Fetch the agent
	var agentList agentregistryv1alpha1.AgentCatalogList
	if err := s.cache.List(ctx, &agentList, client.MatchingFields{
		controller.IndexAgentName:     name,
		controller.IndexAgentIsLatest: "true",
	}); err != nil {
		return errorResult(fmt.Sprintf("Failed to get agent: %v", err)), nil
	}
	if len(agentList.Items) == 0 {
		return errorResult(fmt.Sprintf("Agent '%s' not found", name)), nil
	}

	agent := agentList.Items[0]
	agentJSON, _ := json.MarshalIndent(agent.Spec, "", "  ")

	// Gather related resources
	var serverList agentregistryv1alpha1.MCPServerCatalogList
	_ = s.cache.List(ctx, &serverList)
	serversJSON, _ := json.MarshalIndent(summarizeServers(serverList.Items), "", "  ")

	var modelList agentregistryv1alpha1.ModelCatalogList
	_ = s.cache.List(ctx, &modelList)
	modelsJSON, _ := json.MarshalIndent(summarizeModels(modelList.Items), "", "  ")

	userMsg := fmt.Sprintf("Analyze the dependency tree for this agent:\n\nAgent:\n%s\n\nAvailable MCP Servers:\n%s\n\nAvailable Models:\n%s\n\nProduce a dependency analysis: what MCP servers does this agent need, what model does it use, are all dependencies available, and is there anything missing?",
		string(agentJSON), string(serversJSON), string(modelsJSON))

	result, err := s.requestSampling(ctx, "You are an Agent Registry dependency analyzer. Analyze an agent's dependencies and produce a health report.", userMsg)
	if err != nil {
		return textResult(fmt.Sprintf("Sampling unavailable. Agent spec:\n%s", string(agentJSON))), nil
	}

	return textResult(result), nil
}

func (s *MCPServer) handleGenerateDeploymentPlan(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	resourcesStr := getStringArg(args, "resources")
	namespace := getStringArg(args, "namespace")
	if namespace == "" {
		namespace = "agentregistry"
	}

	resourceNames := strings.Split(resourcesStr, ",")
	for i := range resourceNames {
		resourceNames[i] = strings.TrimSpace(resourceNames[i])
	}

	// Gather catalog data
	var serverList agentregistryv1alpha1.MCPServerCatalogList
	_ = s.cache.List(ctx, &serverList)

	var agentList agentregistryv1alpha1.AgentCatalogList
	_ = s.cache.List(ctx, &agentList)

	var deploymentList agentregistryv1alpha1.RegistryDeploymentList
	_ = s.cache.List(ctx, &deploymentList)

	var envList agentregistryv1alpha1.DiscoveryConfigList
	_ = s.client.List(ctx, &envList, client.InNamespace("agentregistry"))

	catalogData := map[string]interface{}{
		"requestedResources":  resourceNames,
		"targetNamespace":     namespace,
		"servers":             summarizeServers(serverList.Items),
		"agents":              summarizeAgents(agentList.Items),
		"existingDeployments": summarizeDeployments(deploymentList.Items),
	}
	catalogJSON, _ := json.MarshalIndent(catalogData, "", "  ")

	userMsg := fmt.Sprintf("Generate a deployment plan for the following resources:\n%s\n\nProduce a step-by-step plan including: order of deployment, configuration recommendations, namespace strategy, and any dependencies to resolve first.",
		string(catalogJSON))

	result, err := s.requestSampling(ctx, "You are an Agent Registry deployment planner. Generate a step-by-step deployment plan for the requested resources.", userMsg)
	if err != nil {
		return textResult(fmt.Sprintf("Sampling unavailable. Requested resources: %s\nTarget namespace: %s", resourcesStr, namespace)), nil
	}

	return textResult(result), nil
}

// --- Shared helpers ---

type registryStats struct {
	TotalServers     int `json:"totalServers"`
	TotalAgents      int `json:"totalAgents"`
	TotalSkills      int `json:"totalSkills"`
	TotalModels      int `json:"totalModels"`
	TotalDeployments int `json:"totalDeployments"`
}

func (s *MCPServer) getStats(ctx context.Context) (*registryStats, error) {
	stats := &registryStats{}

	var serverList agentregistryv1alpha1.MCPServerCatalogList
	if err := s.cache.List(ctx, &serverList); err != nil {
		return nil, err
	}
	stats.TotalServers = len(serverList.Items)

	var agentList agentregistryv1alpha1.AgentCatalogList
	if err := s.cache.List(ctx, &agentList); err != nil {
		return nil, err
	}
	stats.TotalAgents = len(agentList.Items)

	var skillList agentregistryv1alpha1.SkillCatalogList
	if err := s.cache.List(ctx, &skillList); err != nil {
		return nil, err
	}
	stats.TotalSkills = len(skillList.Items)

	var modelList agentregistryv1alpha1.ModelCatalogList
	if err := s.cache.List(ctx, &modelList); err != nil {
		return nil, err
	}
	stats.TotalModels = len(modelList.Items)

	var deploymentList agentregistryv1alpha1.RegistryDeploymentList
	if err := s.cache.List(ctx, &deploymentList); err != nil {
		return nil, err
	}
	stats.TotalDeployments = len(deploymentList.Items)

	return stats, nil
}

func sanitizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "@", "-")
	name = strings.ReplaceAll(name, " ", "-")
	if len(name) > 63 {
		name = name[:63]
	}
	return strings.Trim(name, "-")
}

type serverBrief struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Title   string `json:"title,omitempty"`
}

func summarizeServers(items []agentregistryv1alpha1.MCPServerCatalog) []serverBrief {
	result := make([]serverBrief, 0, len(items))
	for _, item := range items {
		result = append(result, serverBrief{
			Name:    item.Spec.Name,
			Version: item.Spec.Version,
			Title:   item.Spec.Title,
		})
	}
	return result
}

type modelBrief struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

func summarizeModels(items []agentregistryv1alpha1.ModelCatalog) []modelBrief {
	result := make([]modelBrief, 0, len(items))
	for _, item := range items {
		result = append(result, modelBrief{
			Name:     item.Spec.Name,
			Provider: item.Spec.Provider,
			Model:    item.Spec.Model,
		})
	}
	return result
}

type agentBrief struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Title     string `json:"title,omitempty"`
	AgentType string `json:"agentType,omitempty"`
}

func summarizeAgents(items []agentregistryv1alpha1.AgentCatalog) []agentBrief {
	result := make([]agentBrief, 0, len(items))
	for _, item := range items {
		result = append(result, agentBrief{
			Name:      item.Spec.Name,
			Version:   item.Spec.Version,
			Title:     item.Spec.Title,
			AgentType: item.Spec.AgentType,
		})
	}
	return result
}

type deployBrief struct {
	Name         string `json:"name"`
	ResourceName string `json:"resourceName"`
	Version      string `json:"version"`
	Phase        string `json:"phase"`
}

func summarizeDeployments(items []agentregistryv1alpha1.RegistryDeployment) []deployBrief {
	result := make([]deployBrief, 0, len(items))
	for _, item := range items {
		result = append(result, deployBrief{
			Name:         item.Name,
			ResourceName: item.Spec.ResourceName,
			Version:      item.Spec.Version,
			Phase:        string(item.Status.Phase),
		})
	}
	return result
}

// --- Catalog Management Handlers ---

// requireAdmin provides defense-in-depth for write operations.
// When auth is enabled, this blocks write tools even if the HTTP auth middleware is somehow bypassed.
func (s *MCPServer) requireAdmin() *mcp.CallToolResult {
	if s.authEnabled {
		return errorResult("Admin operations require authentication. Use the HTTP admin API (/admin/v0/*) with a Bearer token, or set AGENTREGISTRY_DISABLE_AUTH=true for development.")
	}
	return nil
}

func (s *MCPServer) handleCreateCatalog(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireAdmin(); err != nil {
		return err, nil
	}

	args := request.GetArguments()
	catalogType := getStringArg(args, "type")
	name := getStringArg(args, "name")
	version := getStringArg(args, "version")
	title := getStringArg(args, "title")
	description := getStringArg(args, "description")

	if name == "" || version == "" {
		return errorResult("name and version are required"), nil
	}

	namespace := config.GetNamespace()
	crName := handlers.GenerateCRName(name, version)
	labels := map[string]string{
		"agentregistry.dev/name":    handlers.SanitizeK8sName(name),
		"agentregistry.dev/version": handlers.SanitizeK8sName(version),
	}

	switch catalogType {
	case "servers":
		obj := &agentregistryv1alpha1.MCPServerCatalog{
			ObjectMeta: metav1.ObjectMeta{
				Name:      crName,
				Namespace: namespace,
				Labels:    labels,
			},
			Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
				Name:        name,
				Version:     version,
				Title:       title,
				Description: description,
			},
		}
		if err := s.client.Create(ctx, obj); err != nil {
			return errorResult(fmt.Sprintf("Failed to create server: %v", err)), nil
		}
		return textResult(fmt.Sprintf("Server '%s' v%s created", name, version)), nil

	case "agents":
		obj := &agentregistryv1alpha1.AgentCatalog{
			ObjectMeta: metav1.ObjectMeta{
				Name:      crName,
				Namespace: namespace,
				Labels:    labels,
			},
			Spec: agentregistryv1alpha1.AgentCatalogSpec{
				Name:        name,
				Version:     version,
				Title:       title,
				Description: description,
			},
		}
		if err := s.client.Create(ctx, obj); err != nil {
			return errorResult(fmt.Sprintf("Failed to create agent: %v", err)), nil
		}
		return textResult(fmt.Sprintf("Agent '%s' v%s created", name, version)), nil

	case "skills":
		category := getStringArg(args, "category")
		obj := &agentregistryv1alpha1.SkillCatalog{
			ObjectMeta: metav1.ObjectMeta{
				Name:      crName,
				Namespace: namespace,
				Labels:    labels,
			},
			Spec: agentregistryv1alpha1.SkillCatalogSpec{
				Name:        name,
				Version:     version,
				Title:       title,
				Description: description,
				Category:    category,
			},
		}
		if err := s.client.Create(ctx, obj); err != nil {
			return errorResult(fmt.Sprintf("Failed to create skill: %v", err)), nil
		}
		return textResult(fmt.Sprintf("Skill '%s' v%s created", name, version)), nil

	case "models":
		provider := getStringArg(args, "provider")
		model := getStringArg(args, "model")
		if provider == "" || model == "" {
			return errorResult("provider and model are required for models"), nil
		}
		obj := &agentregistryv1alpha1.ModelCatalog{
			ObjectMeta: metav1.ObjectMeta{
				Name:      crName,
				Namespace: namespace,
				Labels:    labels,
			},
			Spec: agentregistryv1alpha1.ModelCatalogSpec{
				Name:        name,
				Provider:    provider,
				Model:       model,
				Description: description,
			},
		}
		if err := s.client.Create(ctx, obj); err != nil {
			return errorResult(fmt.Sprintf("Failed to create model: %v", err)), nil
		}
		return textResult(fmt.Sprintf("Model '%s' (%s/%s) created", name, provider, model)), nil

	default:
		return errorResult("Invalid type: must be servers, agents, skills, or models"), nil
	}
}

func (s *MCPServer) handleDeleteCatalog(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.requireAdmin(); err != nil {
		return err, nil
	}

	args := request.GetArguments()
	catalogType := getStringArg(args, "type")
	name := getStringArg(args, "name")

	if name == "" {
		return errorResult("name is required"), nil
	}

	switch catalogType {
	case "servers":
		var list agentregistryv1alpha1.MCPServerCatalogList
		if err := s.cache.List(ctx, &list, client.MatchingFields{
			controller.IndexMCPServerName: name,
		}); err != nil {
			return errorResult(fmt.Sprintf("Failed to find server: %v", err)), nil
		}
		if len(list.Items) == 0 {
			return errorResult(fmt.Sprintf("Server '%s' not found", name)), nil
		}
		deleted := 0
		for i := range list.Items {
			if err := s.client.Delete(ctx, &list.Items[i]); err != nil {
				return errorResult(fmt.Sprintf("Failed to delete server '%s': %v", list.Items[i].Name, err)), nil
			}
			deleted++
		}
		return textResult(fmt.Sprintf("Deleted %d version(s) of server '%s'", deleted, name)), nil

	case "agents":
		var list agentregistryv1alpha1.AgentCatalogList
		if err := s.cache.List(ctx, &list, client.MatchingFields{
			controller.IndexAgentName: name,
		}); err != nil {
			return errorResult(fmt.Sprintf("Failed to find agent: %v", err)), nil
		}
		if len(list.Items) == 0 {
			return errorResult(fmt.Sprintf("Agent '%s' not found", name)), nil
		}
		deleted := 0
		for i := range list.Items {
			if err := s.client.Delete(ctx, &list.Items[i]); err != nil {
				return errorResult(fmt.Sprintf("Failed to delete agent '%s': %v", list.Items[i].Name, err)), nil
			}
			deleted++
		}
		return textResult(fmt.Sprintf("Deleted %d version(s) of agent '%s'", deleted, name)), nil

	case "skills":
		var list agentregistryv1alpha1.SkillCatalogList
		if err := s.cache.List(ctx, &list, client.MatchingFields{
			controller.IndexSkillName: name,
		}); err != nil {
			return errorResult(fmt.Sprintf("Failed to find skill: %v", err)), nil
		}
		if len(list.Items) == 0 {
			return errorResult(fmt.Sprintf("Skill '%s' not found", name)), nil
		}
		deleted := 0
		for i := range list.Items {
			if err := s.client.Delete(ctx, &list.Items[i]); err != nil {
				return errorResult(fmt.Sprintf("Failed to delete skill '%s': %v", list.Items[i].Name, err)), nil
			}
			deleted++
		}
		return textResult(fmt.Sprintf("Deleted %d version(s) of skill '%s'", deleted, name)), nil

	case "models":
		var list agentregistryv1alpha1.ModelCatalogList
		if err := s.cache.List(ctx, &list, client.MatchingFields{
			controller.IndexModelName: name,
		}); err != nil {
			return errorResult(fmt.Sprintf("Failed to find model: %v", err)), nil
		}
		if len(list.Items) == 0 {
			return errorResult(fmt.Sprintf("Model '%s' not found", name)), nil
		}
		for i := range list.Items {
			if err := s.client.Delete(ctx, &list.Items[i]); err != nil {
				return errorResult(fmt.Sprintf("Failed to delete model '%s': %v", list.Items[i].Name, err)), nil
			}
		}
		return textResult(fmt.Sprintf("Deleted model '%s'", name)), nil

	default:
		return errorResult("Invalid type: must be servers, agents, skills, or models"), nil
	}
}
