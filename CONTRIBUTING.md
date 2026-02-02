# Contributing to Agent Registry

Thank you for your interest in contributing to Agent Registry! This document provides guidelines and instructions for contributing.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/agentregistry.git`
3. Add upstream remote: `git remote add upstream https://github.com/agentregistry-dev/agentregistry.git`
4. Create a branch: `git checkout -b feature/my-feature`

## Development Setup

### Prerequisites

- **Go 1.25+** (for controller development)
- **Node.js 18+** (for UI development)
- **Kubernetes cluster** (local via kind/minikube or remote)
- **kubectl** configured to access your cluster
- **make** (for build automation)

### Initial Setup

```bash
# Clone and setup
git clone https://github.com/agentregistry-dev/agentregistry.git
cd agentregistry

# Download Go dependencies
go mod download

# Install UI dependencies
cd ui && npm install && cd ..

# Build controller
make build-controller
```

## Development Workflow

### Quick Start - Run Everything

```bash
# Start controller + UI in one command
make dev

# Access:
# - UI: http://localhost:3000
# - API: http://localhost:8080
# - Metrics: http://localhost:8081
```

### Working on the Controller

```bash
# Make changes to cmd/controller/ or internal/controller/

# Quick build (Go only)
go build -o bin/controller cmd/controller/main.go

# Run controller
./bin/controller --enable-http-api=true

# Or use make
make dev-controller
```

### Working on the UI

```bash
# Start UI development server (hot reload)
make dev-ui

# Opens at http://localhost:3000
# Connects to controller API at localhost:8080

# Make changes to ui/app/**/*.tsx
# Changes auto-reload in browser
```

### Working on Both

```bash
# Terminal 1: Controller
make dev-controller

# Terminal 2: UI
make dev-ui

# Or run both in parallel
make dev
```

## Code Style

### Go

- Follow standard Go conventions
- Use `gofmt` for formatting
- Use `golangci-lint` for linting
- Write meaningful comments for exported functions
- Keep functions small and focused

```bash
# Format code
make fmt

# Run linter
make lint
```

### TypeScript/React

- Follow Next.js 14 and React best practices
- Use TypeScript for type safety
- Use functional components with hooks
- Keep components small and reusable

```bash
# Lint UI code
cd ui && npm run lint
```

## Testing

### Controller Tests

```bash
# Run all tests with coverage
make test

# Run controller tests only
make test-controller

# View coverage report
go tool cover -html=coverage.out
```

### Test Structure

- Unit tests: `internal/controller/*_test.go`
- Use `envtest` for controller tests with fake K8s API
- Current coverage: 22%

## Adding New Features

### New Kubernetes Reconciler

1. Create `internal/controller/myresource_controller.go`:

```go
package controller

import (
    "context"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
)

type MyResourceReconciler struct {
    client.Client
    Scheme *runtime.Scheme
    Logger zerolog.Logger
}

func (r *MyResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Implementation
    return ctrl.Result{}, nil
}

func (r *MyResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&MyResourceCRD{}).
        Complete(r)
}
```

2. Register in `cmd/controller/main.go`
3. Add tests in `internal/controller/myresource_controller_test.go`
4. Update documentation

### New CRD

1. Define CRD in `api/v1alpha1/myresource_types.go`
2. Run code generation (if using kubebuilder markers)
3. Create YAML in `api/v1alpha1/`
4. Add reconciler (see above)
5. Update Helm chart

### New HTTP API Endpoint

1. Add handler in `internal/httpapi/handlers/myhandler.go`:

```go
package handlers

func NewMyHandler(c client.Client, cache cache.Cache, logger zerolog.Logger) *MyHandler {
    return &MyHandler{client: c, cache: cache, logger: logger}
}

func (h *MyHandler) RegisterRoutes(api huma.API, prefix string, admin bool) {
    huma.Register(api, huma.Operation{
        OperationID: "getMyResources",
        Method:      http.MethodGet,
        Path:        prefix + "/my-resources",
    }, h.GetMyResources)
}
```

2. Register in `internal/httpapi/server.go`
3. Add tests
4. Update API documentation

### New UI Component

1. Create component in `ui/components/`:

```tsx
import { Card } from "@/components/ui/card"

export function MyComponent() {
  return (
    <Card>
      <h2>My Component</h2>
    </Card>
  )
}
```

2. Use in page:

```tsx
import { MyComponent } from "@/components/MyComponent"

export default function Page() {
  return <MyComponent />
}
```

3. Test in browser with `make dev-ui`

## CRD Changes

When modifying CRD schemas:

1. Update `api/v1alpha1/*_types.go`
2. Update corresponding reconciler in `internal/controller/`
3. Update API handlers in `internal/httpapi/handlers/`
4. Update UI components if needed
5. Test reconciliation logic
6. Update examples in `docs/` or `charts/`

## Documentation

Update documentation when adding features:

- `README.md` - Overview, quick start, examples
- `CLAUDE.md` - Developer guide for Claude Code
- `DEVELOPMENT.md` - Architecture details
- Inline code comments for exported functions
- OpenAPI docs (auto-generated from Huma)

## Commit Messages

Follow conventional commits:

```
type(scope): subject

body

footer
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Formatting
- `refactor`: Code restructuring
- `test`: Tests
- `chore`: Maintenance

**Examples:**
```
feat(controller): add ModelCatalog reconciler
fix(api): handle empty catalog lists
docs: update CRD examples in README
refactor(httpapi): simplify handler registration
test(controller): add AgentCatalog reconciler tests
```

## Pull Request Process

1. Update documentation
2. Add/update tests
3. Ensure tests pass (`make test`)
4. Ensure code is formatted (`make fmt`)
5. Ensure linting passes (`make lint`)
6. Request review

### PR Checklist

- [ ] Code follows style guidelines
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] Commits follow conventional commits
- [ ] PR description explains the change

## Building for Release

```bash
# Full clean build
make all

# Test the binary
./bin/controller --version

# Build container image
make ko-controller

# Tag and push
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

## Common Issues

### Go Module Issues

```bash
go mod tidy
go mod download
```

### UI Build Failures

```bash
cd ui
rm -rf node_modules package-lock.json .next
npm install
npm run build
```

### Controller Not Connecting to K8s

```bash
# Check kubeconfig
kubectl cluster-info

# Verify CRDs are installed
kubectl get crds | grep agentregistry.dev

# Check controller logs
kubectl logs -n agentregistry deployment/agentregistry-controller
```

### NextAuth Errors in UI

For development, ensure `ui/.env.local` exists:

```bash
AUTH_SECRET=dev-secret-bypass-only-not-for-production
NEXT_PUBLIC_API_URL=http://localhost:8080
```

## Getting Help

- 🐛 **Report bugs**: [GitHub Issues](https://github.com/agentregistry-dev/agentregistry/issues)
- 💬 **Ask questions**: [GitHub Discussions](https://github.com/agentregistry-dev/agentregistry/discussions)
- 💬 **Chat**: [Discord Server](https://discord.gg/HTYNjF2y2t)

## Code of Conduct

- Be respectful and inclusive
- Welcome newcomers
- Give constructive feedback
- Focus on what's best for the project
- Follow the [Contributor Covenant](https://www.contributor-covenant.org/)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to Agent Registry! 🎉
