<div align="center">
  <picture>
    <img alt="Agent Inventory" src="docs/img/arlogo.png" height="90"/>
  </picture>

  <h2>The Control Plane for AI Infrastructure</h2>

  <p>
    <h3>Kubernetes-native registry for MCP servers, agents, skills & models</h3>
  </p>

  <p>
    <a href="https://golang.org/doc/install"><img src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go 1.25+"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License: MIT"></a>
    <a href="https://github.com/den-vasyliev/agentregistry-inventory"><img src="https://img.shields.io/badge/coverage-40.2%25-yellow?style=flat-square" alt="Coverage"></a>
    <a href="https://discord.gg/HTYNjF2y2t"><img src="https://img.shields.io/discord/1435836734666707190?style=flat-square&label=Discord&logo=discord&logoColor=white&color=5865F2" alt="Discord"></a>
  </p>

  <p>
    <a href="#quick-start">🚀 Quick Start</a> •
    <a href="#features">✨ Features</a> •
    <a href="#architecture">🏗️ Architecture</a> •
    <a href="#docs">📚 Docs</a> •
    <a href="https://medium.com/@den.vasyliev/your-ai-infrastructure-is-sprawling-you-just-dont-know-it-yet-e5c85d32060a">📝 Blog</a>
  </p>

  <p><a href="https://medium.com/@den.vasyliev/your-ai-infrastructure-is-sprawling-you-just-dont-know-it-yet-e5c85d32060a"><img src="https://img.shields.io/badge/Medium-Read%20the%20story-black?style=flat-square&logo=medium&logoColor=white" alt="Read on Medium"></a>  <a href="https://youtu.be/sCut0CEHRr0">
    <img src="https://img.shields.io/badge/▶_Watch_Demo-red?style=for-the-badge&logo=youtube&logoColor=white" alt="Watch Demo on YouTube"/>
  </a>
  </p>

</div>

<a id="features"></a>

## ✨ What You Get
<p><h3>Automatically indexes MCP servers, agents, skills, and models across clusters.

If it's running, it's in the catalog.</h3>
</p>

> **No CLI needed.** Agent Registry ships an [MCP server](#-mcp-server) — connect it to Claude Code, Cursor, or any MCP-compatible client and manage your registry conversationally.
<center><img src="docs/img/inventory.png"></center>

| Capability | What It Means |
|------------|---------------|
| 🔍 **Auto-Discovery** | Scans your clusters for AI workloads — MCP servers, agents, skills, models — and catalogs them automatically. Zero manual work. |
| 📦 **Unified Inventory** | Everything in one place across dev, staging, prod. Git as the single source of truth. |
| ✍️ **Create & Publish** | Generate manifests via UI/API, submit for review, open PRs — or deploy directly. |
| 🚀 **One-Click Deploy** | Deploy from catalog to any environment. Controller handles the lifecycle. |
| 🔒 **GitOps Native** | GitOps and Gitless Ops workflows built-in. |
| 🌐 **Multi-Cluster** | Discover and deploy across clusters with workload identity. |


<a id="quick-start"></a>

## 🚀 Quick Start

### One command to run dev environment

```bash
git clone https://github.com/den-vasyliev/agentregistry-inventory.git
cd agentregistry-inventory && make dev
```

> 🎯 **That's it.** UI opens at http://localhost:3000 with sample data pre-loaded.
>
> No Kubernetes cluster needed — uses envtest (embedded etcd + kube-apiserver).

**☸️ Have a cluster?**

```bash
kubectl apply -k https://github.com/den-vasyliev/agentregistry-inventory/config/crd
helm install agentregistry-inventory ./charts/agentregistry -n agentregistry --create-namespace
```

---

## 🌟 Why Agent Inventory?

| Without Agent Inventory | With Agent Inventory |
|------------------------|---------------------|
| 😵 Sprawl of AI tools across clusters | 📦 Single source of truth |
| 🔍 Manual discovery of MCP servers | 🤖 Auto-discovery & cataloging |
| 😰 No version control for AI configs | 📝 GitOps-native workflows |
| 🤷 "What agents are running in prod?" | 📊 Real-time inventory & status |
| 😱 Direct K8s yaml edits | 🚀 One-click deploy from UI/API |

<a id="architecture"></a>

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      WEB UI (Next.js)                       │
│              embedded in controller at :8080                │
└───────────────────────────┬─────────────────────────────────┘
                            │ REST
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    CONTROLLER (Go Controller Runtime)       │
│  ┌─────────────┐ ┌────────────────┐ ┌─────────────────────┐ │
│  │  HTTP API   │ │  Reconcilers   │ │  Auto-Discovery     │ │
│  │   :8080     │ │                │ │                     │ │
│  └─────────────┘ └────────────────┘ └─────────────────────┘ │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐    │
│  │  MCP Server │ │   Metrics   │ │   Health            │    │
│  │   :8083     │ │   :8081     │ │   :8082             │    │
│  └─────────────┘ └─────────────┘ └─────────────────────┘    │
└───────────────────────────┬─────────────────────────────────┘
                            │ K8s API
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                     CRDs (etcd)                             │
│  ┌─────────────────┐ ┌─────────────────┐ ┌──────────────┐   │
│  │ MCPServerCatalog│ │  AgentCatalog   │ │ SkillCatalog │   │
│  └─────────────────┘ └─────────────────┘ └──────────────┘   │
│  ┌────────────────────┐ ┌─────────────────┐                 │
│  │RegistryDeployment  | │ DiscoveryConfig │                 │
│  └────────────────────┘ └─────────────────┘                 │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│              RUNTIME (Kagent + KMCP +...)                   │
│         Agent ↔ MCP Server ↔ Model ↔ Skills                 │
└─────────────────────────────────────────────────────────────┘
```

<a id="docs"></a>

## 📚 CRD Reference
- [Kgateway](https://kgateway.dev) — Gateway API for AI traffic
- [Kagent](https://github.com/kagent-dev/kagent) — Kubernetes AI agent runtime
- [KMCP](https://github.com/kagent-dev/kmcp) — MCP server operator

---

## 📋 Inventory CRD Examples

### 📦 Publish a Catalog Entry

```yaml
apiVersion: agentregistry.dev/v1alpha1
kind: MCPServerCatalog          # Also: AgentCatalog, SkillCatalog, ModelCatalog
metadata:
  name: filesystem-v1-0-0
  namespace: agentregistry
spec:
  name: "filesystem"
  version: "1.0.0"
  title: "Filesystem MCP Server"
  description: "Provides file system access tools"
  websiteUrl: "https://github.com/modelcontextprotocol/servers"
  repository:
    url: "https://github.com/modelcontextprotocol/servers"
    source: github
  packages:
    - registryType: npm
      identifier: "@modelcontextprotocol/server-filesystem"
      version: "0.6.1"
      transport:
        type: stdio
  remotes:                        # Optional: streamable-http endpoints
    - type: streamable-http
      url: "https://mcp.example.com/filesystem"
```

### 🚀 Deploy to Runtime

```yaml
apiVersion: agentregistry.dev/v1alpha1
kind: RegistryDeployment
metadata:
  name: filesystem-deployment
  namespace: agentregistry
spec:
  resourceName: "filesystem"
  version: "1.0.0"
  resourceType: mcp             # mcp | agent
  runtime: kubernetes           # Required: deployment runtime
  namespace: default            # Target namespace
  preferRemote: false           # Use local package vs remote endpoint
  environment: ""               # Target environment (from DiscoveryConfig), empty = local cluster
  config:                       # Optional: deployment configuration
    LOG_LEVEL: "info"
```

The controller reconciles this → creates MCPServer/Agent CRs → tracks status.

### 🌍 Multi-Cluster Discovery

```yaml
apiVersion: agentregistry.dev/v1alpha1
kind: DiscoveryConfig
metadata:
  name: multi-cluster-discovery
  namespace: agentregistry
spec:
  environments:
    - name: production
      cluster:
        name: prod-gke
        projectId: my-gcp-project
        zone: us-central1
        useWorkloadIdentity: true
      provider: gcp
      discoveryEnabled: true
      deployEnabled: false
      namespaces: [ai-workloads, agents]
      resourceTypes: [MCPServer, Agent, ModelConfig]
```

[→ Full Autodiscovery Docs](docs/AUTODISCOVERY.md)

---

## 🔌 API Reference

### Public API (Read-Only, no auth required)

```bash
curl http://localhost:8080/v0/servers
curl http://localhost:8080/v0/agents
curl http://localhost:8080/v0/skills
```

### Admin API (Write)

```bash
# Auth disabled by default — just POST
curl -X POST http://localhost:8080/admin/v0/servers \
  -H "Content-Type: application/json" \
  -d @server.json

# With auth enabled (AGENTREGISTRY_AUTH_ENABLED=true)
curl -X POST http://localhost:8080/admin/v0/servers \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d @server.json
```

---

## 🤖 MCP Server

The controller embeds an MCP server on `:8083`. Connect any MCP-compatible client (Claude Code, Cursor, etc.) to browse, deploy, and manage registry resources conversationally.

### Connect (auth disabled, default)

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

### Connect (auth enabled)

```json
{
  "mcpServers": {
    "agentregistry": {
      "type": "streamable-http",
      "url": "http://localhost:8083/mcp",
      "headers": {
        "Authorization": "Bearer your-token"
      }
    }
  }
}
```

### Tools

| Tool | Description |
|------|-------------|
| `list_catalog` | List catalog entries (servers/agents/skills/models) |
| `get_catalog` | Get entry details |
| `get_registry_stats` | Counts of all resource types |
| `list_deployments` | List active deployments |
| `get_deployment` | Deployment details by name |
| `deploy_catalog_item` | Deploy a catalog item to Kubernetes |
| `delete_deployment` | Remove a deployment |
| `update_deployment_config` | Update deployment config |
| `list_environments` | Discovered environments from DiscoveryConfig |
| `get_discovery_map` | Cluster topology and resource counts |
| `trigger_discovery` | Force re-scan of discovery |
| `recommend_servers` | AI-powered server recommendations |
| `analyze_agent_dependencies` | AI-powered dependency analysis |
| `generate_deployment_plan` | AI-powered deployment planning |

[→ Full MCP Docs](docs/MCP_SERVER.md) | [→ MCP Auth](docs/mcp-auth.md)

---

## ☸️ Production Deployment

```bash
helm install agentregistry ./charts/agentregistry \
  --namespace agentregistry \
  --create-namespace \
  --set replicaCount=2 \
  --set controller.leaderElection=true
```

### Key Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `replicaCount` | `1` | Set to 2+ for HA |
| `controller.leaderElection` | `false` | Required for multi-replica |
| `controller.logLevel` | `info` | Use `debug` for troubleshooting |
| `httpApi.serviceType` | `ClusterIP` | Use `LoadBalancer` for external access |
| `disableAuth` | `true` | Set to `false` to enable Bearer token auth |

---

## 🔐 Authentication

Auth is **disabled by default**. When enabled (`disableAuth: false`), the admin API and MCP server require a Bearer token from the `agentregistry-api-tokens` Kubernetes Secret.

For production deployments with a gateway, use [agentgateway](charts/agentgateway/) to handle Azure AD JWT validation in front of the controller — keep `disableAuth: true` on the backend and let the gateway enforce auth.

### Enable Bearer Token Auth

```bash
# Create token secret
kubectl create secret generic agentregistry-api-tokens \
  -n agentregistry \
  --from-literal=admin-token=$(openssl rand -hex 32)

# Install with auth enabled
helm install agentregistry ./charts/agentregistry \
  --namespace agentregistry \
  --set disableAuth=false
```

### Azure AD (MSAL Browser PKCE)

The embedded UI supports Azure AD login via MSAL.js — no client secret required.

```yaml
# charts/agentregistry/values.yaml
azure:
  tenantId: "your-tenant-id"
  clientId: "your-client-id"
```

[→ Full Azure AD Setup](docs/azure-ad-setup.md)

---

## 🧪 Development

```bash
make dev          # Full stack: controller + UI dev server
make run          # Controller only with embedded UI at :8080
make test         # Run test suite
make lint         # gofmt + go vet
make build        # Build UI + controller binary
make generate     # Regenerate CRD manifests + deepcopy
```

---

## 🔗 Ecosystem

<div align="center">

| Project | Description | Link |
|---------|-------------|------|
| **Agentgateway** | MCP/HTTP auth gateway | [agentgateway.dev](https://agentgateway.dev/) |
| **MCP** | Model Context Protocol specification | [modelcontextprotocol.io](https://modelcontextprotocol.io) |
| **Kagent** | Kubernetes AI agent runtime | [github.com/kagent-dev/kagent](https://github.com/kagent-dev/kagent) |
| **KMCP** | MCP server operator for Kubernetes | [github.com/kagent-dev/kmcp](https://github.com/kagent-dev/kmcp) |
| **MCP Go SDK** | Official Go SDK | [github.com/modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk) |

</div>

---

## 💬 Join the Community

- 💬 [Discord](https://discord.gg/HTYNjF2y2t) — Chat with the team
- 🐛 [Issues](https://github.com/den-vasyliev/agentregistry-inventory/issues) — Report bugs or request features
- 🤝 [PRs Welcome](CONTRIBUTING.md) — We love contributions!

---

<div align="center">

**[⬆ Back to Top](#the-control-plane-for-ai-infrastructure)**

Made with ❤️ by the Agent Inventory team

[MIT License](LICENSE)

</div>
