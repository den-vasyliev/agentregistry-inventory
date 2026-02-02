# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/claude-code) when working with code in this repository.

## Project Overview

Agent Registry is a **Kubernetes-native controller** that provides a centralized registry to securely curate, discover, deploy, and manage agentic infrastructure including MCP servers, agents, and skills.

### Key Features
- **CRD-based storage** - Catalog entries stored as Kubernetes Custom Resources
- **Auto-discovery** - Automatically indexes deployed resources
- **HTTP API** - REST API embedded in controller for catalog access
- **Web UI** - Next.js interface for browsing and managing catalogs
- **Declarative deployment** - GitOps-ready resource management

## Architecture

The project consists of two main components:

1. **Controller** - Kubernetes controller with embedded HTTP API
2. **UI** - Next.js web interface

### Directory Structure

```
agentregistry/
├── cmd/
│   └── controller/          # Controller entry point (ONLY binary)
├── internal/
│   ├── controller/          # 8 K8s reconcilers
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

1. **8 Kubernetes Reconcilers:**
   - `MCPServerCatalogReconciler` - Manages MCP server catalog entries
   - `AgentCatalogReconciler` - Manages agent catalog entries
   - `SkillCatalogReconciler` - Manages skill catalog entries
   - `RegistryDeploymentReconciler` - Deploys catalog items to K8s
   - `MCPServerDiscoveryReconciler` - Auto-discovers deployed MCPServers
   - `AgentDiscoveryReconciler` - Auto-discovers deployed Agents
   - `SkillDiscoveryReconciler` - Auto-discovers skills from agents
   - `ModelDiscoveryReconciler` - Auto-discovers ModelConfig resources

2. **HTTP API Server** (embedded, port 8080):
   - Public endpoints: `/v0/servers`, `/v0/agents`, `/v0/skills`, `/v0/models`
   - Admin endpoints: `/admin/v0/*` for management
   - **Authentication disabled by default** (`authEnabled: false`)

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

kagent.dev/v1alpha2:
- Agent
- RemoteMCPServer
- ModelConfig

kmcp.agentregistry.dev/v1alpha1:
- MCPServer
```

## Build Commands

```bash
# Build controller binary
make build-controller

# Build UI (static export)
make build-ui

# Build both
make build

# Run in development (controller + UI)
make dev

# Run tests
make test

# Run controller tests only
make test-controller

# Build container image
make ko-controller
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
4. **No authentication by default** - `authEnabled: false` in `internal/httpapi/server.go:53`

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

### What Needs Tests
- ❌ HTTP API handlers (0% coverage)
- ❌ Most controller reconcilers
- ❌ MCP server implementation

## Common Tasks

### Adding a New CRD Field

1. Update `api/v1alpha1/*_types.go`
2. Run `make generate` (if using kubebuilder)
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

## Notes

- **No CLI** - All management via K8s CRs or HTTP API
- **No database** - CRDs are the source of truth
- **Controller-only** - Single binary for all functionality
- **UI is separate** - Runs independently, connects via HTTP

IMPORTANT: This context may or may not be relevant to your tasks. You should not respond to this context unless it is highly relevant to your task.
