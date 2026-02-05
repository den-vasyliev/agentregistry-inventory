# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/claude-code) when working with code in this repository.

## Git Workflow

**CRITICAL: Pre-commit checklist - ALWAYS run these commands before EVERY commit:**

1. `make fmt` - Format all code (gofmt, prettier)
2. `make lint` - Check for linting issues
3. `make test` - Run all tests

If ANY of these fail, fix the issues before committing. NEVER commit or push code that fails formatting, linting, or tests.

**CRITICAL: NEVER push before user tests!** After committing, WAIT for the user to test and explicitly ask to push.


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
│   ├── controller/          # 5 K8s reconcilers (catalog + deployment + discovery)
│   ├── httpapi/             # HTTP API server (embedded in controller)
│   │   ├── handlers/        # API endpoint handlers
│   │   └── server.go        # Huma API setup + UI embedding
│   ├── runtime/             # Runtime deployment logic (translation to K8s resources)
│   ├── utils/               # Shared utilities
│   └── version/             # Version info
├── ui/                      # Next.js 16 frontend (static export)
│   ├── app/                 # Next.js app router pages
│   ├── components/          # React components
│   ├── lib/                 # API clients and utilities
│   └── out/                 # Static export output (embedded into controller)
├── api/v1alpha1/            # CRD definitions (Go types)
├── config/                  # Kubernetes manifests
│   ├── crd/                 # Generated CRD YAML
│   ├── rbac/                # RBAC manifests
│   ├── manager/             # Controller deployment
│   └── samples/             # Example resources
└── charts/agentregistry/    # Helm chart
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

1. **5 Kubernetes Reconcilers:**
   - `MCPServerCatalogReconciler` - Manages MCP server catalog entries
   - `AgentCatalogReconciler` - Manages agent catalog entries
   - `SkillCatalogReconciler` - Manages skill catalog entries
   - `RegistryDeploymentReconciler` - Deploys catalog items to K8s
   - `DiscoveryConfigReconciler` - Multi-cluster discovery with workload identity (see [docs/AUTODISCOVERY.md](docs/AUTODISCOVERY.md))

2. **HTTP API Server** (embedded, port 8080):
   - Public endpoints: `/v0/servers`, `/v0/agents`, `/v0/skills`, `/v0/models` (read-only)
   - Admin endpoints: `/admin/v0/*` for management (requires auth)
   - **Authentication enabled by default** - tokens read from Secret `agentregistry-api-tokens`
   - UI assets embedded in binary and served at `/` (static files from `ui/out`)

3. **Metrics & Health:**
   - Metrics: `:8081`
   - Health probe: `:8082`

### UI (`ui/`)

- **Next.js 16** with TypeScript (static export mode)
- **Tailwind CSS** + **shadcn/ui** components
- Connects to controller API at `localhost:8080` in dev, or embedded/served by controller in production
- **NextAuth.js** for authentication (optional, can be bypassed for dev)
- Built as static export (`next export`) and embedded into controller binary

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
# Build UI (static export to ui/out/)
make build-ui

# Prepare UI for embedding (copies ui/out/ to internal/httpapi/ui_dist/)
make prepare-ui-embed

# Build controller binary (includes embedded UI)
make build-controller

# Build everything (UI + prepare + controller)
make build

# Run in development mode
make dev              # Starts both controller and UI in parallel
make dev-controller   # Controller only (without UI embedding)
make dev-ui          # UI dev server only (:3000)

# Testing & Quality
make test            # Run all tests
make test-coverage   # Run tests with coverage report
make lint            # Run golangci-lint
make fmt             # Format Go code (gofmt)

# Container images
make ko-controller   # Build and push with ko (uses KO_DOCKER_REPO env)
make image          # Build container image locally
make push           # Prepare UI and push with ko

# Code generation
make generate       # Generate deepcopy methods and CRD manifests
make manifests      # Generate CRD and RBAC manifests only
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
- **Next.js 16**: React framework (static export)
- **NextAuth.js**: Authentication (optional, can be bypassed)
- **Tailwind CSS**: Styling
- **shadcn/ui**: Component library
- **Lucide React**: Icons
- **Sonner**: Toast notifications

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

- **Static export** - Next.js built as static HTML/CSS/JS (no Node.js runtime needed)
- **Client-side rendering** - All components use `"use client"` directive
- **API calls** -
  - Dev: Direct to controller at `localhost:8080` (CORS enabled)
  - Production: Proxied through controller serving the UI
- **Optional auth** - NextAuth can be bypassed for development (set `AUTH_SECRET=dev-secret...`)
- **Embedded in controller** - UI assets copied to `internal/httpapi/ui_dist/` and embedded via `//go:embed`

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
- ❌ Most controller reconcilers (MCPServerCatalog, AgentCatalog, SkillCatalog, RegistryDeployment)
- ❌ UI components and pages

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

## Important Notes

### Architecture Decisions

- **No CLI** - All management via K8s CRs or HTTP API (removed `arctl` CLI)
- **No database** - CRDs are the source of truth (removed SQLite/PostgreSQL)
- **Single binary** - Controller includes HTTP API + embedded UI
- **UI embedded** - Static Next.js export bundled into controller binary (no separate deployment needed)
- **Kubernetes-native** - Everything runs in-cluster, no local runtime mode

### UI Embedding Process

1. `make build-ui` - Builds Next.js static export to `ui/out/`
2. `make prepare-ui-embed` - Copies `ui/out/` to `internal/httpapi/ui_dist/`
3. `make build-controller` - Embeds `ui_dist/` into controller binary using `//go:embed`
4. Controller serves UI at `/` and API at `/v0/*` and `/admin/v0/*`

### Service Accounts

- **agentregistry-controller** - Main controller service account
- **agentregistry-inventory** - Discovery service account (for multi-cluster workload identity)

### Default Ports

- `:8080` - HTTP API and UI
- `:8081` - Metrics
- `:8082` - Health probes

---

IMPORTANT: This context may or may not be relevant to your tasks. You should not respond to this context unless it is highly relevant to your task.
