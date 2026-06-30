# Application Passport — Agent Registry

## 1. Identity

Agent Registry is a Kubernetes-native controller that provides a centralized
registry to curate, discover, deploy, and manage agentic infrastructure — MCP
servers, agents, skills, and model configs. It ships as a single Go binary that
runs an in-cluster controller-runtime manager (reconciling six own CRDs and
deploying kagent/kmcp resources), an embedded HTTP REST API with a static
Next.js UI, and an embedded MCP server exposing registry tools, prompts, and
resources. Catalog recommendation tools delegate inference to the connected MCP
client via MCP **sampling** rather than bundling a model. The admin/write
surface is authenticated and fail-closed; public catalog-read endpoints are
open by design.

| Field | Value |
|---|---|
| Component | Agent Registry (controller) |
| Module | `github.com/agentregistry-dev/agentregistry` |
| Repository | `github.com/den-vasyliev/agentregistry-inventory` |
| Registry / image | `ghcr.io/den-vasyliev/agentregistry-inventory` |
| Version | `v0.5.4` (chart `0.1.0`, appVersion `latest`) |
| Commit | `e2454cc` |
| Owner | den.vasyliev@gmail.com |
| License | Apache-2.0 |
| Audit status | Remediated — all HIGH/CRITICAL findings fixed; 0 HIGH/CRITICAL CVEs (2026-06-30) |
| Code review | All findings remediated — 2026-06-30 |

## 2. Classification

| Field | Value |
|---|---|
| Class | Kubernetes controller (operator) + embedded HTTP API + embedded MCP server |
| Type | Registry / catalog + deployment manager for MCP servers, agents, skills, models |
| Capabilities | Reconciles 6 own CRDs; deploys kagent/kmcp resources; multi-cluster discovery + deployment; serves REST API + static UI; serves MCP server (tools, prompts, resources) |
| Model role | Calls an LLM via MCP **sampling** (delegates inference to the connected MCP client) for catalog recommendation tools |
| Runtime | Single Go binary; in-cluster controller-runtime manager |
| Public accessibility | HTTP API + UI and MCP server are HTTP services; exposure depends on Service type (`ClusterIP` default) / optional Gateway API HTTPRoute |
| Prod data access | Read/write to Kubernetes API across all namespaces (own CRDs, kagent CRDs, ConfigMaps); read to remote/target clusters |
| PII | None identified in source |
| Sensitivity | Manages deployment of cluster workloads and reads cluster Secrets |

## 3. Artifact

| Field | Value |
|---|---|
| Form | OCI container image (built with `ko`) |
| Languages | Go 1.26 (controller/API/MCP); TypeScript / Next.js 16 (UI, static export embedded in binary) |
| Build tool | `ko` (controller), `next build` static export (UI) |
| Base image | `cgr.dev/chainguard/static:latest-glibc` |
| Runtime user | Non-root UID/GID 65532; `runAsNonRoot: true`; `readOnlyRootFilesystem: true`; all capabilities dropped; `seccompProfile: RuntimeDefault`; `allowPrivilegeEscalation: false` |

## 4. AI / LLM Supply Chain

| Field | Value |
|---|---|
| Inference model | None bundled; inference performed by the connected MCP client via MCP `sampling` |
| Sampling call cap | `MaxTokens: 4096` per call |
| Global token / cost budget | Rate limit (calls/sec, token-bucket) + aggregate token budget on the sampling path; env-tunable. Write/mutating MCP tools are fail-closed (refused when auth disabled) |
| LLM-calling tools | `recommend_servers`, `recommend_agents` (catalog data + free-text user `description` composed into the sampling prompt) |
| MCP prompts exposed | `agentregistry_skill`, `deploy_server`, `find_agents`, `registry_overview`, `catalog_schema`, `deployment_workflow` (+ others) |
| Egress destinations | `container.googleapis.com` (GKE cluster discovery, workload-identity OAuth) — auth: GCP default credentials; `raw.githubusercontent.com` / `gitlab.com` (submit manifest fetch) — auth: none; user-supplied URL (admin import) — https-only, private/loopback/link-local/metadata IPs blocked, no caller headers forwarded |
| External AI dependencies | `github.com/mark3labs/mcp-go`, `github.com/modelcontextprotocol/registry`, `github.com/kagent-dev/kagent`, `github.com/kagent-dev/kmcp` |

## 5. Privileges

### Kubernetes RBAC (ClusterRole, cluster-wide)

| API group | Resources | Verbs |
|---|---|---|
| `agentregistry.dev` | mcpservercatalogs, agentcatalogs, skillcatalogs, modelcatalogs, registrydeployments, discoveryconfigs (+ `/status`, `/finalizers`) | get, list, watch, create, update, patch, delete |
| `kagent.dev` | agents, remotemcpservers, mcpservers, modelconfigs | get, list, watch, create, update, patch, delete |
| `""` (core) | configmaps | get, list, watch, create, update, patch, delete |
| `""` (core) | events | create, patch |
| `coordination.k8s.io` | leases | get, list, watch, create, update, patch, delete |

Secret access is **namespaced** (controller namespace only), granted via a `Role`/`RoleBinding` rather than cluster-wide — only the `agentregistry-api-tokens` Secret is read.

ServiceAccount created by chart. On GKE, uses workload identity / GCP default credentials with `container.CloudPlatformScope` for remote-cluster discovery and deployment.

### API / MCP token scope

| Item | Value |
|---|---|
| Token source | Kubernetes Secret `agentregistry-api-tokens` (each key = one valid bearer token) in the controller namespace |
| Model | Single shared static-token allowlist; no per-user identity, roles, or scopes |
| UI auth | Azure AD (MSAL browser PKCE, public client) when configured |

## 6. Network Surface

| Port | Service | Deployed auth state |
|---|---|---|
| `:8080` | HTTP REST API (`/v0`, `/admin/v0`) + embedded static UI | `/admin/*` routes always authenticated and fail-closed (mandatory bearer token, independent of toggle); `/v0/*` public read open by design |
| `:8083` | MCP server (HTTP) | Bearer-token middleware; chart default `disableAuth: false` (auth enabled). Write/mutating tools fail-closed when auth disabled |
| `:8081` | Prometheus metrics | None |
| `:8082` | Health / readiness probes | None |

CORS: configurable via `AGENTREGISTRY_CORS_ORIGINS` (empty by default = same-origin only). Credentials are only enabled with an explicit origin allowlist and never combined with a `*` wildcard. Admin import is restricted to https with private/loopback/link-local/metadata IP egress blocked and no caller-header forwarding (SSRF hardening). Optional Gateway API HTTPRoute (disabled by default).

## 7. Dependency Vulnerabilities

Counts from `trivy fs` (vuln scanner), post-remediation. SBOMs: CycloneDX 1.6 via syft.

| Ecosystem | Manifest | CRITICAL | HIGH | MEDIUM | LOW |
|---|---|---|---|---|---|
| Go | `go.mod` | 0 | 0 | 0 | 0 |
| npm | `ui/package-lock.json` | 0 | 0 | 1 | 0 |

All CRITICAL and HIGH CVEs eliminated. The single residual npm MEDIUM is a transitive `postcss` advisory bundled inside `next`'s own dependency tree; the only available "fix" downgrades `next` to a vulnerable 9.x line, so it is intentionally not applied. Key bumps: `grpc`, `golang.org/x/crypto`, `golang.org/x/net`, `go.opentelemetry.io/otel`, `buger/jsonparser`, `modelcontextprotocol/registry` (Go); `next` → 16.2.x (npm).
