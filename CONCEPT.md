# Your AI Infrastructure Is Sprawling. You Just Don't Know It Yet.

[![README](https://img.shields.io/badge/README-docs-blue)](README.md)

> Built at the [AI + MCP Agents Hackathon](https://aihackathon.dev). Inspired by the [solo.io](https://solo.io) team and their work on kagent, kgateway, kmcp and agent registry — pushing us to make AI infrastructure safe and reliable.

![Agent Inventory](docs/img/inventory.png)

Every enterprise building with AI today has the same invisible problem. Agents are scattered across clusters. MCP servers are deployed by different teams who don't talk to each other. Models are running in three environments, and nobody can tell you which version is in production. There's no central view of what exists, what's available for reuse, or what depends on what.

We know because we lived it. Running kagent across multiple clusters, deploying MCP servers alongside kgateway, managing model configs in different environments — at some point we realized we'd lost track of what was running where. The tooling we needed didn't exist, so we built it.

It's the same sprawl we saw with microservices five years ago — except now the components are smarter, more autonomous, and harder to track.

## The Inventory Problem

Think about how your organization manages AI resources today. Someone on the ML platform team deploys an MCP server for filesystem access. A product team builds an agent that uses it. Another team builds a different agent that needs the same server but doesn't know it exists, so they deploy their own. Meanwhile, a third team has a skill that would save both of them a week of work — sitting in a namespace nobody looks at.

This isn't a tooling problem. It's a visibility problem. And it compounds fast.

The moment you have more than one cluster, more than one team, or more than one environment — you've lost the plot. Nobody knows what's running where. Nobody knows what's safe to remove. Nobody knows what's available to reuse.

## What If Your Catalog Built Itself?

Agent Inventory takes a different approach: **if it's running, it's in the catalog.**

No publish step. No registration forms. No separate marketplace that drifts from reality. The inventory watches your clusters and automatically indexes every MCP server, agent, skill, and model it finds. You deploy something — it appears in the catalog. You remove it — it disappears.

```
┌──────────────────────────────────────┐
│  Agent Inventory                     │  discovery + catalog + UI
├────────────┬───────────┬─────────────┤
│   kagent   │ kgateway  │    llm-d    │  AI runtime
├────────────┴───────────┴─────────────┤
│           Kubernetes                 │  infrastructure
└──────────────────────────────────────┘
```

This is the inventory layer — it sits on top of whatever AI runtime stack you're already running. It doesn't replace kagent, kgateway, or llm-d. It indexes them.

## Why This Architecture

We started with inspiration from Agent Registry by solo.io, a clean CLI for browsing agent registries. Great foundation — but enterprise reality pushed us further. And in some cases, in a different direction entirely. Instead of a CLI, we leaned into MCP prompts and skills, so agents could talk to the registry themselves. Instead of semantic search, we explored agentic search, where the AI reasons over the catalog rather than matching embeddings. Instead of a database, we found that Kubernetes informer caches already had everything we needed. And instead of a publish step — well, auto-discovery turned out to be the better paradigm. Let the cluster be the source of truth.

Three design decisions came out of this.

**CRDs are the source of truth.** There's no database. Every catalog entry is a Kubernetes Custom Resource. This means your entire AI inventory is declarative, versionable, and manageable with the same GitOps workflows you already use for infrastructure. `kubectl get mcpservercatalog` shows you everything.

**Discovery is automatic.** A DiscoveryConfig resource tells the controller which namespaces and clusters to watch. It uses workload identity federation to reach across cluster boundaries — no shared credentials, no VPN tunnels. Point it at your prod GKE cluster, your staging EKS cluster, and your dev kind cluster. It indexes all of them.

**The controller is a single binary.** Reconcilers, HTTP API, web UI, and an embedded MCP server — all in one process. 

For local development or even cli-like option no cluster required for development - `make dev` starts everything locally with an embedded etcd and kube-apiserver. The full catalog is available at `localhost:3000` in seconds.

## What You Actually Get

### Governance without overhead

All lifecycle management flows through Git. The audit trail is Git history — who changed what, when, and why. Approval workflows are pull request reviews. Supply chain trust is signed commits and image attestation. The deploy button in the UI is gated by OIDC authentication with group-based access.

You don't need a separate governance layer. You already have one. It's called Git.

### Dependencies you can see

An agent's dependencies are already declared in its runtime spec — which MCP servers it needs, which skills it uses, which model it talks to. Agent Inventory surfaces this from the catalog so you can navigate the graph: click an agent, see its MCP servers. Click an MCP server, see which agents depend on it.

This is impact analysis for free. Before you remove a resource, you know exactly what breaks.

### Search that reflects reality

Filtering is simple and fast — no database, no Elasticsearch. Filter by resource type, category, deployment status, verified publisher. Every filter operates on live catalog data that auto-updates as the cluster state changes.

When someone asks "what MCP servers are running in production?" the answer is one filter away. Not a Slack thread.

### Reuse you can measure

Usage statistics come from the observability layer — trace-based metrics per agent and MCP server, call counts, latency, error rates. Resource popularity becomes a trust signal. When a skill says "used by 12 agents across 4 teams," that's a reason to reuse it instead of building your own.

### Multi-tenancy that already works

OIDC authentication with group-based access control. Namespace scoping so teams see resources in their authorized namespaces. Environment isolation through DiscoveryConfig — separate dev, staging, and prod catalogs that reflect what's actually running in each environment.

## The MCP Server Inside

Here's the part that changes how you interact with the registry. Agent Inventory embeds an MCP server directly in the controller. Connect Claude Code, Cursor, or any MCP-compatible client and manage your entire inventory conversationally.

Browse the catalog. Deploy a server. Check dependencies. Analyze an agent's dependency tree. Generate a deployment plan. All through natural language, all hitting the same live catalog data.

No CLI to install. No commands to memorize. Just connect and talk to your infrastructure.

## Who This Is For

**Platform teams** who need a single view of all AI resources across clusters — without maintaining a separate catalog by hand.

**Developers** who want to find and reuse existing agents, MCP servers, and skills — without asking on Slack.

**Security teams** who need an audit trail, verified publisher identity, and namespace-based isolation — without bolting on another tool.

## Try It

```bash
git clone https://github.com/den-vasyliev/agentregistry-inventory.git
cd agentregistry-inventory && make dev
```

That's it. UI opens at `localhost:3000` with sample data pre-loaded. No Kubernetes cluster needed.

If you're running AI workloads on Kubernetes and you don't know exactly what's deployed where — Invetrory this is the missing piece.
