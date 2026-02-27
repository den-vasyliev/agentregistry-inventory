# Azure AD Setup

Authentication uses **MSAL.js PKCE** — browser-side OAuth, no client secret required.

Auth is **disabled by default**. Enable it via Helm (`disableAuth: false`) or env var (`AGENTREGISTRY_AUTH_ENABLED=true`).

## Azure AD App Registration

1. Go to [Azure Portal](https://portal.azure.com) → **Azure Active Directory** → **App registrations** → **New registration**
2. Configure:
   - **Name**: `Agent Registry`
   - **Supported account types**: Accounts in this organizational directory only
   - **Redirect URI**: Platform **Single-page application (SPA)**
     - `http://localhost:8080/` (local dev)
     - `https://your-domain.com/` (prod)
3. Click **Register**

> **Important**: Use **Single-page application** platform type, not Web. This enables PKCE and removes the client secret requirement.

## Get Values

After registration note:
- **Application (client) ID** → `clientId`
- **Directory (tenant) ID** → `tenantId`

No client secret needed.

## API Permissions

1. Go to **API permissions** → **Add a permission** → **Microsoft Graph** → **Delegated**
2. Add: `openid`, `profile`, `email`
3. Click **Grant admin consent**

## Auth Flow

```
Browser → /             (no auth, loads UI + config.js)
Browser → Azure AD      (PKCE loginRedirect, auto-triggered)
Azure AD → /?code=…     (redirect back with auth code)
MSAL → exchanges code   (PKCE verifier, returns id_token)
Browser → /api/*        (Authorization: Bearer <id_token>)
Gateway → validates JWT (Azure AD JWKS, no secret)
Go backend ← request    (passthrough, auth handled by gateway)
```

## Helm Configuration

```yaml
# charts/agentregistry/values.yaml
disableAuth: false   # enable backend token checks

azure:
  tenantId: "your-tenant-id"
  clientId: "your-client-id"
  # redirectUri: override if window.location.origin doesn't match registered URI
  # redirectUri: "https://your-domain.com/"
```

The Go binary reads these at startup and serves them via `/config.js`. MSAL loads the config on page load and redirects to Azure AD if no session exists.

## Gateway (agentgateway)

For production, deploy [agentgateway](../charts/agentgateway/) in front of the controller to handle JWT validation. The controller backend runs with `disableAuth: true` and trusts the gateway to enforce auth.

See [config/agentgateway/config.yaml](../config/agentgateway/config.yaml) for the full gateway config with JWT rules.

## Local Development

```bash
export AZURE_AD_TENANT_ID=your-tenant-id
export AZURE_AD_CLIENT_ID=your-client-id
make run
```

Open `http://localhost:8080` — the browser redirects to Azure AD automatically.

Add `http://localhost:8080/` as a SPA redirect URI in your Azure AD app registration.

## Redirect URI Override

If the registered redirect URI doesn't match `window.location.origin` (e.g. during migration from an old NextAuth setup), set an override:

```bash
export AZURE_AD_REDIRECT_URI=https://your-domain.com/old/callback/path
make run
```

Or via Helm:
```yaml
azure:
  redirectUri: "https://your-domain.com/old/callback/path"
```

## Troubleshooting

**`AADSTS900971: No reply address provided`** — The redirect URI sent to Azure AD is empty or not registered. Check that the URI listed in **Authentication → Single-page application** exactly matches your app's origin (including trailing slash).

**`redirect_uri_mismatch`** — Add the URI with trailing slash as a SPA redirect URI in Azure AD.

**`AADSTS700054: response_type 'token' is not enabled`** — Ensure the app is registered as SPA, not Web.

**No redirect to Azure AD** — Check browser console for `[MSAL]` errors. Verify `/config.js` returns the correct config: `curl http://localhost:8080/config.js`

**Token not sent to backend** — Check Network tab; `Authorization: Bearer` header should appear on `/api/admin/v0/` requests.
