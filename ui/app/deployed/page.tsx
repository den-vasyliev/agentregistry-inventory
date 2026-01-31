"use client"

import { useEffect, useState } from "react"
import Link from "next/link"
import { useSession } from "next-auth/react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { adminApiClient, createAuthenticatedClient } from "@/lib/admin-api"
import { Trash2, AlertCircle, Calendar, Package, Copy, Check, Search, LogIn, Bot } from "lucide-react"
import MCPIcon from "@/components/icons/mcp"
import { toast } from "sonner"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"

type DeploymentResponse = {
  serverName: string
  version: string
  deployedAt: string
  updatedAt: string
  status: string
  config: Record<string, string>
  preferRemote: boolean
  resourceType: string // "mcp" or "agent"
  runtime: string
  isExternal?: boolean // true if not managed by registry
}

export default function DeployedPage() {
  const { data: session } = useSession()
  const [activeTab, setActiveTab] = useState("mcp")
  const [deployments, setDeployments] = useState<DeploymentResponse[]>([])
  const [filteredDeployments, setFilteredDeployments] = useState<DeploymentResponse[]>([])
  const [searchQuery, setSearchQuery] = useState("")
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [removing, setRemoving] = useState(false)
  const [serverToRemove, setServerToRemove] = useState<{ name: string, version: string, resourceType: string } | null>(null)
  const [copied, setCopied] = useState(false)

  // Pagination state
  const [currentPageServers, setCurrentPageServers] = useState(1)
  const [currentPageAgents, setCurrentPageAgents] = useState(1)
  const itemsPerPage = 5

  // Gateway URL - get from controller config or use default
  const gatewayUrl = process.env.NEXT_PUBLIC_GATEWAY_URL || "https://kagent.ops.gfknewron.com"

  const copyToClipboard = () => {
    navigator.clipboard.writeText(gatewayUrl)
    setCopied(true)
    toast.success("Gateway URL copied to clipboard!")
    setTimeout(() => setCopied(false), 2000)
  }

  const fetchDeployments = async () => {
    try {
      setError(null)
      // Use authenticated client if session available
      const client = session?.accessToken
        ? createAuthenticatedClient(session.accessToken)
        : adminApiClient
      // listDeployments now returns both managed and external K8s resources
      const deployData = await client.listDeployments()
      setDeployments(deployData)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch deployments')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchDeployments()
    // Refresh every 30 seconds
    const interval = setInterval(fetchDeployments, 30000)
    return () => clearInterval(interval)
  }, [])

  // Filter deployments by search query
  useEffect(() => {
    const query = searchQuery.toLowerCase()
    setFilteredDeployments(
      deployments.filter(d =>
        d.serverName.toLowerCase().includes(query) ||
        d.version.toLowerCase().includes(query)
      )
    )
  }, [searchQuery, deployments])

  const handleRemove = async (serverName: string, version: string, resourceType: string) => {
    if (!session) {
      toast.error("Please sign in to remove deployments")
      return
    }
    setServerToRemove({ name: serverName, version, resourceType })
  }

  const confirmRemove = async () => {
    if (!serverToRemove) return

    try {
      setRemoving(true)
      // Use authenticated client for admin operation
      const client = session?.accessToken
        ? createAuthenticatedClient(session.accessToken)
        : adminApiClient
      await client.removeDeployment(serverToRemove.name, serverToRemove.version, serverToRemove.resourceType)
      // Remove from local state
      setDeployments(prev => prev.filter(d => d.serverName !== serverToRemove.name || d.version !== serverToRemove.version || d.resourceType !== serverToRemove.resourceType))
      setServerToRemove(null)
      fetchDeployments()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to remove deployment')
    } finally {
      setRemoving(false)
    }
  }

  const runningCount = filteredDeployments.length
  const agents = filteredDeployments.filter(d => d.resourceType === 'agent')
  const mcpServers = filteredDeployments.filter(d => d.resourceType === 'mcp')

  return (
    <main className="min-h-screen bg-background">
      {/* Stats Section */}
      <div className="bg-muted/30 border-b">
        <div className="container mx-auto px-6 py-6">
          <div className="grid gap-4 md:grid-cols-3">
            <Card className="p-4 hover:shadow-md transition-all duration-200 border hover:border-primary/20">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-green-500/10 rounded-lg flex items-center justify-center">
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                    strokeWidth={2}
                    stroke="currentColor"
                    className="h-5 w-5 text-green-600"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                    />
                  </svg>
                </div>
                <div>
                  <p className="text-2xl font-bold">{runningCount}</p>
                  <p className="text-xs text-muted-foreground">Total Resources</p>
                </div>
              </div>
            </Card>

            <Card className="p-4 hover:shadow-md transition-all duration-200 border hover:border-primary/20">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-blue-500/10 rounded-lg flex items-center justify-center">
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                    strokeWidth={2}
                    stroke="currentColor"
                    className="h-5 w-5 text-blue-600"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      d="M5.25 14.25h13.5m-13.5 0a3 3 0 01-3-3m3 3a3 3 0 100 6h13.5a3 3 0 100-6m-16.5-3a3 3 0 013-3h13.5a3 3 0 013 3m-19.5 0a4.5 4.5 0 01.9-2.7L5.737 5.1a3.375 3.375 0 012.7-1.35h7.126c1.062 0 2.062.5 2.7 1.35l2.587 3.45a4.5 4.5 0 01.9 2.7m0 0a3 3 0 01-3 3m0 3h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008zm-3 6h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008z"
                    />
                  </svg>
                </div>
                <div>
                  <p className="text-2xl font-bold">
                    {mcpServers.length}
                  </p>
                  <p className="text-xs text-muted-foreground">MCP Servers</p>
                </div>
              </div>
            </Card>

            <Card className="p-4 hover:shadow-md transition-all duration-200 border hover:border-primary/20">
              <div className="flex items-center justify-between gap-3">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-purple-500/10 rounded-lg flex items-center justify-center">
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      fill="none"
                      viewBox="0 0 24 24"
                      strokeWidth={2}
                      stroke="currentColor"
                      className="h-5 w-5 text-purple-600"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z"
                      />
                    </svg>
                  </div>
                  <div>
                    <code className="text-sm font-mono font-semibold">{gatewayUrl}</code>
                    <p className="text-xs text-muted-foreground">Gateway Endpoint</p>
                  </div>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={copyToClipboard}
                  className="shrink-0"
                  title="Copy gateway URL"
                >
                  {copied ? (
                    <Check className="h-4 w-4 text-green-600" />
                  ) : (
                    <Copy className="h-4 w-4" />
                  )}
                </Button>
              </div>
            </Card>
          </div>
        </div>
      </div>

      <div className="container mx-auto px-6 py-8">
        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          <div className="flex items-center gap-4 mb-8">
            <TabsList>
              <TabsTrigger value="mcp" className="gap-2">
                <span className="h-4 w-4 flex items-center justify-center">
                  <MCPIcon />
                </span>
                Servers
              </TabsTrigger>
              <TabsTrigger value="agents" className="gap-2">
                <Bot className="h-4 w-4" />
                Agents
              </TabsTrigger>
            </TabsList>

            {/* Search */}
            <div className="flex-1 max-w-md">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Search deployed resources..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-10 h-9"
                />
              </div>
            </div>
          </div>

          {error && (
            <Card className="p-4 mb-6 bg-destructive/10 border-destructive/20">
              <div className="flex items-center gap-2 text-destructive">
                <AlertCircle className="h-4 w-4" />
                <p className="text-sm font-medium">{error}</p>
              </div>
            </Card>
          )}

          {loading ? (
            <Card className="p-12">
              <div className="text-center text-muted-foreground">
                <p className="text-lg font-medium">Loading resources...</p>
              </div>
            </Card>
          ) : (agents.length === 0 && mcpServers.length === 0) ? (
            <Card className="p-12">
              <div className="text-center text-muted-foreground">
                <div className="w-16 h-16 mx-auto mb-4 opacity-50 flex items-center justify-center">
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                    strokeWidth={1.5}
                    stroke="currentColor"
                    className="w-16 h-16"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      d="M5.25 14.25h13.5m-13.5 0a3 3 0 01-3-3m3 3a3 3 0 100 6h13.5a3 3 0 100-6m-16.5-3a3 3 0 013-3h13.5a3 3 0 013 3m-19.5 0a4.5 4.5 0 01.9-2.7L5.737 5.1a3.375 3.375 0 012.7-1.35h7.126c1.062 0 2.062.5 2.7 1.35l2.587 3.45a4.5 4.5 0 01.9 2.7m0 0a3 3 0 01-3 3m0 3h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008zm-3 6h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008z"
                    />
                  </svg>
                </div>
                <p className="text-lg font-medium mb-2">
                  No resources found
                </p>
                <p className="text-sm mb-6">
                  Deploy MCP servers from the Registry to monitor them here.
                </p>
                <Link
                  href="/"
                  className="inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none ring-offset-background bg-primary text-primary-foreground hover:bg-primary/90 h-10 py-2 px-4"
                >
                  Go to Registry
                </Link>
              </div>
            </Card>
          ) : (
            <>

              <TabsContent value="mcp" className="space-y-4">
                {mcpServers
                  .slice((currentPageServers - 1) * itemsPerPage, currentPageServers * itemsPerPage)
                  .map((item) => (
                  <Card key={`${item.serverName}-${item.version}`} className="p-6 hover:shadow-md transition-all duration-200">
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-3 mb-3">
                          <h3 className="text-xl font-semibold">{item.serverName}</h3>
                          <Badge variant="outline">
                            {item.runtime || "local"}
                          </Badge>
                          {!item.isExternal ? (
                            <Badge className="bg-primary/10 text-primary hover:bg-primary/20 border-primary/20">
                              Managed
                            </Badge>
                          ) : (
                            <Badge variant="secondary" className="bg-purple-500/10 text-purple-600 hover:bg-purple-500/20 border-purple-500/20">
                              External
                            </Badge>
                          )}
                        </div>

                        <div className="grid grid-cols-2 gap-4 text-sm">
                          <div className="flex items-center gap-2 text-muted-foreground">
                            <Calendar className="h-4 w-4" />
                            <span>
                              Deployed: {new Date(item.deployedAt).toLocaleString()}
                            </span>
                          </div>
                          <div className="flex items-center gap-2 text-muted-foreground">
                            <Package className="h-4 w-4" />
                            <span>
                              Version: {item.version}
                              {item.config?.namespace && ` • ${item.config.namespace}`}
                            </span>
                          </div>
                        </div>

                        {Object.keys(item.config || {}).length > 0 && (
                          <div className="mt-3 pt-3 border-t">
                            <p className="text-xs text-muted-foreground mb-2">Configuration:</p>
                            <div className="flex flex-wrap gap-2">
                              {Object.entries(item.config || {}).slice(0, 3).map(([key]) => (
                                <span key={key} className="text-xs px-2 py-1 bg-muted rounded">
                                  {key}
                                </span>
                              ))}
                              {Object.keys(item.config || {}).length > 3 && (
                                <span className="text-xs px-2 py-1 bg-muted rounded text-muted-foreground">
                                  +{Object.keys(item.config || {}).length - 3} more
                                </span>
                              )}
                            </div>
                          </div>
                        )}
                      </div>

                      {!item.isExternal && (
                        <Button
                          variant={session ? "destructive" : "outline"}
                          size="sm"
                          className="ml-4"
                          onClick={() => handleRemove(item.serverName, item.version, item.resourceType)}
                          disabled={removing}
                          title={!session ? "Sign in to remove" : undefined}
                        >
                          {session ? (
                            <><Trash2 className="h-4 w-4 mr-2" />Remove</>
                          ) : (
                            <><LogIn className="h-4 w-4 mr-2" />Sign in to remove</>
                          )}
                        </Button>
                      )}
                    </div>
                  </Card>
                ))}
                {mcpServers.length > itemsPerPage && (
                  <div className="flex items-center justify-center gap-2 mt-6">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCurrentPageServers(p => Math.max(1, p - 1))}
                      disabled={currentPageServers === 1}
                    >
                      Previous
                    </Button>
                    <span className="text-sm text-muted-foreground">
                      Page {currentPageServers} of {Math.ceil(mcpServers.length / itemsPerPage)}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCurrentPageServers(p => Math.min(Math.ceil(mcpServers.length / itemsPerPage), p + 1))}
                      disabled={currentPageServers >= Math.ceil(mcpServers.length / itemsPerPage)}
                    >
                      Next
                    </Button>
                  </div>
                )}
              </TabsContent>

              <TabsContent value="agents" className="space-y-4">
                {agents
                  .slice((currentPageAgents - 1) * itemsPerPage, currentPageAgents * itemsPerPage)
                  .map((item) => (
                  <Card key={`${item.serverName}-${item.version}`} className="p-6 hover:shadow-md transition-all duration-200">
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-3 mb-3">
                          <h3 className="text-xl font-semibold">{item.serverName}</h3>
                          <Badge variant="outline">
                            {item.runtime || "local"}
                          </Badge>
                          {!item.isExternal ? (
                            <Badge className="bg-primary/10 text-primary hover:bg-primary/20 border-primary/20">
                              Managed
                            </Badge>
                          ) : (
                            <Badge variant="secondary" className="bg-purple-500/10 text-purple-600 hover:bg-purple-500/20 border-purple-500/20">
                              External
                            </Badge>
                          )}
                        </div>

                        <div className="grid grid-cols-2 gap-4 text-sm">
                          <div className="flex items-center gap-2 text-muted-foreground">
                            <Calendar className="h-4 w-4" />
                            <span>
                              Deployed: {new Date(item.deployedAt).toLocaleString()}
                            </span>
                          </div>
                          <div className="flex items-center gap-2 text-muted-foreground">
                            <Package className="h-4 w-4" />
                            <span>
                              Version: {item.version}
                              {item.config?.namespace && ` • ${item.config.namespace}`}
                            </span>
                          </div>
                        </div>

                        {Object.keys(item.config || {}).length > 0 && (
                          <div className="mt-3 pt-3 border-t">
                            <p className="text-xs text-muted-foreground mb-2">Configuration:</p>
                            <div className="flex flex-wrap gap-2">
                              {Object.entries(item.config || {}).slice(0, 3).map(([key]) => (
                                <span key={key} className="text-xs px-2 py-1 bg-muted rounded">
                                  {key}
                                </span>
                              ))}
                              {Object.keys(item.config || {}).length > 3 && (
                                <span className="text-xs px-2 py-1 bg-muted rounded text-muted-foreground">
                                  +{Object.keys(item.config || {}).length - 3} more
                                </span>
                              )}
                            </div>
                          </div>
                        )}
                      </div>

                      {!item.isExternal && (
                        <Button
                          variant={session ? "destructive" : "outline"}
                          size="sm"
                          className="ml-4"
                          onClick={() => handleRemove(item.serverName, item.version, item.resourceType)}
                          disabled={removing}
                          title={!session ? "Sign in to remove" : undefined}
                        >
                          {session ? (
                            <><Trash2 className="h-4 w-4 mr-2" />Remove</>
                          ) : (
                            <><LogIn className="h-4 w-4 mr-2" />Sign in to remove</>
                          )}
                        </Button>
                      )}
                    </div>
                  </Card>
                ))}
                {agents.length > itemsPerPage && (
                  <div className="flex items-center justify-center gap-2 mt-6">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCurrentPageAgents(p => Math.max(1, p - 1))}
                      disabled={currentPageAgents === 1}
                    >
                      Previous
                    </Button>
                    <span className="text-sm text-muted-foreground">
                      Page {currentPageAgents} of {Math.ceil(agents.length / itemsPerPage)}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCurrentPageAgents(p => Math.min(Math.ceil(agents.length / itemsPerPage), p + 1))}
                      disabled={currentPageAgents >= Math.ceil(agents.length / itemsPerPage)}
                    >
                      Next
                    </Button>
                  </div>
                )}
              </TabsContent>
            </>
          )}
        </Tabs>
      </div>

      {/* Remove Confirmation Dialog */}
      <Dialog open={!!serverToRemove} onOpenChange={(open) => !open && setServerToRemove(null)}>
        <DialogContent onClose={() => setServerToRemove(null)}>
          <DialogHeader>
            <DialogTitle>Remove Deployment</DialogTitle>
            <DialogDescription>
              Are you sure you want to remove <strong>{serverToRemove?.name}</strong> (version {serverToRemove?.version}) ({serverToRemove?.resourceType})?
              <br />
              <br />
              This will stop the server and remove it from your deployments. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setServerToRemove(null)}
              disabled={removing}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={confirmRemove}
              disabled={removing}
            >
              {removing ? 'Removing...' : 'Remove Deployment'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </main>
  )
}
