import { signIn } from "@/auth"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { AlertCircle } from "lucide-react"

const isAzureADConfigured =
  process.env.AZURE_AD_CLIENT_ID &&
  process.env.AZURE_AD_CLIENT_SECRET &&
  process.env.AZURE_AD_TENANT_ID

export default function SignInPage() {
  if (!isAzureADConfigured) {
    return (
      <main className="min-h-screen bg-background flex items-center justify-center p-6">
        <Card className="max-w-md w-full p-8">
          <div className="text-center mb-6">
            <AlertCircle className="h-12 w-12 mx-auto mb-4 text-yellow-600" />
            <h1 className="text-2xl font-bold mb-2">Authentication Not Configured</h1>
            <p className="text-muted-foreground mb-4">
              Azure AD authentication is not set up yet.
            </p>
            <div className="text-left bg-muted p-4 rounded-md text-sm space-y-2">
              <p className="font-medium">To enable authentication:</p>
              <ol className="list-decimal list-inside space-y-1 text-muted-foreground">
                <li>Set up Azure AD app registration</li>
                <li>Configure environment variables</li>
                <li>Restart the application</li>
              </ol>
            </div>
            <p className="text-sm text-muted-foreground mt-4">
              See <code className="bg-muted px-2 py-1 rounded">docs/azure-ad-setup.md</code> for details.
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
            await signIn("azure-ad", { redirectTo: "/" })
          }}
        >
          <Button type="submit" className="w-full" size="lg">
            Sign in with Azure AD
          </Button>
        </form>

        <p className="text-sm text-muted-foreground text-center mt-6">
          You can browse resources without signing in. Authentication is only required for deploying to infrastructure.
        </p>
      </Card>
    </main>
  )
}
