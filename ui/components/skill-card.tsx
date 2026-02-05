"use client"

import { SkillResponse } from "@/lib/admin-api"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { Package, Calendar, Tag, ExternalLink, GitBranch, Github, Globe, Zap, BadgeCheck, ShieldCheck, CheckCircle2 } from "lucide-react"
import { Badge } from "@/components/ui/badge"

interface SkillCardProps {
  skill: SkillResponse
  showExternalLinks?: boolean
  onClick?: () => void
}

export function SkillCard({ skill, showExternalLinks = true, onClick }: SkillCardProps) {
  const { skill: skillData, _meta } = skill

  // Extract metadata
  const publisherMetadata = _meta?.['io.modelcontextprotocol.registry/publisher-provided'] as Record<string, unknown> | undefined
  const aregistryMetadata = publisherMetadata?.['aregistry.ai/metadata'] as Record<string, unknown> | undefined
  const identityData = aregistryMetadata?.['identity'] as Record<string, unknown> | undefined

  // Get owner from metadata or extract from repository URL
  const getOwner = (): string | null => {
    // Try to get email from metadata first
    const contactEmail = publisherMetadata?.contact_email as string | undefined
    if (contactEmail) return contactEmail
    const identityEmail = identityData?.email as string | undefined
    if (identityEmail) return identityEmail

    // Fallback to extracting owner/org from GitHub repository URL
    if (skillData.repository?.url) {
      const match = skillData.repository.url.match(/github\.com\/([^\/]+)/)
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
          <div className="w-10 h-10 rounded bg-primary/10 flex items-center justify-center flex-shrink-0 mt-1">
            <Zap className="h-5 w-5 text-primary" />
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1 flex-wrap">
              <h3 className="font-semibold text-lg">{skillData.title || skillData.name}</h3>
              <Badge variant="outline" className="bg-yellow-500/10 text-yellow-600 border-yellow-500/20 text-xs">
                Skill
              </Badge>
              {/* Skills are metadata - show as Active */}
              <Badge variant="outline" className="bg-green-500/10 text-green-600 border-green-500/20 text-xs">
                <CheckCircle2 className="h-3 w-3 mr-1" />
                Active
              </Badge>
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
            <p className="text-sm text-muted-foreground">{skillData.name}</p>
          </div>
        </div>
        <div className="flex items-center gap-1 ml-2">
          {showExternalLinks && skillData.repository?.url && (
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={(e) => {
                e.stopPropagation()
                window.open(skillData.repository?.url || '', '_blank')
              }}
              title="View on GitHub"
            >
              <Github className="h-4 w-4" />
            </Button>
          )}
          {showExternalLinks && skillData.websiteUrl && (
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={(e) => {
                e.stopPropagation()
                window.open(skillData.websiteUrl, '_blank')
              }}
              title="Visit website"
            >
              <Globe className="h-4 w-4" />
            </Button>
          )}
        </div>
      </div>

      <p className="text-sm text-muted-foreground mb-3 line-clamp-2">
        {skillData.description}
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
          <span>{skillData.version}</span>
        </div>

        {skillData.packages && skillData.packages.length > 0 && (
          <div className="flex items-center gap-1">
            <Package className="h-3 w-3" />
            <span>{skillData.packages.length} package{skillData.packages.length !== 1 ? 's' : ''}</span>
          </div>
        )}

        {skillData.remotes && skillData.remotes.length > 0 && (
          <div className="flex items-center gap-1">
            <ExternalLink className="h-3 w-3" />
            <span>{skillData.remotes.length} remote{skillData.remotes.length !== 1 ? 's' : ''}</span>
          </div>
        )}

        {skillData.repository && (
          <div className="flex items-center gap-1">
            <GitBranch className="h-3 w-3" />
            <span>{skillData.repository.source}</span>
          </div>
        )}
      </div>
      </Card>
    </TooltipProvider>
  )
}
