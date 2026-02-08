# Master Agent MCP and A2A Integration

This document describes how to interact with the Master Agent via MCP tools and A2A protocol.

## Overview

The Master Agent processes infrastructure events using an LLM and maintains a world state with incidents. You can interact with it in two ways:

1. **MCP Tools** - Simple, tool-based queries (added in this update)
2. **A2A Protocol** - Conversational interface for asking questions

## Architecture

```
┌─────────────────┐
│  MCP Client     │ ──── MCP Tools ────┐
└─────────────────┘                    │
                                       ▼
┌─────────────────┐              ┌──────────────┐
│  A2A Client     │ ── A2A ───▶  │ Master Agent │
└─────────────────┘              │              │
                                 │  - EventHub  │
┌─────────────────┐              │  - LLM       │
│  HTTP API       │ ── REST ──▶  │  - WorldState│
└─────────────────┘              └──────────────┘
```

## MCP Tools (Port 8083)

### 1. `get_master_agent_status`

Get the current state of the master agent including world state, active incidents, and queue status.

**No parameters required**

**Returns:**
```json
{
  "running": true,
  "worldState": {
    "lastUpdated": "2026-02-08T10:30:00Z",
    "summary": "All systems operational",
    "totalEvents": 42,
    "pendingEvents": 2,
    "activeIncidents": 0
  },
  "incidents": [],
  "queueDepth": 2,
  "queueTotal": 42,
  "a2aEndpoint": "http://localhost:8084"
}
```

### 2. `emit_event`

Emit an infrastructure event to the master agent for LLM analysis.

**Parameters:**
- `type` (required): Event type (e.g., "pod-crash", "node-pressure", "deployment-failed")
- `message` (required): Human-readable event description
- `severity` (optional): "info", "warning", or "critical" (default: "info")
- `source` (optional): Event source identifier (default: "mcp-client")

**Example:**
```json
{
  "type": "pod-crash",
  "message": "Pod nginx-deployment-abc123 crashed with OOMKilled",
  "severity": "critical",
  "source": "k8s/pod/production/nginx-deployment-abc123"
}
```

**Returns:**
```
✓ Event emitted successfully

Event ID: 550e8400-e29b-41d4-a716-446655440000
Type: pod-crash
Severity: critical
Message: Pod nginx-deployment-abc123 crashed with OOMKilled

The master agent will process this event and update its world state.
```

### 3. `get_recent_events`

Get recent events processed by the master agent.

**Parameters:**
- `limit` (optional): Max events to return (default: 20, max: 100)

**Returns:**
```json
{
  "count": 5,
  "limit": 20,
  "events": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "source": "k8s/pod/production/nginx",
      "type": "pod-crash",
      "severity": "critical",
      "message": "Pod crashed with OOMKilled",
      "timestamp": "2026-02-08T10:25:00Z"
    }
  ]
}
```

## A2A Protocol (Port 8084)

The A2A server exposes the master agent as a conversational interface. Use an A2A-compatible client to ask questions.

### Example Queries:

**Q: "Is anything wrong in the infrastructure?"**

A: "Currently investigating 1 critical incident: Pod nginx-deployment-abc123 in production namespace crashed with OOMKilled at 10:25 UTC. Memory limit is set to 256Mi. Recommend increasing memory limit or investigating memory leak."

**Q: "What happened to the nginx deployment?"**

A: "The nginx deployment experienced a pod crash at 10:25 UTC. The pod was killed due to Out of Memory (OOMKilled). This is the 3rd occurrence in the past hour, suggesting a memory leak or insufficient memory allocation."

**Q: "Show me recent events"**

A: "In the last hour I've processed 12 events: 3 pod crashes (nginx), 2 node pressure warnings, 1 deployment update success, and 6 routine health checks."

## HTTP API (Port 8080)

Alternative REST endpoints:

### GET /v0/agent/status
Returns master agent status (same as MCP `get_master_agent_status`)

### POST /v0/agent/events
Emit an event (same as MCP `emit_event`)

**Example:**
```bash
curl -X POST http://localhost:8080/v0/agent/events \
  -H "Content-Type: application/json" \
  -d '{
    "type": "test-event",
    "severity": "warning",
    "message": "Testing master agent",
    "source": "manual-test"
  }'
```

## Configuration

Enable the master agent via `MasterAgentConfig` CRD:

```yaml
apiVersion: agentregistry.dev/v1alpha1
kind: MasterAgentConfig
metadata:
  name: default
  namespace: agentregistry
spec:
  enabled: true
  models:
    default: gemini-2.0-flash-exp  # ModelCatalog name
  mcpServers:
    - name: filesystem
      url: http://mcp-filesystem:8080
  a2a:
    enabled: true
    port: 8084  # A2A server port
  maxConcurrentEvents: 5
  systemPrompt: |
    You are an autonomous infrastructure observer...
```

## Event Flow

1. **Event Creation** → `emit_event` (MCP) or HTTP POST → EventHub queue
2. **Master Agent** → Worker processes event with LLM
3. **LLM Analysis** → Uses MCP tools from configured servers
4. **State Update** → Updates WorldState summary, creates/resolves incidents
5. **Query Results** → `get_master_agent_status` (MCP) or A2A chat

## Testing

```bash
# MCP client (using Python SDK or Go SDK)
# Call tool: get_master_agent_status
# Call tool: emit_event with type="test", message="Testing"

# A2A client
# Connect to http://localhost:8084
# Ask: "What's the current state?"

# HTTP
curl http://localhost:8080/v0/agent/status
```

## Implementation Files

- `internal/mcp/server.go` - MCP server with master agent accessors
- `internal/mcp/masteragent_tools.go` - MCP tool handlers
- `internal/controller/masteragent_controller.go` - A2A server startup
- `internal/masteragent/agent.go` - Master agent core
- `internal/masteragent/eventhub.go` - Event queue
- `internal/masteragent/worldstate.go` - State management
