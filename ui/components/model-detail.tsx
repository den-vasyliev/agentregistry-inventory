"use client"

import { useState, useEffect } from "react"
import { ModelResponse } from "@/lib/admin-api"
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
import {
  X,
  Calendar,
  ArrowLeft,
  Brain,
  Server,
  CheckCircle2,
  XCircle,
  Circle,
  Tag,
} from "lucide-react"

interface ModelDetailProps {
  model: ModelResponse
  onClose: () => void
}

export function ModelDetail({ model, onClose }: ModelDetailProps) {
  const [activeTab, setActiveTab] = useState("overview")

  const { model: modelData, _meta } = model
  const official = _meta?.['io.modelcontextprotocol.registry/official']

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

  const formatDate = (dateString: string) => {
    try {
      return new Date(dateString).toLocaleString('en-US', {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      })
    } catch {
      return dateString
    }
  }

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
          Back to Models
        </Button>

        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-start gap-4 flex-1">
            <div className="w-16 h-16 rounded bg-primary/20 flex items-center justify-center flex-shrink-0 mt-1">
              <Brain className="h-8 w-8 text-primary" />
            </div>
            <div className="flex-1">
              <div className="flex items-center gap-3 mb-2 flex-wrap">
                <h1 className="text-3xl font-bold">{modelData.name}</h1>
                {modelData.provider && (
                  <Badge variant="outline" className="text-sm">
                    {modelData.provider}
                  </Badge>
                )}
                {modelData.model && (
                  <Badge variant="secondary" className="text-sm">
                    {modelData.model}
                  </Badge>
                )}
                {_meta?.ready && (
                  <Badge variant="default" className="text-sm bg-green-500/10 text-green-600 hover:bg-green-500/20 border-green-500/20">
                    <CheckCircle2 className="h-3 w-3 mr-1" />
                    Ready
                  </Badge>
                )}
                {_meta?.ready === false && (
                  <Badge variant="destructive" className="text-sm">
                    <XCircle className="h-3 w-3 mr-1" />
                    Not Ready
                  </Badge>
                )}
              </div>
              {modelData.description && (
                <p className="text-muted-foreground">{modelData.description}</p>
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
          <div className="flex items-center gap-2 px-3 py-2 bg-muted rounded-md">
            <Circle className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-muted-foreground">Status:</span>
            <span className="font-medium">{official?.status || 'unknown'}</span>
          </div>

          {official?.publishedAt && (
            <div className="flex items-center gap-2 px-3 py-2 bg-muted rounded-md">
              <Calendar className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="text-muted-foreground">Published:</span>
              <span className="font-medium">{formatDate(official.publishedAt)}</span>
            </div>
          )}

          {modelData.baseUrl && (
            <div className="flex items-center gap-2 px-3 py-2 bg-muted rounded-md">
              <Server className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="text-muted-foreground">Base URL:</span>
              <span className="font-medium font-mono text-xs">{modelData.baseUrl}</span>
            </div>
          )}
        </div>

        {/* Tabs */}
        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList>
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="usage">Usage</TabsTrigger>
            {_meta?.message && <TabsTrigger value="status">Status</TabsTrigger>}
          </TabsList>

          {/* Overview Tab */}
          <TabsContent value="overview" className="mt-6">
            <div className="grid gap-6">
              <Card className="p-6">
                <h3 className="text-lg font-semibold mb-4">Model Information</h3>
                <dl className="space-y-3">
                  <div className="flex items-start">
                    <dt className="text-sm font-medium text-muted-foreground w-32">Name:</dt>
                    <dd className="text-sm flex-1">{modelData.name}</dd>
                  </div>
                  <div className="flex items-start">
                    <dt className="text-sm font-medium text-muted-foreground w-32">Provider:</dt>
                    <dd className="text-sm flex-1">{modelData.provider}</dd>
                  </div>
                  <div className="flex items-start">
                    <dt className="text-sm font-medium text-muted-foreground w-32">Model:</dt>
                    <dd className="text-sm flex-1 font-mono">{modelData.model}</dd>
                  </div>
                  {modelData.baseUrl && (
                    <div className="flex items-start">
                      <dt className="text-sm font-medium text-muted-foreground w-32">Base URL:</dt>
                      <dd className="text-sm flex-1 font-mono break-all">{modelData.baseUrl}</dd>
                    </div>
                  )}
                  {modelData.description && (
                    <div className="flex items-start">
                      <dt className="text-sm font-medium text-muted-foreground w-32">Description:</dt>
                      <dd className="text-sm flex-1">{modelData.description}</dd>
                    </div>
                  )}
                </dl>
              </Card>

              <Card className="p-6">
                <h3 className="text-lg font-semibold mb-4">Registry Metadata</h3>
                <dl className="space-y-3">
                  {official?.status && (
                    <div className="flex items-start">
                      <dt className="text-sm font-medium text-muted-foreground w-32">Status:</dt>
                      <dd className="text-sm flex-1">
                        <Badge variant={official.status === 'active' ? 'default' : 'secondary'}>
                          {official.status}
                        </Badge>
                      </dd>
                    </div>
                  )}
                  {official?.publishedAt && (
                    <div className="flex items-start">
                      <dt className="text-sm font-medium text-muted-foreground w-32">Published At:</dt>
                      <dd className="text-sm flex-1">{formatDate(official.publishedAt)}</dd>
                    </div>
                  )}
                  {official?.updatedAt && (
                    <div className="flex items-start">
                      <dt className="text-sm font-medium text-muted-foreground w-32">Updated At:</dt>
                      <dd className="text-sm flex-1">{formatDate(official.updatedAt)}</dd>
                    </div>
                  )}
                  <div className="flex items-start">
                    <dt className="text-sm font-medium text-muted-foreground w-32">Ready:</dt>
                    <dd className="text-sm flex-1">
                      {_meta?.ready ? (
                        <Badge variant="default" className="bg-green-500/10 text-green-600 hover:bg-green-500/20 border-green-500/20">
                          <CheckCircle2 className="h-3 w-3 mr-1" />
                          Yes
                        </Badge>
                      ) : (
                        <Badge variant="destructive">
                          <XCircle className="h-3 w-3 mr-1" />
                          No
                        </Badge>
                      )}
                    </dd>
                  </div>
                </dl>
              </Card>
            </div>
          </TabsContent>

          {/* Usage Tab */}
          <TabsContent value="usage" className="mt-6">
            <Card className="p-6">
              <h3 className="text-lg font-semibold mb-4">Used By</h3>
              {_meta?.usedBy && _meta.usedBy.length > 0 ? (
                <div className="space-y-3">
                  {_meta.usedBy.map((ref, idx) => (
                    <div key={idx} className="flex items-center gap-3 p-3 bg-muted rounded-md">
                      <Tag className="h-4 w-4 text-muted-foreground" />
                      <div>
                        <div className="text-sm font-medium">{ref.name}</div>
                        <div className="text-xs text-muted-foreground">
                          {ref.kind && `${ref.kind} in `}{ref.namespace}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">
                  This model is not currently used by any resources.
                </p>
              )}
            </Card>
          </TabsContent>

          {/* Status Tab */}
          {_meta?.message && (
            <TabsContent value="status" className="mt-6">
              <Card className="p-6">
                <h3 className="text-lg font-semibold mb-4">Status Message</h3>
                <div className="bg-muted p-4 rounded-md">
                  <p className="text-sm font-mono whitespace-pre-wrap">{_meta.message}</p>
                </div>
              </Card>
            </TabsContent>
          )}
        </Tabs>
      </div>
    </div>
  )
}
