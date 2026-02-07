"use client"

import { useMemo } from "react"
import {
  ReactFlow,
  Background,
  Handle,
  Position,
  type Node,
  type Edge,
  type NodeProps,
  useNodesState,
  useEdgesState,
} from "@xyflow/react"
import "@xyflow/react/dist/style.css"
import {
  Bot,
  Server,
  Blocks,
  Settings2,
  Package,
  Globe,
} from "lucide-react"
import type { AgentJSON } from "@/lib/admin-api"

// ── Node type config ────────────────────────────────────────────────

type DepNodeType =
  | "agent"
  | "mcpTool"
  | "agentTool"
  | "skill"
  | "modelConfig"
  | "mcpServer"
  | "package"
  | "remote"

const nodeConfig: Record<
  DepNodeType,
  { border: string; bg: string; icon: typeof Bot }
> = {
  agent: { border: "border-purple-500", bg: "bg-purple-500/10", icon: Bot },
  mcpTool: { border: "border-blue-500", bg: "bg-blue-500/10", icon: Server },
  agentTool: { border: "border-green-500", bg: "bg-green-500/10", icon: Bot },
  skill: { border: "border-orange-500", bg: "bg-orange-500/10", icon: Blocks },
  modelConfig: {
    border: "border-pink-500",
    bg: "bg-pink-500/10",
    icon: Settings2,
  },
  mcpServer: {
    border: "border-sky-500",
    bg: "bg-sky-500/10",
    icon: Server,
  },
  package: {
    border: "border-teal-500",
    bg: "bg-teal-500/10",
    icon: Package,
  },
  remote: {
    border: "border-yellow-500",
    bg: "bg-yellow-500/10",
    icon: Globe,
  },
}

// ── Custom node component ───────────────────────────────────────────

type DepNodeData = {
  label: string
  subtitle?: string
  depType: DepNodeType
  isRoot?: boolean
}

function DependencyNode({ data }: NodeProps<Node<DepNodeData>>) {
  const cfg = nodeConfig[data.depType]
  const Icon = cfg.icon

  return (
    <div
      className={`rounded-lg border-2 ${cfg.border} ${cfg.bg} px-3 py-2 min-w-[160px] max-w-[220px] bg-card shadow-sm`}
      title={data.subtitle ? `${data.label}\n${data.subtitle}` : data.label}
    >
      {!data.isRoot && <Handle type="target" position={Position.Top} className="!bg-muted-foreground/50 !w-2 !h-2" />}
      <Handle type="source" position={Position.Bottom} className="!bg-muted-foreground/50 !w-2 !h-2" />
      <div className="flex items-center gap-2">
        <Icon className="h-4 w-4 shrink-0 text-foreground/70" />
        <div className="min-w-0">
          <p className="text-xs font-semibold truncate text-foreground">
            {data.label}
          </p>
          {data.subtitle && (
            <p className="text-[10px] text-muted-foreground truncate">
              {data.subtitle}
            </p>
          )}
        </div>
      </div>
    </div>
  )
}

const nodeTypes = { dependency: DependencyNode }

// ── Graph data builder ──────────────────────────────────────────────

function buildGraphData(agent: AgentJSON): { nodes: Node<DepNodeData>[]; edges: Edge[] } {
  const nodes: Node<DepNodeData>[] = []
  const edges: Edge[] = []

  // Root agent node
  nodes.push({
    id: "agent-root",
    type: "dependency",
    position: { x: 0, y: 0 },
    data: { label: agent.name, subtitle: agent.agentType, depType: "agent", isRoot: true },
  })

  // Tools
  const tools = agent.tools || []
  tools.forEach((tool, i) => {
    const id = `tool-${i}`
    const depType: DepNodeType = tool.type === "McpServer" ? "mcpTool" : "agentTool"
    const subtitle =
      tool.toolNames && tool.toolNames.length > 0
        ? `${tool.toolNames.length} tool${tool.toolNames.length > 1 ? "s" : ""}`
        : tool.type
    nodes.push({
      id,
      type: "dependency",
      position: { x: 0, y: 0 },
      data: { label: tool.name, subtitle, depType },
    })
    edges.push({ id: `e-root-${id}`, source: "agent-root", target: id, type: "smoothstep" })
  })

  // Skills
  const skills = agent.skills || []
  skills.forEach((skill, i) => {
    const id = `skill-${i}`
    const label = skill.split("/").pop() || skill
    nodes.push({
      id,
      type: "dependency",
      position: { x: 0, y: 0 },
      data: { label, subtitle: "Skill (OCI)", depType: "skill" },
    })
    edges.push({ id: `e-root-${id}`, source: "agent-root", target: id, type: "smoothstep" })
  })

  // Model config
  if (agent.modelConfigRef) {
    const id = "model-config"
    nodes.push({
      id,
      type: "dependency",
      position: { x: 0, y: 0 },
      data: { label: agent.modelConfigRef, subtitle: "Model Config", depType: "modelConfig" },
    })
    edges.push({ id: `e-root-${id}`, source: "agent-root", target: id, type: "smoothstep" })
  }

  // MCP Servers (legacy)
  const mcpServers = agent.mcpServers || []
  mcpServers.forEach((mcp, i) => {
    const id = `mcp-${i}`
    nodes.push({
      id,
      type: "dependency",
      position: { x: 0, y: 0 },
      data: { label: mcp.name, subtitle: mcp.type, depType: "mcpServer" },
    })
    edges.push({ id: `e-root-${id}`, source: "agent-root", target: id, type: "smoothstep" })
  })

  // Packages
  const packages = agent.packages || []
  packages.forEach((pkg, i) => {
    const id = `pkg-${i}`
    nodes.push({
      id,
      type: "dependency",
      position: { x: 0, y: 0 },
      data: {
        label: pkg.identifier,
        subtitle: pkg.version ? `v${pkg.version}` : pkg.registryType,
        depType: "package",
      },
    })
    edges.push({ id: `e-root-${id}`, source: "agent-root", target: id, type: "smoothstep" })
  })

  // Remotes
  const remotes = agent.remotes || []
  remotes.forEach((remote, i) => {
    const id = `remote-${i}`
    nodes.push({
      id,
      type: "dependency",
      position: { x: 0, y: 0 },
      data: { label: remote.url || remote.type, subtitle: remote.type, depType: "remote" },
    })
    edges.push({ id: `e-root-${id}`, source: "agent-root", target: id, type: "smoothstep" })
  })

  return { nodes, edges }
}

// ── Dagre layout ────────────────────────────────────────────────────

function getLayoutedElements(
  nodes: Node<DepNodeData>[],
  edges: Edge[],
): { nodes: Node<DepNodeData>[]; edges: Edge[] } {
  const Dagre = require("@dagrejs/dagre")
  const g = new Dagre.graphlib.Graph().setDefaultEdgeLabel(() => ({}))
  g.setGraph({ rankdir: "TB", nodesep: 50, ranksep: 80 })

  nodes.forEach((node) => g.setNode(node.id, { width: 200, height: 60 }))
  edges.forEach((edge) => g.setEdge(edge.source, edge.target))

  Dagre.layout(g)

  const layoutedNodes = nodes.map((node) => {
    const pos = g.node(node.id)
    return { ...node, position: { x: pos.x - 100, y: pos.y - 30 } }
  })

  return { nodes: layoutedNodes, edges }
}

// ── Exported component ──────────────────────────────────────────────

export function AgentDependencyGraph({ agent }: { agent: AgentJSON }) {
  const { nodes: layoutedNodes, edges: layoutedEdges } = useMemo(() => {
    const { nodes, edges } = buildGraphData(agent)
    return getLayoutedElements(nodes, edges)
  }, [agent])

  const [nodes, , onNodesChange] = useNodesState(layoutedNodes)
  const [edges] = useEdgesState(layoutedEdges)

  return (
    <div className="h-[450px] w-full rounded-lg border bg-card">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        nodeTypes={nodeTypes}
        fitView
        minZoom={0.3}
        maxZoom={1.5}
        nodesConnectable={false}
        proOptions={{ hideAttribution: false }}
      >
        <Background gap={16} size={1} color="rgba(128,128,128,0.3)" />
      </ReactFlow>
    </div>
  )
}
