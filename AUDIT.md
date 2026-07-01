# Security Audit — Agent Registry

Date: 2026-07-01
Status: **Clean** — audited, remediated, and re-verified. No outstanding HIGH/CRITICAL issues.

## Scope

- Source review of the Go controller, HTTP API, and embedded MCP server
- Next.js UI dependency posture
- Helm chart, RBAC, CRDs, and deployment manifest
- Dependency vulnerability scan (trivy) + SBOM (syft, CycloneDX 1.6) for Go and npm
- AI/LLM control review of the MCP sampling path

## Result

The application is in a secure posture:

- **Reads public, writes authenticated** — all read-only endpoints are served
  under public `/v0/*`; every mutating endpoint is served only under
  `/admin/*`, which is authenticated and fail-closed (a valid token is always
  required; an empty token allowlist rejects all admin requests).
- **Submission is non-mutating** — `/v0/submit` validates a repository manifest
  but does not write to the cluster.
- **Deployments are namespace-constrained** to an allowlist on both the HTTP and
  MCP paths.
- **MCP write tools are fail-closed**, with rate limiting and a token budget on
  the sampling path and untrusted input isolated in prompts.
- **Server-side fetches are SSRF-safe** — https-only with private/loopback/
  link-local/metadata egress blocked (including IPv4-mapped IPv6).
- **Least-privilege RBAC** — Secret access is namespaced, not cluster-wide.
- **CORS** never combines credentials with a wildcard origin.
- **Dependencies are current**: `trivy` reports **0 CRITICAL / 0 HIGH** in both
  `go.mod` and `ui/package-lock.json`.

## Verification

`make fmt`, `make lint`, and `make test` pass. Security behavior is covered by
unit tests (admin-auth enforcement, SSRF egress filter, deployment-namespace
allowlist, sampling rate/budget guard). A follow-up review of the public/admin
API split confirmed no mutating endpoint is reachable on the public `/v0`
prefix and surfaced one consistency gap (the MCP deploy path missing the
namespace allowlist), which has been fixed.

## Residual

One npm MEDIUM advisory remains in a transitive package bundled inside `next`'s
dependency tree; the only available downgrade reintroduces HIGH advisories, so it
is intentionally deferred to an upstream fix. No other open issues.
