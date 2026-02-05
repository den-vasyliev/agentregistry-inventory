"use client"

import { ModelResponse } from "@/lib/admin-api"
import { Card } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { Calendar, Tag, Trash2, Brain, CheckCircle2, XCircle, Server } from "lucide-react"

interface ModelCardProps {
  model: ModelResponse
  onDelete?: (model: ModelResponse) => void
  showDelete?: boolean
  onClick?: () => void
}

export function ModelCard({ model, onDelete, showDelete = true, onClick }: ModelCardProps) {
  const { model: modelData, _meta } = model

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

  // Determine status badge
  const getStatusBadge = () => {
    if (_meta?.ready === false) {
      return (
        <Tooltip>
          <TooltipTrigger asChild>
            <Badge variant="destructive" className="text-xs">
              <XCircle className="h-3 w-3 mr-1" />
              Not Ready
            </Badge>
          </TooltipTrigger>
          <TooltipContent>
            <p>{_meta?.message || 'Model is not ready'}</p>
          </TooltipContent>
        </Tooltip>
      )
    }

    if (_meta?.ready === true) {
      return (
        <Badge variant="default" className="text-xs bg-green-500/10 text-green-600 hover:bg-green-500/20 border-green-500/20">
          <CheckCircle2 className="h-3 w-3 mr-1" />
          Ready
        </Badge>
      )
    }

    return null
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
            <Brain className="h-5 w-5 text-primary" />
          </div>
          <div className="flex-1 min-w-0">
            <h3 className="font-semibold text-lg mb-1">{modelData.name}</h3>
            <div className="text-xs text-muted-foreground flex items-center gap-1 flex-wrap">
              {modelData.provider && (
                <Badge variant="outline" className="text-xs">
                  {modelData.provider}
                </Badge>
              )}
              {modelData.model && (
                <Badge variant="secondary" className="text-xs">
                  {modelData.model}
                </Badge>
              )}
              {/* Deployment status badge */}
              {_meta?.deployment && (
                <Badge 
                  variant="outline" 
                  className={`text-xs ${_meta.deployment.ready 
                    ? 'bg-green-500/10 text-green-600 border-green-500/20' 
                    : 'bg-red-500/10 text-red-600 border-red-500/20'}`}
                >
                  {_meta.deployment.ready ? 'Running' : 'Not Ready'}
                </Badge>
              )}
              {/* External badge for discovered resources */}
              {_meta?.isDiscovered && (
                <Badge variant="outline" className="bg-teal-500/10 text-teal-600 border-teal-500/20 text-xs">
                  External
                </Badge>
              )}
              {getStatusBadge()}
            </div>
          </div>
        </div>
        <div className="flex items-center gap-1 ml-2">
          {showDelete && onDelete && (
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 text-destructive hover:text-destructive hover:bg-destructive/10"
              onClick={(e) => {
                e.stopPropagation()
                onDelete(model)
              }}
              title="Remove from registry"
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          )}
        </div>
      </div>

      {modelData.description && (
        <p className="text-sm text-muted-foreground mb-3 line-clamp-2">
          {modelData.description}
        </p>
      )}

      <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
        {modelData.baseUrl && (
          <div className="flex items-center gap-1">
            <Server className="h-3 w-3" />
            <span className="font-mono text-xs truncate max-w-[200px]" title={modelData.baseUrl}>
              {modelData.baseUrl}
            </span>
          </div>
        )}

        {_meta?.usedBy && _meta.usedBy.length > 0 && (
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="flex items-center gap-1">
                <Tag className="h-3 w-3" />
                <span>Used by {_meta.usedBy.length} resource{_meta.usedBy.length !== 1 ? 's' : ''}</span>
              </div>
            </TooltipTrigger>
            <TooltipContent>
              <div className="text-xs">
                {_meta.usedBy.map((ref, idx) => (
                  <div key={idx}>
                    {ref.kind}: {ref.namespace}/{ref.name}
                  </div>
                ))}
              </div>
            </TooltipContent>
          </Tooltip>
        )}
      </div>
      </Card>
    </TooltipProvider>
  )
}
