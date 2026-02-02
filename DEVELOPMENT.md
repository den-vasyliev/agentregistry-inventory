# Development Guide

This document provides architectural details and development guidance for the Agent Registry controller.

## Architecture Overview

Agent Registry is a **Kubernetes-native controller** that manages AI infrastructure using Custom Resource Definitions (CRDs). It consists of three main components:

### 1. Controller (cmd/controller/)

Built with [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime), provides:

- **8 Kubernetes Reconcilers** - Manage catalog entries and deployments
- **HTTP API Server** - REST API embedded in controller process
- **Auto-Discovery** - Automatically index deployed resources
- **Leader Election** - High availability support
- **Metrics** - Prometheus metrics on port 8081
- **Health Probes** - Liveness/readiness on port 8082

**Reconcilers:**

1. **MCPServerCatalog** - Manages MCP server catalog entries
2. **AgentCatalog** - Manages agent catalog entries
3. **SkillCatalog** - Manages skill catalog entries
4. **RegistryDeployment** - Deploys catalog entries to runtime
5. **MCPServerDiscovery** - Auto-discovers deployed MCPServers
6. **AgentDiscovery** - Auto-discovers deployed Agents
7. **SkillDiscovery** - Auto-discovers skills from Agent specs
8. **ModelDiscovery** - Auto-discovers ModelConfig resources

### 2. HTTP API (internal/httpapi/)

Built with [Huma v2](https://github.com/danielgtaylor/huma), provides REST API:

**Public Endpoints (Read-Only):**
- `GET /v0/servers` - List MCP servers
- `GET /v0/servers/{name}/{version}` - Get specific server
- `GET /v0/agents` - List agents
- `GET /v0/skills` - List skills
- `GET /v0/models` - List models

**Admin Endpoints (Management):**
- `POST /admin/v0/servers` - Create/update server catalog
- `POST /admin/v0/agents` - Create/update agent catalog
- `POST /admin/v0/skills` - Create/update skill catalog
- `POST /admin/v0/deploy` - Deploy to runtime

**Port:** 8080 (configurable with `--http-api-addr`)

**Authentication:** Disabled by default (`authEnabled: false` in server.go:53)

### 3. Web UI (ui/)

Built with:
- **Framework:** Next.js 14 (App Router)
- **Language:** TypeScript
- **Styling:** Tailwind CSS
- **Components:** shadcn/ui
- **Auth:** NextAuth.js (bypassed in development)

**Features:**
- Browse published catalogs (MCP servers, agents, skills, models)
- View deployed resources
- Health status indicators
- Real-time data from controller API

**Development Server:** `npm run dev` (port 3000)

**Environment:** Requires `ui/.env.local` with `AUTH_SECRET` for NextAuth

## Data Storage

### CRD-Based Storage

All data is stored as Kubernetes Custom Resources:

**Catalog CRDs:**
- `MCPServerCatalog` - MCP server definitions
- `AgentCatalog` - Agent definitions
- `SkillCatalog` - Skill definitions
- `ModelCatalog` - Model definitions

**Runtime CRDs:**
- `RegistryDeployment` - Deployment requests
- `MCPServer` (kagent.dev) - Deployed MCP servers
- `Agent` (kagent.dev) - Deployed agents
- `ModelConfig` (kagent.dev) - Model configurations

**API Version:** `agentregistry.dev/v1alpha1`

**Storage:** CRDs are stored in Kubernetes etcd, no external database required

## Data Flow

### Catalog Management

```
User/GitOps
    ↓
kubectl apply (MCPServerCatalog YAML)
    ↓
K8s API Server
    ↓
Controller Watch
    ↓
MCPServerCatalog Reconciler
    ↓
Update Status + Cache
```

### Deployment Flow

```
User/GitOps
    ↓
kubectl apply (RegistryDeployment YAML)
    ↓
K8s API Server
    ↓
Controller Watch
    ↓
RegistryDeployment Reconciler
    ├─→ Lookup Catalog Entry (MCPServerCatalog/AgentCatalog)
    ├─→ Translate to Runtime CR (MCPServer/Agent)
    └─→ Create Runtime CR in target namespace
        ↓
Runtime Controller (kmcp/kagent)
    ↓
Running Pod
```

### Auto-Discovery Flow

```
Runtime Controller (kmcp/kagent)
    ↓
Create MCPServer/Agent CR
    ↓
K8s API Server
    ↓
Controller Watch
    ↓
Discovery Reconciler (MCPServerDiscovery/AgentDiscovery)
    ├─→ Check if catalog entry exists
    └─→ Create catalog entry if missing
        ↓
Auto-cataloged resource
```

### HTTP API Request

```
Browser/CLI Request
    ↓
HTTP API Server (:8080)
    ↓
Huma Handler (internal/httpapi/handlers/)
    ↓
Controller Cache (cache.Cache)
    ↓
CRD List/Get
    ↓
JSON Response
```

## Build Process

### Development Workflow

```bash
# Start controller + UI in one command
make dev

# Access:
# - UI: http://localhost:3000
# - API: http://localhost:8080
# - Metrics: http://localhost:8081
# - Health: http://localhost:8082
```

The `make dev` target:
1. Builds controller binary with version info
2. Starts controller with HTTP API enabled
3. Starts UI development server with hot reload
4. Runs both in parallel (Ctrl+C stops both)

### Component-Specific Development

```bash
# Controller only (fast iteration)
make dev-controller
# or
go build -o bin/controller cmd/controller/main.go
./bin/controller --enable-http-api=true

# UI only (hot reload)
make dev-ui
# or
cd ui && npm run dev

# Full build (UI + controller)
make build
```

### Production Build

```bash
# Build controller binary
make build-controller

# Build UI static files
make build-ui

# Build container image with ko
make ko-controller

# Full build (all targets)
make all
```

### Container Images

Uses [ko](https://ko.build/) for building container images:

```bash
# Build controller image
make ko-controller
# Builds: localhost:5001/agentregistry-controller:latest

# Push to registry
KO_DOCKER_REPO=ghcr.io/myorg make ko-controller
```

## Extension Points

### Adding a New Reconciler

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
    // 1. Fetch the resource
    // 2. Implement reconciliation logic
    // 3. Update status
    // 4. Return result
    return ctrl.Result{}, nil
}

func (r *MyResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&MyResourceCRD{}).
        Complete(r)
}
```

2. Register in `cmd/controller/main.go`:

```go
if err = (&controller.MyResourceReconciler{
    Client: mgr.GetClient(),
    Scheme: mgr.GetScheme(),
    Logger: logger.With().Str("controller", "myresource").Logger(),
}).SetupWithManager(mgr); err != nil {
    setupLog.Error(err, "unable to create controller", "controller", "MyResource")
    os.Exit(1)
}
```

3. Add tests in `internal/controller/myresource_controller_test.go`

### Adding a New CRD

1. Define CRD in `api/v1alpha1/myresource_types.go`:

```go
package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MyResourceSpec struct {
    Name    string `json:"name"`
    Version string `json:"version"`
}

type MyResourceStatus struct {
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type MyResource struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   MyResourceSpec   `json:"spec,omitempty"`
    Status MyResourceStatus `json:"status,omitempty"`
}

type MyResourceList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []MyResource `json:"items"`
}
```

2. Create YAML in `api/v1alpha1/myresource.yaml`

3. Add reconciler (see above)

4. Update Helm chart to install CRD

### Adding a New HTTP API Endpoint

1. Create handler in `internal/httpapi/handlers/myhandler.go`:

```go
package handlers

import (
    "context"
    "github.com/danielgtaylor/huma/v2"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/cache"
)

type MyHandler struct {
    client client.Client
    cache  cache.Cache
    logger zerolog.Logger
}

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

func (h *MyHandler) GetMyResources(ctx context.Context, input *struct{}) (*struct{
    Body []MyResourceResponse
}, error) {
    // 1. List CRs from cache
    // 2. Transform to response format
    // 3. Return
}
```

2. Register in `internal/httpapi/server.go`:

```go
myHandler := handlers.NewMyHandler(s.client, s.cache, s.logger)
myHandler.RegisterRoutes(api, "/v0", false)
```

3. Add tests

### Adding a New UI Page

1. Create page in `ui/app/mypage/page.tsx`:

```tsx
import { Card } from "@/components/ui/card"

export default function MyPage() {
  const [data, setData] = useState([])

  useEffect(() => {
    fetch('/v0/my-resources')
      .then(res => res.json())
      .then(setData)
  }, [])

  return (
    <div>
      <h1>My Resources</h1>
      {data.map(item => (
        <Card key={item.id}>{item.name}</Card>
      ))}
    </div>
  )
}
```

2. Test with `make dev-ui`

## Testing

### Running Tests

```bash
# All tests with coverage
make test

# Controller tests only
make test-controller

# View coverage report
go tool cover -html=coverage.out
```

### Test Structure

- **Unit tests:** `internal/controller/*_test.go`
- **Framework:** [envtest](https://book.kubebuilder.io/reference/envtest.html) - Provides fake Kubernetes API
- **Current coverage:** 22% (controller 17%, runtime 4.4%, translation 81.7%)

### Writing Controller Tests

```go
package controller_test

import (
    "testing"
    "context"
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("MyResource Controller", func() {
    var (
        ctx context.Context
        resource *v1alpha1.MyResource
    )

    BeforeEach(func() {
        ctx = context.Background()
        resource = &v1alpha1.MyResource{
            ObjectMeta: metav1.ObjectMeta{
                Name:      "test-resource",
                Namespace: "default",
            },
            Spec: v1alpha1.MyResourceSpec{
                Name: "test",
            },
        }
    })

    It("should reconcile the resource", func() {
        Expect(k8sClient.Create(ctx, resource)).To(Succeed())

        // Wait for reconciliation
        Eventually(func() bool {
            err := k8sClient.Get(ctx, client.ObjectKeyFromObject(resource), resource)
            return err == nil && len(resource.Status.Conditions) > 0
        }).Should(BeTrue())
    })
})
```

## Configuration

### Controller Flags

```bash
./bin/controller \
  --metrics-addr=:8081 \
  --probe-addr=:8082 \
  --leader-elect=true \
  --enable-http-api=true \
  --http-api-addr=:8080 \
  --log-level=info
```

### Environment Variables

```bash
# Controller
KUBECONFIG=/path/to/kubeconfig

# UI (.env.local)
AUTH_SECRET=dev-secret-bypass-only-not-for-production
NEXT_PUBLIC_API_URL=http://localhost:8080
```

## Debugging

### Controller Logs

```bash
# Local development
./bin/controller --log-level=debug

# Kubernetes deployment
kubectl logs -n agentregistry deployment/agentregistry-controller -f
```

### API Debugging

```bash
# Test API endpoints
curl http://localhost:8080/v0/servers
curl http://localhost:8080/v0/agents

# Check health
curl http://localhost:8082/healthz
curl http://localhost:8082/readyz

# Check metrics
curl http://localhost:8081/metrics
```

### CRD Inspection

```bash
# List catalog entries
kubectl get mcpservercatalogs -A
kubectl get agentcatalogs -A
kubectl get skillcatalogs -A

# View specific resource
kubectl describe mcpservercatalog filesystem-v1-0-0 -n agentregistry

# Watch reconciliation
kubectl get mcpservercatalogs -A -w
```

## Performance Considerations

### Controller Cache

The controller uses controller-runtime's cache:
- **Read operations:** Served from cache (fast)
- **Write operations:** Go to K8s API (slower)
- **Cache refresh:** Automatic via informers
- **Cache initialization:** Waits for sync on startup

### HTTP API Cache

The HTTP API uses the same controller cache:
- **GET requests:** Read from cache (no K8s API calls)
- **POST requests:** Write to K8s API, cache updates via watch
- **Consistency:** Eventually consistent (typical delay < 100ms)

### Resource Limits

Recommended resource requests/limits:

```yaml
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 1000m
    memory: 512Mi
```

## Security Considerations

### RBAC

The controller requires these permissions:
- `get`, `list`, `watch` on all CRDs
- `create`, `update`, `patch`, `delete` on catalog CRDs
- `update` on status subresources
- `create` on runtime CRDs (MCPServer, Agent)

See `charts/agentregistry/templates/rbac.yaml` for full RBAC configuration.

### API Authentication

**Current state:** Authentication is disabled by default (`authEnabled: false`)

To enable authentication:
1. Set `AUTH_ENABLED=true` environment variable
2. Configure allowed tokens via `AUTH_TOKENS` (comma-separated)
3. Clients must send `Authorization: Bearer <token>` header

### Network Policies

Recommended network policies:
- Controller can access K8s API
- Controller exposes metrics to Prometheus
- UI can access controller HTTP API
- Block external access to controller API

## Common Issues

### Controller Not Starting

```bash
# Check CRDs are installed
kubectl get crds | grep agentregistry.dev

# Check RBAC permissions
kubectl auth can-i list mcpservercatalogs --as=system:serviceaccount:agentregistry:agentregistry

# Check logs
kubectl logs -n agentregistry deployment/agentregistry-controller
```

### UI Not Connecting to API

```bash
# Check API is running
curl http://localhost:8080/v0/servers

# Check NEXT_PUBLIC_API_URL in ui/.env.local
cat ui/.env.local

# Check NextAuth secret is set
grep AUTH_SECRET ui/.env.local
```

### Reconciliation Not Happening

```bash
# Check controller is running
kubectl get pods -n agentregistry

# Check controller logs for errors
kubectl logs -n agentregistry deployment/agentregistry-controller | grep -i error

# Check resource status
kubectl describe mcpservercatalog my-server -n agentregistry
```

## Contributing

When adding features:

1. Create feature branch
2. Add reconciler/handler/UI component
3. Add tests (aim for >80% coverage)
4. Update documentation
5. Test with `make dev`
6. Run `make test` and `make lint`
7. Submit PR with conventional commit messages

## Resources

- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) - Controller framework
- [Huma v2](https://github.com/danielgtaylor/huma) - HTTP API framework
- [Next.js Documentation](https://nextjs.org/docs) - UI framework
- [envtest](https://book.kubebuilder.io/reference/envtest.html) - Testing framework
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) - MCP protocol
- [kagent](https://github.com/kagent-dev/kagent) - Agent runtime
- [kmcp](https://github.com/kagent-dev/kmcp) - MCP server operator
