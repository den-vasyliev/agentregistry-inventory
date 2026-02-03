# End-to-End Test Plan: Agent Registry

## Prerequisites

- Kubernetes cluster (kind, minikube, or remote)
- kubectl configured
- Helm 3.x installed
- Docker (for building images if needed)

---

## Phase 1: Installation

### 1.1 Build Controller Image

```bash
# Build the controller
go build -o bin/controller ./cmd/controller

# Or build Docker image
docker build -t agentregistry-controller:dev -f Dockerfile .
```

### 1.2 Install CRDs

```bash
# Apply CRDs directly
kubectl apply -f charts/agentregistry/templates/crds/

# Verify CRDs are installed
kubectl get crds | grep agentregistry
```

**Expected output:**
```
agentcatalogs.agentregistry.dev
mcpservercatalogs.agentregistry.dev
registrydeployments.agentregistry.dev
skillcatalogs.agentregistry.dev
```

### 1.3 Install via Helm

```bash
# Create namespace
kubectl create namespace agentregistry

# Install chart
helm install agentregistry ./charts/agentregistry \
  -n agentregistry \
  --set image.repository=agentregistry-controller \
  --set image.tag=dev

# Verify pods are running
kubectl get pods -n agentregistry
```

### 1.4 (Alternative) Run Controller Locally

```bash
# Run controller against local cluster
./bin/controller --kubeconfig=$HOME/.kube/config
```

### 1.5 Port Forward API

```bash
# Forward HTTP API
kubectl port-forward -n agentregistry svc/agentregistry 8080:8080
```

### 1.6 Verify Installation

```bash
# Health check
curl http://localhost:8080/admin/v0/health

# Stats (should be empty)
curl http://localhost:8080/admin/v0/stats
```

**Expected:**
```json
{"status":"healthy"}
{"total_servers":0,"total_server_names":0,...}
```

---

## Phase 2: Import MCP Server

### 2.1 Create MCP Server from GitHub Repo

```bash
# Example: Filesystem MCP server
curl -X POST http://localhost:8080/admin/v0/servers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "github/modelcontextprotocol/filesystem",
    "version": "1.0.0",
    "title": "Filesystem MCP Server",
    "description": "MCP server for filesystem access",
    "repository": {
      "url": "https://github.com/modelcontextprotocol/servers",
      "source": "github"
    },
    "packages": [{
      "registryType": "npm",
      "identifier": "@modelcontextprotocol/server-filesystem",
      "version": "latest",
      "runtimeHint": "npx",
      "transport": {
        "type": "stdio"
      },
      "packageArguments": [{
        "name": "allowed_directories",
        "type": "string",
        "description": "Directories to allow access to",
        "required": true
      }]
    }]
  }'
```

### 2.2 Verify MCP Server Created

```bash
# List all servers (admin)
curl http://localhost:8080/admin/v0/servers | jq

# Check via kubectl
kubectl get mcpservercatalogs -A
```

### 2.3 Publish MCP Server

```bash
curl -X POST "http://localhost:8080/admin/v0/servers/github%2Fmodelcontextprotocol%2Ffilesystem/versions/1.0.0/publish"
```

### 2.4 Verify Published

```bash
# List published servers (public API)
curl http://localhost:8080/v0/servers | jq

# Check isLatest flag
kubectl get mcpservercatalog -A -o yaml | grep -A5 "status:"
```

---

## Phase 3: Import KAgent Agent

### 3.1 Create Agent Entry

```bash
# Example: ADK Base Agent
curl -X POST http://localhost:8080/admin/v0/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "kagent/adk-base",
    "version": "0.7.12",
    "title": "ADK Base Agent",
    "description": "Base agent using Google ADK framework",
    "image": "ghcr.io/kagent-dev/adk-base:0.7.12",
    "language": "python",
    "framework": "adk",
    "modelProvider": "google",
    "modelName": "gemini-2.0-flash",
    "repository": {
      "url": "https://github.com/kagent-dev/kagent",
      "source": "github"
    }
  }'
```

### 3.2 Verify Agent Created

```bash
# List agents
curl http://localhost:8080/admin/v0/agents | jq

# Check via kubectl
kubectl get agentcatalogs -A
```

### 3.3 Publish Agent

```bash
curl -X POST "http://localhost:8080/admin/v0/agents/kagent%2Fadk-base/versions/0.7.12/publish"
```

---

## Phase 4: Verify in UI

### 4.1 Start UI

```bash
cd ui
npm install
npm run dev
```

### 4.2 Configure API URL

Set `NEXT_PUBLIC_API_URL=http://localhost:8080` in `.env.local` or environment.

### 4.3 UI Verification Checklist

- [ ] Navigate to http://localhost:3000
- [ ] See MCP server "filesystem" in servers list
- [ ] See Agent "adk-base" in agents list
- [ ] Click on server to see details
- [ ] Click on agent to see details
- [ ] Verify version info, status, publish date shown

---

## Phase 5: Test Deployment

### 5.1 Create Deployment (MCP Server)

```bash
curl -X POST http://localhost:8080/admin/v0/deployments \
  -H "Content-Type: application/json" \
  -d '{
    "resourceName": "github/modelcontextprotocol/filesystem",
    "version": "1.0.0",
    "resourceType": "mcp",
    "runtime": "kubernetes",
    "namespace": "kagent",
    "config": {
      "ALLOWED_DIRECTORIES": "/data"
    }
  }'
```

### 5.2 Verify KMCP Resources Created

```bash
# Check RegistryDeployment
kubectl get registrydeployments -A

# Check KMCP MCPServer created
kubectl get mcpservers -n kagent
```

### 5.3 Create Deployment (Agent)

```bash
curl -X POST http://localhost:8080/admin/v0/deployments \
  -H "Content-Type: application/json" \
  -d '{
    "resourceName": "kagent/adk-base",
    "version": "0.7.12",
    "resourceType": "agent",
    "runtime": "kubernetes",
    "namespace": "kagent"
  }'
```

### 5.4 Verify KAgent Resources Created

```bash
# Check KAgent Agent created
kubectl get agents -n kagent
```

---

## Phase 6: Cleanup

```bash
# Delete deployments
kubectl delete registrydeployments --all -n agentregistry

# Uninstall Helm chart
helm uninstall agentregistry -n agentregistry

# Delete CRDs (optional)
kubectl delete crds mcpservercatalogs.agentregistry.dev \
  agentcatalogs.agentregistry.dev \
  skillcatalogs.agentregistry.dev \
  registrydeployments.agentregistry.dev

# Delete namespace
kubectl delete namespace agentregistry
```

---

## Quick Test Script

```bash
#!/bin/bash
set -e

API_URL="http://localhost:8080"

echo "=== Testing Agent Registry ==="

# Health check
echo "1. Health check..."
curl -s "$API_URL/admin/v0/health" | jq

# Create MCP server
echo "2. Creating MCP server..."
curl -s -X POST "$API_URL/admin/v0/servers" \
  -H "Content-Type: application/json" \
  -d '{"name":"test/mcp-server","version":"1.0.0","description":"Test server"}' | jq

# Publish
echo "3. Publishing..."
curl -s -X POST "$API_URL/admin/v0/servers/test%2Fmcp-server/versions/1.0.0/publish" | jq

# List published
echo "4. Listing published servers..."
curl -s "$API_URL/v0/servers" | jq

# Stats
echo "5. Stats..."
curl -s "$API_URL/admin/v0/stats" | jq

echo "=== Test Complete ==="
```
