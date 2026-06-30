# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project aims to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Security

- Admin HTTP API (`/admin/*`) is now authenticated and **fail-closed**: a valid
  bearer token is always required, independent of the auth toggle, and requests
  are rejected when no tokens are configured. Public `/v0/*` read endpoints
  remain open by design.
- The `/admin/v0/submit` endpoint (registered directly on the mux) is now wrapped
  with the same mandatory admin auth.
- Hardened the admin import fetcher against SSRF: https-only, egress to
  private/loopback/link-local/cloud-metadata addresses blocked after DNS
  resolution, redirects no longer followed, caller-supplied headers no longer
  forwarded, upstream response bodies no longer reflected, and response size
  capped.
- Restricted catalog deployment to an allowlist of target namespaces
  (`AGENTREGISTRY_ALLOWED_DEPLOY_NAMESPACES`, defaulting to the controller
  namespace) so the controller's RBAC cannot be used to schedule workloads into
  arbitrary namespaces.
- MCP write/mutating tools are now fail-closed (refused when auth is disabled).
- Added a call-rate limit and aggregate token budget on the MCP sampling path
  (env-tunable) to bound LLM cost and abuse.
- Isolated untrusted user/catalog input in MCP recommendation prompts to reduce
  prompt-injection exposure.
- CORS credentials are enabled only with an explicit origin allowlist and never
  combined with a wildcard origin.
- Scoped Secret read access to a namespaced `Role`/`RoleBinding` (controller
  namespace) instead of cluster-wide.

### Changed

- Bumped Go dependencies to remediate HIGH/CRITICAL CVEs: `google.golang.org/grpc`,
  `golang.org/x/crypto`, `golang.org/x/net`, `go.opentelemetry.io/otel`,
  `github.com/buger/jsonparser`, and `github.com/modelcontextprotocol/registry`.
- Bumped `next` to a fixed release line (≥ 16.2.6) to remediate UI dependency CVEs.
- Minimum Go toolchain is now **1.26** (inherited from an upgraded dependency).
- Helm chart now defaults to auth-enabled (`disableAuth: false`).

### Added

- Optional `Application` (`app.k8s.io/v1beta1`) manifest for the Helm chart,
  gated by `application.create` (default `false`), grouping all chart components
  under a single Application object for GKE Marketplace / tooling.
- `make`-verified security regression tests for admin-auth enforcement, the SSRF
  egress filter, the deployment-namespace allowlist, and the sampling guard.

### Fixed

- Resolved a latent duplicate operation-ID collision in the environments API
  handler that surfaced as a router panic when both the public and admin route
  groups were registered.
