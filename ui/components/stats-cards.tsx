"use client"

import { Card } from "@/components/ui/card"
import MCPIcon from "@/components/icons/mcp"
import { Zap, Bot, Brain } from "lucide-react"
import { ServerStats } from "@/lib/admin-api"

interface StatsCardsProps {
  stats: ServerStats
  skillCount: number
  agentCount: number
  modelCount: number
}

export function StatsCards({ stats, skillCount, agentCount, modelCount }: StatsCardsProps) {
  return (
    <div className="bg-muted/30 border-b">
      <div className="container mx-auto px-6 py-6">
        <div className="grid gap-4 md:grid-cols-4">
          <Card className="p-4 hover:shadow-md transition-all duration-200 border hover:border-primary/20">
            <div className="flex items-center gap-3">
              <div className="p-2 bg-primary/10 rounded-lg flex items-center justify-center">
                <span className="h-5 w-5 text-primary flex items-center justify-center">
                  <MCPIcon />
                </span>
              </div>
              <div>
                <p className="text-2xl font-bold">{stats.total_server_names}</p>
                <p className="text-xs text-muted-foreground">MCP</p>
              </div>
            </div>
          </Card>

          <Card className="p-4 hover:shadow-md transition-all duration-200 border hover:border-primary/20">
            <div className="flex items-center gap-3">
              <div className="p-2 bg-primary/20 rounded-lg flex items-center justify-center">
                <Zap className="h-5 w-5 text-primary" />
              </div>
              <div>
                <p className="text-2xl font-bold">{skillCount}</p>
                <p className="text-xs text-muted-foreground">Skills</p>
              </div>
            </div>
          </Card>

          <Card className="p-4 hover:shadow-md transition-all duration-200 border hover:border-primary/20">
            <div className="flex items-center gap-3">
              <div className="p-2 bg-primary/30 rounded-lg flex items-center justify-center">
                <Bot className="h-5 w-5 text-primary" />
              </div>
              <div>
                <p className="text-2xl font-bold">{agentCount}</p>
                <p className="text-xs text-muted-foreground">Agents</p>
              </div>
            </div>
          </Card>

          <Card className="p-4 hover:shadow-md transition-all duration-200 border hover:border-primary/20">
            <div className="flex items-center gap-3">
              <div className="p-2 bg-primary/40 rounded-lg flex items-center justify-center">
                <Brain className="h-5 w-5 text-primary" />
              </div>
              <div>
                <p className="text-2xl font-bold">{modelCount}</p>
                <p className="text-xs text-muted-foreground">Models</p>
              </div>
            </div>
          </Card>
        </div>
      </div>
    </div>
  )
}
