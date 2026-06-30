# Security Audit — Agent Registry

Date: 2026-06-30
Status: **Clean** — audited and remediated. No outstanding HIGH/CRITICAL issues.

## Scope

- Source review of the Go controller, HTTP API, and embedded MCP server
- Next.js UI dependency posture
- Helm chart, RBAC, CRDs, and deployment manifest
- Dependency vulnerability scan (trivy) + SBOM (syft, CycloneDX 1.6) for Go and npm
- AI/LLM control review of the MCP sampling path

## Result

The application is in a secure posture:

- **Admin API is authenticated and fail-closed** — `/admin/*` always requires a
  valid token; public `/v0/*` read endpoints are open by design.
- **MCP write tools are fail-closed**, with rate limiting and a token budget on
  the sampling path and untrusted input isolated in prompts.
- **Server-side fetches are SSRF-safe** — https-only with private/loopback/
  link-local/metadata egress blocked.
- **Least-privilege RBAC** — Secret access is namespaced, not cluster-wide.
- **Deployments are namespace-constrained** to an allowlist.
- **CORS** never combines credentials with a wildcard origin.
- **Dependencies are current**: `trivy` reports **0 CRITICAL / 0 HIGH** in both
  `go.mod` and `ui/package-lock.json`.

## Verification

`make fmt`, `make lint`, and `make test` pass. Security behavior is covered by
unit tests (admin-auth enforcement, SSRF egress filter, deployment-namespace
allowlist, sampling rate/budget guard).

## Residual

One npm MEDIUM advisory remains in a transitive package bundled inside `next`'s
dependency tree; the only available downgrade reintroduces HIGH advisories, so it
is intentionally deferred to an upstream fix. No other open issues.
