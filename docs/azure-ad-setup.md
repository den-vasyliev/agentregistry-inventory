# Azure AD Authentication Setup

This guide explains how to configure Azure AD authentication for the Agent Registry web UI.

## Prerequisites

- Azure AD tenant
- Admin access to Azure Portal

## Azure AD Configuration

### 1. Register Application

1. Go to [Azure Portal](https://portal.azure.com)
2. Navigate to **Azure Active Directory** → **App registrations**
3. Click **New registration**
4. Configure:
   - **Name**: `Agent Registry`
   - **Supported account types**: Accounts in this organizational directory only
   - **Redirect URI**:
     - Platform: **Web**
     - URL: `http://localhost:3000/api/auth/callback/azure-ad` (for dev)
     - URL: `https://your-domain.com/api/auth/callback/azure-ad` (for prod)
5. Click **Register**

### 2. Get Credentials

After registration, note these values:

- **Application (client) ID** → `AZURE_AD_CLIENT_ID`
- **Directory (tenant) ID** → `AZURE_AD_TENANT_ID`

### 3. Create Client Secret

1. Go to **Certificates & secrets**
2. Click **New client secret**
3. Add description: `Agent Registry Auth`
4. Choose expiration (recommended: 12-24 months)
5. Click **Add**
6. **Copy the secret value immediately** → `AZURE_AD_CLIENT_SECRET`
   - ⚠️ This value won't be shown again!

### 4. Configure API Permissions (Optional)

If you need to read user groups/roles:

1. Go to **API permissions**
2. Click **Add a permission**
3. Select **Microsoft Graph**
4. Select **Delegated permissions**
5. Add: `User.Read`, `email`, `profile`, `openid`
6. Click **Add permissions**
7. Click **Grant admin consent** (if you have admin rights)

## Environment Configuration

### Local Development

1. Copy the example env file:
   ```bash
   cd ui
   cp .env.example .env.local
   ```

2. Fill in your Azure AD values in `.env.local`:
   ```bash
   AUTH_SECRET=$(openssl rand -base64 32)
   AUTH_URL=http://localhost:3000

   AZURE_AD_CLIENT_ID=your-client-id-here
   AZURE_AD_CLIENT_SECRET=your-client-secret-here
   AZURE_AD_TENANT_ID=your-tenant-id-here

   NEXT_PUBLIC_API_URL=http://localhost:8080
   ```

### Production

Set environment variables in your deployment platform:

- Vercel: Project Settings → Environment Variables
- Docker: Pass via `-e` flags or docker-compose
- Kubernetes: ConfigMap/Secret

## Security Notes

- **Never commit** `.env.local` or `.env` files
- Client secrets should be rotated periodically
- Use separate Azure AD apps for dev/staging/prod
- Consider using managed identities in production

## Testing

1. Start the UI: `npm run dev`
2. Visit http://localhost:3000
3. Click **Sign In** in the navigation
4. Sign in with your Azure AD credentials
5. You should be redirected back with authentication

## Troubleshooting

### "Redirect URI mismatch"
- Verify the redirect URI in Azure AD matches exactly
- Check that you're using the correct environment

### "Invalid client secret"
- Generate a new client secret
- Update `AZURE_AD_CLIENT_SECRET` in your env

### "AADSTS50011: The reply URL is not configured"
- Add all callback URLs to Azure AD redirect URIs
- Include both `http://localhost:3000` and production URL

### User can't deploy/remove
- Check that the session is being set correctly
- Verify middleware is protecting the routes
- Check browser console for errors
