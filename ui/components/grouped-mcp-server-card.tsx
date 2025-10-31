"use client"

import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { GroupedMCPServer } from "@/lib/types"
import { ExternalLink, Github, Download, Trash2, Clock, Package } from "lucide-react"
import { formatDistanceToNow } from "@/lib/utils"

interface GroupedMCPServerCardProps {
  groupedServer: GroupedMCPServer
  onInstall?: (groupedServer: GroupedMCPServer) => void
  onUninstall?: (groupedServer: GroupedMCPServer) => void
  onClick?: (groupedServer: GroupedMCPServer) => void
}

export function GroupedMCPServerCard({
  groupedServer,
  onInstall,
  onUninstall,
  onClick,
}: GroupedMCPServerCardProps) {
  const { latestVersion, versions, hasInstalledVersion } = groupedServer
  const updatedAt = latestVersion._meta?.["io.modelcontextprotocol.registry/official"]?.updatedAt
  const isLatest = latestVersion._meta?.["io.modelcontextprotocol.registry/official"]?.isLatest
  const status = latestVersion._meta?.["io.modelcontextprotocol.registry/official"]?.status

  return (
    <Card
      className="group relative overflow-hidden transition-all hover:shadow-md cursor-pointer"
      onClick={() => onClick?.(groupedServer)}
    >
      <div className="p-6">
        <div className="flex gap-4">
          {/* Icon/Image */}
          <div className="flex-shrink-0">
            <div className="w-16 h-16 rounded-lg bg-muted flex items-center justify-center overflow-hidden">
              {groupedServer.icon ? (
                <img
                  src={groupedServer.icon}
                  alt={groupedServer.title || groupedServer.name}
                  className="w-full h-full object-cover"
                />
              ) : (
                <div className="text-2xl font-bold text-muted-foreground">
                  {(groupedServer.title || groupedServer.name).charAt(0).toUpperCase()}
                </div>
              )}
            </div>
          </div>

          {/* Content */}
          <div className="flex-1 min-w-0">
            <div className="flex items-start justify-between gap-2 mb-2">
              <div className="flex-1 min-w-0">
                <h3 className="font-semibold text-lg truncate">
                  {groupedServer.title || groupedServer.name}
                </h3>
                <p className="text-sm text-muted-foreground truncate">
                  {groupedServer.name}
                </p>
              </div>
              <div className="flex items-center gap-2 flex-shrink-0">
                {hasInstalledVersion && (
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
              {groupedServer.description}
            </p>

            <div className="flex items-center gap-4 text-xs text-muted-foreground mb-4">
              {versions.length > 1 ? (
                <span className="flex items-center gap-1">
                  <Package className="w-3 h-3" />
                  <Badge variant="secondary">{versions.length} versions</Badge>
                  <span className="text-muted-foreground">
                    (latest: v{latestVersion.server.version})
                  </span>
                </span>
              ) : (
                <span className="flex items-center gap-1">
                  <Badge variant="secondary">v{latestVersion.server.version}</Badge>
                </span>
              )}
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
              {groupedServer.repository?.url && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-8 px-2"
                  onClick={(e) => {
                    e.stopPropagation()
                    window.open(groupedServer.repository!.url, "_blank")
                  }}
                >
                  <Github className="w-4 h-4 mr-1" />
                  Repo
                </Button>
              )}

              {/* Website Link */}
              {groupedServer.websiteUrl && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-8 px-2"
                  onClick={(e) => {
                    e.stopPropagation()
                    window.open(groupedServer.websiteUrl!, "_blank")
                  }}
                >
                  <ExternalLink className="w-4 h-4 mr-1" />
                  Website
                </Button>
              )}

              <div className="flex-1" />

              {/* Install/Uninstall Button */}
              {hasInstalledVersion ? (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={(e) => {
                    e.stopPropagation()
                    onUninstall?.(groupedServer)
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
                    onInstall?.(groupedServer)
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

