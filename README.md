<div align="center">
  <picture>
    <img alt="Agent Inventory" src="./img/arlogo.png" height="90"/>
  </picture>

  <h3>The Control Plane for AI Infrastructure</h3>
  
  <p>
    <strong>Kubernetes-native registry for MCP servers, agents, skills & models</strong>
  </p>

  <p>
    <a href="https://golang.org/doc/install"><img src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go 1.25+"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License: MIT"></a>
    <a href="https://github.com/agentregistry-dev/agentregistry"><img src="https://img.shields.io/badge/coverage-24.7%25-yellow?style=flat-square" alt="Coverage"></a>
    <a href="https://discord.gg/HTYNjF2y2t"><img src="https://img.shields.io/discord/1435836734666707190?style=flat-square&label=Discord&logo=discord&logoColor=white&color=5865F2" alt="Discord"></a>
  </p>

  <p>
    <a href="#-quick-start">🚀 Quick Start</a> •
    <a href="#-features">✨ Features</a> •
    <a href="#-architecture">🏗️ Architecture</a> •
    <a href="#-documentation">📚 Docs</a>
  </p>
</div>

---

## 🚀 Quick Start

### One command to rule them all

```bash
git clone https://github.com/den-vasyliev/agentregistry-inventory.git
cd agentregistry && make dev
```

> 🎯 **That's it.** UI opens at http://localhost:3000 with sample data pre-loaded.
> 
> No Kubernetes cluster needed — uses envtest (embedded etcd + kube-apiserver).

**☸️ Have a cluster?**

```bash
kubectl apply -f https://raw.githubusercontent.com/agentregistry-dev/agentregistry/main/config/crd/
helm install agentregistry ./charts/agentregistry -n agentregistry --create-namespace
```

---

## ✨ What You Get

```
Discover → Inventory → Deploy → Monitor
     ↑_________________________________________↓
              (Auto-discovery loop)
```

| Capability | What It Means |
|------------|---------------|
| 🔍 **Auto-Discovery** | Scans your clusters for AI workloads — MCP servers, agents, skills, models — and catalogs them automatically. Zero manual work. |
| 📦 **Unified Inventory** | Everything in one place across dev, staging, prod. Git as the single source of truth. |
| ✍️ **Create & Publish** | Generate manifests via UI/API, submit for review, open PRs — or deploy directly. |
| 🚀 **One-Click Deploy** | Deploy from catalog to any environment. Controller handles the lifecycle. |
| 🔒 **GitOps Native** | GitOps and Gitless Ops workflows built-in. |
| 🌐 **Multi-Cluster** | Discover and deploy across clusters with workload identity. |



---

## 🌟 Why Agent Inventory?

| Without Agent Inventory | With Agent Inventory |
|------------------------|---------------------|
| 😵 Sprawl of AI tools across clusters | 📦 Single source of truth |
| 🔍 Manual discovery of MCP servers | 🤖 Auto-discovery & cataloging |
| 😰 No version control for AI configs | 📝 GitOps-native workflows |
| 🤷 "What agents are running in prod?" | 📊 Real-time inventory & status |
| 😱 Direct K8s yaml edits | 🚀 One-click deploy from UI/API |

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      WEB UI (Next.js)                       │
│                    http://localhost:3000                    │
└───────────────────────────┬─────────────────────────────────┘
                            │ REST
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    CONTROLLER (Go Controller Runtime)       │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐    │
│  │  HTTP API   │ │ 9 Reconcilers│ │  Auto-Discovery    │    │
│  │   :8080     │ │              │ │                    │    │
│  └─────────────┘ └─────────────┘ └─────────────────────┘    │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐    │
│  │  Metrics    │ │   Health    │ │   Leader Election   │    │
│  │   :8081     │ │   :8082     │ │                     │    │
│  └─────────────┘ └─────────────┘ └─────────────────────┘    │
└───────────────────────────┬─────────────────────────────────┘
                            │ K8s API
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                     CRDs (etcd)                             │
│  ┌─────────────────┐ ┌─────────────────┐ ┌──────────────┐   │
│  │ MCPServerCatalog│ │  AgentCatalog   │ │ SkillCatalog │   │
│  └─────────────────┘ └─────────────────┘ └──────────────┘   │
│  ┌─────────────────┐ ┌─────────────────┐                    │
│  │RegistryDeployment│ │ DiscoveryConfig │                   │
│  └─────────────────┘ └─────────────────┘                    │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│              RUNTIME (Kagent + KMCP +...)                   │
│         Agent ↔ MCP Server ↔ Model ↔ Skills                 │
└─────────────────────────────────────────────────────────────┘
```

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
  packages:
    - registryType: npm
      identifier: "@modelcontextprotocol/server-filesystem"
      version: "0.6.1"
      transport:
        type: stdio
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
  resourceType: mcp             # mcp | agent | skill
  namespace: default            # Target namespace
  preferRemote: false           # Use local package vs remote endpoint
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
      namespaces: [ai-workloads, agents]
      resourceTypes: [MCPServer, Agent, ModelConfig]
      labels:
        environment: production
        tier: critical
```

[→ Full Autodiscovery Docs](docs/AUTODISCOVERY.md)

---

## 🔌 API Reference

### Public API (Read-Only)

```bash
# Browse catalog
curl http://localhost:8080/v0/servers
curl http://localhost:8080/v0/servers/filesystem/1.0.0

# Search agents & skills
curl http://localhost:8080/v0/agents?framework=langchain
curl http://localhost:8080/v0/skills?category=code-generation
```

### Admin API (Write)

```bash
# Create catalog entry
curl -X POST http://localhost:8080/admin/v0/servers \
  -H "Content-Type: application/json" \
  -d @server.json

# Deploy to cluster
curl -X POST http://localhost:8080/admin/v0/deploy \
  -H "Content-Type: application/json" \
  -d '{
    "resourceName": "filesystem",
    "version": "1.0.0",
    "resourceType": "mcp",
    "namespace": "default"
  }'
```

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

| Setting | Default | When to Change |
|---------|---------|----------------|
| `replicaCount` | 1 | Set to 2+ for HA |
| `controller.leaderElection` | true | Required for multi-replica |
| `controller.logLevel` | info | Use `debug` for troubleshooting |
| `httpApi.serviceType` | ClusterIP | Use `LoadBalancer` for external access |

---

## 🔐 Authentication (OIDC)

Agent Inventory supports OIDC-based authentication with group-based authorization.

### Quick Setup

```bash
# Install with OIDC enabled
helm install agentregistry ./charts/agentregistry \
  --namespace agentregistry \
  --create-namespace \
  --set oidc.enabled=true \
  --set oidc.issuer=https://your-oidc-provider.com \
  --set oidc.audience=agentregistry \
  --set oidc.adminGroup=agentregistry-admins
```

### Configuration

| Setting | Description | Required |
|---------|-------------|----------|
| `oidc.enabled` | Enable OIDC authentication | Yes |
| `oidc.issuer` | OIDC provider URL (e.g., `https://keycloak.example.com/realms/myrealm`) | Yes |
| `oidc.audience` | Expected JWT audience (client ID) | Yes |
| `oidc.adminGroup` | Group required for deployment operations | No |
| `oidc.groupClaim` | Claim name containing user groups (default: `groups`) | No |

### Supported Providers

- **Generic OIDC** (Keycloak, Auth0, Okta)
- **Google OAuth** (backward compatible)
- **Azure AD** (backward compatible)

[→ Full OIDC Setup Guide](docs/oidc-setup.md)

---

## 🧪 Development

```bash
make dev          # Full stack: controller + UI + sample data
make test         # Run test suite with coverage
make lint         # gofmt + go vet + eslint
make build        # Build controller binary
make image        # Build container image (KO)
```

[→ Development Guide](DEVELOPMENT.md) | [→ Contributing](CONTRIBUTING.md)


---

## 🔗 Ecosystem

<div align="center">

| Project | Description | Link |
|---------|-------------|------|
| **KGateway** | AI Gateway | [kgateway.dev](https://kgateway.dev/) |
| **MCP** | Model Context Protocol specification | [modelcontextprotocol.io](https://modelcontextprotocol.io) |
| **Kagent** | Kubernetes AI agent runtime | [github.com/kagent-dev/kagent](https://github.com/kagent-dev/kagent) |
| **KMCP** | MCP server operator for Kubernetes | [github.com/kagent-dev/kmcp](https://github.com/kagent-dev/kmcp) |
| **MCP Go SDK** | Official Go SDK | [github.com/modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk) |

</div>

---

## 💬 Join the Community

- 💬 [Discord](https://discord.gg/HTYNjF2y2t) — Chat with the team
- 🐛 [Issues](https://github.com/agentregistry-dev/agentregistry/issues) — Report bugs or request features
- 🤝 [PRs Welcome](CONTRIBUTING.md) — We love contributions!

---

<div align="center">

**[⬆ Back to Top](#-the-control-plane-for-ai-infrastructure)**

Made with ❤️ by the Agent Inventory team

[MIT License](LICENSE)

</div>
