import NextAuth from "next-auth"
import GoogleProvider from "next-auth/providers/google"
import type { OIDCConfig } from "next-auth/providers"

// Check for generic OIDC configuration (recommended)
const isOIDCConfigured =
  process.env.OIDC_ISSUER &&
  process.env.OIDC_CLIENT_ID &&
  process.env.OIDC_CLIENT_SECRET

// Fallback to provider-specific configurations
const isGoogleConfigured =
  process.env.GOOGLE_CLIENT_ID &&
  process.env.GOOGLE_CLIENT_SECRET

const isAzureConfigured =
  process.env.AZURE_AD_CLIENT_ID &&
  process.env.AZURE_AD_CLIENT_SECRET &&
  process.env.AZURE_AD_TENANT_ID

// Build provider list with priority: OIDC > Google > Azure
const providers = []

if (isOIDCConfigured) {
  // Generic OIDC provider (works with Keycloak, Auth0, Okta, etc.)
  providers.push({
    id: "oidc",
    name: process.env.OIDC_PROVIDER_NAME || "OIDC",
    type: "oidc",
    issuer: process.env.OIDC_ISSUER,
    clientId: process.env.OIDC_CLIENT_ID!,
    clientSecret: process.env.OIDC_CLIENT_SECRET!,
    authorization: {
      params: {
        scope: process.env.OIDC_SCOPE || "openid email profile groups"
      }
    },
  } as OIDCConfig<any>)
} else if (isGoogleConfigured) {
  // Fallback to Google OAuth
  providers.push(GoogleProvider({
    clientId: process.env.GOOGLE_CLIENT_ID!,
    clientSecret: process.env.GOOGLE_CLIENT_SECRET!,
  }))
} else if (isAzureConfigured) {
  // Fallback to Azure AD (using generic OIDC with Azure endpoint)
  providers.push({
    id: "azure-ad",
    name: "Azure AD",
    type: "oidc",
    issuer: `https://login.microsoftonline.com/${process.env.AZURE_AD_TENANT_ID}/v2.0`,
    clientId: process.env.AZURE_AD_CLIENT_ID!,
    clientSecret: process.env.AZURE_AD_CLIENT_SECRET!,
  } as OIDCConfig<any>)
}

export const { handlers, signIn, signOut, auth } = NextAuth({
  providers,
  callbacks: {
    async jwt({ token, account, profile }) {
      // Add OIDC info to token
      if (account) {
        token.accessToken = account.access_token
        token.idToken = account.id_token
      }
      if (profile) {
        token.email = profile.email
        token.name = profile.name
        if (Array.isArray((profile as { groups?: unknown }).groups)) {
          token.groups = (profile as { groups: string[] }).groups
        }
      }
      return token
    },
    async session({ session, token }) {
      // Add user info to session
      if (token.email) session.user.email = token.email as string
      if (token.name) session.user.name = token.name as string
      ;(session as { accessToken?: string; idToken?: string }).accessToken = token.accessToken as string | undefined
      ;(session as { accessToken?: string; idToken?: string }).idToken = token.idToken as string | undefined
      ;(session.user as { groups?: string[] }).groups = token.groups as string[] | undefined
      return session
    },
  },
  pages: {
    signIn: "/auth/signin",
  },
})
