"use client"

import { useEffect, useState } from "react"
import Link from "next/link"
import { useSession } from "next-auth/react"
import { Card } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { adminApiClient, createAuthenticatedClient, ServerResponse, SkillResponse, AgentResponse } from "@/lib/admin-api"
import { Trash2, AlertCircle, Calendar, Package, Rocket, Plus, Search, LogIn, BadgeCheck, Bot, Zap } from "lucide-react"
import MCPIcon from "@/components/icons/mcp"
import { SubmitResourceDialog } from "@/components/submit-resource-dialog"
import { ServerDetail } from "@/components/server-detail"
import { SkillDetail } from "@/components/skill-detail"
import { AgentDetail } from "@/components/agent-detail"
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
  resourceType: string
}

export default function PublishedPage() {
  const { data: session } = useSession()
  const [activeTab, setActiveTab] = useState("mcp")
  const [servers, setServers] = useState<ServerResponse[]>([])
  const [skills, setSkills] = useState<SkillResponse[]>([])
  const [agents, setAgents] = useState<AgentResponse[]>([])
  const [filteredServers, setFilteredServers] = useState<ServerResponse[]>([])
  const [filteredSkills, setFilteredSkills] = useState<SkillResponse[]>([])
  const [filteredAgents, setFilteredAgents] = useState<AgentResponse[]>([])
  const [searchQuery, setSearchQuery] = useState("")
  const [deployments, setDeployments] = useState<DeploymentResponse[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [unpublishing, setUnpublishing] = useState(false)
  const [deploying, setDeploying] = useState(false)
  const [deployRuntime] = useState<'kubernetes'>('kubernetes')
  const [itemToUnpublish, setItemToUnpublish] = useState<{ name: string, version: string, type: 'server' | 'skill' | 'agent' } | null>(null)
  const [itemToDeploy, setItemToDeploy] = useState<{ name: string, version: string, type: 'server' | 'agent' } | null>(null)
  const [selectedServer, setSelectedServer] = useState<ServerResponse | null>(null)
  const [selectedSkill, setSelectedSkill] = useState<SkillResponse | null>(null)
  const [selectedAgent, setSelectedAgent] = useState<AgentResponse | null>(null)

  // Pagination state
  const [currentPageServers, setCurrentPageServers] = useState(1)
  const [currentPageSkills, setCurrentPageSkills] = useState(1)
  const [currentPageAgents, setCurrentPageAgents] = useState(1)
  const [submitResourceDialogOpen, setSubmitResourceDialogOpen] = useState(false)
  const itemsPerPage = 5

  const fetchPublished = async () => {
    try {
      setLoading(true)
      setError(null)

      // Fetch all published servers (with pagination if needed)
      const allServers: ServerResponse[] = []
      let serverCursor: string | undefined

      do {
        const response = await adminApiClient.listPublishedServers({
          cursor: serverCursor,
          limit: 100,
        })
        allServers.push(...response.servers)
        serverCursor = response.metadata.nextCursor
      } while (serverCursor)

      setServers(allServers)

      // Fetch all published skills (with pagination if needed)
      const allSkills: SkillResponse[] = []
      let skillCursor: string | undefined

      do {
        const response = await adminApiClient.listPublishedSkills({
          cursor: skillCursor,
          limit: 100,
        })
        allSkills.push(...response.skills)
        skillCursor = response.metadata.nextCursor
      } while (skillCursor)

      setSkills(allSkills)

      // Fetch all published agents (with pagination if needed)
      const allAgents: AgentResponse[] = []
      let agentCursor: string | undefined

      do {
        const response = await adminApiClient.listPublishedAgents({
          cursor: agentCursor,
          limit: 100,
        })
        allAgents.push(...response.agents)
        agentCursor = response.metadata.nextCursor
      } while (agentCursor)

      setAgents(allAgents)

      // Fetch deployments to check what's currently deployed
      const deploymentData = await adminApiClient.listDeployments()
      setDeployments(deploymentData)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch published resources')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchPublished()
    // Refresh every 30 seconds
    const interval = setInterval(fetchPublished, 30000)
    return () => clearInterval(interval)
  }, [])

  // Filter by search query
  useEffect(() => {
    const query = searchQuery.toLowerCase()
    setFilteredServers(
      servers.filter(s =>
        s.server.name.toLowerCase().includes(query) ||
        s.server.description?.toLowerCase().includes(query)
      )
    )
    setFilteredSkills(
      skills.filter(s =>
        s.skill.name.toLowerCase().includes(query) ||
        s.skill.description?.toLowerCase().includes(query)
      )
    )
    setFilteredAgents(
      agents.filter(a =>
        a.agent.name.toLowerCase().includes(query) ||
        a.agent.description?.toLowerCase().includes(query)
      )
    )
  }, [searchQuery, servers, skills, agents])

  // Check if a resource is currently deployed (check both name and version)
  const isDeployed = (name: string, version: string, type: 'server' | 'agent') => {
    return deployments.some(d => d.serverName === name && d.version === version && d.resourceType === (type === 'server' ? 'mcp' : 'agent'))
  }

  const handleUnpublish = async (name: string, version: string, type: 'server' | 'skill' | 'agent') => {
    // Check if the resource is deployed (check specific version)
    if (type !== 'skill' && isDeployed(name, version, type)) {
      toast.error(`Cannot unpublish ${name} version ${version} while it's deployed. Remove it from the Deployed page first.`)
      return
    }
    setItemToUnpublish({ name, version, type })
  }

  const handleDeploy = async (name: string, version: string, type: 'server' | 'agent') => {
    if (!session) {
      toast.error("Please sign in to deploy resources")
      return
    }
    setItemToDeploy({ name, version, type })
  }

  const confirmDeploy = async () => {
    if (!itemToDeploy) return

    try {
      setDeploying(true)

      // Use admin client (auth is disabled by default in controller)
      const client = adminApiClient

      // Deploy server or agent
      await client.deployServer({
        serverName: itemToDeploy.name,
        version: itemToDeploy.version,
        config: {},
        preferRemote: false,
        resourceType: itemToDeploy.type === 'agent' ? 'agent' : 'mcp',
      })

      setItemToDeploy(null)
      toast.success(`Successfully deployed ${itemToDeploy.name} to ${deployRuntime}!`)
      // Refresh deployments to update the UI
      await fetchPublished()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to deploy resource')
    } finally {
      setDeploying(false)
    }
  }

  const confirmUnpublish = async () => {
    if (!itemToUnpublish) return

    try {
      setUnpublishing(true)

      // Use admin client (auth is disabled by default in controller)
      const client = adminApiClient

      if (itemToUnpublish.type === 'server') {
        await client.unpublishServerStatus(itemToUnpublish.name, itemToUnpublish.version)
        setServers(prev => prev.filter(s => s.server.name !== itemToUnpublish.name || s.server.version !== itemToUnpublish.version))
      } else if (itemToUnpublish.type === 'skill') {
        await client.unpublishSkillStatus(itemToUnpublish.name, itemToUnpublish.version)
        setSkills(prev => prev.filter(s => s.skill.name !== itemToUnpublish.name || s.skill.version !== itemToUnpublish.version))
      } else if (itemToUnpublish.type === 'agent') {
        await client.unpublishAgentStatus(itemToUnpublish.name, itemToUnpublish.version)
        setAgents(prev => prev.filter(a => a.agent.name !== itemToUnpublish.name || a.agent.version !== itemToUnpublish.version))
      }

      setItemToUnpublish(null)
      toast.success(`Successfully unpublished ${itemToUnpublish.name}`)
      // Refresh deployments to update the UI
      await fetchPublished()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to unpublish resource')
    } finally {
      setUnpublishing(false)
    }
  }

  const handlePublish = async (server: ServerResponse) => {
    try {
      const client = adminApiClient
      await client.publishServerStatus(server.server.name, server.server.version)
      await fetchPublished() // Refresh data
      toast.success(`Successfully published ${server.server.name}`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to publish server")
    }
  }

  const handlePublishSkill = async (skill: SkillResponse) => {
    try {
      const client = adminApiClient
      await client.publishSkillStatus(skill.skill.name, skill.skill.version)
      await fetchPublished() // Refresh data
      toast.success(`Successfully published ${skill.skill.name}`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to publish skill")
    }
  }

  const handlePublishAgent = async (agentResponse: AgentResponse) => {
    const { agent } = agentResponse

    try {
      const client = adminApiClient
      await client.publishAgentStatus(agent.name, agent.version)
      await fetchPublished() // Refresh data
      toast.success(`Successfully published ${agent.name}`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to publish agent")
    }
  }

  const totalPublished = servers.length + skills.length + agents.length

  // Show server detail view if a server is selected
  if (selectedServer) {
    return (
      <ServerDetail
        server={selectedServer}
        onClose={() => setSelectedServer(null)}
        onServerCopied={fetchPublished}
        onPublish={handlePublish}
      />
    )
  }

  // Show skill detail view if a skill is selected
  if (selectedSkill) {
    return (
      <SkillDetail
        skill={selectedSkill}
        onClose={() => setSelectedSkill(null)}
        onPublish={handlePublishSkill}
      />
    )
  }

  // Show agent detail view if an agent is selected
  if (selectedAgent) {
    return (
      <AgentDetail
        agent={selectedAgent}
        onClose={() => setSelectedAgent(null)}
        onPublish={handlePublishAgent}
      />
    )
  }

  return (
    <main className="min-h-screen bg-background">
      {/* Stats Section */}
      <div className="bg-muted/30 border-b">
        <div className="container mx-auto px-6 py-6">
          <div className="grid gap-4 md:grid-cols-4">
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
                  <p className="text-2xl font-bold">{totalPublished}</p>
                  <p className="text-xs text-muted-foreground">Total Published</p>
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
                  <p className="text-2xl font-bold">{servers.length}</p>
                  <p className="text-xs text-muted-foreground">MCP Servers</p>
                </div>
              </div>
            </Card>

            <Card className="p-4 hover:shadow-md transition-all duration-200 border hover:border-primary/20">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-yellow-500/10 rounded-lg flex items-center justify-center">
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                    strokeWidth={2}
                    stroke="currentColor"
                    className="h-5 w-5 text-yellow-600"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z"
                    />
                  </svg>
                </div>
                <div>
                  <p className="text-2xl font-bold">{skills.length}</p>
                  <p className="text-xs text-muted-foreground">Skills</p>
                </div>
              </div>
            </Card>

            <Card className="p-4 hover:shadow-md transition-all duration-200 border hover:border-primary/20">
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
                      d="M8.25 3v1.5M4.5 8.25H3m18 0h-1.5M4.5 12H3m18 0h-1.5m-15 3.75H3m18 0h-1.5M8.25 19.5V21M12 3v1.5m0 15V21m3.75-18v1.5m0 15V21m-9-1.5h10.5a2.25 2.25 0 002.25-2.25V6.75a2.25 2.25 0 00-2.25-2.25H6.75A2.25 2.25 0 004.5 6.75v10.5a2.25 2.25 0 002.25 2.25zm.75-12h9v9h-9v-9z"
                    />
                  </svg>
                </div>
                <div>
                  <p className="text-2xl font-bold">{agents.length}</p>
                  <p className="text-xs text-muted-foreground">Agents</p>
                </div>
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
              <TabsTrigger value="skills" className="gap-2">
                <Zap className="h-4 w-4" />
                Skills
              </TabsTrigger>
            </TabsList>

            {/* Search */}
            <div className="flex-1 max-w-md">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Search published resources..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-10 h-9"
                />
              </div>
            </div>

            {/* Action Button */}
            <div className="ml-auto">
              <Button
                className="gap-2"
                onClick={() => setSubmitResourceDialogOpen(true)}
              >
                <Plus className="h-4 w-4" />
                Submit Resource
              </Button>
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
                <p className="text-lg font-medium">Loading published resources...</p>
              </div>
            </Card>
          ) : totalPublished === 0 ? (
            <Card className="p-12">
              <div className="text-center text-muted-foreground">
                <div className="w-16 h-16 mx-auto mb-4 opacity-50 flex items-center justify-center">
                  <Package className="w-16 h-16" />
                </div>
                <p className="text-lg font-medium mb-2">
                  No published resources
                </p>
                <p className="text-sm mb-6">
                  Publish MCP servers, skills, or agents from the Inventory to see them here.
                </p>
                <Link
                  href="/"
                  className="inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none ring-offset-background bg-primary text-primary-foreground hover:bg-primary/90 h-10 py-2 px-4"
                >
                  Go to Inventory
                </Link>
              </div>
            </Card>
          ) : (
            <>

              <TabsContent value="mcp" className="space-y-4">
                {filteredServers
                  .slice((currentPageServers - 1) * itemsPerPage, currentPageServers * itemsPerPage)
                  .map((serverResponse) => {
                  const server = serverResponse.server
                  const meta = serverResponse._meta?.['io.modelcontextprotocol.registry/official']
                  const deployed = isDeployed(server.name, server.version, 'server')

                  // Extract owner from metadata or repository URL
                  const publisherMetadata = server._meta?.['io.modelcontextprotocol.registry/publisher-provided']?.['aregistry.ai/metadata'] as any
                  const identityData = publisherMetadata?.identity
                  const owner = publisherMetadata?.contact_email || identityData?.email || (meta as any)?.submitter ||
                    (server.repository?.url?.match(/github\.com\/([^\/]+)/)?.[1]) || null

                  return (
                    <Card
                      key={`${server.name}-${server.version}`}
                      className="p-6 hover:shadow-md transition-all duration-200 cursor-pointer"
                      onClick={() => setSelectedServer(serverResponse)}
                    >
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="flex items-center gap-3 mb-3">
                            <h3 className="text-xl font-semibold">{server.name}</h3>
                            {deployed ? (
                              <Badge className="bg-green-500/10 text-green-600 hover:bg-green-500/20 border-green-500/20">
                                Running
                              </Badge>
                            ) : (
                              <Badge variant="secondary" className="bg-muted">
                                Not Deployed
                              </Badge>
                            )}
                          </div>

                          <p className="text-sm text-muted-foreground mb-3">{server.description}</p>

                          <div className="flex flex-wrap items-center gap-4 text-sm">
                            <div className="flex items-center gap-2 text-muted-foreground">
                              <Package className="h-4 w-4" />
                              <span>Version: {server.version}</span>
                            </div>
                            {meta?.publishedAt && (
                              <div className="flex items-center gap-2 text-muted-foreground">
                                <Calendar className="h-4 w-4" />
                                <span>Published: {new Date(meta.publishedAt).toLocaleDateString()}</span>
                              </div>
                            )}
                            {owner && (
                              <div className="flex items-center gap-2 text-primary font-medium">
                                <BadgeCheck className="h-4 w-4" />
                                <span>Owner: {owner}</span>
                              </div>
                            )}
                          </div>
                        </div>

                        <div className="flex gap-2 ml-4">
                          <Button
                            variant="default"
                            size="sm"
                            onClick={(e) => {
                              e.stopPropagation()
                              handleDeploy(server.name, server.version, 'server')
                            }}
                            disabled={deploying || deployed}
                            title={!session ? "Sign in to deploy" : undefined}
                          >
                            {!session ? (
                              <><LogIn className="h-4 w-4 mr-2" />Sign in to deploy</>
                            ) : deployed ? (
                              <><Rocket className="h-4 w-4 mr-2" />Already Deployed</>
                            ) : (
                              <><Rocket className="h-4 w-4 mr-2" />Deploy</>
                            )}
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={(e) => {
                              e.stopPropagation()
                              handleUnpublish(server.name, server.version, 'server')
                            }}
                            disabled={unpublishing}
                          >
                            Unpublish
                          </Button>
                        </div>
                      </div>
                    </Card>
                  )
                })}
                {filteredServers.length > itemsPerPage && (
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
                      Page {currentPageServers} of {Math.ceil(filteredServers.length / itemsPerPage)}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCurrentPageServers(p => Math.min(Math.ceil(filteredServers.length / itemsPerPage), p + 1))}
                      disabled={currentPageServers >= Math.ceil(filteredServers.length / itemsPerPage)}
                    >
                      Next
                    </Button>
                  </div>
                )}
              </TabsContent>

              <TabsContent value="agents" className="space-y-4">
                {filteredAgents
                  .slice((currentPageAgents - 1) * itemsPerPage, currentPageAgents * itemsPerPage)
                  .map((agentResponse) => {
                  const agent = agentResponse.agent
                  const meta = agentResponse._meta?.['io.modelcontextprotocol.registry/official']
                  const deployed = isDeployed(agent.name, agent.version, 'agent')

                  // Extract owner from metadata or repository URL
                  const publisherMetadata = (agent as any)._meta?.['io.modelcontextprotocol.registry/publisher-provided']?.['aregistry.ai/metadata']
                  const identityData = publisherMetadata?.identity
                  const owner = publisherMetadata?.contact_email || identityData?.email || (meta as any)?.submitter ||
                    (agent.repository?.url?.match(/github\.com\/([^\/]+)/)?.[1]) || null

                  return (
                    <Card
                      key={`${agent.name}-${agent.version}`}
                      className="p-6 hover:shadow-md transition-all duration-200 cursor-pointer"
                      onClick={() => setSelectedAgent(agentResponse)}
                    >
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="flex items-center gap-3 mb-3">
                            <h3 className="text-xl font-semibold">{agent.name}</h3>
                            {deployed ? (
                              <Badge className="bg-green-500/10 text-green-600 hover:bg-green-500/20 border-green-500/20">
                                Running
                              </Badge>
                            ) : (
                              <Badge variant="secondary" className="bg-muted">
                                Not Deployed
                              </Badge>
                            )}
                          </div>

                          <p className="text-sm text-muted-foreground mb-3">{agent.description}</p>

                          <div className="flex flex-wrap items-center gap-4 text-sm">
                            <div className="flex items-center gap-2 text-muted-foreground">
                              <Package className="h-4 w-4" />
                              <span>Version: {agent.version}</span>
                            </div>
                            {meta?.publishedAt && (
                              <div className="flex items-center gap-2 text-muted-foreground">
                                <Calendar className="h-4 w-4" />
                                <span>Published: {new Date(meta.publishedAt).toLocaleDateString()}</span>
                              </div>
                            )}
                            {owner && (
                              <div className="flex items-center gap-2 text-primary font-medium">
                                <BadgeCheck className="h-4 w-4" />
                                <span>Owner: {owner}</span>
                              </div>
                            )}
                          </div>
                        </div>

                        <div className="flex gap-2 ml-4">
                          <Button
                            variant="default"
                            size="sm"
                            onClick={(e) => {
                              e.stopPropagation()
                              handleDeploy(agent.name, agent.version, 'agent')
                            }}
                            disabled={deploying || deployed}
                            title={!session ? "Sign in to deploy" : undefined}
                          >
                            {!session ? (
                              <><LogIn className="h-4 w-4 mr-2" />Sign in to deploy</>
                            ) : deployed ? (
                              <><Rocket className="h-4 w-4 mr-2" />Already Deployed</>
                            ) : (
                              <><Rocket className="h-4 w-4 mr-2" />Deploy</>
                            )}
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={(e) => {
                              e.stopPropagation()
                              handleUnpublish(agent.name, agent.version, 'agent')
                            }}
                            disabled={unpublishing}
                          >
                            Unpublish
                          </Button>
                        </div>
                      </div>
                    </Card>
                  )
                })}
                {filteredAgents.length > itemsPerPage && (
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
                      Page {currentPageAgents} of {Math.ceil(filteredAgents.length / itemsPerPage)}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCurrentPageAgents(p => Math.min(Math.ceil(filteredAgents.length / itemsPerPage), p + 1))}
                      disabled={currentPageAgents >= Math.ceil(filteredAgents.length / itemsPerPage)}
                    >
                      Next
                    </Button>
                  </div>
                )}
              </TabsContent>

              <TabsContent value="skills" className="space-y-4">
                {filteredSkills
                  .slice((currentPageSkills - 1) * itemsPerPage, currentPageSkills * itemsPerPage)
                  .map((skillResponse) => {
                  const skill = skillResponse.skill
                  const meta = skillResponse._meta?.['io.modelcontextprotocol.registry/official']

                  // Extract owner from metadata or repository URL
                  const publisherMetadata = (skill as any)._meta?.['io.modelcontextprotocol.registry/publisher-provided']?.['aregistry.ai/metadata']
                  const identityData = publisherMetadata?.identity
                  const owner = publisherMetadata?.contact_email || identityData?.email || (meta as any)?.submitter ||
                    (skill.repository?.url?.match(/github\.com\/([^\/]+)/)?.[1]) || null

                  return (
                    <Card
                      key={`${skill.name}-${skill.version}`}
                      className="p-6 hover:shadow-md transition-all duration-200 cursor-pointer"
                      onClick={() => setSelectedSkill(skillResponse)}
                    >
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="flex items-center gap-3 mb-3">
                            <h3 className="text-xl font-semibold">{skill.title || skill.name}</h3>
                          </div>

                          <p className="text-sm text-muted-foreground mb-3">{skill.description}</p>

                          <div className="flex flex-wrap items-center gap-4 text-sm">
                            <div className="flex items-center gap-2 text-muted-foreground">
                              <Package className="h-4 w-4" />
                              <span>Version: {skill.version}</span>
                            </div>
                            {meta?.publishedAt && (
                              <div className="flex items-center gap-2 text-muted-foreground">
                                <Calendar className="h-4 w-4" />
                                <span>Published: {new Date(meta.publishedAt).toLocaleDateString()}</span>
                              </div>
                            )}
                            {owner && (
                              <div className="flex items-center gap-2 text-primary font-medium">
                                <BadgeCheck className="h-4 w-4" />
                                <span>Owner: {owner}</span>
                              </div>
                            )}
                          </div>
                        </div>

                        <Button
                          variant="outline"
                          size="sm"
                          className="ml-4"
                          onClick={(e) => {
                            e.stopPropagation()
                            handleUnpublish(skill.name, skill.version, 'skill')
                          }}
                          disabled={unpublishing}
                        >
                          Unpublish
                        </Button>
                      </div>
                    </Card>
                  )
                })}
                {filteredSkills.length > itemsPerPage && (
                  <div className="flex items-center justify-center gap-2 mt-6">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCurrentPageSkills(p => Math.max(1, p - 1))}
                      disabled={currentPageSkills === 1}
                    >
                      Previous
                    </Button>
                    <span className="text-sm text-muted-foreground">
                      Page {currentPageSkills} of {Math.ceil(filteredSkills.length / itemsPerPage)}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCurrentPageSkills(p => Math.min(Math.ceil(filteredSkills.length / itemsPerPage), p + 1))}
                      disabled={currentPageSkills >= Math.ceil(filteredSkills.length / itemsPerPage)}
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

      {/* Unpublish Confirmation Dialog */}
      <Dialog open={!!itemToUnpublish} onOpenChange={(open) => !open && setItemToUnpublish(null)}>
        <DialogContent onClose={() => setItemToUnpublish(null)}>
          <DialogHeader>
            <DialogTitle>Unpublish Resource</DialogTitle>
            <DialogDescription>
              Are you sure you want to unpublish <strong>{itemToUnpublish?.name}</strong> (version {itemToUnpublish?.version})?
              <br />
              <br />
              This will change its status to unpublished. The resource will still exist in the inventory but won&apos;t be visible to public users.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setItemToUnpublish(null)}
              disabled={unpublishing}
            >
              Cancel
            </Button>
            <Button
              variant="default"
              onClick={confirmUnpublish}
              disabled={unpublishing}
            >
              {unpublishing ? 'Unpublishing...' : 'Unpublish'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Deploy Confirmation Dialog */}
      <Dialog open={!!itemToDeploy} onOpenChange={(open) => !open && setItemToDeploy(null)}>
        <DialogContent onClose={() => setItemToDeploy(null)}>
          <DialogHeader>
            <DialogTitle>Deploy Resource</DialogTitle>
            <DialogDescription>
              Deploy <strong>{itemToDeploy?.name}</strong> (version {itemToDeploy?.version})?
              <br />
              <br />
              This will start the {itemToDeploy?.type === 'server' ? 'MCP server' : 'agent'} on your system.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2 py-4">
            <Label>Deployment destination</Label>
            <p className="text-sm text-muted-foreground">Kubernetes</p>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setItemToDeploy(null)}
              disabled={deploying}
            >
              Cancel
            </Button>
            <Button
              variant="default"
              onClick={confirmDeploy}
              disabled={deploying}
            >
              {deploying ? 'Deploying...' : 'Deploy'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Submit Resource Dialog (GitOps Workflow) */}
      <SubmitResourceDialog
        open={submitResourceDialogOpen}
        onOpenChange={setSubmitResourceDialogOpen}
      />
    </main>
  )
}
