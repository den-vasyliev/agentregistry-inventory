import { signIn } from "@/auth"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { AlertCircle } from "lucide-react"
import { redirect } from "next/navigation"

// Detect configured provider
const isOIDCConfigured =
  process.env.OIDC_ISSUER &&
  process.env.OIDC_CLIENT_ID &&
  process.env.OIDC_CLIENT_SECRET

const isGoogleConfigured =
  process.env.GOOGLE_CLIENT_ID &&
  process.env.GOOGLE_CLIENT_SECRET

const isAzureConfigured =
  process.env.AZURE_AD_CLIENT_ID &&
  process.env.AZURE_AD_CLIENT_SECRET &&
  process.env.AZURE_AD_TENANT_ID

const disableAuth = process.env.NEXT_PUBLIC_DISABLE_AUTH !== "false"

// Determine provider name and ID
const providerName = isOIDCConfigured
  ? (process.env.OIDC_PROVIDER_NAME || "OIDC")
  : isGoogleConfigured
  ? "Google"
  : isAzureConfigured
  ? "Azure AD"
  : null

const providerId = isOIDCConfigured
  ? "oidc"
  : isGoogleConfigured
  ? "google"
  : isAzureConfigured
  ? "azure-ad"
  : null

export default function SignInPage() {
  if (disableAuth) {
    redirect("/")
  }

  if (!providerId) {
    return (
      <main className="min-h-screen bg-background flex items-center justify-center p-6">
        <Card className="max-w-md w-full p-8">
          <div className="text-center mb-6">
            <AlertCircle className="h-12 w-12 mx-auto mb-4 text-yellow-600" />
            <h1 className="text-2xl font-bold mb-2">Authentication Not Configured</h1>
            <p className="text-muted-foreground mb-4">
              No authentication provider is configured.
            </p>
            <div className="text-left bg-muted p-4 rounded-md text-sm space-y-2">
              <p className="font-medium">To enable authentication:</p>
              <ol className="list-decimal list-inside space-y-1 text-muted-foreground">
                <li>Choose an OIDC provider (Keycloak, Auth0, Okta, Google, Azure)</li>
                <li>Configure environment variables (see below)</li>
                <li>Restart the application</li>
              </ol>
            </div>
            <div className="text-left bg-muted p-4 rounded-md text-sm mt-4">
              <p className="font-medium mb-2">Generic OIDC (recommended):</p>
              <code className="text-xs block">
                OIDC_ISSUER=https://your-provider.com<br />
                OIDC_CLIENT_ID=your-client-id<br />
                OIDC_CLIENT_SECRET=your-secret<br />
                OIDC_PROVIDER_NAME=Your Provider
              </code>
            </div>
            <p className="text-sm text-muted-foreground mt-4">
              See <code className="bg-muted px-2 py-1 rounded">docs/oidc-setup.md</code> for complete setup guide.
            </p>
          </div>
        </Card>
      </main>
    )
  }

  return (
    <main className="min-h-screen bg-background flex items-center justify-center p-6">
      <Card className="max-w-md w-full p-8">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold mb-2">Sign In</h1>
          <p className="text-muted-foreground">
            Sign in to deploy and manage resources
          </p>
        </div>

        <form
          action={async () => {
            "use server"
            await signIn(providerId, { redirectTo: "/" })
          }}
        >
          <Button type="submit" className="w-full" size="lg">
            Sign in with {providerName}
          </Button>
        </form>

        <p className="text-sm text-muted-foreground text-center mt-6">
          You can browse resources without signing in. Authentication is only required for deploying to infrastructure.
        </p>
      </Card>
    </main>
  )
}
