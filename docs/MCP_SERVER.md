# Agent Registry MCP Server

The Agent Registry exposes an MCP (Model Context Protocol) server that allows AI agents and tools like Claude Code to browse, manage, and deploy resources in the registry directly.

The MCP server runs embedded in the controller on port `:8083` (configurable via `--mcp-addr`).

## Configuration

### Claude Code (`.mcp.json`)

```json
{
  "mcpServers": {
    "agentregistry": {
      "type": "streamable-http",
      "url": "http://localhost:8083/mcp"
    }
  }
}
```

### Port-forward to cluster

```bash
kubectl port-forward -n agentregistry svc/agentregistry-controller 8083:8083
```

## Capabilities

The MCP server provides **tools**, **resources**, and **prompts**.

### Tools

#### Catalog Browsing (read-only)

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `list_catalog` | List catalog entries by type | `type` (servers/agents/skills/models), `search?`, `version?`, `category?`, `provider?`, `limit?` |
| `get_catalog` | Get catalog entry details | `type`, `name`, `version?` |
| `get_registry_stats` | Get counts of all resource types | _(none)_ |

#### Catalog Management (requires auth disabled or dev mode)

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `create_catalog` | Create a new catalog entry | `type`, `name`, `version`, `title?`, `description?`, `category?` (skills), `provider?` + `model?` (models) |
| `delete_catalog` | Delete a catalog entry (all versions) | `type`, `name` |

#### Deployment Management

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `list_deployments` | List deployments | `resourceType?`, `limit?` |
| `get_deployment` | Get deployment details | `name` |
| `deploy_catalog_item` | Deploy a catalog item to K8s | `resourceName`, `version`, `resourceType` (mcp/agent), `namespace?`, `config?` |
| `update_deployment_config` | Merge config into deployment | `name`, `config` |
| `delete_deployment` | Delete a deployment | `name` |

#### Discovery

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `list_environments` | List discovered environments | _(none)_ |
| `get_discovery_map` | Get topology map with clusters, environments, resource counts | _(none)_ |
| `trigger_discovery` | Force re-scan of DiscoveryConfig | `configName?` |

#### AI-Powered (uses MCP sampling)

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `recommend_servers` | Recommend MCP servers for a use case | `description` |
| `analyze_agent_dependencies` | Analyze an agent's dependency tree | `name` |
| `generate_deployment_plan` | Generate a deployment plan | `resources` (comma-separated), `namespace?` |

> **Note:** Sampling-powered tools require the MCP client to support sampling. When unavailable, they gracefully degrade and return raw catalog data instead.

### Resources

| URI | Description |
|-----|-------------|
| `registry://stats` | Registry statistics (static resource) |
| `registry://servers` | All MCP servers |
| `registry://servers/{name}` | MCP server details |
| `registry://agents` | All agents |
| `registry://agents/{name}` | Agent details |
| `registry://skills` | All skills |
| `registry://skills/{name}` | Skill details |
| `registry://models` | All models |
| `registry://models/{name}` | Model details |
| `registry://deployments` | All deployments |
| `registry://deployments/{name}` | Deployment details |
| `registry://environments` | All environments |

### Prompts

| Prompt | Description | Arguments |
|--------|-------------|-----------|
| `agentregistry_skill` | Complete guide to all tools, workflows, and best practices | _(none)_ |
| `deploy_server` | Guided workflow for deploying an MCP server | `server_name?` |
| `find_agents` | Guided workflow for finding agents | `use_case?` |
| `registry_overview` | Comprehensive overview of registry contents and status | _(none)_ |

## Authentication

The MCP server uses two layers of authentication:

1. **HTTP Bearer token middleware** - Validates tokens at the transport level before any MCP processing
2. **Per-tool `requireAdmin()` gating** - Defense in depth for all write/mutating tools

Tokens are read from the `agentregistry-api-tokens` Kubernetes Secret (same as the HTTP API).

### Quick Start

```bash
# 1. Create a token
kubectl create secret generic agentregistry-api-tokens \
  -n agentregistry \
  --from-literal=my-token=$(openssl rand -hex 32)

# 2. Get the token value
kubectl get secret agentregistry-api-tokens -n agentregistry \
  -o jsonpath='{.data.my-token}' | base64 -d

# 3. Configure your MCP client with the token (see below)
```

### Client Configuration (with auth)

```json
{
  "mcpServers": {
    "agentregistry": {
      "type": "streamable-http",
      "url": "http://localhost:8083/mcp",
      "headers": {
        "Authorization": "Bearer your-secret-token"
      }
    }
  }
}
```

### Disabling Auth (Development)

```bash
AGENTREGISTRY_DISABLE_AUTH=true
```

When disabled, no headers are needed and all tools are accessible.

For full details on the auth model, token management, error responses, troubleshooting, and security considerations, see [MCP Authentication](./mcp-auth.md).

## Test Results

Tested via Claude Code MCP client against a live cluster (auth disabled).

| Capability | Status | Notes |
|---|---|---|
| `list_catalog` (all 4 types) | OK | servers, agents, skills, models with filtering |
| `get_catalog` (all 4 types) | OK | by name + version |
| `get_registry_stats` | OK | counts |
| `create_catalog` (all 4 types) | OK | servers, agents, skills, models |
| `delete_catalog` (all 4 types) | OK | deletes all versions of a resource |
| `deploy_catalog_item` | OK | creates RegistryDeployment CR |
| `get_deployment` | OK | |
| `list_deployments` | OK | |
| `update_deployment_config` | OK | merges config |
| `delete_deployment` | OK | |
| `list_environments` | OK | |
| `get_discovery_map` | OK | |
| `trigger_discovery` | OK | |
| `recommend_servers` | OK (degraded) | Sampling unavailable in client, returns raw data |
| `analyze_agent_dependencies` | OK (degraded) | Sampling unavailable in client, returns raw data |
| `generate_deployment_plan` | OK (degraded) | Sampling unavailable in client, returns raw data |
| Resources (static + templates) | Registered | `registry://stats` works, 11 templates registered |
| Prompts (4 prompts) | Registered | |

**18 tools total - all functional.**

## Known Limitations

| Limitation | Description | Workaround |
|------------|-------------|------------|
| No `update_catalog` | Cannot update an existing catalog entry in-place | Delete + re-create |
| No version-specific delete | `delete_catalog` removes **all versions** of a resource | Use HTTP admin API for version-specific delete |
| Limited create fields | `create_catalog` only supports basic fields (name, version, title, description). Cannot set packages, transports, endpoints (servers), systemMessage, tools, modelConfigRef (agents), etc. | Use HTTP admin API or `kubectl apply` for full spec |
| Sampling degradation | AI-powered tools (`recommend_servers`, `analyze_agent_dependencies`, `generate_deployment_plan`) fall back to raw data when the MCP client doesn't support sampling | Results are still useful, just not AI-summarized |
| Auth gating | When auth is enabled, all MCP requests require a Bearer token from the `agentregistry-api-tokens` Secret | Configure token in MCP client headers |

## Architecture

```
Controller Binary
├── Kubernetes Reconcilers (port :8081 metrics, :8082 health)
├── HTTP API Server (:8080) ── REST API + embedded UI
└── MCP Server (:8083) ── Streamable HTTP
    ├── Tools (18 tools)
    ├── Resources (1 static + 11 templates)
    └── Prompts (4 prompts)
```

The MCP server uses:
- **`controller-runtime` cache** for read operations (consistent with controller's view)
- **`controller-runtime` client** for write operations (direct to API server)
- **`mcp-go`** library (`github.com/mark3labs/mcp-go`) for the MCP protocol implementation
