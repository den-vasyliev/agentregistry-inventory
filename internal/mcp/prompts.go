package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *MCPServer) registerPrompts() {
	s.mcpServer.AddPrompt(
		mcp.NewPrompt("agentregistry_skill",
			mcp.WithPromptDescription("Complete guide for managing a Kubernetes-native registry of MCP servers, agents, skills, and models. Use when browsing the catalog, deploying resources, analyzing dependencies, or exploring multi-cluster discovery."),
		),
		s.handlePromptSkill,
	)

	s.mcpServer.AddPrompt(
		mcp.NewPrompt("deploy_server",
			mcp.WithPromptDescription("Step-by-step workflow for deploying an MCP server from the catalog to Kubernetes. Use when asked to deploy, install, or set up an MCP server."),
			mcp.WithArgument("server_name",
				mcp.ArgumentDescription("Optional: MCP server name to deploy"),
			),
		),
		s.handlePromptDeployServer,
	)

	s.mcpServer.AddPrompt(
		mcp.NewPrompt("find_agents",
			mcp.WithPromptDescription("Guided workflow for discovering agents by use case or capability. Use when asked to find, search, or recommend agents."),
			mcp.WithArgument("use_case",
				mcp.ArgumentDescription("Optional: describe your use case"),
			),
		),
		s.handlePromptFindAgents,
	)

	s.mcpServer.AddPrompt(
		mcp.NewPrompt("registry_overview",
			mcp.WithPromptDescription("Comprehensive snapshot of registry contents, deployments, and multi-cluster discovery status. Use when asked for a summary or overview of what's in the registry."),
		),
		s.handlePromptRegistryOverview,
	)
}

func (s *MCPServer) handlePromptSkill(_ context.Context, _ mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: "Agent Registry MCP Server - Complete Guide",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: `You have access to Agent Registry, a Kubernetes-native registry for MCP servers, agents, skills, and models. Use these tools to browse the catalog, deploy resources to clusters, and explore multi-cluster discovery.

## Workflows

### Browse the catalog
1. Call get_registry_stats to see total counts
2. Call list_catalog with type="servers", "agents", "skills", or "models"
   - Add version="latest" to see only the latest version of each entry
   - Add search="keyword" to filter by name
3. Call get_catalog with type and name for full details (packages, transports, endpoints, tools, etc.)

### Deploy a resource to Kubernetes
1. Find the resource: list_catalog type="servers" (or "agents") to browse, then get_catalog for details
2. Check existing deployments: list_deployments to avoid duplicates
3. Deploy: deploy_catalog_item with resourceName, version, and resourceType="mcp" (for servers) or "agent" (for agents)
4. Verify: get_deployment with the deployment name (pattern: sanitized-name-version)
5. Update config later with update_deployment_config if needed

### Explore multi-cluster discovery
1. Call list_environments to see discovered environments (dev, staging, prod, etc.)
2. Call get_discovery_map for the full topology: clusters, namespaces, resource counts, connection status
3. Call trigger_discovery to force a re-scan if data looks stale

### Analyze dependencies (AI-powered)
These tools use MCP sampling to invoke your LLM for analysis. They degrade gracefully to raw data if sampling is unavailable.
- recommend_servers: describe a use case, get matching server recommendations
- analyze_agent_dependencies: provide an agent name, get its full dependency tree (MCP servers, models, missing deps)
- generate_deployment_plan: provide comma-separated resource names, get ordered deployment steps

### Manage catalog entries
- create_catalog: create a new entry (basic fields only — use kubectl apply for full spec)
- delete_catalog: remove all versions of an entry by name

## Key details
- All resources live in the "agentregistry" namespace by default
- Catalog types: servers, agents, skills, models
- Deployment resourceType is "mcp" for servers, "agent" for agents
- Write tools (deploy, delete, update, trigger, create, delete catalog) require auth when enabled
- Discovery watches environment namespaces and auto-creates catalog entries in the agentregistry namespace`,
				},
			},
		},
	}, nil
}

func (s *MCPServer) handlePromptDeployServer(_ context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	serverName := request.Params.Arguments["server_name"]

	var steps string
	if serverName != "" {
		steps = fmt.Sprintf(`Deploy the MCP server "%s" from the Agent Registry.

Steps:
1. get_catalog type="servers" name="%s" — review details (version, packages, transports)
2. list_deployments — check if already deployed (avoid duplicates)
3. deploy_catalog_item resourceName="%s" version=<latest> resourceType="mcp" — deploy it
4. get_deployment name=<deployment-name> — verify status

If the server is not found in the catalog, search with list_catalog type="servers" to find the correct name.`, serverName, serverName, serverName)
	} else {
		steps = `Help me deploy an MCP server from the Agent Registry.

Steps:
1. list_catalog type="servers" version="latest" — show available servers
2. Ask me which server to deploy (or recommend one if I describe my use case)
3. get_catalog type="servers" name=<chosen> — review details
4. list_deployments — check for existing deployments
5. deploy_catalog_item resourceName=<name> version=<version> resourceType="mcp" — deploy
6. get_deployment name=<deployment-name> — verify status`
	}

	return &mcp.GetPromptResult{
		Description: "Deploy MCP Server Workflow",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: steps,
				},
			},
		},
	}, nil
}

func (s *MCPServer) handlePromptFindAgents(_ context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	useCase := request.Params.Arguments["use_case"]

	var steps string
	if useCase != "" {
		steps = fmt.Sprintf(`Find agents in the Agent Registry for this use case: %s

Steps:
1. list_catalog type="agents" version="latest" — browse available agents
2. Identify agents whose description or agentType matches the use case
3. get_catalog type="agents" name=<agent> — check details: mcpServers, tools, modelConfigRef, systemMessage
4. Verify dependencies are available:
   - list_catalog type="servers" — check required MCP servers exist
   - list_catalog type="models" — check model requirements
5. Summarize: agent name, what it does, its dependencies, and whether all deps are satisfied`, useCase)
	} else {
		steps = `Help me find agents in the Agent Registry.

Steps:
1. list_catalog type="agents" version="latest" — show all available agents
2. Present a summary of each agent: name, type, description
3. When I pick one, get_catalog type="agents" name=<agent> for full details
4. Check its dependencies:
   - list_catalog type="servers" — verify required MCP servers exist
   - list_catalog type="models" — verify model requirements
5. Report any missing dependencies`
	}

	return &mcp.GetPromptResult{
		Description: "Find Agents Workflow",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: steps,
				},
			},
		},
	}, nil
}

func (s *MCPServer) handlePromptRegistryOverview(_ context.Context, _ mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: "Registry Overview",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: `Give me a comprehensive overview of the Agent Registry.

Gather data by calling these tools (run independent calls in parallel where possible):
1. get_registry_stats — total resource counts
2. list_catalog type="servers" version="latest" — available MCP servers
3. list_catalog type="agents" version="latest" — available agents
4. list_catalog type="skills" — available skills
5. list_catalog type="models" — model configurations
6. list_deployments — active deployments
7. get_discovery_map — multi-cluster topology

Present a structured summary:
- Resource counts (servers, agents, skills, models, deployments)
- Key resources in each category (name, version, brief description)
- Deployment status (what's deployed, in which namespaces)
- Discovery status (connected clusters, environments, any errors)`,
				},
			},
		},
	}, nil
}
