# MCP Server Authentication

This guide covers authentication for the Agent Registry MCP server. For HTTP API and UI authentication, see [azure-ad-setup.md](./azure-ad-setup.md).

## Overview

Auth is **disabled by default**. When enabled (`AGENTREGISTRY_AUTH_ENABLED=true`), the MCP server uses a two-layer model:

1. **Layer 1: HTTP Bearer token** — validates token at transport level before any MCP processing
2. **Layer 2: Per-tool `requireAdmin()`** — defense in depth, blocks write tools even if Layer 1 is bypassed

Both layers read tokens from the `agentregistry-api-tokens` Kubernetes Secret.

### Authentication Flow

```
MCP Client Request
│
├─ Auth disabled (default)? ──→ Pass through
│
├─ Layer 1: HTTP Middleware
│   ├─ Missing Authorization header? ──→ 401 Unauthorized
│   ├─ Invalid format? ──→ 401 Unauthorized
│   ├─ Token not in allowlist? ──→ 401 Unauthorized
│   └─ Valid token ──→ Continue
│
└─ Layer 2: Per-tool requireAdmin() (write tools only)
    └─ Blocks if auth enabled and token missing
```

## Token Source

The MCP server and HTTP API share the same Secret: `agentregistry-api-tokens` in the controller namespace (`agentregistry`).

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: agentregistry-api-tokens
  namespace: agentregistry
type: Opaque
data:
  admin-token: <base64-encoded-token>
  ci-token: <base64-encoded-token>
```

### Creating Tokens

```bash
kubectl create secret generic agentregistry-api-tokens \
  -n agentregistry \
  --from-literal=admin-token=$(openssl rand -hex 32) \
  --from-literal=ci-token=$(openssl rand -hex 32)
```

> **Note:** Token changes require a controller restart — tokens are loaded once at startup.

## Client Configuration

### Auth Disabled (default)

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

### Auth Enabled

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

## Auth Model by Tool

| Tool | Type | Auth Required |
|------|------|---------------|
| `list_catalog` | Read | No |
| `get_catalog` | Read | No |
| `get_registry_stats` | Read | No |
| `list_deployments` | Read | No |
| `get_deployment` | Read | No |
| `list_environments` | Read | No |
| `get_discovery_map` | Read | No |
| `recommend_servers` | Read | No |
| `analyze_agent_dependencies` | Read | No |
| `generate_deployment_plan` | Read | No |
| `create_catalog` | Write | Yes (when auth enabled) |
| `delete_catalog` | Write | Yes |
| `deploy_catalog_item` | Write | Yes |
| `delete_deployment` | Write | Yes |
| `update_deployment_config` | Write | Yes |
| `trigger_discovery` | Write | Yes |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AGENTREGISTRY_AUTH_ENABLED` | unset (auth off) | Set to `true` to enable Bearer token auth |
| `POD_NAMESPACE` | `agentregistry` | Namespace where the token Secret is read from |

## Production Checklist

- [ ] `agentregistry-api-tokens` Secret exists with strong random tokens
- [ ] MCP port (`:8083`) is not exposed via Ingress or LoadBalancer
- [ ] NetworkPolicy restricts access to the MCP port
- [ ] RBAC restricts read access to the `agentregistry-api-tokens` Secret
- [ ] Separate tokens for each MCP client for audit trail

## Troubleshooting

**"Missing Authorization header"** — Add `Authorization: Bearer <token>` header in your MCP client config.

**"Invalid token"** — Check the token matches a value in the Secret: `kubectl get secret agentregistry-api-tokens -n agentregistry -o jsonpath='{.data.admin-token}' | base64 -d`

**Connection refused** — Check the controller is running: `kubectl get pods -n agentregistry` and port-forward: `kubectl port-forward -n agentregistry svc/agentregistry-controller 8083:8083`
