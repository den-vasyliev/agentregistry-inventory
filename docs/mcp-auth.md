# MCP Server Authentication

This guide covers the authentication system for the Agent Registry MCP server. For general MCP server usage, see [MCP_SERVER.md](./MCP_SERVER.md). For HTTP API and UI authentication (OIDC), see [oidc-setup.md](./oidc-setup.md).

## Overview

The MCP server uses a two-layer authentication model:

1. **Layer 1: HTTP Bearer token middleware** - Validates tokens at the transport level before any MCP protocol processing occurs
2. **Layer 2: Per-tool `requireAdmin()` gating** - Defense in depth that blocks write/mutating tools even if the HTTP layer is bypassed

Both layers read configuration from the same source: the `AGENTREGISTRY_DISABLE_AUTH` environment variable and the `agentregistry-api-tokens` Kubernetes Secret.

### Authentication Flow

```
MCP Client Request
│
├─ Auth disabled? ──→ Pass through to MCP server
│
├─ Layer 1: HTTP Middleware
│   ├─ Missing Authorization header? ──→ 401 Unauthorized
│   ├─ Invalid format (not "Bearer <token>")? ──→ 401 Unauthorized
│   ├─ Token not in allowedTokens? ──→ 401 Unauthorized
│   └─ Valid token ──→ Continue to MCP server
│
└─ Layer 2: Per-tool requireAdmin() (write tools only)
    ├─ authEnabled=true? ──→ Error: "Admin operations require authentication"
    └─ authEnabled=false? ──→ Allow
```

> **Note:** Layer 2 (`requireAdmin()`) currently acts as an unconditional block when auth is enabled, providing defense in depth. The HTTP middleware (Layer 1) is the primary authentication mechanism. In the future, Layer 2 may be extended to support role-based access within authenticated sessions.

## Token Source

The MCP server and HTTP API share the same token source: a Kubernetes Secret named `agentregistry-api-tokens` in the controller namespace (default: `agentregistry`).

Each key in the Secret represents a named token. The key name is used for logging; the value is the actual token string.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: agentregistry-api-tokens
  namespace: agentregistry
type: Opaque
data:
  # Each key is a token name, value is the base64-encoded token
  admin-token: <base64-encoded-token>
  ci-token: <base64-encoded-token>
  mcp-readonly: <base64-encoded-token>
```

### Creating Tokens

```bash
# Single token
kubectl create secret generic agentregistry-api-tokens \
  -n agentregistry \
  --from-literal=admin-token=$(openssl rand -hex 32)

# Multiple tokens
kubectl create secret generic agentregistry-api-tokens \
  -n agentregistry \
  --from-literal=admin-token=$(openssl rand -hex 32) \
  --from-literal=ci-token=$(openssl rand -hex 32) \
  --from-literal=mcp-client=$(openssl rand -hex 32)
```

### Updating Tokens

```bash
# Replace the entire secret
kubectl delete secret agentregistry-api-tokens -n agentregistry
kubectl create secret generic agentregistry-api-tokens \
  -n agentregistry \
  --from-literal=admin-token=$(openssl rand -hex 32)
```

> **Important:** Token changes require a controller restart. Tokens are loaded once at startup and cached in memory.

## Auth Model by Tool

| Tool | Type | HTTP Auth | Tool Auth | Notes |
|------|------|-----------|-----------|-------|
| `list_catalog` | Read | Bearer token | None | |
| `get_catalog` | Read | Bearer token | None | |
| `get_registry_stats` | Read | Bearer token | None | |
| `list_deployments` | Read | Bearer token | None | |
| `get_deployment` | Read | Bearer token | None | |
| `list_environments` | Read | Bearer token | None | |
| `get_discovery_map` | Read | Bearer token | None | |
| `recommend_servers` | Read | Bearer token | None | Uses sampling |
| `analyze_agent_dependencies` | Read | Bearer token | None | Uses sampling |
| `generate_deployment_plan` | Read | Bearer token | None | Uses sampling |
| `create_catalog` | Write | Bearer token | `requireAdmin()` | |
| `delete_catalog` | Write | Bearer token | `requireAdmin()` | |
| `deploy_catalog_item` | Write | Bearer token | `requireAdmin()` | |
| `delete_deployment` | Write | Bearer token | `requireAdmin()` | |
| `update_deployment_config` | Write | Bearer token | `requireAdmin()` | |
| `trigger_discovery` | Write | Bearer token | `requireAdmin()` | |

Resources and prompts are also gated by the HTTP middleware (Layer 1) when auth is enabled.

## Client Configuration

### Claude Code

Add to `.mcp.json` in your project root or `~/.claude.json`:

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

### Development (Auth Disabled)

When auth is disabled, no headers are needed:

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

### Generic MCP Client

Any MCP client that supports Streamable HTTP transport can authenticate by including the `Authorization` header:

```
Authorization: Bearer your-secret-token
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AGENTREGISTRY_DISABLE_AUTH` | `false` (auth enabled) | Set to `true` to disable both layers of auth |
| `POD_NAMESPACE` | `agentregistry` | Namespace where the `agentregistry-api-tokens` Secret is read from |

## Error Responses

When auth is enabled, the MCP server returns standard HTTP error responses at the transport level (before MCP protocol processing):

### Missing Authorization Header

```
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{"error":"Missing Authorization header"}
```

### Invalid Header Format

```
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{"error":"Invalid Authorization header format"}
```

### Invalid Token

```
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{"error":"Invalid token"}
```

### Write Tool Blocked (Layer 2)

If a write tool is called and `authEnabled=true`, the tool returns an MCP error result (not an HTTP error):

```json
{
  "isError": true,
  "content": [{
    "type": "text",
    "text": "Admin operations require authentication. Use the HTTP admin API (/admin/v0/*) with a Bearer token, or set AGENTREGISTRY_DISABLE_AUTH=true for development."
  }]
}
```

## Comparison with HTTP API Auth

| Aspect | HTTP API (`:8080`) | MCP Server (`:8083`) |
|--------|-------------------|---------------------|
| Token source | `agentregistry-api-tokens` Secret | Same Secret |
| Auth toggle | `AGENTREGISTRY_DISABLE_AUTH` | Same variable |
| Public endpoints | `/v0/*` (no auth required) | None (all-or-nothing) |
| Admin endpoints | `/admin/v0/*` (Bearer token required) | All tools (when auth enabled) |
| OIDC support | Yes (deploy write endpoints) | No |
| Auth granularity | Per-endpoint (public vs admin) | Per-request (all tools) + per-tool (write tools) |

The key difference: the HTTP API has separate public and admin endpoint prefixes, allowing unauthenticated read access. The MCP server operates on a single endpoint (`/mcp`), so authentication is all-or-nothing at the HTTP level. The `requireAdmin()` tool-level check provides additional write protection.

## Security Considerations

### Network Access

- The MCP port (`:8083`) should **not** be exposed publicly
- Use Kubernetes NetworkPolicy or service mesh to restrict access
- In development, access via `kubectl port-forward`

```bash
kubectl port-forward -n agentregistry svc/agentregistry-controller 8083:8083
```

### Token Management

- Tokens are loaded at controller startup and cached in memory
- Tokens are stored in plaintext in the Secret (standard K8s Secret behavior)
- Use Kubernetes RBAC to restrict access to the `agentregistry-api-tokens` Secret
- Rotate tokens by updating the Secret and restarting the controller
- Use separate tokens for different clients/environments for audit trail

### Defense in Depth

The two-layer model ensures:

1. **If the HTTP middleware is misconfigured** - `requireAdmin()` still blocks write tools
2. **If auth is accidentally disabled** - Network-level controls still protect the MCP port
3. **If the MCP port is accidentally exposed** - Bearer tokens prevent unauthorized access

### Production Checklist

- [ ] `AGENTREGISTRY_DISABLE_AUTH` is unset or `false`
- [ ] `agentregistry-api-tokens` Secret exists with strong tokens
- [ ] MCP port (`:8083`) is not exposed via Ingress or LoadBalancer
- [ ] NetworkPolicy restricts access to MCP port
- [ ] RBAC restricts read access to the `agentregistry-api-tokens` Secret
- [ ] Separate tokens configured for each MCP client

## Troubleshooting

### "Missing Authorization header"

- Verify your MCP client config includes the `headers` field
- Check that `Authorization` header is being sent (not stripped by proxy)

### "Invalid token"

- Verify the token matches a value in the `agentregistry-api-tokens` Secret
- Check for leading/trailing whitespace in the Secret value
- Decode the Secret to verify: `kubectl get secret agentregistry-api-tokens -n agentregistry -o jsonpath='{.data.admin-token}' | base64 -d`

### "Admin operations require authentication"

- This is the Layer 2 (`requireAdmin()`) error for write tools
- Either disable auth for development (`AGENTREGISTRY_DISABLE_AUTH=true`) or use the HTTP admin API for production write operations

### Connection refused

- Check the controller is running: `kubectl get pods -n agentregistry`
- Verify port-forward is active: `kubectl port-forward -n agentregistry svc/agentregistry-controller 8083:8083`
- Check controller logs: `kubectl logs -n agentregistry -l app=agentregistry-controller`

### Tokens not loading

- Check controller logs for: `loaded MCP API tokens from secret` or error messages
- Verify Secret exists: `kubectl get secret agentregistry-api-tokens -n agentregistry`
- Verify namespace matches: check `POD_NAMESPACE` env var on the controller pod
