"use client"

import { AgentResponse } from "@/lib/admin-api"
import { Card } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { Calendar, Tag, Bot, Trash2, Container, Cpu, Brain, Github, BadgeCheck, Play, StopCircle, CheckCircle2, XCircle } from "lucide-react"

interface AgentCardProps {
  agent: AgentResponse
  onDelete?: (agent: AgentResponse) => void
  onDeploy?: (agent: AgentResponse) => void
  onUndeploy?: (agent: AgentResponse) => void
  showDelete?: boolean
  showDeploy?: boolean
  showExternalLinks?: boolean
  onClick?: () => void
}

export function AgentCard({ agent, onDelete, onDeploy, onUndeploy, showDelete = true, showDeploy = true, showExternalLinks = true, onClick }: AgentCardProps) {
  const { agent: agentData, _meta } = agent

  // Get deployment status
  const deployment = _meta?.deployment
  const isExternal = _meta?.isDiscovered || _meta?.source === 'discovery'
  const deploymentStatus = deployment?.ready ? "Running" : deployment ? "Failed" : "Not Deployed"

  // Extract metadata
  const publisherMetadata = (agentData as any)._meta?.['io.modelcontextprotocol.registry/publisher-provided']?.['aregistry.ai/metadata']
  const identityData = publisherMetadata?.identity

  // Get owner from metadata or extract from repository URL
  const getOwner = () => {
    // Try to get email from metadata first
    if (publisherMetadata?.contact_email) return publisherMetadata.contact_email
    if (identityData?.email) return identityData.email

    // Fallback to extracting owner/org from GitHub repository URL
    if (agentData.repository?.url) {
      const match = agentData.repository.url.match(/github\.com\/([^\/]+)/)
      if (match) return match[1]
    }

    return null
  }

  const owner = getOwner()

  const handleClick = () => {
    if (onClick) {
      onClick()
    }
  }

  // Format date
  const formatDate = (dateString: string) => {
    try {
      return new Date(dateString).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      })
    } catch {
      return dateString
    }
  }

  // Get status badge styles
  const getStatusBadgeStyles = (status: string) => {
    switch (status) {
      case "Deployed":
        return 'bg-green-500/10 text-green-600 border-green-500/20'
      case "Failed":
        return 'bg-red-500/10 text-red-600 border-red-500/20'
      case "Deploying":
        return 'bg-yellow-500/10 text-yellow-600 border-yellow-500/20'
      default:
        return 'bg-gray-500/10 text-gray-600 border-gray-500/20'
    }
  }

  return (
    <TooltipProvider>
      <Card
        className="p-4 hover:shadow-md transition-all duration-200 cursor-pointer border hover:border-primary/20"
        onClick={handleClick}
      >
      <div className="flex items-start justify-between mb-2">
        <div className="flex items-start gap-3 flex-1">
          <div className="w-10 h-10 rounded bg-primary/20 flex items-center justify-center flex-shrink-0 mt-1">
            <Bot className="h-5 w-5 text-primary" />
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1 flex-wrap">
              <h3 className="font-semibold text-lg">{agentData.name}</h3>
              <Badge variant="outline" className="bg-purple-500/10 text-purple-600 border-purple-500/20 text-xs">
                Agent
              </Badge>
              {/* Deployment status badge */}
              <Badge 
                variant="outline" 
                className={`text-xs ${getStatusBadgeStyles(deploymentStatus)}`}
              >
                {deploymentStatus === "Running" && <CheckCircle2 className="h-3 w-3 mr-1" />}
                {deploymentStatus === "Failed" && <XCircle className="h-3 w-3 mr-1" />}
                {deploymentStatus}
              </Badge>
              {/* External badge for discovered resources */}
              {_meta?.isDiscovered && (
                <Badge variant="outline" className="bg-teal-500/10 text-teal-600 border-teal-500/20 text-xs">
                  External
                </Badge>
              )}
            </div>
            <div className="text-xs text-muted-foreground flex items-center gap-1 flex-wrap">
              {agentData.framework && (
                <Badge variant="outline" className="text-xs">
                  {agentData.framework}
                </Badge>
              )}
              {agentData.language && (
                <Badge variant="secondary" className="text-xs">
                  {agentData.language}
                </Badge>
              )}
            </div>
          </div>
        </div>
        <div className="flex items-center gap-1 ml-2">
          {/* Deploy/Undeploy buttons - only for managed (non-external) resources */}
          {!isExternal && showDeploy && deploymentStatus === "Not Deployed" && onDeploy && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="default"
                  size="sm"
                  className="h-8 gap-1.5"
                  onClick={(e) => {
                    e.stopPropagation()
                    onDeploy(agent)
                  }}
                >
                  <Play className="h-3.5 w-3.5" />
                  Deploy
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                <p>Deploy this agent</p>
              </TooltipContent>
            </Tooltip>
          )}
          {!isExternal && showDeploy && deploymentStatus === "Running" && onUndeploy && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-8 gap-1.5"
                  onClick={(e) => {
                    e.stopPropagation()
                    onUndeploy(agent)
                  }}
                >
                  <StopCircle className="h-3.5 w-3.5" />
                  Undeploy
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                <p>Undeploy this agent</p>
              </TooltipContent>
            </Tooltip>
          )}
          {/* Delete button - only for managed (non-external) resources */}
          {!isExternal && showDelete && onDelete && (
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 text-destructive hover:text-destructive hover:bg-destructive/10"
              onClick={(e) => {
                e.stopPropagation()
                onDelete(agent)
              }}
              title="Remove from registry"
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          )}
        </div>
      </div>

      {agentData.description && (
        <p className="text-sm text-muted-foreground mb-3 line-clamp-2">
          {agentData.description}
        </p>
      )}

      <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
        {owner && (
          <div className="flex items-center gap-1 text-primary font-medium">
            <BadgeCheck className="h-3 w-3" />
            <span>{owner}</span>
          </div>
        )}

        <div className="flex items-center gap-1">
          <Tag className="h-3 w-3" />
          <span>{agentData.version}</span>
        </div>

        {agentData.modelProvider && (
          <div className="flex items-center gap-1">
            <Brain className="h-3 w-3" />
            <span>{agentData.modelProvider}</span>
          </div>
        )}

        {agentData.modelName && (
          <div className="flex items-center gap-1">
            <Cpu className="h-3 w-3" />
            <span className="font-mono text-xs">{agentData.modelName}</span>
          </div>
        )}

        {agentData.image && (
          <div className="flex items-center gap-1">
            <Container className="h-3 w-3" />
            <span className="font-mono text-xs truncate max-w-[200px]" title={agentData.image}>
              {agentData.image}
            </span>
          </div>
        )}

        {showExternalLinks && agentData.repository?.url && (
          <a
            href={agentData.repository.url}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-1 hover:text-primary transition-colors"
            onClick={(e) => e.stopPropagation()}
          >
            <Github className="h-3 w-3" />
            <span>Repository</span>
          </a>
        )}
      </div>
      </Card>
    </TooltipProvider>
  )
}
