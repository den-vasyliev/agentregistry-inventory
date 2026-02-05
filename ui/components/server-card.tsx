"use client"

import { ServerResponse } from "@/lib/admin-api"
import { Card } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { Package, Calendar, Tag, ExternalLink, GitBranch, Star, Github, Globe, Play, StopCircle, ShieldCheck, BadgeCheck, CheckCircle2, XCircle } from "lucide-react"

interface ServerCardProps {
  server: ServerResponse
  onDeploy?: (server: ServerResponse) => void
  onUndeploy?: (server: ServerResponse) => void
  showDeploy?: boolean
  showExternalLinks?: boolean
  onClick?: () => void
  versionCount?: number
}

export function ServerCard({ server, onDeploy, onUndeploy, showDeploy = true, showExternalLinks = true, onClick, versionCount }: ServerCardProps) {
  const { server: serverData, _meta } = server

  // Get deployment status
  const deployment = _meta?.deployment
  const isExternal = _meta?.isDiscovered || _meta?.source === 'discovery'
  const deploymentStatus = deployment?.ready ? "Running" : deployment ? "Failed" : "Not Deployed"

  // Extract metadata
  const publisherMetadata = (serverData as any)._meta?.['io.modelcontextprotocol.registry/publisher-provided']?.['aregistry.ai/metadata']
  const githubStars = publisherMetadata?.stars
  const identityData = publisherMetadata?.identity

  // Get owner from metadata or extract from repository URL
  const getOwner = () => {
    // Try to get email from metadata first
    if (publisherMetadata?.contact_email) return publisherMetadata.contact_email
    if (identityData?.email) return identityData.email

    // Fallback to extracting owner/org from GitHub repository URL
    if (serverData.repository?.url) {
      const match = serverData.repository.url.match(/github\.com\/([^\/]+)/)
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

  // Get the first icon if available
  const icon = serverData.icons?.[0]

  // Get status badge styles
  const getStatusBadgeStyles = (status: string) => {
    switch (status) {
      case "Running":
        return 'bg-green-500/10 text-green-600 border-green-500/20'
      case "Failed":
        return 'bg-red-500/10 text-red-600 border-red-500/20'
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
          {icon && (
            <img 
              src={icon.src} 
              alt="Server icon" 
              className="w-10 h-10 rounded flex-shrink-0 mt-1"
            />
          )}
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1 flex-wrap">
              <h3 className="font-semibold text-lg">{serverData.title || serverData.name}</h3>
              <Badge variant="outline" className="bg-blue-500/10 text-blue-600 border-blue-500/20 text-xs">
                {serverData.remotes && serverData.remotes.length > 0 ? "Remote MCP" : "MCP Server"}
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
                <Badge variant="outline" className="bg-purple-500/10 text-purple-600 border-purple-500/20 text-xs">
                  External
                </Badge>
              )}
              <Tooltip>
                <TooltipTrigger asChild>
                  <ShieldCheck
                    className={`h-4 w-4 flex-shrink-0 ${
                      identityData?.org_is_verified
                        ? 'text-blue-600 dark:text-blue-400'
                        : 'text-gray-400 dark:text-gray-600 opacity-40'
                    }`}
                  />
                </TooltipTrigger>
                <TooltipContent>
                  <p>{identityData?.org_is_verified ? 'Verified Organization' : 'Organization Not Verified'}</p>
                </TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <BadgeCheck
                    className={`h-4 w-4 flex-shrink-0 ${
                      identityData?.publisher_identity_verified_by_jwt
                        ? 'text-green-600 dark:text-green-400'
                        : 'text-gray-400 dark:text-gray-600 opacity-40'
                    }`}
                  />
                </TooltipTrigger>
                <TooltipContent>
                  <p>{identityData?.publisher_identity_verified_by_jwt ? 'Verified Publisher' : 'Publisher Not Verified'}</p>
                </TooltipContent>
              </Tooltip>
            </div>
            <p className="text-sm text-muted-foreground">{serverData.name}</p>
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
                    onDeploy(server)
                  }}
                >
                  <Play className="h-3.5 w-3.5" />
                  Deploy
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                <p>Deploy this server</p>
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
                    onUndeploy(server)
                  }}
                >
                  <StopCircle className="h-3.5 w-3.5" />
                  Undeploy
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                <p>Undeploy this server</p>
              </TooltipContent>
            </Tooltip>
          )}
          {showExternalLinks && serverData.repository?.url && (
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={(e) => {
                e.stopPropagation()
                window.open(serverData.repository?.url || '', '_blank')
              }}
              title="View on GitHub"
            >
              <Github className="h-4 w-4" />
            </Button>
          )}
          {showExternalLinks && serverData.websiteUrl && (
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={(e) => {
                e.stopPropagation()
                window.open(serverData.websiteUrl, '_blank')
              }}
              title="Visit website"
            >
              <Globe className="h-4 w-4" />
            </Button>
          )}
        </div>
      </div>

      <p className="text-sm text-muted-foreground mb-3 line-clamp-2">
        {serverData.description}
      </p>

      <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
        {owner && (
          <div className="flex items-center gap-1 text-primary font-medium">
            <BadgeCheck className="h-3 w-3" />
            <span>{owner}</span>
          </div>
        )}

        <div className="flex items-center gap-1">
          <Tag className="h-3 w-3" />
          <span>{serverData.version}</span>
          {versionCount && versionCount > 1 && (
            <span className="ml-1 text-primary font-medium">
              (+{versionCount - 1} more)
            </span>
          )}
        </div>

        {serverData.packages && serverData.packages.length > 0 && (
          <div className="flex items-center gap-1">
            <Package className="h-3 w-3" />
            <span>{serverData.packages.length} package{serverData.packages.length !== 1 ? 's' : ''}</span>
          </div>
        )}

        {serverData.remotes && serverData.remotes.length > 0 && (
          <div className="flex items-center gap-1">
            <ExternalLink className="h-3 w-3" />
            <span>{serverData.remotes.length} remote{serverData.remotes.length !== 1 ? 's' : ''}</span>
          </div>
        )}

        {serverData.repository && (
          <div className="flex items-center gap-1">
            <GitBranch className="h-3 w-3" />
            <span>{serverData.repository.source}</span>
          </div>
        )}

        {githubStars !== undefined && (
          <div className="flex items-center gap-1 text-yellow-600 dark:text-yellow-400">
            <Star className="h-3 w-3 fill-yellow-600 dark:fill-yellow-400" />
            <span className="font-medium">{githubStars.toLocaleString()}</span>
          </div>
        )}
      </div>
      </Card>
    </TooltipProvider>
  )
}
