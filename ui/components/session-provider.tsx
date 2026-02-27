"use client"

import { createContext, useContext, useEffect, useState } from "react"
import { type AccountInfo } from "@azure/msal-browser"
import { initializeMsal, acquireToken, getAccount, isAuthEnabled } from "@/lib/msal"

interface AuthState {
  account: AccountInfo | null
  token: string | null
  status: "loading" | "authenticated" | "unauthenticated"
}

const AuthContext = createContext<AuthState>({
  account: null,
  token: null,
  status: "loading",
})

export function SessionProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AuthState>(() => {
    // Initialize state based on auth configuration to avoid sync setState in effect
    if (!isAuthEnabled()) {
      return { account: null, token: null, status: "unauthenticated" }
    }
    return { account: null, token: null, status: "loading" }
  })

  useEffect(() => {
    // Skip MSAL entirely when auth is disabled (AGENTREGISTRY_DISABLE_AUTH=true)
    if (!isAuthEnabled()) {
      return
    }

    initializeMsal()
      .then(async (redirectResult) => {
        // After redirect back from Azure AD, redirectResult has the tokens
        if (redirectResult) {
          setState({
            account: redirectResult.account,
            token: redirectResult.idToken,
            status: "authenticated",
          })
          return
        }

        // Check if we already have an account (e.g. returning user with cached session)
        const account = getAccount()
        if (!account) {
          // No account — trigger login redirect immediately (no button needed)
          await acquireToken()
          // acquireToken calls loginRedirect, page will navigate away
          return
        }

        // Have an account — acquire token silently
        const token = await acquireToken()
        if (token) {
          setState({ account, token, status: "authenticated" })
        }
        // If acquireToken returned null it triggered a redirect — state stays loading
      })
      .catch((err) => {
        // Config not loaded = /ui/config.js not served or missing env vars
        console.error("MSAL init error:", err)
        setState({ account: null, token: null, status: "unauthenticated" })
      })
  }, [])

  return <AuthContext.Provider value={state}>{children}</AuthContext.Provider>
}

export function useSession() {
  return useContext(AuthContext)
}
