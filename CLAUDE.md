# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/claude-code) when working with code in this repository.

## Git Workflow

**CRITICAL: Pre-commit checklist - ALWAYS run these commands before EVERY commit:**

1. `make fmt` - Format all code (gofmt, prettier)
2. `make lint` - Check for linting issues (gofmt + go vet)
3. `make test` - Run all tests

If ANY of these fail, fix the issues before committing. NEVER commit or push code that fails formatting, linting, or tests.

**CRITICAL: NEVER push before user tests!** After committing, WAIT for the user to test and explicitly ask to push.

**IMPORTANT: Commit Messages**

- Follow conventional commits (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`)
- Do NOT add `Co-Authored-By:` lines to commit messages

## Project Overview

Agent Registry is a **Kubernetes-native controller** that provides a centralized registry to curate, discover, deploy, and manage agentic infrastructure including MCP servers, agents, and skills.

## Architecture

Single binary controller with embedded HTTP API and static UI.

### Directory Structure

```
agentregistry/
├── cmd/controller/              # Controller entry point (ONLY binary)
├── api/v1alpha1/                # CRD type definitions
├── internal/
│   ├── controller/              # 5 K8s reconcilers
│   ├── httpapi/                 # HTTP API server (Huma) + UI embedding
│   │   ├── handlers/            # API endpoint handlers
│   │   └── server.go            # Huma API setup + import logic
│   ├── runtime/                 # Deployment logic (translation to K8s resources)
│   ├── cluster/                 # Remote cluster client factory (GCP workload identity)
│   ├── config/                  # Namespace and config helpers
│   ├── validation/              # Input validation
│   ├── utils/                   # Shared utilities
│   └── version/                 # Version info
├── ui/                          # Next.js 16 frontend (static export)
│   ├── app/                     # Next.js app router pages
│   ├── components/              # React components
│   └── lib/                     # API client (admin-api.ts)
├── config/
│   ├── crd/                     # Generated CRD YAML
│   ├── external-crds/           # Vendored upstream CRDs (kagent, kmcp)
│   ├── rbac/                    # RBAC manifests
│   └── samples/                 # Example resources
└── charts/agentregistry/        # Helm chart
```

### CRDs

```
agentregistry.dev/v1alpha1:
  MCPServerCatalog, AgentCatalog, SkillCatalog, ModelCatalog
  RegistryDeployment
  DiscoveryConfig (namespace-scoped, multi-cluster autodiscovery)

External (vendored):
  kagent.dev/v1alpha2: Agent, RemoteMCPServer, ModelConfig
  kmcp.agentregistry.dev/v1alpha1: MCPServer
```

### Default Namespace

All Agent Registry resources live in the `agentregistry` namespace. Use `config.GetNamespace()` (respects `POD_NAMESPACE` env var) when referencing it in code.

### Ports

- `:8080` - HTTP API + UI
- `:8081` - Metrics
- `:8082` - Health probes

## Build Commands

```bash
make build            # Build everything (UI + controller)
make dev              # Start controller + UI dev server
make dev-controller   # Controller only
make dev-ui           # UI dev server only (:3000)
make test             # Run all tests
make lint             # gofmt + go vet
make fmt              # Format Go + JS code
make generate         # Regenerate deepcopy + CRD manifests
make push             # Build UI + push container with ko
```

## Testing

**Coverage: 37.7%**

| Package | Coverage |
|---|---|
| `internal/validation` | 100% |
| `internal/config` | 100% |
| `internal/runtime/translation/kagent` | 81.7% |
| `internal/httpapi` | 47.7% |
| `internal/controller` | 39.4% |
| `internal/cluster` | 28.3% |
| `internal/httpapi/handlers` | 25.9% |
| `internal/runtime` | 8.7% |

**Use envtest, NOT fake clients.** All controller tests use `SetupTestEnv` from `testhelper_test.go`:

```go
helper := SetupTestEnv(t, 60*time.Second, false)
defer helper.Cleanup(t)
```

## Common Tasks

### Adding a New CRD Field

1. Update `api/v1alpha1/*_types.go`
2. `make generate`
3. Update controller in `internal/controller/`
4. Update API handlers in `internal/httpapi/handlers/`
5. Update UI in `ui/components/`

### Adding a New API Endpoint

1. Create handler in `internal/httpapi/handlers/`
2. Register route in handler's `RegisterRoutes()` method

### Adding a New Reconciler

1. Create reconciler in `internal/controller/`
2. Register in `cmd/controller/main.go`
3. Add tests in `internal/controller/*_test.go`

## Environment

- **Go**: 1.25+
- **Node.js**: 18+
- **Kubernetes**: 1.27+
