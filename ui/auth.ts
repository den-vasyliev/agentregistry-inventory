import NextAuth from "next-auth"
import AzureADProvider from "next-auth/providers/azure-ad"

// Check if Azure AD is configured
const isAzureADConfigured =
  process.env.AZURE_AD_CLIENT_ID &&
  process.env.AZURE_AD_CLIENT_SECRET &&
  process.env.AZURE_AD_TENANT_ID

export const { handlers, signIn, signOut, auth } = NextAuth({
  providers: isAzureADConfigured ? [
    AzureADProvider({
      clientId: process.env.AZURE_AD_CLIENT_ID!,
      clientSecret: process.env.AZURE_AD_CLIENT_SECRET!,
      issuer: `https://login.microsoftonline.com/${process.env.AZURE_AD_TENANT_ID}/v2.0`,
    }),
  ] : [],
  callbacks: {
    async jwt({ token, account, profile }) {
      // Add Azure AD info to token
      if (account) {
        token.accessToken = account.access_token
        token.idToken = account.id_token
      }
      if (profile) {
        token.email = profile.email
        token.name = profile.name
      }
      return token
    },
    async session({ session, token }) {
      // Add user info to session
      if (token.email) session.user.email = token.email as string
      if (token.name) session.user.name = token.name as string
      return session
    },
  },
  pages: {
    signIn: "/auth/signin",
  },
})
