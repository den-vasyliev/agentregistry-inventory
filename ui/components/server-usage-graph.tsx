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
  Package,
  Globe,
} from "lucide-react"
import type { ServerJSON } from "@/lib/admin-api"

// ── Node type config ────────────────────────────────────────────────

type UsageNodeType = "server" | "agent" | "package" | "remote"

const nodeConfig: Record<
  UsageNodeType,
  { border: string; bg: string; icon: typeof Bot }
> = {
  server: { border: "border-blue-500", bg: "bg-blue-500/10", icon: Server },
  agent: { border: "border-purple-500", bg: "bg-purple-500/10", icon: Bot },
  package: { border: "border-teal-500", bg: "bg-teal-500/10", icon: Package },
  remote: { border: "border-yellow-500", bg: "bg-yellow-500/10", icon: Globe },
}

// ── Custom node component ───────────────────────────────────────────

type UsageNodeData = {
  label: string
  subtitle?: string
  nodeType: UsageNodeType
  isRoot?: boolean
}

function UsageNode({ data }: NodeProps<Node<UsageNodeData>>) {
  const cfg = nodeConfig[data.nodeType]
  const Icon = cfg.icon

  return (
    <div
      className={`rounded-lg border-2 ${cfg.border} ${cfg.bg} px-3 py-2 min-w-[160px] max-w-[220px] bg-card shadow-sm`}
      title={data.subtitle ? `${data.label}\n${data.subtitle}` : data.label}
    >
      <Handle type="target" position={Position.Bottom} className="!bg-muted-foreground/50 !w-2 !h-2" />
      <Handle type="source" position={Position.Top} className="!bg-muted-foreground/50 !w-2 !h-2" />
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

const nodeTypes = { usage: UsageNode }

// ── Graph data builder ──────────────────────────────────────────────

interface UsageRef {
  namespace: string
  name: string
  kind?: string
}

function buildGraphData(
  server: ServerJSON,
  usedBy: UsageRef[],
): { nodes: Node<UsageNodeData>[]; edges: Edge[] } {
  const nodes: Node<UsageNodeData>[] = []
  const edges: Edge[] = []

  // Root server node (center)
  nodes.push({
    id: "server-root",
    type: "usage",
    position: { x: 0, y: 0 },
    data: { label: server.title || server.name, subtitle: `v${server.version}`, nodeType: "server", isRoot: true },
  })

  // Agents that use this server (above)
  usedBy.forEach((ref, i) => {
    const id = `agent-${i}`
    nodes.push({
      id,
      type: "usage",
      position: { x: 0, y: 0 },
      data: { label: ref.name, subtitle: ref.kind ? `${ref.kind} in ${ref.namespace}` : ref.namespace, nodeType: "agent" },
    })
    edges.push({ id: `e-${id}-root`, source: id, target: "server-root", type: "smoothstep" })
  })

  // Packages (below)
  const packages = server.packages || []
  packages.forEach((pkg, i) => {
    const id = `pkg-${i}`
    nodes.push({
      id,
      type: "usage",
      position: { x: 0, y: 0 },
      data: {
        label: pkg.identifier,
        subtitle: pkg.version ? `v${pkg.version}` : pkg.registryType,
        nodeType: "package",
      },
    })
    edges.push({ id: `e-root-${id}`, source: "server-root", target: id, type: "smoothstep" })
  })

  // Remotes (below)
  const remotes = server.remotes || []
  remotes.forEach((remote, i) => {
    const id = `remote-${i}`
    nodes.push({
      id,
      type: "usage",
      position: { x: 0, y: 0 },
      data: { label: remote.url || remote.type, subtitle: remote.type, nodeType: "remote" },
    })
    edges.push({ id: `e-root-${id}`, source: "server-root", target: id, type: "smoothstep" })
  })

  return { nodes, edges }
}

// ── Dagre layout ────────────────────────────────────────────────────

function getLayoutedElements(
  nodes: Node<UsageNodeData>[],
  edges: Edge[],
): { nodes: Node<UsageNodeData>[]; edges: Edge[] } {
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

export function ServerUsageGraph({
  server,
  usedBy,
}: {
  server: ServerJSON
  usedBy: UsageRef[]
}) {
  const { nodes: layoutedNodes, edges: layoutedEdges } = useMemo(() => {
    const { nodes, edges } = buildGraphData(server, usedBy)
    return getLayoutedElements(nodes, edges)
  }, [server, usedBy])

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
