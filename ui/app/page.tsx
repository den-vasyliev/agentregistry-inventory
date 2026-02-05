"use client"

import { useEffect, useState, useRef, useCallback } from "react"
import { useSession } from "next-auth/react"
import { useRouter } from "next/navigation"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { Label } from "@/components/ui/label"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { ServerCard } from "@/components/server-card"
import { SkillCard } from "@/components/skill-card"
import { AgentCard } from "@/components/agent-card"
import { ModelCard } from "@/components/model-card"
import { ServerDetail } from "@/components/server-detail"
import { SkillDetail } from "@/components/skill-detail"
import { AgentDetail } from "@/components/agent-detail"
import { ModelDetail } from "@/components/model-detail"
import { SubmitResourceDialog } from "@/components/submit-resource-dialog"
import { DeployDialog, UndeployDialog } from "@/components/deploy-dialog"
import { StatsCards } from "@/components/stats-cards"
import { Pagination } from "@/components/pagination"
import { adminApiClient, ServerResponse, SkillResponse, AgentResponse, ModelResponse, ServerStats } from "@/lib/admin-api"
import MCPIcon from "@/components/icons/mcp"
import { toast } from "sonner"
import {
  Search,
  RefreshCw,
  Zap,
  Bot,
  Brain,
  GitPullRequest,
  ShieldCheck,
  BadgeCheck,
} from "lucide-react"

// Grouped server type
interface GroupedServer extends ServerResponse {
  versionCount: number
  allVersions: ServerResponse[]
}

// Deployment status filter type
type DeploymentStatus = "all" | "external" | "running" | "not_deployed" | "failed"

export default function AdminPage() {
  const { data: session, status } = useSession()
  const router = useRouter()
  const disableAuth = process.env.NEXT_PUBLIC_DISABLE_AUTH !== "false"
  const [activeTab, setActiveTab] = useState("servers")
  const [deploymentStatusFilter, setDeploymentStatusFilter] = useState<DeploymentStatus>("all")
  const [servers, setServers] = useState<ServerResponse[]>([])
  const [groupedServers, setGroupedServers] = useState<GroupedServer[]>([])
  const [skills, setSkills] = useState<SkillResponse[]>([])
  const [agents, setAgents] = useState<AgentResponse[]>([])
  const [models, setModels] = useState<ModelResponse[]>([])
  const [filteredServers, setFilteredServers] = useState<GroupedServer[]>([])
  const [filteredSkills, setFilteredSkills] = useState<SkillResponse[]>([])
  const [filteredAgents, setFilteredAgents] = useState<AgentResponse[]>([])
  const [filteredModels, setFilteredModels] = useState<ModelResponse[]>([])
  const [stats, setStats] = useState<ServerStats | null>(null)
  const [searchQuery, setSearchQuery] = useState("")
  const [debouncedSearch, setDebouncedSearch] = useState("")
  const [filterVerifiedOrg, setFilterVerifiedOrg] = useState(false)
  const [filterVerifiedPublisher, setFilterVerifiedPublisher] = useState(false)
  const [submitResourceDialogOpen, setSubmitResourceDialogOpen] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedServer, setSelectedServer] = useState<ServerResponse | null>(null)
  const [selectedSkill, setSelectedSkill] = useState<SkillResponse | null>(null)
  const [selectedAgent, setSelectedAgent] = useState<AgentResponse | null>(null)
  const [selectedModel, setSelectedModel] = useState<ModelResponse | null>(null)

  useEffect(() => {
    if (!disableAuth && status === "unauthenticated") {
      router.push("/auth/signin")
    }
  }, [status, router, disableAuth])

  // Debounce search query (300ms)
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(searchQuery), 300)
    return () => clearTimeout(timer)
  }, [searchQuery])

  // Deploy/Undeploy dialog state
  const [deployDialogOpen, setDeployDialogOpen] = useState(false)
  const [undeployDialogOpen, setUndeployDialogOpen] = useState(false)
  const [itemToDeploy, setItemToDeploy] = useState<{ name: string, version: string, type: 'server' | 'agent' } | null>(null)
  const [itemToUndeploy, setItemToUndeploy] = useState<{ name: string, version: string, type: 'server' | 'agent' } | null>(null)
  const [deploying, setDeploying] = useState(false)
  const [undeploying, setUndeploying] = useState(false)
  const [deployNamespace, setDeployNamespace] = useState("agentregistry")
  const [environments, setEnvironments] = useState<Array<{name: string, namespace: string}>>([
    { name: "agentregistry", namespace: "agentregistry" }
  ])
  const [loadingEnvironments, setLoadingEnvironments] = useState(false)

  // Pagination state
  const [currentPageServers, setCurrentPageServers] = useState(1)
  const [currentPageSkills, setCurrentPageSkills] = useState(1)
  const [currentPageAgents, setCurrentPageAgents] = useState(1)
  const [currentPageModels, setCurrentPageModels] = useState(1)
  const itemsPerPage = 5

  // Track scroll position for restoring after navigation
  const scrollPositionRef = useRef<number>(0)
  const shouldRestoreScrollRef = useRef<boolean>(false)

  // Helper function to get resource creation date (for sorting)
  const getResourceDate = (item: ServerResponse | AgentResponse | SkillResponse | ModelResponse): Date | null => {
    const dateStr = item._meta?.['io.modelcontextprotocol.registry/official']?.updatedAt
    if (!dateStr) return null
    try {
      return new Date(dateStr)
    } catch {
      return null
    }
  }

  // Extract the canonical name from any resource type
  const getResourceName = (item: ServerResponse | AgentResponse | SkillResponse | ModelResponse): string => {
    if ('server' in item) return item.server.name
    if ('agent' in item) return item.agent.name
    if ('skill' in item) return item.skill.name
    if ('model' in item) return item.model.name
    return ''
  }

  // Sort newest first by creation date
  const sortByCreatedDesc = <T extends ServerResponse | AgentResponse | SkillResponse | ModelResponse>(list: T[]): T[] =>
    [...list].sort((a, b) => {
      const dateA = getResourceDate(a)
      const dateB = getResourceDate(b)
      if (!dateA && !dateB) return getResourceName(a).localeCompare(getResourceName(b))
      if (!dateA) return 1
      if (!dateB) return -1
      const diff = dateB.getTime() - dateA.getTime()
      return diff !== 0 ? diff : getResourceName(a).localeCompare(getResourceName(b))
    })

  const matchesDeploymentFilter = (item: ServerResponse | AgentResponse, filter: DeploymentStatus): boolean => {
    if (filter === "all") return true

    const isExternal = item._meta?.isDiscovered || item._meta?.source === 'discovery'
    const deployment = item._meta?.deployment

    switch (filter) {
      case "external":
        return isExternal
      case "running":
        return deployment?.ready === true
      case "failed":
        return !!deployment && deployment.ready === false
      case "not_deployed":
        return !isExternal && !deployment
    }
  }

  // Group servers by name, keeping the latest version as the representative
  const groupServersByName = (servers: ServerResponse[]): GroupedServer[] => {
    const grouped = new Map<string, ServerResponse[]>()

    servers.forEach((server) => {
      const name = server.server.name
      if (!grouped.has(name)) {
        grouped.set(name, [])
      }
      grouped.get(name)!.push(server)
    })

    return Array.from(grouped.entries()).map(([name, versions]) => {
      const sortedVersions = [...versions].sort((a, b) => {
        const dateA = getResourceDate(a)
        const dateB = getResourceDate(b)
        if (dateA && dateB) {
          return dateB.getTime() - dateA.getTime()
        }
        return b.server.version.localeCompare(a.server.version)
      })

      const latestVersion = sortedVersions[0]
      return {
        ...latestVersion,
        versionCount: versions.length,
        allVersions: sortedVersions,
      }
    })
  }

  // Fetch data from API
  const fetchData = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)

      const allServers: ServerResponse[] = []
      let serverCursor: string | undefined

      do {
        const response = await adminApiClient.listServers({
          cursor: serverCursor,
          limit: 100,
        })
        allServers.push(...response.servers)
        serverCursor = response.metadata.nextCursor
      } while (serverCursor)

      setServers(allServers)

      const allSkills: SkillResponse[] = []
      let skillCursor: string | undefined

      do {
        const response = await adminApiClient.listSkills({
          cursor: skillCursor,
          limit: 100,
        })
        allSkills.push(...response.skills)
        skillCursor = response.metadata.nextCursor
      } while (skillCursor)

      setSkills(allSkills)

      const allAgents: AgentResponse[] = []
      let agentCursor: string | undefined

      do {
        const response = await adminApiClient.listAgents({
          cursor: agentCursor,
          limit: 100,
        })
        allAgents.push(...response.agents)
        agentCursor = response.metadata.nextCursor
      } while (agentCursor)

      setAgents(allAgents)

      const allModels: ModelResponse[] = []
      let modelCursor: string | undefined

      do {
        const response = await adminApiClient.listModels({
          cursor: modelCursor,
          limit: 100,
        })
        allModels.push(...response.models)
        modelCursor = response.metadata.nextCursor
      } while (modelCursor)

      setModels(allModels)

      const grouped = groupServersByName(allServers)
      setGroupedServers(grouped)

      setStats({
        total_servers: allServers.length,
        total_server_names: grouped.length,
        active_servers: allServers.length,
        deprecated_servers: 0,
        deleted_servers: 0,
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch data")
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  // Restore scroll position when returning from server detail
  useEffect(() => {
    if (!selectedServer && shouldRestoreScrollRef.current) {
      setTimeout(() => {
        window.scrollTo({
          top: scrollPositionRef.current,
          behavior: 'instant' as ScrollBehavior
        })
        shouldRestoreScrollRef.current = false
      }, 0)
    }
  }, [selectedServer])

  const handleServerClick = useCallback((server: GroupedServer) => {
    scrollPositionRef.current = window.scrollY
    shouldRestoreScrollRef.current = true
    setSelectedServer(server)
  }, [])

  const handleCloseServerDetail = useCallback(() => {
    setSelectedServer(null)
  }, [])

  const handleDeploy = useCallback((item: ServerResponse | AgentResponse, type: 'server' | 'agent') => {
    const name = type === 'server' ? (item as ServerResponse).server.name : (item as AgentResponse).agent.name
    const version = type === 'server' ? (item as ServerResponse).server.version : (item as AgentResponse).agent.version
    setItemToDeploy({ name, version, type })
    setDeployDialogOpen(true)
    fetchEnvironments()
  }, [])

  const handleUndeploy = useCallback((item: ServerResponse | AgentResponse, type: 'server' | 'agent') => {
    const name = type === 'server' ? (item as ServerResponse).server.name : (item as AgentResponse).agent.name
    const version = type === 'server' ? (item as ServerResponse).server.version : (item as AgentResponse).agent.version
    setItemToUndeploy({ name, version, type })
    setUndeployDialogOpen(true)
  }, [])

  const fetchEnvironments = async () => {
    setLoadingEnvironments(true)
    try {
      const envs = await adminApiClient.listEnvironments()
      if (envs && envs.length > 0) {
        const unique = envs.filter((env, idx, arr) => arr.findIndex(e => e.namespace === env.namespace) === idx)
        setEnvironments(unique)
        setDeployNamespace(unique[0].namespace)
      }
    } catch (err) {
      console.error("Failed to fetch environments:", err)
    } finally {
      setLoadingEnvironments(false)
    }
  }

  const confirmDeploy = async () => {
    if (!itemToDeploy) return

    try {
      setDeploying(true)

      await adminApiClient.deployServer({
        serverName: itemToDeploy.name,
        version: itemToDeploy.version,
        config: {},
        preferRemote: false,
        resourceType: itemToDeploy.type === 'agent' ? 'agent' : 'mcp',
        namespace: deployNamespace,
      })

      setDeployDialogOpen(false)
      setItemToDeploy(null)
      toast.success(`Successfully deployed ${itemToDeploy.name} to ${deployNamespace}!`)
      await fetchData()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to deploy resource')
    } finally {
      setDeploying(false)
    }
  }

  const confirmUndeploy = async () => {
    if (!itemToUndeploy) return

    try {
      setUndeploying(true)

      await adminApiClient.removeDeployment(
        itemToUndeploy.name,
        itemToUndeploy.version,
        itemToUndeploy.type === 'agent' ? 'agent' : 'mcp'
      )

      setUndeployDialogOpen(false)
      setItemToUndeploy(null)
      toast.success(`Successfully undeployed ${itemToUndeploy.name}`)
      await fetchData()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to undeploy resource')
    } finally {
      setUndeploying(false)
    }
  }

  // Reset all page numbers when search query changes
  useEffect(() => {
    setCurrentPageServers(1)
    setCurrentPageSkills(1)
    setCurrentPageAgents(1)
    setCurrentPageModels(1)
  }, [debouncedSearch, deploymentStatusFilter])

  // Filter and sort servers
  useEffect(() => {
    let filtered = [...groupedServers]

    if (debouncedSearch) {
      const query = debouncedSearch.toLowerCase()
      filtered = filtered.filter(
        (s) =>
          s.server.name.toLowerCase().includes(query) ||
          s.server.title?.toLowerCase().includes(query) ||
          s.server.description?.toLowerCase().includes(query)
      )
    }

    if (filterVerifiedOrg) {
      filtered = filtered.filter((s) => {
        const identityData = s._meta?.['io.modelcontextprotocol.registry/publisher-provided']?.['aregistry.ai/metadata']?.identity
        return identityData?.org_is_verified === true
      })
    }

    if (filterVerifiedPublisher) {
      filtered = filtered.filter((s) => {
        const identityData = s._meta?.['io.modelcontextprotocol.registry/publisher-provided']?.['aregistry.ai/metadata']?.identity
        return identityData?.publisher_identity_verified_by_jwt === true
      })
    }

    filtered = filtered.filter((s) => matchesDeploymentFilter(s, deploymentStatusFilter))

    setFilteredServers(sortByCreatedDesc(filtered))
  }, [debouncedSearch, groupedServers, filterVerifiedOrg, filterVerifiedPublisher, deploymentStatusFilter])

  // Filter skills, agents, and models
  useEffect(() => {
    const query = debouncedSearch.toLowerCase()

    let filteredSk = skills
    if (debouncedSearch) {
      filteredSk = filteredSk.filter(
        (s) =>
          s.skill.name.toLowerCase().includes(query) ||
          s.skill.title?.toLowerCase().includes(query) ||
          s.skill.description?.toLowerCase().includes(query)
      )
    }
    if (filterVerifiedOrg) {
      filteredSk = filteredSk.filter((s) => {
        const identityData = s._meta?.['io.modelcontextprotocol.registry/publisher-provided']?.['aregistry.ai/metadata']?.identity
        return identityData?.org_is_verified === true
      })
    }
    if (filterVerifiedPublisher) {
      filteredSk = filteredSk.filter((s) => {
        const identityData = s._meta?.['io.modelcontextprotocol.registry/publisher-provided']?.['aregistry.ai/metadata']?.identity
        return identityData?.publisher_identity_verified_by_jwt === true
      })
    }
    setFilteredSkills(sortByCreatedDesc(filteredSk))

    let filteredA = agents
    if (debouncedSearch) {
      filteredA = filteredA.filter(
        ({agent}) =>
          agent.name?.toLowerCase().includes(query) ||
          agent.modelProvider?.toLowerCase().includes(query) ||
          agent.description?.toLowerCase().includes(query)
      )
    }
    if (filterVerifiedOrg) {
      filteredA = filteredA.filter((a) => {
        const identityData = a._meta?.['io.modelcontextprotocol.registry/publisher-provided']?.['aregistry.ai/metadata']?.identity
        return identityData?.org_is_verified === true
      })
    }
    if (filterVerifiedPublisher) {
      filteredA = filteredA.filter((a) => {
        const identityData = a._meta?.['io.modelcontextprotocol.registry/publisher-provided']?.['aregistry.ai/metadata']?.identity
        return identityData?.publisher_identity_verified_by_jwt === true
      })
    }
    filteredA = filteredA.filter((a) => matchesDeploymentFilter(a, deploymentStatusFilter))
    setFilteredAgents(sortByCreatedDesc(filteredA))

    let filteredM = models
    if (debouncedSearch) {
      filteredM = filteredM.filter(
        ({ model }) =>
          model.name?.toLowerCase().includes(query) ||
          model.provider?.toLowerCase().includes(query) ||
          model.model?.toLowerCase().includes(query) ||
          model.description?.toLowerCase().includes(query)
      )
    }
    setFilteredModels(sortByCreatedDesc(filteredM))
  }, [debouncedSearch, skills, agents, models, filterVerifiedOrg, filterVerifiedPublisher, deploymentStatusFilter])

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-muted-foreground">Loading inventory data...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="text-red-500 text-6xl mb-4">⚠️</div>
          <h2 className="text-xl font-bold mb-2">Error Loading Inventory</h2>
          <p className="text-muted-foreground mb-4">{error}</p>
          <Button onClick={fetchData}>Retry</Button>
        </div>
      </div>
    )
  }

  return (
    <>
      <DeployDialog
        open={deployDialogOpen}
        onOpenChange={setDeployDialogOpen}
        itemName={itemToDeploy?.name}
        itemVersion={itemToDeploy?.version}
        itemType={itemToDeploy?.type}
        deploying={deploying}
        onConfirm={confirmDeploy}
        environments={environments}
        loadingEnvironments={loadingEnvironments}
        deployNamespace={deployNamespace}
        onNamespaceChange={setDeployNamespace}
      />

      <UndeployDialog
        open={undeployDialogOpen}
        onOpenChange={setUndeployDialogOpen}
        itemName={itemToUndeploy?.name}
        itemVersion={itemToUndeploy?.version}
        itemType={itemToUndeploy?.type}
        undeploying={undeploying}
        onConfirm={confirmUndeploy}
      />

      {selectedServer && (
        <ServerDetail
          server={selectedServer as ServerResponse & { allVersions?: ServerResponse[] }}
          onClose={handleCloseServerDetail}
          onServerCopied={fetchData}
        />
      )}

      {selectedSkill && (
        <SkillDetail
          skill={selectedSkill}
          onClose={() => setSelectedSkill(null)}
        />
      )}

      {selectedAgent && (
        <AgentDetail
          agent={selectedAgent}
          onClose={() => setSelectedAgent(null)}
        />
      )}

      {selectedModel && (
        <ModelDetail
          model={selectedModel}
          onClose={() => setSelectedModel(null)}
        />
      )}

      {!selectedServer && !selectedSkill && !selectedAgent && !selectedModel && (
    <main className="min-h-screen bg-background">
      {stats && (
        <StatsCards
          stats={stats}
          skillCount={skills.length}
          agentCount={agents.length}
          modelCount={models.length}
        />
      )}

      <div className="container mx-auto px-6 py-8">
        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          <div className="flex items-center gap-4 mb-8">
            <TabsList>
              <TabsTrigger value="servers" className="gap-2">
                <span className="h-4 w-4 flex items-center justify-center">
                  <MCPIcon />
                </span>
                MCP
              </TabsTrigger>
              <TabsTrigger value="agents" className="gap-2">
                <Bot className="h-4 w-4" />
                Agents
              </TabsTrigger>
              <TabsTrigger value="skills" className="gap-2">
                <Zap className="h-4 w-4" />
                Skills
              </TabsTrigger>
              <TabsTrigger value="models" className="gap-2">
                <Brain className="h-4 w-4" />
                Models
              </TabsTrigger>
            </TabsList>

            {/* Search */}
            <div className="flex-1 max-w-md">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Search..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-10 h-9"
                />
              </div>
            </div>

            {/* Deployment Status Filter - only for Servers and Agents */}
            {(activeTab === 'servers' || activeTab === 'agents') && (
              <Tabs value={deploymentStatusFilter} onValueChange={(v) => setDeploymentStatusFilter(v as DeploymentStatus)}>
                <TabsList>
                  <TabsTrigger value="all">All</TabsTrigger>
                  <TabsTrigger value="external">External</TabsTrigger>
                  <TabsTrigger value="running">Running</TabsTrigger>
                  <TabsTrigger value="not_deployed">Not Deployed</TabsTrigger>
                  <TabsTrigger value="failed">Failed</TabsTrigger>
                </TabsList>
              </Tabs>
            )}

            {/* Verified Filters */}
            {(activeTab === 'servers' || activeTab === 'agents' || activeTab === 'skills') && (
              <TooltipProvider>
                <div className="flex items-center gap-3">
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <div className="flex items-center space-x-2">
                        <Checkbox
                          id="filter-verified-org"
                          checked={filterVerifiedOrg}
                          onCheckedChange={(checked: boolean) => setFilterVerifiedOrg(checked)}
                        />
                        <Label
                          htmlFor="filter-verified-org"
                          className="cursor-pointer"
                        >
                          <ShieldCheck className={`h-5 w-5 ${filterVerifiedOrg ? 'text-blue-600' : 'text-gray-400'}`} />
                        </Label>
                      </div>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>Verified Organization</p>
                    </TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <div className="flex items-center space-x-2">
                        <Checkbox
                          id="filter-verified-publisher"
                          checked={filterVerifiedPublisher}
                          onCheckedChange={(checked: boolean) => setFilterVerifiedPublisher(checked)}
                        />
                        <Label
                          htmlFor="filter-verified-publisher"
                          className="cursor-pointer"
                        >
                          <BadgeCheck className={`h-5 w-5 ${filterVerifiedPublisher ? 'text-green-600' : 'text-gray-400'}`} />
                        </Label>
                      </div>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>Verified Publisher</p>
                    </TooltipContent>
                  </Tooltip>
                </div>
              </TooltipProvider>
            )}

            {/* Action Buttons */}
            <div className="flex items-center gap-3 ml-auto">
              <Button
                variant="default"
                className="gap-2"
                onClick={() => setSubmitResourceDialogOpen(true)}
              >
                <GitPullRequest className="h-4 w-4" />
                Submit
              </Button>

              <Button
                variant="ghost"
                size="icon"
                onClick={fetchData}
                title="Refresh"
              >
                <RefreshCw className="h-4 w-4" />
              </Button>
            </div>
          </div>

          {/* MCP Tab */}
          <TabsContent value="servers">
            <div>
              <h2 className="text-lg font-semibold mb-4">
                MCP
                <span className="text-muted-foreground ml-2">
                  ({filteredServers.length})
                </span>
              </h2>

              {filteredServers.length === 0 ? (
                <Card className="p-12">
                  <div className="text-center text-muted-foreground">
                    <div className="w-12 h-12 mx-auto mb-4 opacity-50 flex items-center justify-center">
                      <MCPIcon />
                    </div>
                    <p className="text-lg font-medium mb-2">
                      {groupedServers.length === 0
                        ? "No MCP servers in inventory"
                        : "No MCP servers match your filters"}
                    </p>
                    <p className="text-sm mb-4">
                      {groupedServers.length === 0
                        ? "Submit an MCP server to get started"
                        : "Try adjusting your search or filter criteria"}
                    </p>
                    {groupedServers.length === 0 && (
                      <Button
                        variant="outline"
                        className="gap-2"
                        onClick={() => setSubmitResourceDialogOpen(true)}
                      >
                        <GitPullRequest className="h-4 w-4" />
                        Submit MCP Server
                      </Button>
                    )}
                  </div>
                </Card>
              ) : (
                <>
                  <div className="grid gap-4">
                    {filteredServers
                      .slice((currentPageServers - 1) * itemsPerPage, currentPageServers * itemsPerPage)
                      .map((server, index) => (
                        <ServerCard
                          key={`${server.server.name}-${server.server.version}-${index}`}
                          server={server}
                          versionCount={server.versionCount}
                          onClick={() => handleServerClick(server)}
                          onDeploy={(s) => handleDeploy(s, 'server')}
                          onUndeploy={(s) => handleUndeploy(s, 'server')}
                        />
                      ))}
                  </div>
                  <Pagination
                    currentPage={currentPageServers}
                    totalItems={filteredServers.length}
                    itemsPerPage={itemsPerPage}
                    onPageChange={setCurrentPageServers}
                  />
                </>
              )}
            </div>
          </TabsContent>

          {/* Skills Tab */}
          <TabsContent value="skills">
            <div>
              <h2 className="text-lg font-semibold mb-4">
                Skills
                <span className="text-muted-foreground ml-2">
                  ({filteredSkills.length})
                </span>
              </h2>

              {filteredSkills.length === 0 ? (
                <Card className="p-12">
                  <div className="text-center text-muted-foreground">
                    <div className="w-12 h-12 mx-auto mb-4 opacity-50 flex items-center justify-center text-primary">
                      <Zap className="w-12 h-12" />
                    </div>
                    <p className="text-lg font-medium mb-2">
                      {skills.length === 0
                        ? "No skills in inventory"
                        : "No skills match your filters"}
                    </p>
                    <p className="text-sm mb-4">
                      {skills.length === 0
                        ? "Submit a skill to get started"
                        : "Try adjusting your search or filter criteria"}
                    </p>
                    {skills.length === 0 && (
                      <Button
                        variant="outline"
                        className="gap-2"
                        onClick={() => setSubmitResourceDialogOpen(true)}
                      >
                        <GitPullRequest className="h-4 w-4" />
                        Submit Skill
                      </Button>
                    )}
                  </div>
                </Card>
              ) : (
                <>
                  <div className="grid gap-4">
                    {filteredSkills
                      .slice((currentPageSkills - 1) * itemsPerPage, currentPageSkills * itemsPerPage)
                      .map((skill, index) => (
                        <SkillCard
                          key={`${skill.skill.name}-${skill.skill.version}-${index}`}
                          skill={skill}
                          onClick={() => setSelectedSkill(skill)}
                        />
                      ))}
                  </div>
                  <Pagination
                    currentPage={currentPageSkills}
                    totalItems={filteredSkills.length}
                    itemsPerPage={itemsPerPage}
                    onPageChange={setCurrentPageSkills}
                  />
                </>
              )}
            </div>
          </TabsContent>

          {/* Agents Tab */}
          <TabsContent value="agents">
            <div>
              <h2 className="text-lg font-semibold mb-4">
                Agents
                <span className="text-muted-foreground ml-2">
                  ({filteredAgents.length})
                </span>
              </h2>

              {filteredAgents.length === 0 ? (
                <Card className="p-12">
                  <div className="text-center text-muted-foreground">
                    <div className="w-12 h-12 mx-auto mb-4 opacity-50 flex items-center justify-center text-primary">
                      <Bot className="w-12 h-12" />
                    </div>
                    <p className="text-lg font-medium mb-2">
                      {agents.length === 0
                        ? "No agents in inventory"
                        : "No agents match your filters"}
                    </p>
                    <p className="text-sm mb-4">
                      {agents.length === 0
                        ? "Submit an agent to get started"
                        : "Try adjusting your search or filter criteria"}
                    </p>
                    {agents.length === 0 && (
                      <Button
                        variant="outline"
                        className="gap-2"
                        onClick={() => setSubmitResourceDialogOpen(true)}
                      >
                        <GitPullRequest className="h-4 w-4" />
                        Submit Agent
                      </Button>
                    )}
                  </div>
                </Card>
              ) : (
                <>
                  <div className="grid gap-4">
                    {filteredAgents
                      .slice((currentPageAgents - 1) * itemsPerPage, currentPageAgents * itemsPerPage)
                      .map((agent, index) => (
                        <AgentCard
                          key={`${agent.agent.name}-${agent.agent.version}-${index}`}
                          agent={agent}
                          onClick={() => setSelectedAgent(agent)}
                          onDeploy={(a) => handleDeploy(a, 'agent')}
                          onUndeploy={(a) => handleUndeploy(a, 'agent')}
                        />
                      ))}
                  </div>
                  <Pagination
                    currentPage={currentPageAgents}
                    totalItems={filteredAgents.length}
                    itemsPerPage={itemsPerPage}
                    onPageChange={setCurrentPageAgents}
                  />
                </>
              )}
            </div>
          </TabsContent>

          {/* Models Tab */}
          <TabsContent value="models">
            <div>
              <h2 className="text-lg font-semibold mb-4">
                Models
                <span className="text-muted-foreground ml-2">
                  ({filteredModels.length})
                </span>
              </h2>

              {filteredModels.length === 0 ? (
                <Card className="p-12">
                  <div className="text-center text-muted-foreground">
                    <div className="w-12 h-12 mx-auto mb-4 opacity-50 flex items-center justify-center text-primary">
                      <Brain className="w-12 h-12" />
                    </div>
                    <p className="text-lg font-medium mb-2">
                      {models.length === 0
                        ? "No models in inventory"
                        : "No models match your filters"}
                    </p>
                    <p className="text-sm mb-4">
                      {models.length === 0
                        ? "Add models to the inventory to get started"
                        : "Try adjusting your search or filter criteria"}
                    </p>
                  </div>
                </Card>
              ) : (
                <>
                  <div className="grid gap-4">
                    {filteredModels
                      .slice((currentPageModels - 1) * itemsPerPage, currentPageModels * itemsPerPage)
                      .map((model, index) => (
                        <ModelCard
                          key={`${model.model.name}-${index}`}
                          model={model}
                          onClick={() => setSelectedModel(model)}
                        />
                      ))}
                  </div>
                  <Pagination
                    currentPage={currentPageModels}
                    totalItems={filteredModels.length}
                    itemsPerPage={itemsPerPage}
                    onPageChange={setCurrentPageModels}
                  />
                </>
              )}
            </div>
          </TabsContent>
        </Tabs>
      </div>

      {/* Submit Resource Dialog */}
      <SubmitResourceDialog
        open={submitResourceDialogOpen}
        onOpenChange={setSubmitResourceDialogOpen}
      />
      </main>
      )}
    </>
  )
}
