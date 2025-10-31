"use client"

import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { MCPServerWithStatus } from "@/lib/types"
import { ExternalLink, Github, Download, Trash2, Clock } from "lucide-react"
import { formatDistanceToNow } from "@/lib/utils"

interface MCPServerCardProps {
  server: MCPServerWithStatus
  onInstall?: (server: MCPServerWithStatus) => void
  onUninstall?: (server: MCPServerWithStatus) => void
  onClick?: (server: MCPServerWithStatus) => void
}

export function MCPServerCard({
  server,
  onInstall,
  onUninstall,
  onClick,
}: MCPServerCardProps) {
  const { server: serverData, _meta, installed } = server
  const updatedAt = _meta?.["io.modelcontextprotocol.registry/official"]?.updatedAt
  const isLatest = _meta?.["io.modelcontextprotocol.registry/official"]?.isLatest
  const status = _meta?.["io.modelcontextprotocol.registry/official"]?.status

  // Get icon or use default
  const icon = serverData.icons?.[0]?.src

  return (
    <Card
      className="group relative overflow-hidden transition-all hover:shadow-md cursor-pointer"
      onClick={() => onClick?.(server)}
    >
      <div className="p-6">
        <div className="flex gap-4">
          {/* Icon/Image */}
          <div className="flex-shrink-0">
            <div className="w-16 h-16 rounded-lg bg-muted flex items-center justify-center overflow-hidden">
              {icon ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={icon}
                  alt={serverData.title || serverData.name}
                  className="w-full h-full object-cover"
                />
              ) : (
                <div className="text-2xl font-bold text-muted-foreground">
                  {(serverData.title || serverData.name).charAt(0).toUpperCase()}
                </div>
              )}
            </div>
          </div>

          {/* Content */}
          <div className="flex-1 min-w-0">
            <div className="flex items-start justify-between gap-2 mb-2">
              <div className="flex-1 min-w-0">
                <h3 className="font-semibold text-lg truncate">
                  {serverData.title || serverData.name}
                </h3>
                <p className="text-sm text-muted-foreground truncate">
                  {serverData.name}
                </p>
              </div>
              <div className="flex items-center gap-2 flex-shrink-0">
                {installed && (
                  <Badge variant="default" className="bg-green-600">
                    Installed
                  </Badge>
                )}
                {isLatest && (
                  <Badge variant="outline">Latest</Badge>
                )}
              </div>
            </div>

            <p className="text-sm text-muted-foreground line-clamp-2 mb-3">
              {serverData.description}
            </p>

            <div className="flex items-center gap-4 text-xs text-muted-foreground mb-4">
              <span className="flex items-center gap-1">
                <Badge variant="secondary">v{serverData.version}</Badge>
              </span>
              {updatedAt && (
                <span className="flex items-center gap-1">
                  <Clock className="w-3 h-3" />
                  {formatDistanceToNow(new Date(updatedAt))}
                </span>
              )}
              {status && status !== "active" && (
                <Badge variant="outline" className="text-yellow-600">
                  {status}
                </Badge>
              )}
            </div>

            <div className="flex items-center gap-2">
              {/* Repository Link */}
              {serverData.repository?.url && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-8 px-2"
                  onClick={(e) => {
                    e.stopPropagation()
                    window.open(serverData.repository!.url, "_blank")
                  }}
                >
                  <Github className="w-4 h-4 mr-1" />
                  Repo
                </Button>
              )}

              {/* Website Link */}
              {serverData.websiteUrl && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-8 px-2"
                  onClick={(e) => {
                    e.stopPropagation()
                    window.open(serverData.websiteUrl!, "_blank")
                  }}
                >
                  <ExternalLink className="w-4 h-4 mr-1" />
                  Website
                </Button>
              )}

              <div className="flex-1" />

              {/* Install/Uninstall Button */}
              {installed ? (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={(e) => {
                    e.stopPropagation()
                    onUninstall?.(server)
                  }}
                >
                  <Trash2 className="w-4 h-4 mr-1" />
                  Uninstall
                </Button>
              ) : (
                <Button
                  variant="default"
                  size="sm"
                  onClick={(e) => {
                    e.stopPropagation()
                    onInstall?.(server)
                  }}
                >
                  <Download className="w-4 h-4 mr-1" />
                  Install
                </Button>
              )}
            </div>
          </div>
        </div>
      </div>
    </Card>
  )
}

