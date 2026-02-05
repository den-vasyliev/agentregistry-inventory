"use client"

import { useState, useEffect } from "react"
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
} from "lucide-react"

interface AgentDetailProps {
  agent: AgentResponse
  onClose: () => void
}

export function AgentDetail({ agent, onClose }: AgentDetailProps) {
  const [activeTab, setActiveTab] = useState("overview")

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
                <Badge variant="outline" className="text-sm">
                  {agentData.framework}
                </Badge>
                <Badge variant="secondary" className="text-sm">
                  {agentData.language}
                </Badge>
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

