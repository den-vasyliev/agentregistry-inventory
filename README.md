<div align="center">
  <picture>
    <img alt="agentregistry enterprise" src="./img/agentregistry-enterprise-logo.svg" height="180"/>
  </picture>

  [![Go Version](https://img.shields.io/badge/Go-1.25%2B-blue.svg)](https://golang.org/doc/install)
  [![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
  [![Test Coverage](https://img.shields.io/badge/coverage-24.7%25-yellow.svg)](https://github.com/agentregistry-dev/agentregistry)
  [![Discord](https://img.shields.io/discord/1435836734666707190?label=Join%20Discord&logo=discord&logoColor=white&color=5865F2)](https://discord.gg/HTYNjF2y2t)

  ### A Kubernetes-native registry to securely curate, discover, deploy, and manage agentic infrastructure including MCP servers, agents, skills, and models.
</div>


## 🏢 Enterprise Scale, Not Just CLI

In enterprise, you need **scale and multi-user access for AI resources**. Agent Registry runs as a **Kubernetes service** that provides a centralized catalog for teams to discover, deploy, and manage AI resources.

##  What is Agent Registry?

Agent Registry is a **Kubernetes controller** that brings governance and control to AI artifacts and infrastructure. It provides a secure, centralized registry where teams can publish, discover, and deploy AI artifacts using Kubernetes Custom Resource Definitions (CRDs).

### Agent Registry provides:

- **☸️ Kubernetes-Native**: Built on controller-runtime with CRD-based storage
- **📦 Centralized Catalog**: Discover and curate MCP servers, agents, skills, and models
- **🔒 Control and Governance**: Manage and control custom collections of artifacts
- **📊 Auto-Discovery**: Automatically index deployed resources into the catalog
- **🌐 Multi-Cluster Discovery**: Discover resources across multiple clusters with workload identity
- **🚀 Declarative Deployment**: GitOps-ready resource management
- **🌐 HTTP API + UI**: Browse and manage catalogs via REST API and web interface

### 🏢 Enterprise Benefits

- **📋 Centralized Inventory**: Complete catalog of all AI resources across the organization
- **👥 Developer Platform**: Teams can discover, reuse, and contribute new AI resources
- **✅ Review & Approval**: Built-in workflow for reviewing and approving resources before deployment
- **📈 Usage Analytics**: Track usage statistics and popularity rankings for AI resources
- **🔴 Real-Time Status**: Monitor health and status of deployed resources in real-time
- **🎯 Managed & Custom**: Support both managed (curated) and custom (team-owned) resources
- **🔐 OIDC Authentication**: Secure deployments with OIDC-based authentication
- **📝 Git-Based Source**: Resources defined in Git for version control and audit trails
- **🔍 Auto-Discovery**: Automatically discover deployed resources (gitless ops support)
- **☁️ Multi-Cluster Support**: Discover resources across dev, staging, and production clusters using workload identity

## 💼 Usage Scenarios

### A. Operator Workflow

Operators manage and deploy AI resources using GitOps principles:

<div align="center">
  <img src="./img/operator-scenario.png" alt="Operator Workflow" width="800"/>
</div>

### B. Developer Workflow

Developers discover and use AI resources from the catalog:

<div align="center">
  <img src="./img/dev-scenario.png" alt="Developer Workflow" width="800"/>
</div>

### C. GitOps Approval Workflow

Onboard new resources through a controlled GitOps process with review and approval:

1. **Fill the Form** - Click "Submit" in the UI and fill in resource details
2. **Generate Manifest** - UI generates the `.agentregistry.yaml` manifest
3. **Open PR** - Redirects to GitHub to create a PR with the manifest
4. **CI/CD Creates Resource** - After merge, pipeline submits the resource in `pending_review` state
5. **Review & Approve** - Team reviews and approves/rejects in the Inventory UI

This workflow ensures all resources go through proper review before being published to the catalog.

### D. Multi-Cluster Discovery

Automatically discover and catalog resources across multiple clusters:

1. **Configure DiscoveryConfig** - Define clusters, namespaces, and resource types to discover
2. **Workload Identity** - Use cloud provider workload identity for secure authentication
3. **Automatic Cataloging** - Resources are automatically added to the central catalog
4. **Cross-Environment Visibility** - View resources from dev, staging, and production in one place

See [Multi-Cluster Autodiscovery](docs/AUTODISCOVERY.md) for detailed configuration.

## 🏗️ Architecture

Agent Registry consists of:

1. **Kubernetes Controller** - Reconciles CRDs and manages deployments
2. **HTTP API Server** - REST API for catalog access (embedded in controller)
3. **Web UI** - Next.js interface for browsing catalogs
4. **CRD Storage** - Catalog entries stored as Kubernetes CRs

```
┌─────────────────────┐
│   Web UI (Next.js)  │
│   Port: 3000        │
└──────────┬──────────┘
           │ HTTP API
           ▼
┌─────────────────────┐
│   Controller        │
│   - HTTP API :8080  │
│   - 9 Reconcilers   │
└──────────┬──────────┘
           │ K8s API
           ▼
┌─────────────────────┐
│ Kubernetes CRDs     │
│ - MCPServerCatalog  │
│ - AgentCatalog      │
│ - SkillCatalog      │
│ - ModelCatalog      │
│ - RegistryDeployment│
│ - DiscoveryConfig   │
└─────────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- **Kubernetes cluster** (1.27+)
- **kubectl** configured to access your cluster
- **Go 1.25+** (for building from source)
- **Node.js 18+** (for UI development)

### Installation

```bash
# Clone the repository
git clone https://github.com/agentregistry-dev/agentregistry.git
cd agentregistry

# Install CRDs
kubectl apply -f api/v1alpha1/

# Deploy controller
kubectl apply -f charts/agentregistry/

# Verify deployment
kubectl get pods -n agentregistry
kubectl get mcpservercatalogs,agentcatalogs,skillcatalogs,modelcatalogs -A
```

### Building from Source

```bash
# Build controller binary (includes embedded UI)
make build

# Build container image locally
make image

# Build and push to registry
REGISTRY=ghcr.io/yourorg/agentregistry-enterprise make push

# Full release build (clean, test, lint, build, push)
make release
```

Available make targets:
- `make help` - Show all available targets
- `make build` - Build controller binary
- `make demo` - Run demo environment with envtest
- `make demo-stop` - Stop demo environment
- `make dev` - Run controller and UI in development mode
- `make dev-ui` - Run UI dev server only
- `make test` - Run tests with coverage
- `make lint` - Run linters
- `make image` - Build container image
- `make push` - Push to registry
- `make release` - Full release workflow

### Development Setup

```bash
# Build controller (includes UI)
make build

# Run in development mode
make dev

# Run UI dev server (in another terminal for live reload)
make dev-ui

# Access:
# - UI: http://localhost:3000
# - API: http://localhost:8080
# - Metrics: http://localhost:8081
# - Health: http://localhost:8082
```

See [`DEVELOPMENT.md`](DEVELOPMENT.md) for detailed development instructions.

### Demo Mode

Try Agent Registry locally without a Kubernetes cluster using the built-in demo environment:

```bash
# Start demo with envtest, sample data, and UI
make demo

# Access:
# - UI: http://localhost:3000
# - API: http://localhost:8080
# - Kubeconfig: ./demo-kubeconfig.yaml

# Stop demo
make demo-stop

# Use kubectl with demo cluster
kubectl --kubeconfig=./demo-kubeconfig.yaml get mcpservercatalog
```

The demo environment includes:
- **envtest** - Lightweight Kubernetes API server + etcd
- **Sample resources** - Pre-populated MCP servers, agents, skills, and models
- **No authentication** - Deploy and remove resources without sign-in
- **Isolated** - Separate from your real Kubernetes clusters

Perfect for:
- 🎯 Trying out Agent Registry features
- 👨‍💻 Local development without cluster access
- 🧪 Testing changes quickly
- 📚 Learning how it works

## 📚 Core Concepts

### Catalog CRDs

Define catalog entries using Kubernetes CRDs:

#### MCPServerCatalog

```yaml
apiVersion: agentregistry.dev/v1alpha1
kind: MCPServerCatalog
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

#### AgentCatalog

```yaml
apiVersion: agentregistry.dev/v1alpha1
kind: AgentCatalog
metadata:
  name: my-agent-v1-0-0
  namespace: agentregistry
spec:
  name: "my-agent"
  version: "1.0.0"
  title: "My Agent"
  description: "Custom AI agent"
  image: "ghcr.io/myorg/my-agent:1.0.0"
```

#### SkillCatalog

```yaml
apiVersion: agentregistry.dev/v1alpha1
kind: SkillCatalog
metadata:
  name: terraform-skill-v1-0-0
  namespace: agentregistry
spec:
  name: "terraform-skill"
  version: "1.0.0"
  title: "Terraform Management Skill"
  description: "Manage Terraform resources"
  image: "ghcr.io/skills/terraform:1.0.0"
```

#### ModelCatalog

```yaml
apiVersion: agentregistry.dev/v1alpha1
kind: ModelCatalog
metadata:
  name: gemini-2-0-flash
  namespace: agentregistry
spec:
  name: "gemini-2.0-flash"
  version: "1.0.0"
  title: "Gemini 2.0 Flash"
  description: "Google's fast multimodal AI model"
  provider: "google"
  modelType: "chat"
```

### Deployment

Deploy catalog entries to Kubernetes using **RegistryDeployment**:

```yaml
apiVersion: agentregistry.dev/v1alpha1
kind: RegistryDeployment
metadata:
  name: filesystem-deployment
  namespace: agentregistry
spec:
  resourceName: "filesystem"
  version: "1.0.0"
  resourceType: mcp      # mcp | agent | skill
  runtime: kubernetes
  namespace: default     # Target namespace for deployment
```

The controller automatically:
1. Looks up the catalog entry (MCPServerCatalog/AgentCatalog/SkillCatalog)
2. Creates the corresponding runtime CR (MCPServer/Agent)
3. Updates deployment status

### Auto-Discovery

The controller includes **discovery reconcilers** that automatically index deployed resources:

- **DiscoveryConfigReconciler** - Multi-cluster discovery with workload identity
- **MCPServerDiscovery** - Indexes deployed MCPServers
- **AgentDiscovery** - Indexes deployed Agents
- **SkillDiscovery** - Indexes skills referenced by Agents
- **ModelDiscovery** - Indexes ModelConfig resources

Resources deployed directly (without going through the catalog) are automatically discovered and cataloged.

#### Multi-Cluster Discovery

Use **DiscoveryConfig** to discover resources across multiple clusters:

```yaml
apiVersion: agentregistry.dev/v1alpha1
kind: DiscoveryConfig
metadata:
  name: multi-cluster-discovery
spec:
  environments:
    - name: dev
      cluster:
        name: dev-cluster
        projectId: my-gcp-project
        zone: us-central1
        useWorkloadIdentity: true
      provider: gcp
      discoveryEnabled: true
      namespaces:
        - default
        - ai-workloads
      resourceTypes:
        - MCPServer
        - Agent
        - ModelConfig
      labels:
        environment: dev
```

See [Multi-Cluster Autodiscovery](docs/AUTODISCOVERY.md) for complete documentation.

## 🔌 HTTP API

The controller exposes a REST API for catalog access:

### Public Endpoints (Read-Only)

```bash
# List all MCP servers
curl http://localhost:8080/v0/servers

# Get specific server
curl http://localhost:8080/v0/servers/filesystem/1.0.0

# List agents
curl http://localhost:8080/v0/agents

# List skills
curl http://localhost:8080/v0/skills

# List models
curl http://localhost:8080/v0/models
```

### Admin Endpoints (Management)

```bash
# Create/Update catalog entry
curl -X POST http://localhost:8080/admin/v0/servers \
  -H "Content-Type: application/json" \
  -d @server.json

# Deploy to runtime
curl -X POST http://localhost:8080/admin/v0/deploy \
  -H "Content-Type: application/json" \
  -d '{
    "resourceName": "filesystem",
    "version": "1.0.0",
    "resourceType": "mcp",
    "runtime": "kubernetes"
  }'
```

## 🎨 Web UI

Access the web interface at `http://localhost:3000` (in development) to:

- Browse MCP servers, agents, skills, and models
- View detailed metadata and health status
- Manage catalog entries
- Deploy to Kubernetes

## ☸️ Production Deployment

### Using Helm

```bash
# Install from Helm chart
helm install agentregistry ./charts/agentregistry \
  --namespace agentregistry \
  --create-namespace \
  --set controller.leaderElection=true \
  --set httpApi.enabled=true

# Expose API via ingress
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: agentregistry-api
  namespace: agentregistry
spec:
  rules:
  - host: registry.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: agentregistry
            port:
              number: 8080
EOF
```

### Configuration Options

Key `values.yaml` settings:

```yaml
controller:
  leaderElection: true        # Enable for HA
  replicas: 2                 # High availability
  logLevel: info             # debug | info | warn | error

httpApi:
  enabled: true              # Enable HTTP API server
  port: 8080

ui:
  enabled: true              # Deploy web UI
  replicas: 2

crds:
  install: true              # Install CRDs
```

## 🧪 Testing

```bash
# Run all tests with coverage
make test

# Run linters
make lint

# Format code
make fmt
```

Current test coverage: **24.7%** (focused on controller logic and runtime translation)

## 🤝 Contributing

We welcome contributions! Please see [`CONTRIBUTING.md`](CONTRIBUTING.md) for guidelines.

### Development Workflow

```bash
# 1. Make changes
git checkout -b feature/my-feature

# 2. Test locally
make dev

# 3. Run tests
make test

# 4. Build
make build

# 5. Commit and push
git commit -m "feat: add my feature"
git push origin feature/my-feature
```

## 🔗 Related Projects

- [Model Context Protocol](https://modelcontextprotocol.io/) - The protocol for AI-agent tool integration
- [kagent](https://github.com/kagent-dev/kagent) - Kubernetes-native AI agent runtime
- [kmcp](https://github.com/kagent-dev/kmcp) - MCP server operator for Kubernetes
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) - Official MCP SDK

## 📚 Resources

- 📖 [Development Guide](DEVELOPMENT.md)
- 💬 [GitHub Discussions](https://github.com/agentregistry-dev/agentregistry/discussions)
- 🐛 [Issue Tracker](https://github.com/agentregistry-dev/agentregistry/issues)
- 💬 [Discord Server](https://discord.gg/HTYNjF2y2t)

## 📄 License

MIT License - see [`LICENSE`](LICENSE) for details.

---

**Built with ❤️ for the AI agent ecosystem**
