"use client"

import { useEffect, useState, useRef } from "react"
import { useSession } from "next-auth/react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { ServerCard } from "@/components/server-card"
import { SkillCard } from "@/components/skill-card"
import { AgentCard } from "@/components/agent-card"
import { ModelCard } from "@/components/model-card"
import { ServerDetail } from "@/components/server-detail"
import { SkillDetail } from "@/components/skill-detail"
import { AgentDetail } from "@/components/agent-detail"
import { ModelDetail } from "@/components/model-detail"
import { SubmitResourceDialog } from "@/components/submit-resource-dialog"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { adminApiClient, ServerResponse, SkillResponse, AgentResponse, ModelResponse, ServerStats } from "@/lib/admin-api"
import MCPIcon from "@/components/icons/mcp"
import { toast } from "sonner"
import {
  Search,
  RefreshCw,
  Zap,
  Bot,
  Brain,
  ArrowUpDown,
  Filter,
  GitPullRequest,
  Rocket,
  Trash2,
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
  const { data: session } = useSession()
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
  const [sortBy, setSortBy] = useState<"name" | "date">("name")
  const [filterVerifiedOrg, setFilterVerifiedOrg] = useState(false)
  const [filterVerifiedPublisher, setFilterVerifiedPublisher] = useState(false)
  const [filterSkillVerifiedOrg, setFilterSkillVerifiedOrg] = useState(false)
  const [filterSkillVerifiedPublisher, setFilterSkillVerifiedPublisher] = useState(false)
  const [submitResourceDialogOpen, setSubmitResourceDialogOpen] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedServer, setSelectedServer] = useState<ServerResponse | null>(null)
  const [selectedSkill, setSelectedSkill] = useState<SkillResponse | null>(null)
  const [selectedAgent, setSelectedAgent] = useState<AgentResponse | null>(null)
  const [selectedModel, setSelectedModel] = useState<ModelResponse | null>(null)

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

  // Delete dialog state
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [itemToDelete, setItemToDelete] = useState<{ item: ServerResponse | AgentResponse | SkillResponse | ModelResponse, type: string } | null>(null)
  const [deleting, setDeleting] = useState(false)

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
  const getResourceDate = (server: ServerResponse): Date | null => {
    // Backend sets updatedAt to CreationTimestamp (catalog creation time)
    const dateStr = server._meta?.['io.modelcontextprotocol.registry/official']?.updatedAt
    if (!dateStr) return null
    try {
      return new Date(dateStr)
    } catch {
      return null
    }
  }

  // Get deployment status for filtering
  const getDeploymentStatus = (item: ServerResponse | AgentResponse): "external" | "not_deployed" | "running" | "failed" => {
    const isExternal = item._meta?.isDiscovered || item._meta?.source === 'discovery'
    const deployment = item._meta?.deployment

    // Check deployment status first (applies to both managed and discovered)
    // If deployment info exists, check ready status
    if (deployment) {
      // Ready field is now always present (not omitempty), so we can check boolean directly
      if (deployment.ready === true) return "running"
      if (deployment.ready === false) return "failed"
    }

    // No deployment info - distinguish between external and managed
    if (isExternal) return "external"
    return "not_deployed"
  }

  // Group servers by name, keeping the latest version as the representative
  const groupServersByName = (servers: ServerResponse[]): GroupedServer[] => {
    const grouped = new Map<string, ServerResponse[]>()
    
    // Group all versions by server name
    servers.forEach((server) => {
      const name = server.server.name
      if (!grouped.has(name)) {
        grouped.set(name, [])
      }
      grouped.get(name)!.push(server)
    })
    
    // Convert to GroupedServer array, using the latest version as representative
    return Array.from(grouped.entries()).map(([name, versions]) => {
      // Sort versions by date (newest first) or version string
      const sortedVersions = [...versions].sort((a, b) => {
        const dateA = getResourceDate(a)
        const dateB = getResourceDate(b)
        if (dateA && dateB) {
          return dateB.getTime() - dateA.getTime()
        }
        // Fallback to version string comparison
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
  const fetchData = async () => {
    try {
      setLoading(true)
      setError(null)
      
      // Fetch all servers (with pagination if needed)
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

      // Fetch all skills (with pagination if needed)
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

      // Fetch all agents (with pagination if needed)
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

      // Fetch all models (with pagination if needed)
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

      // Group servers by name
      const grouped = groupServersByName(allServers)
      setGroupedServers(grouped)

      // Set stats
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
  }

  useEffect(() => {
    fetchData()
  }, [])

  // Restore scroll position when returning from server detail
  useEffect(() => {
    if (!selectedServer && shouldRestoreScrollRef.current) {
      // Use setTimeout to ensure DOM has updated
      setTimeout(() => {
        window.scrollTo({
          top: scrollPositionRef.current,
          behavior: 'instant' as ScrollBehavior
        })
        shouldRestoreScrollRef.current = false
      }, 0)
    }
  }, [selectedServer])

  // Handle server card click - save scroll position before navigating
  const handleServerClick = (server: GroupedServer) => {
    scrollPositionRef.current = window.scrollY
    shouldRestoreScrollRef.current = true
    setSelectedServer(server)
  }

  // Handle closing server detail - flag for scroll restoration
  const handleCloseServerDetail = () => {
    setSelectedServer(null)
  }

  // Handle deploy
  const handleDeploy = (item: ServerResponse | AgentResponse, type: 'server' | 'agent') => {
    const name = type === 'server' ? (item as ServerResponse).server.name : (item as AgentResponse).agent.name
    const version = type === 'server' ? (item as ServerResponse).server.version : (item as AgentResponse).agent.version
    setItemToDeploy({ name, version, type })
    setDeployDialogOpen(true)
    fetchEnvironments()
  }

  // Handle undeploy
  const handleUndeploy = (item: ServerResponse | AgentResponse, type: 'server' | 'agent') => {
    const name = type === 'server' ? (item as ServerResponse).server.name : (item as AgentResponse).agent.name
    const version = type === 'server' ? (item as ServerResponse).server.version : (item as AgentResponse).agent.version
    setItemToUndeploy({ name, version, type })
    setUndeployDialogOpen(true)
  }

  // Fetch environments when deploy dialog opens
  const fetchEnvironments = async () => {
    setLoadingEnvironments(true)
    try {
      const envs = await adminApiClient.listEnvironments()
      if (envs && envs.length > 0) {
        setEnvironments(envs)
        setDeployNamespace(envs[0].namespace)
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
      await fetchData() // Refresh data
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
      await fetchData() // Refresh data
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to undeploy resource')
    } finally {
      setUndeploying(false)
    }
  }

  // Handle delete
  const handleDelete = (item: ServerResponse | AgentResponse | SkillResponse | ModelResponse, type: string) => {
    setItemToDelete({ item, type })
    setDeleteDialogOpen(true)
  }

  const confirmDelete = async () => {
    if (!itemToDelete) return

    try {
      setDeleting(true)
      const { item, type } = itemToDelete

      if (type === 'server') {
        const server = item as ServerResponse
        await adminApiClient.deleteServer(server.server.name, server.server.version)
      } else if (type === 'agent') {
        const agent = item as AgentResponse
        await adminApiClient.deleteAgent(agent.agent.name, agent.agent.version)
      } else if (type === 'skill') {
        const skill = item as SkillResponse
        await adminApiClient.deleteSkill(skill.skill.name, skill.skill.version)
      } else if (type === 'model') {
        const model = item as ModelResponse
        await adminApiClient.deleteModel(model.model.name)
      }

      setDeleteDialogOpen(false)
      setItemToDelete(null)
      toast.success(`Successfully deleted resource`)
      await fetchData() // Refresh data
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to delete resource')
    } finally {
      setDeleting(false)
    }
  }

  // Reset all page numbers when search query changes
  useEffect(() => {
    setCurrentPageServers(1)
    setCurrentPageSkills(1)
    setCurrentPageAgents(1)
    setCurrentPageModels(1)
  }, [searchQuery, deploymentStatusFilter])

  // Filter and sort servers based on search query, sort option, and deployment status
  useEffect(() => {
    let filtered = [...groupedServers]

    // Filter by search query
    if (searchQuery) {
      const query = searchQuery.toLowerCase()
      filtered = filtered.filter(
        (s) =>
          s.server.name.toLowerCase().includes(query) ||
          s.server.title?.toLowerCase().includes(query) ||
          s.server.description?.toLowerCase().includes(query)
      )
    }

    // Filter by verified organization
    if (filterVerifiedOrg) {
      filtered = filtered.filter((s) => {
        const identityData = s.server._meta?.['io.modelcontextprotocol.registry/publisher-provided']?.['aregistry.ai/metadata']?.identity
        return identityData?.org_is_verified === true
      })
    }

    // Filter by verified publisher
    if (filterVerifiedPublisher) {
      filtered = filtered.filter((s) => {
        const identityData = s.server._meta?.['io.modelcontextprotocol.registry/publisher-provided']?.['aregistry.ai/metadata']?.identity
        return identityData?.publisher_identity_verified_by_jwt === true
      })
    }

    // Filter by deployment status
    if (deploymentStatusFilter !== "all") {
      filtered = filtered.filter((s) => getDeploymentStatus(s) === deploymentStatusFilter)
    }

    // Sort servers
    filtered.sort((a, b) => {
      switch (sortBy) {
        case "date": {
          const dateA = getResourceDate(a)
          const dateB = getResourceDate(b)
          if (!dateA && !dateB) return 0
          if (!dateA) return 1
          if (!dateB) return -1
          return dateB.getTime() - dateA.getTime()
        }
        case "name":
        default:
          return a.server.name.localeCompare(b.server.name)
      }
    })

    setFilteredServers(filtered)
  }, [searchQuery, groupedServers, sortBy, filterVerifiedOrg, filterVerifiedPublisher, deploymentStatusFilter])

  // Filter skills, agents, and models based on search query and deployment status
  useEffect(() => {
    const query = searchQuery.toLowerCase()

    // Filter skills (skills don't have deployment status, they're always "active")
    let filteredSk = skills
    if (searchQuery) {
      filteredSk = filteredSk.filter(
        (s) =>
          s.skill.name.toLowerCase().includes(query) ||
          s.skill.title?.toLowerCase().includes(query) ||
          s.skill.description?.toLowerCase().includes(query)
      )
    }
    if (filterSkillVerifiedOrg) {
      filteredSk = filteredSk.filter((s) => {
        const publisherProvided = s._meta?.["io.modelcontextprotocol.registry/publisher-provided"] as Record<string, unknown> | undefined
        const aregistryMetadata = publisherProvided?.["aregistry.ai/metadata"] as Record<string, unknown> | undefined
        const identity = aregistryMetadata?.["identity"] as Record<string, unknown> | undefined
        return identity?.["org_is_verified"] === true
      })
    }
    if (filterSkillVerifiedPublisher) {
      filteredSk = filteredSk.filter((s) => {
        const publisherProvided = s._meta?.["io.modelcontextprotocol.registry/publisher-provided"] as Record<string, unknown> | undefined
        const aregistryMetadata = publisherProvided?.["aregistry.ai/metadata"] as Record<string, unknown> | undefined
        const identity = aregistryMetadata?.["identity"] as Record<string, unknown> | undefined
        return identity?.["publisher_identity_verified_by_jwt"] === true
      })
    }
    setFilteredSkills(filteredSk)

    // Filter agents
    let filteredA = agents
    if (searchQuery) {
      filteredA = filteredA.filter(
        ({agent}) =>
          agent.name?.toLowerCase().includes(query) ||
          agent.modelProvider?.toLowerCase().includes(query) ||
          agent.description?.toLowerCase().includes(query)
      )
    }
    // Filter by deployment status
    if (deploymentStatusFilter !== "all") {
      filteredA = filteredA.filter((a) => getDeploymentStatus(a) === deploymentStatusFilter)
    }
    setFilteredAgents(filteredA)

    // Filter models (models don't have deployment status)
    let filteredM = models
    if (searchQuery) {
      filteredM = filteredM.filter(
        ({ model }) =>
          model.name?.toLowerCase().includes(query) ||
          model.provider?.toLowerCase().includes(query) ||
          model.model?.toLowerCase().includes(query) ||
          model.description?.toLowerCase().includes(query)
      )
    }
    setFilteredModels(filteredM)
  }, [searchQuery, skills, agents, models, filterSkillVerifiedOrg, filterSkillVerifiedPublisher, deploymentStatusFilter])

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

  // Show server detail view if a server is selected
  return (
    <>
      {/* Deploy Dialog */}
      <Dialog open={deployDialogOpen} onOpenChange={setDeployDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Deploy Resource</DialogTitle>
            <DialogDescription>
              Deploy <strong>{itemToDeploy?.name}</strong> (version {itemToDeploy?.version})?
              <br />
              <br />
              This will start the {itemToDeploy?.type === 'server' ? 'MCP server' : 'agent'} on your system.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>Deployment destination</Label>
              <p className="text-sm text-muted-foreground">Kubernetes</p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="deploy-namespace">Namespace / Environment</Label>
              <Select value={deployNamespace} onValueChange={setDeployNamespace} disabled={loadingEnvironments}>
                <SelectTrigger id="deploy-namespace">
                  <SelectValue placeholder={loadingEnvironments ? "Loading..." : "Select namespace"} />
                </SelectTrigger>
                <SelectContent>
                  {environments.map((env) => (
                    <SelectItem key={env.namespace} value={env.namespace}>
                      {env.name}/{env.namespace}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                Choose which namespace/environment to deploy to
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setDeployDialogOpen(false)}
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

      {/* Undeploy Dialog */}
      <Dialog open={undeployDialogOpen} onOpenChange={setUndeployDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Undeploy Resource</DialogTitle>
            <DialogDescription>
              Are you sure you want to undeploy <strong>{itemToUndeploy?.name}</strong> (version {itemToUndeploy?.version})?
              <br />
              <br />
              This will stop the {itemToUndeploy?.type === 'server' ? 'MCP server' : 'agent'} and remove it from your deployments.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setUndeployDialogOpen(false)}
              disabled={undeploying}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={confirmUndeploy}
              disabled={undeploying}
            >
              {undeploying ? 'Undeploying...' : 'Undeploy'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Resource</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete <strong>
                {itemToDelete?.type === 'server' ? (itemToDelete.item as ServerResponse).server.name :
                 itemToDelete?.type === 'agent' ? (itemToDelete.item as AgentResponse).agent.name :
                 itemToDelete?.type === 'skill' ? (itemToDelete.item as SkillResponse).skill.name :
                 itemToDelete?.type === 'model' ? (itemToDelete.item as ModelResponse).model.name : ''}
              </strong>?
              <br />
              <br />
              This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setDeleteDialogOpen(false)}
              disabled={deleting}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={confirmDelete}
              disabled={deleting}
            >
              {deleting ? 'Deleting...' : 'Delete'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

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
      {/* Stats Section */}
      {stats && (
        <div className="bg-muted/30 border-b">
          <div className="container mx-auto px-6 py-6">
            <div className="grid gap-4 md:grid-cols-4">
              <Card className="p-4 hover:shadow-md transition-all duration-200 border hover:border-primary/20">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-primary/10 rounded-lg flex items-center justify-center">
                    <span className="h-5 w-5 text-primary flex items-center justify-center">
                      <MCPIcon />
                    </span>
                  </div>
                  <div>
                    <p className="text-2xl font-bold">{stats.total_server_names}</p>
                    <p className="text-xs text-muted-foreground">MCP</p>
                  </div>
                </div>
              </Card>

              <Card className="p-4 hover:shadow-md transition-all duration-200 border hover:border-primary/20">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-primary/20 rounded-lg flex items-center justify-center">
                    <Zap className="h-5 w-5 text-primary" />
                  </div>
                  <div>
                    <p className="text-2xl font-bold">{skills.length}</p>
                    <p className="text-xs text-muted-foreground">Skills</p>
                  </div>
                </div>
              </Card>

              <Card className="p-4 hover:shadow-md transition-all duration-200 border hover:border-primary/20">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-primary/30 rounded-lg flex items-center justify-center">
                    <Bot className="h-5 w-5 text-primary" />
                  </div>
                  <div>
                    <p className="text-2xl font-bold">{agents.length}</p>
                    <p className="text-xs text-muted-foreground">Agents</p>
                  </div>
                </div>
              </Card>

              <Card className="p-4 hover:shadow-md transition-all duration-200 border hover:border-primary/20">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-primary/40 rounded-lg flex items-center justify-center">
                    <Brain className="h-5 w-5 text-primary" />
                  </div>
                  <div>
                    <p className="text-2xl font-bold">{models.length}</p>
                    <p className="text-xs text-muted-foreground">Models</p>
                  </div>
                </div>
              </Card>
            </div>
          </div>
        </div>
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

            {/* Sort */}
            <div className="flex items-center gap-2">
              <ArrowUpDown className="h-4 w-4 text-muted-foreground" />
              <Select value={sortBy} onValueChange={(value: "name" | "date") => setSortBy(value)}>
                <SelectTrigger className="w-[150px] h-9">
                  <SelectValue placeholder="Sort by..." />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="name">Name</SelectItem>
                  
                  <SelectItem value="date">Date</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Verified Filters - for Servers and Agents */}
            {(activeTab === 'servers' || activeTab === 'agents') && (
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
            {/* Server List */}
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
                          onDelete={(s) => handleDelete(s, 'server')}
                        />
                      ))}
                  </div>
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
                </>
              )}
            </div>
          </TabsContent>

          {/* Skills Tab */}
          <TabsContent value="skills">
            {/* Filter controls */}
            <div className="flex items-center justify-end mb-6">
              <div className="flex items-center gap-4">
                <Filter className="h-4 w-4 text-muted-foreground" />
                <div className="flex items-center space-x-2">
                  <input
                    type="checkbox"
                    id="filter-skill-verified-org"
                    checked={filterSkillVerifiedOrg}
                    onChange={(e) => setFilterSkillVerifiedOrg(e.target.checked)}
                    className="h-4 w-4 rounded border-gray-300"
                  />
                  <label
                    htmlFor="filter-skill-verified-org"
                    className="text-sm font-normal cursor-pointer"
                  >
                    Verified Organization
                  </label>
                </div>
                <div className="flex items-center space-x-2">
                  <input
                    type="checkbox"
                    id="filter-skill-verified-publisher"
                    checked={filterSkillVerifiedPublisher}
                    onChange={(e) => setFilterSkillVerifiedPublisher(e.target.checked)}
                    className="h-4 w-4 rounded border-gray-300"
                  />
                  <label
                    htmlFor="filter-skill-verified-publisher"
                    className="text-sm font-normal cursor-pointer"
                  >
                    Verified Publisher
                  </label>
                </div>
              </div>
            </div>

            {/* Skills List */}
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
                          onDelete={(s) => handleDelete(s, 'skill')}
                        />
                      ))}
                  </div>
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
                </>
              )}
            </div>
          </TabsContent>

          {/* Agents Tab */}
          <TabsContent value="agents">
            {/* Agents List */}
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
                          onDelete={(a) => handleDelete(a, 'agent')}
                        />
                      ))}
                  </div>
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
                </>
              )}
            </div>
          </TabsContent>

          {/* Models Tab */}
          <TabsContent value="models">
            {/* Models List */}
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
                          onDelete={(m) => handleDelete(m, 'model')}
                        />
                      ))}
                  </div>
                  {filteredModels.length > itemsPerPage && (
                    <div className="flex items-center justify-center gap-2 mt-6">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setCurrentPageModels(p => Math.max(1, p - 1))}
                        disabled={currentPageModels === 1}
                      >
                        Previous
                      </Button>
                      <span className="text-sm text-muted-foreground">
                        Page {currentPageModels} of {Math.ceil(filteredModels.length / itemsPerPage)}
                      </span>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setCurrentPageModels(p => Math.min(Math.ceil(filteredModels.length / itemsPerPage), p + 1))}
                        disabled={currentPageModels >= Math.ceil(filteredModels.length / itemsPerPage)}
                      >
                        Next
                      </Button>
                    </div>
                  )}
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
