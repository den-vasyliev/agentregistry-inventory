import { PublicClientApplication, type AccountInfo, type AuthenticationResult } from "@azure/msal-browser"

// Runtime config injected by Go binary at /ui/config.js
// Falls back to empty strings (will cause MSAL to fail gracefully in dev without auth)
declare global {
  interface Window {
    __APP_CONFIG__?: {
      tenantId: string
      clientId: string
      redirectUri?: string
      authEnabled?: boolean
    }
  }
}

function getConfig() {
  const cfg = typeof window !== "undefined" ? window.__APP_CONFIG__ : undefined
  return {
    tenantId: cfg?.tenantId ?? "",
    clientId: cfg?.clientId ?? "",
  }
}

export function isAuthEnabled(): boolean {
  const cfg = typeof window !== "undefined" ? window.__APP_CONFIG__ : undefined
  // Default false — auth is off unless explicitly enabled
  return cfg?.authEnabled === true
}

let _msalInstance: PublicClientApplication | null = null

export function getMsalInstance(): PublicClientApplication {
  // Only cache once config is available — prevents caching a broken instance
  // if called before /ui/config.js has executed.
  if (_msalInstance) return _msalInstance

  const { tenantId, clientId } = getConfig()
  if (!clientId || !tenantId) {
    throw new Error(
      "Azure AD config not loaded. Ensure /ui/config.js is served by the backend with AZURE_AD_CLIENT_ID and AZURE_AD_TENANT_ID set."
    )
  }

  _msalInstance = new PublicClientApplication({
    auth: {
      clientId,
      authority: `https://login.microsoftonline.com/${tenantId}`,
      // redirectUri set per-request in loginRedirect() so it's always current
    },
    cache: {
      cacheLocation: "sessionStorage",
      storeAuthStateInCookie: false,
    },
  })

  return _msalInstance
}

const API_SCOPES = ["openid", "profile", "email"]

// Call once on app start to handle redirect response
export async function initializeMsal(): Promise<AuthenticationResult | null> {
  const msal = getMsalInstance()
  await msal.initialize()
  return msal.handleRedirectPromise()
}

// Get active account
export function getAccount(): AccountInfo | null {
  const msal = getMsalInstance()
  const accounts = msal.getAllAccounts()
  return accounts[0] ?? null
}

// Acquire token silently, redirect to login if needed
export async function acquireToken(): Promise<string | null> {
  const msal = getMsalInstance()
  const account = getAccount()
  const redirectUri = getRedirectUri()

  if (!account) {
    await msal.loginRedirect({ scopes: API_SCOPES, redirectUri })
    return null // page will redirect
  }

  try {
    const result = await msal.acquireTokenSilent({ scopes: API_SCOPES, account })
    return result.idToken
  } catch {
    // Silent acquisition failed (token expired, consent needed, etc.) — redirect
    await msal.loginRedirect({ scopes: API_SCOPES, account, redirectUri })
    return null
  }
}

function getRedirectUri(): string {
  // Allow override via runtime config (served by Go binary from AZURE_AD_REDIRECT_URI env var).
  // Useful when the registered redirect URI differs from window.location.origin (e.g. behind a gateway).
  const cfg = typeof window !== "undefined" ? window.__APP_CONFIG__ : undefined
  if (cfg?.redirectUri) {
    return cfg.redirectUri
  }
  return window.location.pathname.startsWith("/ui/")
    ? `${window.location.origin}/ui/`
    : `${window.location.origin}/`
}

// Sign out and redirect back to app root
export async function signOut(): Promise<void> {
  const msal = getMsalInstance()
  const account = getAccount()
  await msal.logoutRedirect({
    account: account ?? undefined,
    postLogoutRedirectUri: getRedirectUri(),
  })
}
