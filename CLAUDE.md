# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/claude-code) when working with code in this repository.

## Git Workflow

**CRITICAL: Pre-commit checklist - ALWAYS run these commands before EVERY commit:**

1. `make fmt` - Format all code (gofmt, prettier)
2. `make lint` - Check for linting issues
3. `make test` - Run all tests

If ANY of these fail, fix the issues before committing. NEVER commit or push code that fails formatting, linting, or tests.

**CRITICAL: NEVER push before user tests!** After committing, WAIT for the user to test and explicitly ask to push.

**IMPORTANT: Always use this command to push changes (when user approves):**

```bash
git push origin HEAD:enterprise-controller
```

This repository uses `enterprise-controller` as the working branch, not `main`.

**IMPORTANT: Commit Messages**

Do NOT add the following line to commit messages:
```
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

This project does not use co-author attribution in commits.

## Project Overview

Agent Registry is a **Kubernetes-native controller** that provides a centralized registry to securely curate, discover, deploy, and manage agentic infrastructure including MCP servers, agents, and skills.

### Key Features
- **CRD-based storage** - Catalog entries stored as Kubernetes Custom Resources
- **Auto-discovery** - Automatically indexes deployed resources
- **Multi-cluster discovery** - Discover resources across multiple clusters with workload identity
- **HTTP API** - REST API embedded in controller for catalog access
- **Web UI** - Next.js interface for browsing and managing catalogs
- **Declarative deployment** - GitOps-ready resource management

## Architecture

The project consists of two main components:

1. **Controller** - Kubernetes controller with embedded HTTP API
2. **UI** - Next.js web interface

### Namespace Structure

**IMPORTANT**: Agent Registry uses `agentregistry` as the default namespace for all its resources:

- **agentregistry** namespace - Contains all Agent Registry resources:
  - `MCPServerCatalog`, `AgentCatalog`, `SkillCatalog`, `ModelCatalog` - Catalog entries
  - `RegistryDeployment` - Deployment manifests
  - `DiscoveryConfig` - Discovery configuration (namespace-scoped)

- **Environment namespaces** (dev, staging, prod, etc.) - Contain actual deployed resources:
  - `MCPServer` (kmcp) - Deployed MCP servers
  - `Agent` (kagent) - Deployed agents
  - `ModelConfig` (kagent) - Model configurations

The DiscoveryConfig watches environment namespaces and creates catalog entries in the `agentregistry` namespace.

### Directory Structure

```
agentregistry/
├── cmd/
│   └── controller/          # Controller entry point (ONLY binary)
├── internal/
│   ├── controller/          # 9 K8s reconcilers (including DiscoveryConfig)
│   ├── httpapi/            # HTTP API server (embedded in controller)
│   ├── mcp/                # MCP server implementation
│   ├── runtime/            # Runtime deployment logic
│   ├── utils/              # Utilities
│   └── version/            # Version info
├── pkg/
│   └── models/             # Shared data models
├── ui/                     # Next.js frontend
├── api/                    # CRD definitions (v1alpha1)
├── charts/                 # Helm charts
└── docker/
    └── controller.Dockerfile
```

### What Was Removed

This project **NO LONGER HAS**:
- ❌ CLI (`arctl` was removed)
- ❌ Standalone server binary
- ❌ Database (SQLite/PostgreSQL)
- ❌ Local runtime mode

Everything is **Kubernetes-native** now.

## Components

### Controller (`cmd/controller/main.go`)

The controller is a **single binary** that runs:

1. **9 Kubernetes Reconcilers:**
   - `MCPServerCatalogReconciler` - Manages MCP server catalog entries
   - `AgentCatalogReconciler` - Manages agent catalog entries
   - `SkillCatalogReconciler` - Manages skill catalog entries
   - `RegistryDeploymentReconciler` - Deploys catalog items to K8s
   - `MCPServerDiscoveryReconciler` - Auto-discovers deployed MCPServers
   - `AgentDiscoveryReconciler` - Auto-discovers deployed Agents
   - `SkillDiscoveryReconciler` - Auto-discovers skills from agents
   - `ModelDiscoveryReconciler` - Auto-discovers ModelConfig resources
   - `DiscoveryConfigReconciler` - Multi-cluster discovery with workload identity (see [docs/AUTODISCOVERY.md](docs/AUTODISCOVERY.md))

2. **HTTP API Server** (embedded, port 8080):
   - Public endpoints: `/v0/servers`, `/v0/agents`, `/v0/skills`, `/v0/models` (read-only)
   - Admin endpoints: `/admin/v0/*` for management (requires auth)
   - **Authentication enabled by default** - tokens read from Secret `agentregistry-api-tokens`

3. **Metrics & Health:**
   - Metrics: `:8081`
   - Health probe: `:8082`

### UI (`ui/`)

- **Next.js 14** with TypeScript
- **Tailwind CSS** + **shadcn/ui** components
- Connects to controller API at `localhost:8080`
- **NextAuth.js** for authentication (optional, can be bypassed for dev)

### Data Storage

**Kubernetes CRDs** (no database):
```
agentregistry.dev/v1alpha1:
- MCPServerCatalog
- AgentCatalog
- SkillCatalog
- RegistryDeployment
- DiscoveryConfig (for multi-cluster autodiscovery)

kagent.dev/v1alpha2:
- Agent
- RemoteMCPServer
- ModelConfig

kmcp.agentregistry.dev/v1alpha1:
- MCPServer
```

## Build Commands

```bash
# Build UI (static export)
make build-ui

# Build controller binary only
make build-controller

# Build both UI and controller
make build

# Run in development mode
make dev          # Controller only
make dev-ui       # UI dev server only

# Run tests
make test

# Lint code
make lint

# Format code
make fmt

# Build and push container image
make ko-controller

# Build container image locally
make image
```

## Development Workflow

### Running Locally

```bash
# Single command - starts both controller and UI
make dev
```

This starts:
- Controller with HTTP API on **:8080**
- UI on **:3000**

Open http://localhost:3000 in browser.

### Running Separately

**Terminal 1:**
```bash
make dev-controller
```

**Terminal 2:**
```bash
make dev-ui
```

### Environment Configuration

For development, `ui/.env.local` is used:
```bash
# Bypass authentication for development
AUTH_SECRET=dev-secret-bypass-only-not-for-production
NEXT_PUBLIC_API_URL=http://localhost:8080
```

## Code Style

- **Go**: Use `gofmt` and `golangci-lint` (config in `.golangci.yaml`)
- **TypeScript/React**: Follow Next.js and React best practices
- **Commits**: Follow conventional commits (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`)

## Key Dependencies

### Go
- **controller-runtime**: `sigs.k8s.io/controller-runtime` - K8s controller framework
- **client-go**: `k8s.io/client-go` - Kubernetes client
- **Huma**: `github.com/danielgtaylor/huma/v2` - HTTP API framework
- **zerolog**: `github.com/rs/zerolog` - Structured logging
- **kagent**: `github.com/kagent-dev/kagent/go` - Agent CRDs
- **kmcp**: `github.com/kagent-dev/kmcp` - MCP server operator

### UI
- **Next.js 14**: React framework
- **NextAuth.js**: Authentication (optional)
- **Tailwind CSS**: Styling
- **shadcn/ui**: Component library

## Important Patterns

### Controller Patterns

1. **CRD-based storage** - All data stored as Kubernetes CRs
2. **Discovery reconcilers** - Auto-index deployed resources
3. **Embedded HTTP API** - API server runs inside controller
4. **Authentication enabled by default** - Set `AGENTREGISTRY_DISABLE_AUTH=true` to disable for development

### API Patterns

- **Public endpoints** (`/v0/*`) - Read-only catalog access
- **Admin endpoints** (`/admin/v0/*`) - Management operations
- **Huma framework** - OpenAPI docs auto-generated

### UI Patterns

- **Client-side rendering** - Next.js with `"use client"`
- **API calls** - Direct to controller at `localhost:8080`
- **Optional auth** - NextAuth can be bypassed for development

## Testing

### Test Coverage: 22%

Coverage breakdown:
- `internal/controller`: 17.0%
- `internal/runtime`: 4.4%
- `internal/runtime/translation/kagent`: 81.7%

### What's Tested
- ✅ Controller helper functions (name generation, version comparison)
- ✅ Runtime translation to K8s resources
- ✅ Basic reconciler logic
- ✅ DiscoveryConfig multi-namespace and multi-environment discovery

### What Needs Tests
- ❌ HTTP API handlers (0% coverage)
- ❌ Most controller reconcilers
- ❌ MCP server implementation

### Testing Best Practices

**IMPORTANT: Use envtest, NOT fake clients!**

All controller tests MUST use the existing `SetupTestEnv` helper from `testhelper_test.go`:

```go
func TestYourReconciler(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Use envtest helper (NOT fake.NewClientBuilder!)
    helper := SetupTestEnv(t, 60*time.Second, false)
    defer helper.Cleanup(t)

    // Use helper.Client for all operations
    err := helper.Client.Create(ctx, obj)

    // Create reconciler with real client
    reconciler := &YourReconciler{
        Client: helper.Client,
        Scheme: helper.Scheme,
        Logger: logger,
    }
}
```

**Why envtest?**
- ✅ Uses real etcd + kube-apiserver (same as production)
- ✅ Tests with actual Kubernetes behavior
- ✅ Validates CRD schemas and webhooks
- ✅ Proper field indexing and caching
- ❌ Fake clients miss critical K8s behavior

## Common Tasks

### Adding a New CRD Field

1. Update `api/v1alpha1/*_types.go`
2. Run `make generate` to generate deepcopy methods and CRD manifests
3. Update controller reconciler in `internal/controller/`
4. Update API handlers in `internal/httpapi/handlers/`
5. Update UI components in `ui/components/`

### Adding a New API Endpoint

1. Create handler in `internal/httpapi/handlers/`
2. Register route in handler's `RegisterRoutes()` method
3. Add to Huma API in `internal/httpapi/server.go`

### Adding a New Reconciler

1. Create reconciler in `internal/controller/`
2. Implement `Reconcile()` method
3. Add `SetupWithManager()` method
4. Register in `cmd/controller/main.go`
5. Add tests in `internal/controller/*_test.go`

## Environment

- **Go**: 1.25+
- **Node.js**: 18+
- **Kubernetes**: 1.27+
- **Docker**: For container builds (optional)

## Multi-Cluster Discovery

The project supports **autodiscovery across multiple clusters** using workload identity:

### Quick Start

1. **Create DiscoveryConfig:**
   ```bash
   kubectl apply -f config/samples/discoveryconfig_example.yaml
   ```

2. **Check status:**
   ```bash
   kubectl get discoveryconfig multi-cluster-discovery -o yaml
   ```

3. **View discovered resources:**
   ```bash
   kubectl get mcpservercatalog -l agentregistry.dev/discovered=true
   ```

See [docs/AUTODISCOVERY.md](docs/AUTODISCOVERY.md) for complete documentation.

## Notes

- **No CLI** - All management via K8s CRs or HTTP API
- **No database** - CRDs are the source of truth
- **Controller-only** - Single binary for all functionality
- **UI is separate** - Runs independently, connects via HTTP

IMPORTANT: This context may or may not be relevant to your tasks. You should not respond to this context unless it is highly relevant to your task.
