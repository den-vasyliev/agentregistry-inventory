"use client"

import { useEffect, useState } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { GroupedMCPServerCard } from "@/components/grouped-mcp-server-card"
import { ServerDetailView } from "@/components/server-detail-view"
import { InstallDialog } from "@/components/install-dialog"
import { AddRegistryDialog } from "@/components/add-registry-dialog"
import { GroupedMCPServer, MCPServerWithStatus } from "@/lib/types"
import { apiClient, Registry } from "@/lib/api"
import { transformServerList, groupServersByName } from "@/lib/transforms"
import {
  Server,
  Package,
  Database,
  Search,
  Plus,
  Settings,
  HardDrive,
  Globe,
  Link as LinkIcon,
  Trash2,
  RefreshCw,
  Zap,
  Bot,
} from "lucide-react"

type ViewMode = "browse" | "installed"
type ResourceType = "servers" | "skills" | "agents"

export default function Home() {
  const [viewMode, setViewMode] = useState<ViewMode>("browse")
  const [resourceType, setResourceType] = useState<ResourceType>("servers")
  const [registries, setRegistries] = useState<Registry[]>([])
  const [servers, setServers] = useState<MCPServerWithStatus[]>([])
  const [groupedServers, setGroupedServers] = useState<GroupedMCPServer[]>([])
  const [skills, setSkills] = useState<any[]>([])
  const [agents, setAgents] = useState<any[]>([])
  const [filteredGroupedServers, setFilteredGroupedServers] = useState<GroupedMCPServer[]>([])
  const [filteredSkills, setFilteredSkills] = useState<any[]>([])
  const [filteredAgents, setFilteredAgents] = useState<any[]>([])
  const [searchQuery, setSearchQuery] = useState("")
  const [selectedServer, setSelectedServer] = useState<MCPServerWithStatus | null>(null)
  const [installDialogOpen, setInstallDialogOpen] = useState(false)
  const [addRegistryDialogOpen, setAddRegistryDialogOpen] = useState(false)
  const [serverToInstall, setServerToInstall] = useState<GroupedMCPServer | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Fetch data from API
  const fetchData = async () => {
    try {
      setLoading(true)
      setError(null)
      const [registriesData, serversData, skillsData, agentsData] = await Promise.all([
        apiClient.getRegistries(),
        apiClient.getServers(),
        apiClient.getSkills(),
        apiClient.getAgents(),
      ])
      setRegistries(registriesData || [])
      const transformedServers = transformServerList(serversData || [])
      setServers(transformedServers)
      setGroupedServers(groupServersByName(transformedServers))
      setSkills(skillsData || [])
      setAgents(agentsData || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch data")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  // Filter resources based on view mode and search query
  useEffect(() => {
    // Filter grouped servers
    let filteredGS = groupedServers
    if (viewMode === "installed") {
      filteredGS = groupedServers.filter((gs) => gs.hasInstalledVersion)
    }
    if (searchQuery) {
      filteredGS = filteredGS.filter(
        (gs) =>
          gs.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          gs.title?.toLowerCase().includes(searchQuery.toLowerCase()) ||
          gs.description.toLowerCase().includes(searchQuery.toLowerCase())
      )
    }
    setFilteredGroupedServers(filteredGS)

    // Filter skills
    let filteredSk = skills
    if (viewMode === "installed") {
      filteredSk = skills.filter((s) => s.installed)
    }
    if (searchQuery) {
      filteredSk = filteredSk.filter(
        (s) =>
          s.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          s.title?.toLowerCase().includes(searchQuery.toLowerCase()) ||
          s.description.toLowerCase().includes(searchQuery.toLowerCase())
      )
    }
    setFilteredSkills(filteredSk)

    // Filter agents
    let filteredA = agents
    if (viewMode === "installed") {
      filteredA = agents.filter((a) => a.installed)
    }
    if (searchQuery) {
      filteredA = filteredA.filter(
        (a) =>
          a.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          a.title?.toLowerCase().includes(searchQuery.toLowerCase()) ||
          a.description.toLowerCase().includes(searchQuery.toLowerCase())
      )
    }
    setFilteredAgents(filteredA)
  }, [viewMode, searchQuery, groupedServers, skills, agents])

  const handleInstall = (groupedServer: GroupedMCPServer) => {
    setServerToInstall(groupedServer)
    setInstallDialogOpen(true)
  }

  const handleInstallConfirm = async (server: MCPServerWithStatus, config: Record<string, string>) => {
    if (!server._dbId) return

    try {
      await apiClient.installServer(server._dbId, config)
      
      // Update local state
      const updatedServers = servers.map((s) =>
        s._dbId === server._dbId
          ? { ...s, installed: true, installedAt: new Date().toISOString() }
          : s
      )
      setServers(updatedServers)
      setGroupedServers(groupServersByName(updatedServers))

      setServerToInstall(null)
    } catch (err) {
      console.error("Failed to install server:", err)
      alert(err instanceof Error ? err.message : "Failed to install server")
    }
  }

  const handleUninstall = async (groupedServer: GroupedMCPServer) => {
    // Find all installed versions and uninstall them
    const installedVersions = groupedServer.versions.filter(v => v.installed)
    
    try {
      // Uninstall all installed versions
      await Promise.all(
        installedVersions.map(version => 
          version._dbId ? apiClient.uninstallServer(version._dbId) : Promise.resolve()
        )
      )
      
      // Update local state
      const updatedServers = servers.map((s) => {
        const shouldUninstall = installedVersions.some(v => v._dbId === s._dbId)
        return shouldUninstall
          ? { ...s, installed: false, installedAt: undefined }
          : s
      })
      setServers(updatedServers)
      setGroupedServers(groupServersByName(updatedServers))
    } catch (err) {
      console.error("Failed to uninstall server:", err)
      alert(err instanceof Error ? err.message : "Failed to uninstall server")
    }
  }

  const handleAddRegistry = async (name: string, url: string, type: string) => {
    await apiClient.addRegistry(name, url, type)
    // Refresh data after adding registry
    await fetchData()
  }

  const handleRemoveRegistry = async (id: number) => {
    if (!confirm("Are you sure you want to remove this registry?")) return
    
    try {
      await apiClient.removeRegistry(id)
      await fetchData()
    } catch (err) {
      console.error("Failed to remove registry:", err)
      alert(err instanceof Error ? err.message : "Failed to remove registry")
    }
  }

  const installedCount = groupedServers.filter((gs) => gs.hasInstalledVersion).length
  const totalServers = groupedServers.length

  if (selectedServer) {
    return (
      <ServerDetailView
        server={selectedServer}
        onClose={() => setSelectedServer(null)}
      />
    )
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-muted-foreground">Loading...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="text-red-500 text-6xl mb-4">⚠️</div>
          <h2 className="text-xl font-bold mb-2">Error Loading Data</h2>
          <p className="text-muted-foreground mb-4">{error}</p>
          <Button onClick={fetchData}>Retry</Button>
        </div>
      </div>
    )
  }

  return (
    <main className="min-h-screen bg-background">
      <div className="border-b">
        <div className="container mx-auto px-6 py-6">
          <div className="flex items-center justify-between mb-6">
            <div>
              <h1 className="text-3xl font-bold mb-2">arctl</h1>
              <p className="text-muted-foreground">AI Registry and Runtime</p>
            </div>
            <Button
              variant="outline"
              size="icon"
              onClick={fetchData}
              title="Refresh data"
            >
              <RefreshCw className="h-5 w-5" />
            </Button>
          </div>

          {/* Stats */}
          <div className="grid gap-4 md:grid-cols-4 mb-6">
            <Card className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-primary/10 rounded-lg">
                  <Database className="h-5 w-5 text-primary" />
                </div>
                <div>
                  <p className="text-2xl font-bold">{registries.length}</p>
                  <p className="text-xs text-muted-foreground">Registries</p>
                </div>
              </div>
            </Card>

            <Card className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-primary/10 rounded-lg">
                  <Server className="h-5 w-5 text-primary" />
                </div>
                <div>
                  <p className="text-2xl font-bold">{totalServers}</p>
                  <p className="text-xs text-muted-foreground">Total Servers</p>
                </div>
              </div>
            </Card>

            <Card className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-green-600/10 rounded-lg">
                  <HardDrive className="h-5 w-5 text-green-600" />
                </div>
                <div>
                  <p className="text-2xl font-bold">{installedCount}</p>
                  <p className="text-xs text-muted-foreground">Installed</p>
                </div>
              </div>
            </Card>

            <Card className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-blue-600/10 rounded-lg">
                  <Package className="h-5 w-5 text-blue-600" />
                </div>
                <div>
                  <p className="text-2xl font-bold">{totalServers - installedCount}</p>
                  <p className="text-xs text-muted-foreground">Available</p>
                </div>
              </div>
            </Card>
          </div>

          {/* Navigation Tabs */}
          <div className="flex items-center gap-4">
            <div className="flex gap-2">
              <Button
                variant={viewMode === "browse" ? "default" : "outline"}
                onClick={() => setViewMode("browse")}
                className="gap-2"
              >
                <Globe className="h-4 w-4" />
                Browse Registry
              </Button>
              <Button
                variant={viewMode === "installed" ? "default" : "outline"}
                onClick={() => setViewMode("installed")}
                className="gap-2"
              >
                <HardDrive className="h-4 w-4" />
                Installed
                {installedCount > 0 && (
                  <Badge variant="secondary" className="ml-1">
                    {installedCount}
                  </Badge>
                )}
              </Button>
            </div>
          </div>
        </div>
      </div>

      <div className="container mx-auto px-6 py-8">
        {/* Search and Filters */}
        <div className="flex items-center gap-4 mb-8">
          <div className="relative flex-1 max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search servers..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-10"
            />
          </div>
          {viewMode === "browse" && (
            <Button
              variant="outline"
              className="gap-2"
              onClick={() => setAddRegistryDialogOpen(true)}
            >
              <Plus className="h-4 w-4" />
              Connect Registry
            </Button>
          )}
        </div>

        {/* Connected Registries (Browse mode only) */}
        {viewMode === "browse" && (
          <div className="mb-8">
            <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
              <LinkIcon className="h-5 w-5" />
              Connected Registries
            </h2>
            <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-4">
              {registries && registries.length > 0 ? registries.map((registry) => (
                <Card key={registry.id} className="p-4">
                  <div className="flex items-start justify-between mb-2">
                    <div className="flex-1">
                      <h3 className="font-semibold">{registry.name}</h3>
                      <p className="text-sm text-muted-foreground truncate">
                        {registry.url}
                      </p>
                    </div>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8"
                      onClick={() => handleRemoveRegistry(registry.id)}
                      title="Remove registry"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary" className="text-xs">
                      {registry.type}
                    </Badge>
                    <Badge variant="default" className="text-xs bg-green-600">
                      Connected
                    </Badge>
                  </div>
                </Card>
              )) : (
                <Card className="p-8 col-span-full">
                  <div className="text-center text-muted-foreground">
                    <Database className="w-12 h-12 mx-auto mb-4 opacity-50" />
                    <p className="text-lg font-medium mb-2">No Registries Connected</p>
                    <p className="text-sm mb-4">Connect to a registry to browse available servers</p>
                    <Button
                      variant="outline"
                      className="gap-2"
                      onClick={() => setAddRegistryDialogOpen(true)}
                    >
                      <Plus className="h-4 w-4" />
                      Connect Registry
                    </Button>
                  </div>
                </Card>
              )}
            </div>
          </div>
        )}

        {/* Server List */}
        <div>
          <h2 className="text-lg font-semibold mb-4">
            {viewMode === "installed" ? "Installed Servers" : "Available Servers"}
            <span className="text-muted-foreground ml-2">
              ({filteredGroupedServers.length})
            </span>
          </h2>

          {filteredGroupedServers.length === 0 ? (
            <Card className="p-12">
              <div className="text-center text-muted-foreground">
                <Server className="w-12 h-12 mx-auto mb-4 opacity-50" />
                <p className="text-lg font-medium mb-2">
                  {viewMode === "installed"
                    ? "No installed servers"
                    : "No servers found"}
                </p>
                <p className="text-sm">
                  {viewMode === "installed"
                    ? "Install servers from the Browse Registry view"
                    : searchQuery
                    ? "Try a different search term"
                    : "Connect a registry to see available servers"}
                </p>
              </div>
            </Card>
          ) : (
            <div className="grid gap-4">
              {filteredGroupedServers.map((groupedServer) => (
                <GroupedMCPServerCard
                  key={groupedServer.name}
                  groupedServer={groupedServer}
                  onInstall={handleInstall}
                  onUninstall={handleUninstall}
                  onClick={(gs) => setSelectedServer(gs.latestVersion)}
                />
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Install Dialog */}
      {serverToInstall && (
        <InstallDialog
          open={installDialogOpen}
          onOpenChange={setInstallDialogOpen}
          groupedServer={serverToInstall}
          onConfirm={handleInstallConfirm}
        />
      )}

      {/* Add Registry Dialog */}
      <AddRegistryDialog
        open={addRegistryDialogOpen}
        onOpenChange={setAddRegistryDialogOpen}
        onAdd={handleAddRegistry}
      />
    </main>
  )
}
