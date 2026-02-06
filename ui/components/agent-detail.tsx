"use client"

import { useState, useEffect } from "react"
import dynamic from "next/dynamic"
import { AgentResponse } from "@/lib/admin-api"
import { formatDateTime as formatDate } from "@/lib/utils"
import { Card } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { toast } from "sonner"
import {
  X,
  Calendar,
  Tag,
  ArrowLeft,
  Bot,
  Code,
  Container,
  Cpu,
  Brain,
  Languages,
  Box,
  Clock,
  Github,
  ExternalLink,
  CheckCircle2,
  XCircle,
  Circle,
  BadgeCheck,
  Server,
  Package,
  Globe,
  Wrench,
  Activity,
  MessageSquare,
  ChevronDown,
  ChevronUp,
  Blocks,
  Settings2,
  Network,
} from "lucide-react"
const AgentDependencyGraph = dynamic(
  () => import("@/components/agent-dependency-graph").then((m) => m.AgentDependencyGraph),
  { ssr: false },
)

interface AgentDetailProps {
  agent: AgentResponse
  onClose: () => void
}

export function AgentDetail({ agent, onClose }: AgentDetailProps) {
  const [activeTab, setActiveTab] = useState("overview")
  const [systemMessageExpanded, setSystemMessageExpanded] = useState(false)

  const { agent: agentData, _meta } = agent
  const official = _meta?.['io.modelcontextprotocol.registry/official']
  const deployment = _meta?.deployment

  // Extract metadata
  const publisherMetadata = (agentData as any)._meta?.['io.modelcontextprotocol.registry/publisher-provided']?.['aregistry.ai/metadata']
  const identityData = publisherMetadata?.identity

  // Get owner from metadata or extract from repository URL
  const getOwner = () => {
    // Try to get email from metadata first
    if (publisherMetadata?.contact_email) return publisherMetadata.contact_email
    if (identityData?.email) return identityData.email
    if ((official as any)?.submitter) return (official as any).submitter

    // Fallback to extracting owner/org from GitHub repository URL
    if (agentData.repository?.url) {
      const match = agentData.repository.url.match(/github\.com\/([^\/]+)/)
      if (match) return match[1]
    }

    return null
  }

  const owner = getOwner()

  // kagent dependency data
  const tools = agentData.tools || []
  const skills = agentData.skills || []
  const modelConfigRef = agentData.modelConfigRef
  const systemMessage = agentData.systemMessage

  // Legacy dependency counts
  const mcpServers = agentData.mcpServers || []
  const packages = agentData.packages || []
  const remotes = agentData.remotes || []
  const dependencyCount = tools.length + skills.length + mcpServers.length + packages.length + remotes.length + (modelConfigRef ? 1 : 0)
  const hasDependencies = dependencyCount > 0 || !!systemMessage

  // Handle ESC key to close
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose()
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => {
      window.removeEventListener('keydown', handleKeyDown)
    }
  }, [onClose])

  return (
    <div className="fixed inset-0 bg-background z-50 overflow-y-auto">
      <div className="container mx-auto px-6 py-6">
        {/* Back Button */}
        <Button
          variant="ghost"
          onClick={onClose}
          className="mb-4 gap-2"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Agents
        </Button>

        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-start gap-4 flex-1">
            <div className="w-16 h-16 rounded bg-primary/20 flex items-center justify-center flex-shrink-0 mt-1">
              <Bot className="h-8 w-8 text-primary" />
            </div>
            <div className="flex-1">
              <div className="flex items-center gap-3 mb-2 flex-wrap">
                <h1 className="text-3xl font-bold">{agentData.name}</h1>
                {agentData.agentType && (
                  <Badge variant={agentData.agentType === 'Declarative' ? 'default' : 'secondary'} className="text-sm">
                    {agentData.agentType}
                  </Badge>
                )}
                {agentData.framework && (
                  <Badge variant="outline" className="text-sm">
                    {agentData.framework}
                  </Badge>
                )}
                {agentData.language && (
                  <Badge variant="secondary" className="text-sm">
                    {agentData.language}
                  </Badge>
                )}
              </div>
              {agentData.description && (
                <p className="text-muted-foreground">{agentData.description}</p>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="icon" onClick={onClose}>
              <X className="h-5 w-5" />
            </Button>
          </div>
        </div>

        {/* Quick Info */}
        <div className="flex flex-wrap gap-3 mb-6 text-sm">
          {owner && (
            <div className="flex items-center gap-2 px-3 py-2 bg-primary/10 rounded-md border border-primary/20">
              <BadgeCheck className="h-3.5 w-3.5 text-primary" />
              <span className="text-muted-foreground">Owner:</span>
              <span className="font-medium">{owner}</span>
            </div>
          )}

          <div className="flex items-center gap-2 px-3 py-2 bg-muted rounded-md">
            <Tag className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-muted-foreground">Version:</span>
            <span className="font-medium">{agentData.version}</span>
          </div>

          <div className="flex items-center gap-2 px-3 py-2 bg-muted rounded-md">
            <Circle className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-muted-foreground">Status:</span>
            <span className="font-medium">{agentData.status || official?.status || 'unknown'}</span>
          </div>

          {deployment && (
            <div className={`flex items-center gap-2 px-3 py-2 rounded-md ${deployment.ready ? 'bg-green-500/10' : 'bg-red-500/10'}`}>
              {deployment.ready ? (
                <CheckCircle2 className="h-3.5 w-3.5 text-green-600" />
              ) : (
                <XCircle className="h-3.5 w-3.5 text-red-600" />
              )}
              <span className="text-muted-foreground">Deployment:</span>
              <span className={`font-medium ${deployment.ready ? 'text-green-600' : 'text-red-600'}`}>
                {deployment.ready ? 'Running' : 'Not Ready'}
              </span>
              {deployment.namespace && (
                <span className="text-xs text-muted-foreground">({deployment.namespace})</span>
              )}
            </div>
          )}

          {modelConfigRef && (
            <div className="flex items-center gap-2 px-3 py-2 bg-muted rounded-md">
              <Settings2 className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="text-muted-foreground">Model Config:</span>
              <span className="font-medium font-mono">{modelConfigRef}</span>
            </div>
          )}

          {official?.publishedAt && (
            <div className="flex items-center gap-2 px-3 py-2 bg-muted rounded-md">
              <Calendar className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="text-muted-foreground">Published:</span>
              <span className="font-medium">{formatDate(official.publishedAt)}</span>
            </div>
          )}

          {official?.updatedAt && (
            <div className="flex items-center gap-2 px-3 py-2 bg-muted rounded-md">
              <Clock className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="text-muted-foreground">Updated:</span>
              <span className="font-medium">{formatDate(official.updatedAt)}</span>
            </div>
          )}
        </div>

        {/* Detailed Information Tabs */}
        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          <TabsList>
            <TabsTrigger value="overview">Overview</TabsTrigger>
            {hasDependencies && (
              <TabsTrigger value="dependencies">
                Dependencies
                <Badge variant="secondary" className="ml-1.5 text-xs px-1.5 py-0">
                  {dependencyCount}
                </Badge>
              </TabsTrigger>
            )}
            <TabsTrigger value="technical">Technical Details</TabsTrigger>
            <TabsTrigger value="raw">Raw Data</TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-4">
            {/* Description */}
            {agentData.description && (
              <Card className="p-6">
                <h3 className="text-lg font-semibold mb-4">Description</h3>
                <p className="text-base">{agentData.description}</p>
              </Card>
            )}

            {/* Basic Info */}
            <Card className="p-6">
              <h3 className="text-lg font-semibold mb-4">Basic Information</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="flex items-center gap-3">
                  <Languages className="h-5 w-5 text-muted-foreground" />
                  <div>
                    <p className="text-sm text-muted-foreground">Language</p>
                    <p className="font-medium">{agentData.language}</p>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <Box className="h-5 w-5 text-muted-foreground" />
                  <div>
                    <p className="text-sm text-muted-foreground">Framework</p>
                    <p className="font-medium">{agentData.framework}</p>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <Brain className="h-5 w-5 text-muted-foreground" />
                  <div>
                    <p className="text-sm text-muted-foreground">Model Provider</p>
                    <p className="font-medium">{agentData.modelProvider}</p>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <Cpu className="h-5 w-5 text-muted-foreground" />
                  <div>
                    <p className="text-sm text-muted-foreground">Model Name</p>
                    <p className="font-medium font-mono">{agentData.modelName}</p>
                  </div>
                </div>
              </div>
            </Card>

            {/* Connected Resources Summary */}
            {hasDependencies && (
              <Card className="p-6">
                <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
                  <Wrench className="h-5 w-5" />
                  Connected Resources
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  {tools.length > 0 && (
                    <button
                      onClick={() => setActiveTab("dependencies")}
                      className="flex items-center gap-3 p-3 bg-muted rounded-lg hover:bg-muted/80 transition-colors text-left"
                    >
                      <Wrench className="h-5 w-5 text-muted-foreground" />
                      <div>
                        <p className="font-medium">{tools.length} Tool{tools.length > 1 ? 's' : ''}</p>
                        <p className="text-xs text-muted-foreground truncate">
                          {tools.map(t => t.name).join(', ')}
                        </p>
                      </div>
                    </button>
                  )}
                  {skills.length > 0 && (
                    <button
                      onClick={() => setActiveTab("dependencies")}
                      className="flex items-center gap-3 p-3 bg-muted rounded-lg hover:bg-muted/80 transition-colors text-left"
                    >
                      <Blocks className="h-5 w-5 text-muted-foreground" />
                      <div>
                        <p className="font-medium">{skills.length} Skill{skills.length > 1 ? 's' : ''}</p>
                        <p className="text-xs text-muted-foreground truncate">
                          {skills.map(s => s.split('/').pop() || s).join(', ')}
                        </p>
                      </div>
                    </button>
                  )}
                  {modelConfigRef && (
                    <button
                      onClick={() => setActiveTab("dependencies")}
                      className="flex items-center gap-3 p-3 bg-muted rounded-lg hover:bg-muted/80 transition-colors text-left"
                    >
                      <Settings2 className="h-5 w-5 text-muted-foreground" />
                      <div>
                        <p className="font-medium">Model Config</p>
                        <p className="text-xs text-muted-foreground font-mono truncate">{modelConfigRef}</p>
                      </div>
                    </button>
                  )}
                  {mcpServers.length > 0 && (
                    <button
                      onClick={() => setActiveTab("dependencies")}
                      className="flex items-center gap-3 p-3 bg-muted rounded-lg hover:bg-muted/80 transition-colors text-left"
                    >
                      <Server className="h-5 w-5 text-muted-foreground" />
                      <div>
                        <p className="font-medium">{mcpServers.length} MCP Server{mcpServers.length > 1 ? 's' : ''}</p>
                        <p className="text-xs text-muted-foreground">
                          {mcpServers.map(m => m.name).join(', ')}
                        </p>
                      </div>
                    </button>
                  )}
                  {packages.length > 0 && (
                    <button
                      onClick={() => setActiveTab("dependencies")}
                      className="flex items-center gap-3 p-3 bg-muted rounded-lg hover:bg-muted/80 transition-colors text-left"
                    >
                      <Package className="h-5 w-5 text-muted-foreground" />
                      <div>
                        <p className="font-medium">{packages.length} Package{packages.length > 1 ? 's' : ''}</p>
                        <p className="text-xs text-muted-foreground truncate">
                          {packages.map(p => p.identifier).join(', ')}
                        </p>
                      </div>
                    </button>
                  )}
                  {remotes.length > 0 && (
                    <button
                      onClick={() => setActiveTab("dependencies")}
                      className="flex items-center gap-3 p-3 bg-muted rounded-lg hover:bg-muted/80 transition-colors text-left"
                    >
                      <Globe className="h-5 w-5 text-muted-foreground" />
                      <div>
                        <p className="font-medium">{remotes.length} Remote{remotes.length > 1 ? 's' : ''}</p>
                        <p className="text-xs text-muted-foreground">
                          {remotes.map(r => r.type).join(', ')}
                        </p>
                      </div>
                    </button>
                  )}
                </div>
              </Card>
            )}

            {/* Telemetry */}
            {agentData.telemetryEndpoint && (
              <Card className="p-6">
                <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
                  <Activity className="h-5 w-5" />
                  Telemetry
                </h3>
                <div className="bg-muted p-4 rounded-lg">
                  <code className="text-sm break-all">{agentData.telemetryEndpoint}</code>
                </div>
              </Card>
            )}

            {/* Repository */}
            {agentData.repository?.url && (
              <Card className="p-6">
                <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
                  <Github className="h-5 w-5" />
                  Repository
                </h3>
                <a
                  href={agentData.repository.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-2 text-primary hover:underline break-all"
                >
                  <span>{agentData.repository.url}</span>
                  <ExternalLink className="h-4 w-4 flex-shrink-0" />
                </a>
                {agentData.repository.source && (
                  <p className="text-sm text-muted-foreground mt-2">
                    Source: <span className="font-medium">{agentData.repository.source}</span>
                  </p>
                )}
              </Card>
            )}
          </TabsContent>

          {hasDependencies && (
            <TabsContent value="dependencies" className="space-y-4">
              {/* Dependency Graph */}
              {dependencyCount > 0 && (
                <Card className="p-4">
                  <h3 className="text-lg font-semibold mb-3 flex items-center gap-2">
                    <Network className="h-5 w-5" />
                    Dependency Graph
                  </h3>
                  <AgentDependencyGraph agent={agentData} />
                </Card>
              )}

              {/* Tools (from kagent) */}
              {tools.length > 0 && (
                <Card className="p-6">
                  <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
                    <Wrench className="h-5 w-5" />
                    Tools
                    <Badge variant="secondary" className="text-xs">{tools.length}</Badge>
                  </h3>
                  <div className="space-y-3">
                    {tools.map((tool, i) => (
                      <div key={i} className="flex items-start gap-3 p-3 bg-muted rounded-lg">
                        {tool.type === 'McpServer' ? (
                          <Server className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
                        ) : (
                          <Bot className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
                        )}
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="font-medium font-mono">{tool.name}</span>
                            <Badge variant="outline" className="text-xs">{tool.type}</Badge>
                          </div>
                          {tool.toolNames && tool.toolNames.length > 0 && (
                            <div className="mt-2 flex flex-wrap gap-1">
                              {tool.toolNames.map((tn, j) => (
                                <Badge key={j} variant="secondary" className="text-xs font-mono">{tn}</Badge>
                              ))}
                            </div>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </Card>
              )}

              {/* Skills (OCI images) */}
              {skills.length > 0 && (
                <Card className="p-6">
                  <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
                    <Blocks className="h-5 w-5" />
                    Skills
                    <Badge variant="secondary" className="text-xs">{skills.length}</Badge>
                  </h3>
                  <div className="space-y-2">
                    {skills.map((skill, i) => (
                      <div key={i} className="bg-muted p-3 rounded-lg">
                        <code className="text-sm break-all">{skill}</code>
                      </div>
                    ))}
                  </div>
                </Card>
              )}

              {/* Model Config */}
              {modelConfigRef && (
                <Card className="p-6">
                  <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
                    <Settings2 className="h-5 w-5" />
                    Model Config
                  </h3>
                  <div className="bg-muted p-3 rounded-lg">
                    <code className="text-sm">{modelConfigRef}</code>
                  </div>
                </Card>
              )}

              {/* System Message */}
              {systemMessage && (
                <Card className="p-6">
                  <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
                    <MessageSquare className="h-5 w-5" />
                    System Message
                  </h3>
                  <div className="bg-muted p-4 rounded-lg">
                    <p className="text-sm whitespace-pre-wrap">
                      {systemMessageExpanded ? systemMessage : systemMessage.slice(0, 300)}
                      {!systemMessageExpanded && systemMessage.length > 300 && '...'}
                    </p>
                    {systemMessage.length > 300 && (
                      <button
                        onClick={() => setSystemMessageExpanded(!systemMessageExpanded)}
                        className="mt-2 flex items-center gap-1 text-xs text-primary hover:underline"
                      >
                        {systemMessageExpanded ? (
                          <>
                            <ChevronUp className="h-3 w-3" />
                            Show less
                          </>
                        ) : (
                          <>
                            <ChevronDown className="h-3 w-3" />
                            Show full message ({systemMessage.length} chars)
                          </>
                        )}
                      </button>
                    )}
                  </div>
                </Card>
              )}

              {/* MCP Servers (legacy) */}
              {mcpServers.length > 0 && (
                <Card className="p-6">
                  <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
                    <Server className="h-5 w-5" />
                    MCP Servers
                    <Badge variant="secondary" className="text-xs">{mcpServers.length}</Badge>
                  </h3>
                  <div className="space-y-3">
                    {mcpServers.map((mcp, i) => (
                      <div key={i} className="flex items-start gap-3 p-3 bg-muted rounded-lg">
                        <Server className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="font-medium">{mcp.name}</span>
                            <Badge variant="outline" className="text-xs">{mcp.type}</Badge>
                          </div>
                          {mcp.url && (
                            <p className="text-sm text-muted-foreground mt-1 font-mono truncate">{mcp.url}</p>
                          )}
                          {mcp.registryServerName && (
                            <p className="text-sm text-muted-foreground mt-1">
                              Registry: <span className="font-medium">{mcp.registryServerName}</span>
                              {mcp.registryServerVersion && (
                                <span className="ml-1">v{mcp.registryServerVersion}</span>
                              )}
                            </p>
                          )}
                          {mcp.image && (
                            <p className="text-sm text-muted-foreground mt-1 font-mono truncate">{mcp.image}</p>
                          )}
                          {mcp.command && (
                            <p className="text-sm text-muted-foreground mt-1 font-mono">
                              {mcp.command} {mcp.args?.join(' ')}
                            </p>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </Card>
              )}

              {/* Packages */}
              {packages.length > 0 && (
                <Card className="p-6">
                  <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
                    <Package className="h-5 w-5" />
                    Packages
                    <Badge variant="secondary" className="text-xs">{packages.length}</Badge>
                  </h3>
                  <div className="space-y-3">
                    {packages.map((pkg, i) => (
                      <div key={i} className="flex items-start gap-3 p-3 bg-muted rounded-lg">
                        <Package className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="font-medium font-mono">{pkg.identifier}</span>
                            <Badge variant="outline" className="text-xs">{pkg.registryType}</Badge>
                            {pkg.version && (
                              <Badge variant="secondary" className="text-xs">v{pkg.version}</Badge>
                            )}
                          </div>
                          {pkg.transport?.type && (
                            <p className="text-sm text-muted-foreground mt-1">
                              Transport: {pkg.transport.type}
                            </p>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </Card>
              )}

              {/* Remotes */}
              {remotes.length > 0 && (
                <Card className="p-6">
                  <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
                    <Globe className="h-5 w-5" />
                    Remote Endpoints
                    <Badge variant="secondary" className="text-xs">{remotes.length}</Badge>
                  </h3>
                  <div className="space-y-3">
                    {remotes.map((remote, i) => (
                      <div key={i} className="flex items-start gap-3 p-3 bg-muted rounded-lg">
                        <Globe className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <Badge variant="outline" className="text-xs">{remote.type}</Badge>
                          </div>
                          {remote.url && (
                            <p className="text-sm text-muted-foreground mt-1 font-mono truncate">{remote.url}</p>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </Card>
              )}
            </TabsContent>
          )}

          <TabsContent value="technical" className="space-y-4">
            {/* Repository */}
            {agentData.repository?.url && (
              <Card className="p-6">
                <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
                  <Github className="h-5 w-5" />
                  Source Repository
                </h3>
                <div className="space-y-3">
                  <div>
                    <p className="text-sm text-muted-foreground mb-1">URL</p>
                    <a
                      href={agentData.repository.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center gap-2 text-primary hover:underline break-all"
                    >
                      <span className="font-mono text-sm">{agentData.repository.url}</span>
                      <ExternalLink className="h-4 w-4 flex-shrink-0" />
                    </a>
                  </div>
                  {agentData.repository.source && (
                    <div>
                      <p className="text-sm text-muted-foreground mb-1">Source</p>
                      <p className="font-medium">{agentData.repository.source}</p>
                    </div>
                  )}
                </div>
              </Card>
            )}

            {/* Container Image */}
            {agentData.image && (
              <Card className="p-6">
                <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
                  <Container className="h-5 w-5" />
                  Container Image
                </h3>
                <div className="bg-muted p-4 rounded-lg">
                  <code className="text-sm break-all">{agentData.image}</code>
                </div>
              </Card>
            )}

            {/* Website & Telemetry */}
            {(agentData.websiteUrl || agentData.telemetryEndpoint) && (
              <Card className="p-6">
                <h3 className="text-lg font-semibold mb-4">Endpoints</h3>
                <div className="space-y-3">
                  {agentData.websiteUrl && (
                    <div>
                      <p className="text-sm text-muted-foreground mb-1">Website</p>
                      <a
                        href={agentData.websiteUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="flex items-center gap-2 text-primary hover:underline break-all"
                      >
                        <span className="font-mono text-sm">{agentData.websiteUrl}</span>
                        <ExternalLink className="h-4 w-4 flex-shrink-0" />
                      </a>
                    </div>
                  )}
                  {agentData.telemetryEndpoint && (
                    <div>
                      <p className="text-sm text-muted-foreground mb-1">Telemetry Endpoint</p>
                      <code className="text-sm break-all">{agentData.telemetryEndpoint}</code>
                    </div>
                  )}
                </div>
              </Card>
            )}

            {/* Timestamps */}
            <Card className="p-6">
              <h3 className="text-lg font-semibold mb-4">Timestamps</h3>
              <div className="space-y-3">
                {agentData.updatedAt && (
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">Last Updated (Local)</span>
                    <span className="font-medium font-mono text-sm">{formatDate(agentData.updatedAt)}</span>
                  </div>
                )}
                {official?.publishedAt && (
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">Published (Registry)</span>
                    <span className="font-medium font-mono text-sm">{formatDate(official.publishedAt)}</span>
                  </div>
                )}
                {official?.updatedAt && (
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">Updated (Registry)</span>
                    <span className="font-medium font-mono text-sm">{formatDate(official.updatedAt)}</span>
                  </div>
                )}
              </div>
            </Card>
          </TabsContent>

          <TabsContent value="raw">
            <Card className="p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold flex items-center gap-2">
                  <Code className="h-5 w-5" />
                  Raw JSON Data
                </h3>
              </div>
              <pre className="bg-muted p-4 rounded-lg overflow-x-auto text-xs">
                {JSON.stringify(agent, null, 2)}
              </pre>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}

