"use client"

import { useMemo, useCallback, useState } from "react"
import {
  ReactFlow,
  Background,
  Handle,
  Position,
  type Node,
  type Edge,
  type NodeProps,
} from "@xyflow/react"
import "@xyflow/react/dist/style.css"
import {
  Server,
  Globe,
  Bot,
  Zap,
  Brain,
  Layers,
  Compass,
  Shield,
  Container,
  Cog,
} from "lucide-react"
import MCPIcon from "@/components/icons/mcp"
import KubernetesIcon from "@/components/icons/kubernetes"
import type { DiscoveryMapConfig } from "@/lib/admin-api"

// ── Node type config ────────────────────────────────────────────────

type MapNodeType =
  | "gateway"
  | "registry"
  | "controller"
  | "runtime"
  | "cluster"
  | "environment"
  | "namespace"
  | "resourceType"

const nodeConfig: Record<
  MapNodeType,
  { border: string; bg: string; icon: typeof Server }
> = {
  gateway: { border: "border-indigo-500", bg: "bg-indigo-500/10", icon: Shield },
  registry: { border: "border-purple-500", bg: "bg-purple-500/10", icon: Compass },
  controller: { border: "border-purple-400", bg: "bg-purple-400/10", icon: Cog },
  runtime: { border: "border-cyan-500", bg: "bg-cyan-500/10", icon: Container },
  cluster: { border: "border-blue-500", bg: "bg-blue-500/10", icon: Server },
  environment: { border: "border-green-500", bg: "bg-green-500/10", icon: Globe },
  namespace: { border: "border-teal-500", bg: "bg-teal-500/10", icon: Layers },
  resourceType: { border: "border-orange-500", bg: "bg-orange-500/10", icon: Layers },
}

// ── Custom node component ───────────────────────────────────────────

type MapNodeData = {
  label: string
  subtitle?: string
  nodeType: MapNodeType
  isRoot?: boolean
  connected?: boolean
  badge?: string
  counts?: { mcpServers: number; agents: number; skills: number; models: number }
}

function ResourceTypeIcon({ type }: { type: string }) {
  switch (type) {
    case "MCPServer":
    case "RemoteMCPServer":
      return <span className="h-4 w-4 flex items-center justify-center text-foreground/70"><MCPIcon /></span>
    case "Agent":
      return <Bot className="h-4 w-4 shrink-0 text-foreground/70" />
    case "Skill":
      return <Zap className="h-4 w-4 shrink-0 text-foreground/70" />
    case "ModelConfig":
      return <Brain className="h-4 w-4 shrink-0 text-foreground/70" />
    default:
      return <Layers className="h-4 w-4 shrink-0 text-foreground/70" />
  }
}

function MapNode({ data }: NodeProps<Node<MapNodeData>>) {
  const cfg = nodeConfig[data.nodeType]
  const Icon = cfg.icon

  return (
    <div
      className={`rounded-lg border-2 ${cfg.border} ${cfg.bg} px-4 py-2.5 min-w-[200px] max-w-[280px] bg-card shadow-sm`}
      title={data.subtitle ? `${data.label}\n${data.subtitle}` : data.label}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-muted-foreground/50 !w-2 !h-2"
      />
      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-muted-foreground/50 !w-2 !h-2"
      />
      <div className="flex items-center gap-2">
        <Icon className="h-5 w-5 shrink-0 text-foreground/70" />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <p className="text-sm font-semibold truncate text-foreground">
              {data.label}
            </p>
            {data.connected !== undefined && (
              <KubernetesIcon className="h-3.5 w-3.5 shrink-0 text-blue-500" />
            )}
          </div>
          {data.subtitle && (
            <p className="text-xs text-muted-foreground truncate">
              {data.subtitle}
            </p>
          )}
        </div>
        {data.badge && (
          <span className="text-[10px] px-1.5 py-0.5 rounded bg-primary/20 text-primary font-medium shrink-0">
            {data.badge}
          </span>
        )}
      </div>
      {data.counts && (
        <div className="flex gap-2 mt-1.5 pt-1.5 border-t border-border/50">
          {data.counts.mcpServers > 0 && (
            <span className="text-[10px] text-muted-foreground flex items-center gap-0.5">
              <span className="h-3 w-3 flex items-center justify-center"><MCPIcon /></span> {data.counts.mcpServers}
            </span>
          )}
          {data.counts.agents > 0 && (
            <span className="text-[10px] text-muted-foreground flex items-center gap-0.5">
              <Bot className="h-3 w-3" /> {data.counts.agents}
            </span>
          )}
          {data.counts.skills > 0 && (
            <span className="text-[10px] text-muted-foreground flex items-center gap-0.5">
              <Zap className="h-3 w-3" /> {data.counts.skills}
            </span>
          )}
          {data.counts.models > 0 && (
            <span className="text-[10px] text-muted-foreground flex items-center gap-0.5">
              <Brain className="h-3 w-3" /> {data.counts.models}
            </span>
          )}
        </div>
      )}
    </div>
  )
}

function ResourceTypeNode({ data }: NodeProps<Node<MapNodeData>>) {
  return (
    <div
      className="rounded-lg border-2 border-orange-500 bg-orange-500/10 px-4 py-2 min-w-[160px] bg-card shadow-sm"
      title={data.label}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-muted-foreground/50 !w-2 !h-2"
      />
      <div className="flex items-center gap-2">
        <ResourceTypeIcon type={data.label} />
        <p className="text-sm font-semibold truncate text-foreground">
          {data.label}
        </p>
      </div>
    </div>
  )
}

const nodeTypes = { map: MapNode, resourceType: ResourceTypeNode }

// ── Graph data builder ──────────────────────────────────────────────

interface ResourceCounts {
  mcpServers: number
  agents: number
  skills: number
  models: number
}

function buildGraphData(
  configs: DiscoveryMapConfig[],
  resourceCounts?: ResourceCounts,
): { nodes: Node<MapNodeData>[]; edges: Edge[] } {
  const nodes: Node<MapNodeData>[] = []
  const edges: Edge[] = []

  // ── Top level: Agent Gateway ──────────────────────────────────────
  nodes.push({
    id: "gateway",
    type: "map",
    position: { x: 0, y: 0 },
    data: {
      label: "Agent Gateway",
      subtitle: "API gateway & routing",
      nodeType: "gateway",
      isRoot: true,
    },
  })

  // ── Provisioned cluster: Registry + Controller + Auto-Discovery ───
  nodes.push({
    id: "registry",
    type: "map",
    position: { x: 0, y: 0 },
    data: {
      label: "Agent Registry Embedded UI",
      subtitle: `${configs.length} discovery config${configs.length !== 1 ? "s" : ""}`,
      nodeType: "registry",
    },
  })

  nodes.push({
    id: "controller",
    type: "map",
    position: { x: 0, y: 0 },
    data: {
      label: "Inventory Controller",
      subtitle: "Resource reconciliation",
      nodeType: "controller",
    },
  })

  nodes.push({
    id: "autodiscovery",
    type: "map",
    position: { x: 0, y: 0 },
    data: {
      label: "Auto-Discovery",
      subtitle: "Multi-cluster discovery",
      nodeType: "controller",
    },
  })

  // Gateway -> UI; UI -> Controller + Auto-Discovery (siblings)
  edges.push({ id: "e-gateway-registry", source: "gateway", target: "registry", type: "smoothstep" })
  edges.push({ id: "e-registry-controller", source: "registry", target: "controller", type: "smoothstep" })
  edges.push({ id: "e-registry-autodiscovery", source: "registry", target: "autodiscovery", type: "smoothstep" })

  // ── Show managed resources in the registry ────────────────────────
  if (resourceCounts) {
    const types = [
      { key: "mcpServers", label: "MCP Servers", count: resourceCounts.mcpServers },
      { key: "agents", label: "Agents", count: resourceCounts.agents },
      { key: "skills", label: "Skills", count: resourceCounts.skills },
      { key: "models", label: "Models", count: resourceCounts.models },
    ].filter(t => t.count > 0)

    for (const t of types) {
      const id = `managed-${t.key}`
      nodes.push({
        id,
        type: "resourceType",
        position: { x: 0, y: 0 },
        data: { label: `${t.label} (${t.count})`, nodeType: "resourceType" },
      })
      edges.push({ id: `e-controller-${id}`, source: "controller", target: id, type: "smoothstep" })
    }
  }

  // ── When no configs, show placeholder ─────────────────────────────
  if (configs.length === 0) {
    nodes.push({
      id: "no-clusters",
      type: "map",
      position: { x: 0, y: 0 },
      data: {
        label: "No clusters discovered",
        subtitle: "Create a DiscoveryConfig to start",
        nodeType: "cluster",
      },
    })
    edges.push({ id: "e-autodiscovery-no-clusters", source: "autodiscovery", target: "no-clusters", type: "smoothstep" })
    return { nodes, edges }
  }

  // ── Track unique clusters to group environments ───────────────────
  const clusterEnvs: Record<string, {
    cluster: { name: string; provider?: string; zone?: string; region?: string }
    envIds: string[]
    resourceTypes: Set<string>
  }> = {}

  for (const config of configs) {
    for (const env of config.environments) {
      const clusterKey = env.cluster.name
      if (!clusterEnvs[clusterKey]) {
        clusterEnvs[clusterKey] = { cluster: env.cluster, envIds: [], resourceTypes: new Set() }
      }

      const envId = `env-${config.name}-${env.name}`
      clusterEnvs[clusterKey].envIds.push(envId)

      // Collect resource types to determine which runtimes are present
      const rts = env.resourceTypes?.length
        ? env.resourceTypes
        : ["MCPServer", "Agent", "Skill", "ModelConfig"]
      rts.forEach(rt => clusterEnvs[clusterKey].resourceTypes.add(rt))

      // Environment node
      const totalResources =
        env.discoveredResources.mcpServers +
        env.discoveredResources.agents +
        env.discoveredResources.skills +
        env.discoveredResources.models

      nodes.push({
        id: envId,
        type: "map",
        position: { x: 0, y: 0 },
        data: {
          label: env.name,
          subtitle: env.namespaces?.length
            ? `ns: ${env.namespaces.join(", ")}`
            : "all namespaces",
          nodeType: "environment",
          connected: env.connected,
          badge: totalResources > 0 ? `${totalResources} resources` : undefined,
          counts: env.discoveredResources,
        },
      })

      // Resource type nodes under each environment
      for (const rt of rts) {
        const rtId = `${envId}-rt-${rt}`
        nodes.push({
          id: rtId,
          type: "resourceType",
          position: { x: 0, y: 0 },
          data: { label: rt, nodeType: "resourceType" },
        })
        edges.push({
          id: `e-${envId}-${rtId}`,
          source: envId,
          target: rtId,
          type: "smoothstep",
        })
      }
    }
  }

  // ── Create cluster nodes with runtime components ──────────────────
  for (const [clusterName, { cluster, envIds, resourceTypes }] of Object.entries(clusterEnvs)) {
    const clusterId = `cluster-${clusterName}`
    const location = cluster.zone || cluster.region || ""
    const providerLabel = cluster.provider ? cluster.provider.toUpperCase() : ""

    nodes.push({
      id: clusterId,
      type: "map",
      position: { x: 0, y: 0 },
      data: {
        label: clusterName,
        subtitle: [providerLabel, location].filter(Boolean).join(" · "),
        nodeType: "cluster",
        badge: providerLabel || undefined,
      },
    })

    // Auto-Discovery -> Cluster (discovery watches)
    edges.push({
      id: `e-autodiscovery-${clusterId}`,
      source: "autodiscovery",
      target: clusterId,
      type: "smoothstep",
    })

    // Runtime components: kagent and kmcp
    const hasAgentResources = resourceTypes.has("Agent") || resourceTypes.has("Skill") || resourceTypes.has("ModelConfig")
    const hasMCPResources = resourceTypes.has("MCPServer") || resourceTypes.has("RemoteMCPServer")

    if (hasAgentResources) {
      const kagentId = `${clusterId}-kagent`
      nodes.push({
        id: kagentId,
        type: "map",
        position: { x: 0, y: 0 },
        data: {
          label: "kagent",
          subtitle: "Agent runtime operator",
          nodeType: "runtime",
        },
      })
      edges.push({ id: `e-${clusterId}-${kagentId}`, source: clusterId, target: kagentId, type: "smoothstep" })
      // kagent -> environments
      for (const envId of envIds) {
        edges.push({ id: `e-${kagentId}-${envId}`, source: kagentId, target: envId, type: "smoothstep" })
      }
    }

    if (hasMCPResources) {
      const kmcpId = `${clusterId}-kmcp`
      nodes.push({
        id: kmcpId,
        type: "map",
        position: { x: 0, y: 0 },
        data: {
          label: "kmcp",
          subtitle: "MCP server operator",
          nodeType: "runtime",
        },
      })
      edges.push({ id: `e-${clusterId}-${kmcpId}`, source: clusterId, target: kmcpId, type: "smoothstep" })
      // kmcp -> environments (only if no kagent, to avoid double edges)
      if (!hasAgentResources) {
        for (const envId of envIds) {
          edges.push({ id: `e-${kmcpId}-${envId}`, source: kmcpId, target: envId, type: "smoothstep" })
        }
      }
    }

    // If neither runtime matches, connect cluster directly to environments
    if (!hasAgentResources && !hasMCPResources) {
      for (const envId of envIds) {
        edges.push({
          id: `e-${clusterId}-${envId}`,
          source: clusterId,
          target: envId,
          type: "smoothstep",
        })
      }
    }
  }

  return { nodes, edges }
}

// ── Dagre layout ────────────────────────────────────────────────────

function getLayoutedElements(
  nodes: Node<MapNodeData>[],
  edges: Edge[],
): { nodes: Node<MapNodeData>[]; edges: Edge[] } {
  const Dagre = require("@dagrejs/dagre")
  const g = new Dagre.graphlib.Graph().setDefaultEdgeLabel(() => ({}))
  g.setGraph({ rankdir: "TB", nodesep: 80, ranksep: 100 })

  nodes.forEach((node) => {
    const isResourceType = node.type === "resourceType"
    g.setNode(node.id, { width: isResourceType ? 180 : 260, height: isResourceType ? 50 : 70 })
  })
  edges.forEach((edge) => g.setEdge(edge.source, edge.target))

  Dagre.layout(g)

  const layoutedNodes = nodes.map((node) => {
    const pos = g.node(node.id)
    return { ...node, position: { x: pos.x - 110, y: pos.y - 35 } }
  })

  return { nodes: layoutedNodes, edges }
}

// ── Exported component ──────────────────────────────────────────────

export function DiscoveryMapGraph({
  configs,
  resourceCounts,
}: {
  configs: DiscoveryMapConfig[]
  resourceCounts?: ResourceCounts
}) {
  const { nodes: layoutedNodes, edges: layoutedEdges } = useMemo(() => {
    const { nodes, edges } = buildGraphData(configs, resourceCounts)
    return getLayoutedElements(nodes, edges)
  }, [configs, resourceCounts])

  // Track drag overrides separately to avoid calling setState in an effect
  const [dragOverrides, setDragOverrides] = useState<Record<string, { x: number; y: number }>>({})

  const nodes = useMemo(
    () => layoutedNodes.map((n) => (dragOverrides[n.id] ? { ...n, position: dragOverrides[n.id] } : n)),
    [layoutedNodes, dragOverrides],
  )

  const onNodesChange = useCallback((changes: Parameters<Exclude<React.ComponentProps<typeof ReactFlow>["onNodesChange"], undefined>>[0]) => {
    const posUpdates: Record<string, { x: number; y: number }> = {}
    for (const change of changes) {
      if (change.type === "position" && change.position) {
        posUpdates[change.id] = change.position
      }
    }
    if (Object.keys(posUpdates).length > 0) {
      setDragOverrides((prev) => ({ ...prev, ...posUpdates }))
    }
  }, [])

  return (
    <div style={{ width: "100%", height: "calc(100vh - 100px)" }} className="rounded-lg border bg-card">
      <ReactFlow
        nodes={nodes}
        edges={layoutedEdges}
        onNodesChange={onNodesChange}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.15 }}
        minZoom={0.3}
        maxZoom={2}
        nodesConnectable={false}
        proOptions={{ hideAttribution: false }}
      >
        <Background gap={16} size={1} color="rgba(128,128,128,0.3)" />
      </ReactFlow>
    </div>
  )
}
