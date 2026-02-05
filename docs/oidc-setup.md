# OIDC Authentication Setup

This guide explains how to configure OIDC authentication for Agent Registry, covering both the **Web UI** (NextAuth.js) and the **Controller API** (JWT validation).

## Overview

Agent Registry supports a two-tier authentication system:

1. **UI Authentication** (NextAuth.js) - For web interface login
2. **API Authentication** (OIDC JWT) - For controller API access

### Authentication Flow

```
User → Sign in via UI → OIDC Provider → JWT Token → API Requests
```

The UI authenticates users via OIDC and receives JWT tokens, which are then passed to the controller API for validation.

## Supported Providers

Agent Registry supports any OIDC-compliant provider:

- **Generic OIDC** (Keycloak, Auth0, Okta, etc.) - Recommended
- **Google OAuth** - Backward compatibility
- **Azure AD** - Backward compatibility

**Priority:** Generic OIDC > Google > Azure AD

## Architecture

### UI (NextAuth.js)
- Handles user authentication flow
- Manages sessions and tokens
- Proxy requests to controller API with JWT

### Controller (Go)
- Validates JWT tokens from OIDC provider
- Caches validated tokens for performance
- Enforces group-based authorization

## Setup Guide

### 1. Choose Your OIDC Provider

Pick one of the following:

<details>
<summary><b>Keycloak (Recommended for self-hosted)</b></summary>

#### Create a Keycloak Client

1. Login to Keycloak Admin Console
2. Select your realm
3. Go to **Clients** → **Create**
4. Configure:
   - **Client ID**: `agentregistry`
   - **Client Protocol**: `openid-connect`
   - **Access Type**: `confidential`
   - **Valid Redirect URIs**: `http://localhost:3000/api/auth/callback/oidc`
   - **Web Origins**: `http://localhost:3000`

5. Go to **Credentials** tab
6. Copy **Secret** → `OIDC_CLIENT_SECRET`

7. Note these values:
   - **Issuer URL**: `https://your-keycloak.com/realms/your-realm`
   - **Client ID**: `agentregistry`
   - **Client Secret**: From step 6

</details>

<details>
<summary><b>Auth0</b></summary>

#### Create an Auth0 Application

1. Login to [Auth0 Dashboard](https://manage.auth0.com)
2. Go to **Applications** → **Create Application**
3. Choose **Regular Web Application**
4. Configure:
   - **Name**: `Agent Registry`
   - **Allowed Callback URLs**: `http://localhost:3000/api/auth/callback/oidc`
   - **Allowed Logout URLs**: `http://localhost:3000`
   - **Allowed Web Origins**: `http://localhost:3000`

5. Go to **Settings** tab
6. Note these values:
   - **Domain**: `your-tenant.auth0.com` → Issuer: `https://your-tenant.auth0.com`
   - **Client ID** → `OIDC_CLIENT_ID`
   - **Client Secret** → `OIDC_CLIENT_SECRET`

</details>

<details>
<summary><b>Okta</b></summary>

#### Create an Okta Application

1. Login to [Okta Admin Console](https://your-org.okta.com)
2. Go to **Applications** → **Create App Integration**
3. Choose **OIDC - OpenID Connect**
4. Choose **Web Application**
5. Configure:
   - **App integration name**: `Agent Registry`
   - **Sign-in redirect URIs**: `http://localhost:3000/api/auth/callback/oidc`
   - **Sign-out redirect URIs**: `http://localhost:3000`
   - **Assignments**: Choose who can access

6. Note these values:
   - **Issuer**: `https://your-org.okta.com` (or your custom domain)
   - **Client ID** → `OIDC_CLIENT_ID`
   - **Client Secret** → `OIDC_CLIENT_SECRET`

</details>

<details>
<summary><b>Google OAuth</b></summary>

Note: You can also use Google via Generic OIDC (see below).

1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Create OAuth 2.0 credentials
3. Configure authorized redirect URIs
4. Use environment variables:
   ```bash
   GOOGLE_CLIENT_ID=your-client-id
   GOOGLE_CLIENT_SECRET=your-client-secret
   ```

</details>

<details>
<summary><b>Azure AD</b></summary>

Note: You can also use Azure AD via Generic OIDC (see Migration section below). For detailed Azure-specific setup, see the [Azure AD Setup Guide](./azure-ad-setup.md).

1. Register app in Azure Portal
2. Use environment variables:
   ```bash
   AZURE_AD_CLIENT_ID=your-client-id
   AZURE_AD_CLIENT_SECRET=your-client-secret
   AZURE_AD_TENANT_ID=your-tenant-id
   ```

</details>

### 2. Configure the UI (NextAuth.js)

The UI uses NextAuth.js for authentication. Configure via environment variables:

#### Development (.env.local)

```bash
# NextAuth
AUTH_SECRET=$(openssl rand -base64 32)
AUTH_URL=http://localhost:3000

# Generic OIDC (Recommended)
OIDC_ISSUER=https://your-provider.com
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret
OIDC_PROVIDER_NAME="Your Provider"  # Optional, display name
OIDC_SCOPE="openid email profile groups"  # Optional

# API URL
NEXT_PUBLIC_API_URL=http://localhost:8080
```

#### Production

Set the same variables in your deployment platform:

- **Kubernetes**: ConfigMap or Secret
- **Vercel**: Project Settings → Environment Variables
- **Docker**: Pass via `-e` flags or docker-compose

### 3. Configure the Controller (API)

The controller validates JWT tokens from your OIDC provider.

#### Environment Variables

```bash
# OIDC Configuration (Required)
AGENTREGISTRY_OIDC_ISSUER=https://your-provider.com
AGENTREGISTRY_OIDC_AUDIENCE=your-client-id

# Group-based Authorization (Optional)
AGENTREGISTRY_OIDC_ADMIN_GROUP=agentregistry-admins
AGENTREGISTRY_OIDC_GROUP_CLAIM=groups  # Default: "groups"

# Token Cache Configuration (Optional)
AGENTREGISTRY_OIDC_CACHE_MARGIN_SECONDS=300  # Default: 5 minutes

# Disable Auth (Development Only)
AGENTREGISTRY_DISABLE_AUTH=false  # Default: false (auth enabled)
```

#### Kubernetes Deployment

Add to your controller deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agentregistry-controller
  namespace: agentregistry
spec:
  template:
    spec:
      containers:
      - name: controller
        image: your-registry/agentregistry-controller:latest
        env:
        - name: AGENTREGISTRY_OIDC_ISSUER
          value: "https://your-provider.com"
        - name: AGENTREGISTRY_OIDC_AUDIENCE
          value: "agentregistry"
        - name: AGENTREGISTRY_OIDC_ADMIN_GROUP
          value: "agentregistry-admins"
```

### 4. Configure Group-based Authorization (Optional)

If you want to restrict deployment operations to specific groups:

#### Keycloak

1. Create a group: `agentregistry-admins`
2. Add users to the group
3. Configure client mappers:
   - **Mapper Type**: Group Membership
   - **Token Claim Name**: `groups`
   - **Add to ID token**: ON
   - **Add to access token**: ON

#### Auth0

1. Create a role: `agentregistry-admins`
2. Assign users to the role
3. Add a rule to include groups in tokens:

```javascript
function (user, context, callback) {
  const namespace = 'https://your-app.com/';
  context.idToken[namespace + 'groups'] = user.app_metadata?.groups || [];
  callback(null, user, context);
}
```

#### Okta

1. Create a group: `agentregistry-admins`
2. Add users to the group
3. Configure claims:
   - Go to **Security** → **API** → **Authorization Servers**
   - Edit your authorization server
   - Add a claim:
     - **Name**: `groups`
     - **Include in**: ID Token
     - **Value**: `getFilteredGroups(app.profile, "agentregistry", 100)`

#### Controller Configuration

Set the admin group in controller env:

```bash
AGENTREGISTRY_OIDC_ADMIN_GROUP=agentregistry-admins
AGENTREGISTRY_OIDC_GROUP_CLAIM=groups
```

## Testing

### 1. Test UI Authentication

```bash
# Start UI
cd ui
npm run dev
```

1. Visit http://localhost:3000
2. Click **Sign In**
3. Authenticate with your OIDC provider
4. You should be redirected back with a session

### 2. Test API Authentication

```bash
# Start controller
make dev-controller
```

Get a JWT token from your UI session and test the API:

```bash
# Get the JWT from browser DevTools (Application → Storage → Cookies)
# Or from the session

# Test authenticated endpoint
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  http://localhost:8080/admin/v0/deployments
```

### 3. Test Group Authorization (if configured)

```bash
# User WITHOUT admin group should get 403
curl -H "Authorization: Bearer USER_JWT_TOKEN" \
  -X POST \
  http://localhost:8080/admin/v0/deployments \
  -d '{"resourceName":"test/server","version":"1.0.0","resourceType":"mcp"}'

# Expected: 403 Forbidden

# User WITH admin group should succeed
curl -H "Authorization: Bearer ADMIN_JWT_TOKEN" \
  -X POST \
  http://localhost:8080/admin/v0/deployments \
  -d '{"resourceName":"test/server","version":"1.0.0","resourceType":"mcp"}'

# Expected: 200 OK
```

## Architecture Details

### Token Caching

The controller implements an LRU cache for validated JWT tokens:

- **Cache Size**: 1000 entries
- **Eviction**: Least Recently Used (LRU)
- **TTL**: Token expiry - 5 minutes (safety margin)
- **Cleanup**: Background goroutine runs every 1 minute

**Performance Impact**: Reduces OIDC validation calls by >95% under normal load.

### Token Validation Flow

```
1. Request arrives with Bearer token
2. Controller checks cache (SHA-256 hash of token)
3. Cache hit → Return cached claims (fast path)
4. Cache miss → Validate with OIDC provider
5. Store validated claims in cache
6. Check group membership (if required)
7. Allow/deny request
```

### Security Considerations

- **Token Hashing**: Tokens are hashed with SHA-256 before caching (never stored in plaintext)
- **Safety Margin**: Cache expires 5 minutes before token expiry to prevent race conditions
- **Group Validation**: Admin group is checked on every request (even with cache hit)
- **Thread Safety**: Cache uses sync.RWMutex for concurrent access

## Environment Variables Reference

### UI Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `AUTH_SECRET` | Yes | - | NextAuth.js secret (generate with `openssl rand -base64 32`) |
| `AUTH_URL` | Yes | - | Base URL for auth callbacks |
| `OIDC_ISSUER` | No* | - | OIDC provider URL |
| `OIDC_CLIENT_ID` | No* | - | OIDC client ID |
| `OIDC_CLIENT_SECRET` | No* | - | OIDC client secret |
| `OIDC_PROVIDER_NAME` | No | "OIDC" | Display name for provider |
| `OIDC_SCOPE` | No | "openid email profile groups" | OAuth scopes |
| `GOOGLE_CLIENT_ID` | No* | - | Google OAuth client ID (legacy) |
| `GOOGLE_CLIENT_SECRET` | No* | - | Google OAuth secret (legacy) |
| `AZURE_AD_CLIENT_ID` | No* | - | Azure AD client ID (legacy) |
| `AZURE_AD_CLIENT_SECRET` | No* | - | Azure AD secret (legacy) |
| `AZURE_AD_TENANT_ID` | No* | - | Azure AD tenant ID (legacy) |
| `NEXT_PUBLIC_API_URL` | Yes | - | Controller API URL |
| `NEXT_PUBLIC_DISABLE_AUTH` | No | "false" | Disable auth for dev |

**Note**: At least one provider (OIDC, Google, or Azure) must be configured.

### Controller Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `AGENTREGISTRY_OIDC_ISSUER` | Yes* | - | OIDC provider URL for JWT validation |
| `AGENTREGISTRY_OIDC_AUDIENCE` | Yes* | - | Expected JWT audience (client ID) |
| `AGENTREGISTRY_OIDC_ADMIN_GROUP` | No | - | Required group for deploy operations |
| `AGENTREGISTRY_OIDC_GROUP_CLAIM` | No | "groups" | Claim name containing user groups |
| `AGENTREGISTRY_OIDC_CACHE_MARGIN_SECONDS` | No | 300 | Cache expiry safety margin (seconds) |
| `AGENTREGISTRY_DISABLE_AUTH` | No | "false" | Disable auth (dev only) |

**Note**: Required when auth is enabled (`AGENTREGISTRY_DISABLE_AUTH != "true"`).

## Troubleshooting

### UI Issues

#### "No authentication provider configured"

- Verify you've set either `OIDC_ISSUER`, `GOOGLE_CLIENT_ID`, or `AZURE_AD_CLIENT_ID`
- Check `.env.local` is in the `ui/` directory
- Restart the dev server after changing env vars

#### "Redirect URI mismatch"

- Verify redirect URI in provider matches: `http://localhost:3000/api/auth/callback/oidc`
- For production, update to your actual domain
- Check for trailing slashes

#### "Invalid client secret"

- Regenerate secret in your OIDC provider
- Update `OIDC_CLIENT_SECRET` in `.env.local`
- Restart the UI

### Controller Issues

#### "Missing authorization header"

- Ensure the UI is sending the JWT token
- Check browser DevTools → Network → Headers
- Verify the API proxy is working (`ui/app/api/registry/[...path]/route.ts`)

#### "Token expired"

- Token lifetime is controlled by your OIDC provider
- Increase token lifetime in provider settings
- Or reduce cache safety margin: `AGENTREGISTRY_OIDC_CACHE_MARGIN_SECONDS=60`

#### "Missing required admin group"

- Verify user is member of the configured admin group
- Check JWT token includes groups claim (decode at jwt.io)
- Verify `AGENTREGISTRY_OIDC_GROUP_CLAIM` matches claim name in JWT
- If groups claim is nested, you may need to configure your provider

#### "Invalid token signature"

- Verify `AGENTREGISTRY_OIDC_ISSUER` matches exactly (no trailing slash)
- Check controller can reach OIDC provider's `.well-known/openid-configuration`
- Verify `AGENTREGISTRY_OIDC_AUDIENCE` matches the JWT `aud` claim

### Cache Issues

#### High OIDC provider load

- Increase cache size (requires code change in `token_cache.go`)
- Increase token lifetime in provider
- Check logs for cache hit/miss ratio

#### Stale permissions

- Tokens are cached until near-expiry
- Reduce `AGENTREGISTRY_OIDC_CACHE_MARGIN_SECONDS` for faster permission updates
- Or restart controller to clear cache

## Development Mode

For local development without OIDC:

```bash
# UI
NEXT_PUBLIC_DISABLE_AUTH=false  # Keep auth for UI testing
AUTH_SECRET=dev-secret-only

# Controller
AGENTREGISTRY_DISABLE_AUTH=true  # Disable API auth checks
```

⚠️ **Never use `DISABLE_AUTH=true` in production!**

## Migration from Provider-Specific to Generic OIDC

If you're currently using Google or Azure AD, you can migrate to generic OIDC:

### From Google OAuth

```bash
# Old (still supported)
GOOGLE_CLIENT_ID=xxx
GOOGLE_CLIENT_SECRET=yyy

# New (recommended)
OIDC_ISSUER=https://accounts.google.com
OIDC_CLIENT_ID=xxx  # Same as GOOGLE_CLIENT_ID
OIDC_CLIENT_SECRET=yyy  # Same as GOOGLE_CLIENT_SECRET
OIDC_PROVIDER_NAME="Google"
```

### From Azure AD

```bash
# Old (still supported)
AZURE_AD_CLIENT_ID=xxx
AZURE_AD_CLIENT_SECRET=yyy
AZURE_AD_TENANT_ID=zzz

# New (recommended)
OIDC_ISSUER=https://login.microsoftonline.com/zzz/v2.0
OIDC_CLIENT_ID=xxx
OIDC_CLIENT_SECRET=yyy
OIDC_PROVIDER_NAME="Azure AD"
```

## Security Best Practices

1. **Rotate Secrets Regularly**
   - Client secrets should be rotated every 6-12 months
   - Use secret management tools (Vault, AWS Secrets Manager, etc.)

2. **Use Separate Providers for Environments**
   - Dev, staging, and prod should have separate OIDC clients
   - Prevents accidental production access from dev tokens

3. **Enable Group-based Authorization**
   - Don't rely on authentication alone
   - Use groups to control who can deploy

4. **Monitor Token Validation**
   - Check controller logs for authentication failures
   - Set up alerts for high failure rates

5. **Secure Token Storage**
   - Tokens are cached in memory only (never persisted to disk)
   - SHA-256 hashed before caching
   - Cache is cleared on controller restart

6. **Network Security**
   - Controller must reach OIDC provider (allow outbound HTTPS)
   - Use TLS for all production deployments
   - Consider IP allowlisting for admin endpoints

## References

- [NextAuth.js Documentation](https://next-auth.js.org/)
- [OIDC Specification](https://openid.net/specs/openid-connect-core-1_0.html)
- [JWT Best Practices](https://datatracker.ietf.org/doc/html/rfc8725)
- [Keycloak Documentation](https://www.keycloak.org/documentation)
- [Auth0 Documentation](https://auth0.com/docs)
- [Okta Documentation](https://developer.okta.com/docs/)
