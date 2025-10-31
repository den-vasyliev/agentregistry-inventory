"use client"

import { useState } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { MCPServerWithStatus, ServerScore } from "@/lib/types"
import {
  X,
  Download,
  Trash2,
  Github,
  ExternalLink,
  Package,
  Settings,
  Activity,
  Shield,
  TrendingUp,
  FileText,
  Users,
  Wrench,
  PlayCircle,
} from "lucide-react"
import { formatDistanceToNow } from "@/lib/utils"
import { ServerPlayground } from "./server-playground"

interface ServerDetailViewProps {
  server: MCPServerWithStatus
  onClose: () => void
  onInstall?: (server: MCPServerWithStatus) => void
  onUninstall?: (server: MCPServerWithStatus) => void
}

// Mock score data - will be replaced with real data later
const getMockScore = (serverName: string): ServerScore => {
  const hash = serverName.split("").reduce((acc, char) => acc + char.charCodeAt(0), 0)
  return {
    overall: 75 + (hash % 20),
    security: 70 + (hash % 25),
    reliability: 80 + (hash % 15),
    performance: 75 + (hash % 20),
    documentation: 65 + (hash % 30),
    community: 60 + (hash % 35),
    maintenance: 85 + (hash % 10),
  }
}

export function ServerDetailView({
  server,
  onClose,
  onInstall,
  onUninstall,
}: ServerDetailViewProps) {
  const [activeTab, setActiveTab] = useState("overview")
  const { server: serverData, _meta, installed } = server
  const score = getMockScore(serverData.name)
  const updatedAt = _meta?.["io.modelcontextprotocol.registry/official"]?.updatedAt
  const publishedAt = _meta?.["io.modelcontextprotocol.registry/official"]?.publishedAt
  const icon = serverData.icons?.[0]?.src

  const getScoreColor = (score: number) => {
    if (score >= 80) return "text-green-600"
    if (score >= 60) return "text-yellow-600"
    return "text-red-600"
  }

  const getScoreLabel = (score: number) => {
    if (score >= 90) return "Excellent"
    if (score >= 80) return "Very Good"
    if (score >= 70) return "Good"
    if (score >= 60) return "Fair"
    return "Poor"
  }

  return (
    <div className="fixed inset-0 z-50 bg-background overflow-y-auto">
      <div className="container mx-auto px-4 py-8 max-w-5xl">
        {/* Header */}
        <div className="flex items-start justify-between mb-6">
          <div className="flex gap-4 flex-1">
            <div className="flex-shrink-0">
              <div className="w-20 h-20 rounded-lg bg-muted flex items-center justify-center overflow-hidden">
                {icon ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={icon}
                    alt={serverData.title || serverData.name}
                    className="w-full h-full object-cover"
                  />
                ) : (
                  <div className="text-3xl font-bold text-muted-foreground">
                    {(serverData.title || serverData.name).charAt(0).toUpperCase()}
                  </div>
                )}
              </div>
            </div>
            <div className="flex-1">
              <h1 className="text-3xl font-bold mb-2">
                {serverData.title || serverData.name}
              </h1>
              <p className="text-muted-foreground mb-3">{serverData.name}</p>
              <div className="flex items-center gap-2 flex-wrap">
                <Badge variant="secondary">v{serverData.version}</Badge>
                {installed && (
                  <Badge variant="default" className="bg-green-600">
                    Installed
                  </Badge>
                )}
                <Badge variant="outline">
                  Score: {score.overall}/100
                </Badge>
              </div>
            </div>
          </div>
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="h-5 w-5" />
          </Button>
        </div>

        {/* Actions */}
        <div className="flex gap-3 mb-6">
          {installed ? (
            <Button
              variant="outline"
              onClick={() => onUninstall?.(server)}
            >
              <Trash2 className="w-4 h-4 mr-2" />
              Uninstall
            </Button>
          ) : (
            <Button onClick={() => onInstall?.(server)}>
              <Download className="w-4 h-4 mr-2" />
              Install
            </Button>
          )}
          {serverData.repository?.url && (
            <Button
              variant="outline"
              onClick={() => window.open(serverData.repository!.url, "_blank")}
            >
              <Github className="w-4 h-4 mr-2" />
              Repository
            </Button>
          )}
          {serverData.websiteUrl && (
            <Button
              variant="outline"
              onClick={() => window.open(serverData.websiteUrl!, "_blank")}
            >
              <ExternalLink className="w-4 h-4 mr-2" />
              Website
            </Button>
          )}
        </div>

        {/* Tabs */}
        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList className="mb-6">
            <TabsTrigger value="overview">
              <FileText className="w-4 h-4 mr-2" />
              Overview
            </TabsTrigger>
            <TabsTrigger value="configuration">
              <Settings className="w-4 h-4 mr-2" />
              Configuration
            </TabsTrigger>
            <TabsTrigger value="score">
              <TrendingUp className="w-4 h-4 mr-2" />
              Score
            </TabsTrigger>
            <TabsTrigger value="playground" disabled={!installed}>
              <PlayCircle className="w-4 h-4 mr-2" />
              Playground
              {!installed && <span className="ml-1 text-xs">(Install required)</span>}
            </TabsTrigger>
          </TabsList>

          {/* Overview Tab */}
          <TabsContent value="overview" className="space-y-6">
            <Card className="p-6">
              <h3 className="text-lg font-semibold mb-4">Description</h3>
              <p className="text-muted-foreground">{serverData.description}</p>
            </Card>

            <div className="grid md:grid-cols-2 gap-6">
              <Card className="p-6">
                <h3 className="text-lg font-semibold mb-4">Information</h3>
                <dl className="space-y-3">
                  <div>
                    <dt className="text-sm font-medium text-muted-foreground">Version</dt>
                    <dd className="text-sm mt-1">{serverData.version}</dd>
                  </div>
                  {publishedAt && (
                    <div>
                      <dt className="text-sm font-medium text-muted-foreground">Published</dt>
                      <dd className="text-sm mt-1">
                        {formatDistanceToNow(new Date(publishedAt))}
                      </dd>
                    </div>
                  )}
                  {updatedAt && (
                    <div>
                      <dt className="text-sm font-medium text-muted-foreground">Last Updated</dt>
                      <dd className="text-sm mt-1">
                        {formatDistanceToNow(new Date(updatedAt))}
                      </dd>
                    </div>
                  )}
                  {serverData.repository?.source && (
                    <div>
                      <dt className="text-sm font-medium text-muted-foreground">Source</dt>
                      <dd className="text-sm mt-1 capitalize">
                        {serverData.repository.source}
                      </dd>
                    </div>
                  )}
                </dl>
              </Card>

              {serverData.packages && serverData.packages.length > 0 && (
                <Card className="p-6">
                  <h3 className="text-lg font-semibold mb-4">Packages</h3>
                  <div className="space-y-3">
                    {serverData.packages.map((pkg, idx) => (
                      <div key={idx} className="p-3 bg-muted rounded-lg">
                        <div className="flex items-center gap-2 mb-1">
                          <Package className="w-4 h-4" />
                          <Badge variant="secondary">{pkg.registryType}</Badge>
                          <Badge variant="outline">v{pkg.version}</Badge>
                        </div>
                        <p className="text-sm text-muted-foreground">
                          {pkg.identifier}
                        </p>
                        {pkg.runtimeHint && (
                          <p className="text-xs text-muted-foreground mt-1">
                            Runtime: {pkg.runtimeHint}
                          </p>
                        )}
                      </div>
                    ))}
                  </div>
                </Card>
              )}
            </div>
          </TabsContent>

          {/* Configuration Tab */}
          <TabsContent value="configuration" className="space-y-6">
            {serverData.packages?.map((pkg, idx) => (
              <Card key={idx} className="p-6">
                <h3 className="text-lg font-semibold mb-4">
                  {pkg.identifier} Configuration
                </h3>

                {pkg.environmentVariables && pkg.environmentVariables.length > 0 && (
                  <div className="mb-6">
                    <h4 className="font-medium mb-3 flex items-center gap-2">
                      <Settings className="w-4 h-4" />
                      Environment Variables
                    </h4>
                    <div className="space-y-3">
                      {pkg.environmentVariables.map((envVar) => (
                        <div
                          key={envVar.name}
                          className="p-3 bg-muted rounded-lg"
                        >
                          <div className="flex items-center gap-2 mb-1">
                            <code className="text-sm font-mono">
                              {envVar.name}
                            </code>
                            {envVar.isRequired && (
                              <Badge variant="destructive" className="text-xs">
                                Required
                              </Badge>
                            )}
                            {envVar.isSecret && (
                              <Badge variant="outline" className="text-xs">
                                Secret
                              </Badge>
                            )}
                          </div>
                          {envVar.description && (
                            <p className="text-sm text-muted-foreground">
                              {envVar.description}
                            </p>
                          )}
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {pkg.transport && (
                  <div>
                    <h4 className="font-medium mb-3 flex items-center gap-2">
                      <Activity className="w-4 h-4" />
                      Transport
                    </h4>
                    <div className="p-3 bg-muted rounded-lg">
                      <div className="flex items-center gap-2">
                        <Badge>{pkg.transport.type}</Badge>
                        {pkg.transport.url && (
                          <span className="text-sm text-muted-foreground">
                            {pkg.transport.url}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                )}
              </Card>
            ))}
          </TabsContent>

          {/* Score Tab */}
          <TabsContent value="score" className="space-y-6">
            <Card className="p-6">
              <h3 className="text-lg font-semibold mb-4">Overall Quality Score</h3>
              <div className="flex items-center gap-4 mb-6">
                <div className="text-5xl font-bold">{score.overall}</div>
                <div>
                  <div className="text-xl font-semibold">
                    {getScoreLabel(score.overall)}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    Out of 100 points
                  </div>
                </div>
              </div>
              <p className="text-sm text-muted-foreground">
                This score is calculated based on multiple factors including
                security, reliability, performance, documentation, community
                support, and maintenance activity.
              </p>
            </Card>

            <div className="grid md:grid-cols-2 gap-4">
              <Card className="p-6">
                <div className="flex items-center gap-3 mb-2">
                  <Shield className="w-5 h-5 text-blue-600" />
                  <h4 className="font-semibold">Security</h4>
                </div>
                <div className="flex items-baseline gap-2">
                  <span className={`text-2xl font-bold ${getScoreColor(score.security)}`}>
                    {score.security}
                  </span>
                  <span className="text-sm text-muted-foreground">/100</span>
                </div>
                <p className="text-xs text-muted-foreground mt-2">
                  Vulnerability scanning, secure coding practices
                </p>
              </Card>

              <Card className="p-6">
                <div className="flex items-center gap-3 mb-2">
                  <Activity className="w-5 h-5 text-green-600" />
                  <h4 className="font-semibold">Reliability</h4>
                </div>
                <div className="flex items-baseline gap-2">
                  <span className={`text-2xl font-bold ${getScoreColor(score.reliability)}`}>
                    {score.reliability}
                  </span>
                  <span className="text-sm text-muted-foreground">/100</span>
                </div>
                <p className="text-xs text-muted-foreground mt-2">
                  Uptime, error handling, stability
                </p>
              </Card>

              <Card className="p-6">
                <div className="flex items-center gap-3 mb-2">
                  <TrendingUp className="w-5 h-5 text-purple-600" />
                  <h4 className="font-semibold">Performance</h4>
                </div>
                <div className="flex items-baseline gap-2">
                  <span className={`text-2xl font-bold ${getScoreColor(score.performance)}`}>
                    {score.performance}
                  </span>
                  <span className="text-sm text-muted-foreground">/100</span>
                </div>
                <p className="text-xs text-muted-foreground mt-2">
                  Response time, resource usage
                </p>
              </Card>

              <Card className="p-6">
                <div className="flex items-center gap-3 mb-2">
                  <FileText className="w-5 h-5 text-orange-600" />
                  <h4 className="font-semibold">Documentation</h4>
                </div>
                <div className="flex items-baseline gap-2">
                  <span className={`text-2xl font-bold ${getScoreColor(score.documentation)}`}>
                    {score.documentation}
                  </span>
                  <span className="text-sm text-muted-foreground">/100</span>
                </div>
                <p className="text-xs text-muted-foreground mt-2">
                  Completeness, examples, API docs
                </p>
              </Card>

              <Card className="p-6">
                <div className="flex items-center gap-3 mb-2">
                  <Users className="w-5 h-5 text-pink-600" />
                  <h4 className="font-semibold">Community</h4>
                </div>
                <div className="flex items-baseline gap-2">
                  <span className={`text-2xl font-bold ${getScoreColor(score.community)}`}>
                    {score.community}
                  </span>
                  <span className="text-sm text-muted-foreground">/100</span>
                </div>
                <p className="text-xs text-muted-foreground mt-2">
                  Stars, contributors, issues resolution
                </p>
              </Card>

              <Card className="p-6">
                <div className="flex items-center gap-3 mb-2">
                  <Wrench className="w-5 h-5 text-cyan-600" />
                  <h4 className="font-semibold">Maintenance</h4>
                </div>
                <div className="flex items-baseline gap-2">
                  <span className={`text-2xl font-bold ${getScoreColor(score.maintenance)}`}>
                    {score.maintenance}
                  </span>
                  <span className="text-sm text-muted-foreground">/100</span>
                </div>
                <p className="text-xs text-muted-foreground mt-2">
                  Update frequency, active development
                </p>
              </Card>
            </div>
          </TabsContent>

          {/* Playground Tab */}
          <TabsContent value="playground">
            {installed && <ServerPlayground server={server} />}
          </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}

