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
import { Calendar, Tag, Bot, Upload, Container, Cpu, Brain, Github, CheckCircle2, XCircle, Circle, BadgeCheck, Clock, Check, X } from "lucide-react"

interface AgentCardProps {
  agent: AgentResponse
  onDelete?: (agent: AgentResponse) => void
  onPublish?: (agent: AgentResponse) => void
  onApprove?: (agent: AgentResponse) => void
  onReject?: (agent: AgentResponse) => void
  showDelete?: boolean
  showPublish?: boolean
  showExternalLinks?: boolean
  showApproval?: boolean
  onClick?: () => void
}

export function AgentCard({ agent, onDelete, onPublish, onApprove, onReject, showDelete = false, showPublish = false, showApproval = false, onClick }: AgentCardProps) {
  const { agent: agentData, _meta } = agent
  const official = _meta?.['io.modelcontextprotocol.registry/official']
  const deployment = _meta?.deployment

  // Check if this is a pending review submission
  const isPendingReview = official?.status === 'pending_review' || (official as any)?.reviewStatus === 'pending'

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
            <div className="flex items-center gap-2 mb-1">
              <h3 className="font-semibold text-lg">{agentData.name}</h3>
              <Badge variant="outline" className="bg-purple-500/10 text-purple-600 border-purple-500/20 text-xs">
                Agent
              </Badge>
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
              {isPendingReview && (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Badge
                      variant="outline"
                      className="text-xs bg-yellow-500/10 text-yellow-600 hover:bg-yellow-500/20 border-yellow-500/20"
                    >
                      <Clock className="h-3 w-3 mr-1" />
                      Pending Review
                    </Badge>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>This agent is awaiting approval</p>
                  </TooltipContent>
                </Tooltip>
              )}
              {deployment && !isPendingReview && (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Badge
                      variant={deployment.ready ? "default" : "destructive"}
                      className={`text-xs ${deployment.ready ? 'bg-green-500/10 text-green-600 hover:bg-green-500/20 border-green-500/20' : ''}`}
                    >
                      {deployment.ready ? (
                        <CheckCircle2 className="h-3 w-3 mr-1" />
                      ) : (
                        <XCircle className="h-3 w-3 mr-1" />
                      )}
                      {deployment.ready ? 'Running' : 'Not Ready'}
                    </Badge>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>{deployment.message || (deployment.ready ? 'Deployment is healthy' : 'Deployment has issues')}</p>
                    {deployment.namespace && <p className="text-xs text-muted-foreground">Namespace: {deployment.namespace}</p>}
                  </TooltipContent>
                </Tooltip>
              )}
            </div>
          </div>
        </div>
        <div className="flex items-center gap-1 ml-2">
          {showApproval && isPendingReview && onApprove && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="default"
                  size="sm"
                  className="h-8 gap-1.5 bg-green-600 hover:bg-green-700"
                  onClick={(e) => {
                    e.stopPropagation()
                    onApprove(agent)
                  }}
                >
                  <Check className="h-3.5 w-3.5" />
                  Approve
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                <p>Approve and publish this agent</p>
              </TooltipContent>
            </Tooltip>
          )}
          {showApproval && isPendingReview && onReject && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="destructive"
                  size="sm"
                  className="h-8 gap-1.5"
                  onClick={(e) => {
                    e.stopPropagation()
                    onReject(agent)
                  }}
                >
                  <X className="h-3.5 w-3.5" />
                  Reject
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                <p>Reject this submission</p>
              </TooltipContent>
            </Tooltip>
          )}
          {showPublish && onPublish && !isPendingReview && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8"
                  onClick={(e) => {
                    e.stopPropagation()
                    onPublish(agent)
                  }}
                >
                  <Upload className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                <p>Publish this agent to your registry</p>
              </TooltipContent>
            </Tooltip>
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

        {official?.publishedAt && (
          <div className="flex items-center gap-1">
            <Calendar className="h-3 w-3" />
            <span>{formatDate(official.publishedAt)}</span>
          </div>
        )}

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

        {agentData.repository?.url && (
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

