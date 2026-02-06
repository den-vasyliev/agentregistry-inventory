# ADR-001: Inventory-Over-Runtime Architecture for Agentic Infrastructure

- **Status**: Accepted
- **Date**: 2025-02-06
- **Authors**: Agent Registry team

## Context

Enterprise teams building with AI face a fragmented landscape: agents scattered across clusters, MCP servers deployed by different teams, models running in different environments. There is no central view of what exists, what's available for reuse, or how resources relate to each other.

The AI runtime stack already has capable components for execution:
- **kagent** for agent orchestration
- **kgateway** for MCP server management
- **llm-d** for model serving

What's missing is an **inventory layer** that indexes everything running across these runtimes into a single, searchable catalog.

## Decision

Agent Registry is an **inventory layer** that sits on top of the AI runtime stack, not a replacement for any runtime component.

```
+--------------------------------------+
|  Agent Registry (Inventory)          |  discovery + catalog + UI
+------------+-----------+-------------+
|   kagent   | kgateway  |    llm-d    |  AI runtime
+------------+-----------+-------------+
|           Kubernetes                 |  infrastructure
+--------------------------------------+
```

### Core Principle: If It's Running, It's in the Catalog

There is no publish step, no separate marketplace, no manual registration. Agent Registry auto-discovers resources from the runtime layer and presents them in a unified catalog.

### Sub-Decisions

#### 1. Zero-Config Discovery via Kubernetes CRDs

Auto-index resources from kagent, kgateway, and llm-d by watching their CRDs. DiscoveryConfig resources define which namespaces and clusters to watch. Discovered resources are stored as catalog CRDs (MCPServerCatalog, AgentCatalog, SkillCatalog) in the `agentregistry` namespace.

**Multi-cluster support** uses workload identity federation - no shared secrets or service account keys. Each cluster's DiscoveryConfig specifies target clusters with their GKE/EKS workload identity configuration.

#### 2. Governance and Compliance via GitOps

All lifecycle management, governance, and compliance is handled through GitOps workflows rather than built into the registry:

- **Audit trail** - Git history provides immutable record of changes
- **Approval workflows** - PR reviews and branch protection for deployment changes
- **Supply chain trust** - signed commits, image attestation, and policy-as-code in the Git pipeline
- **CI/CD integration** - test, evaluate, and register agents automatically on merge
- **Deploy authorization** - deploy actions in UI gated by OIDC authentication with group-based access

This avoids building a governance system inside the registry. Git is the governance system.

#### 3. Dependency Graph from Runtime Data

Agent dependencies are already defined in the runtime layer (kagent Agent CRs include skills, tools, and MCP server references). Agent Registry surfaces this existing data rather than maintaining a separate dependency model:

- Agent detail view shows linked skills, tools, and MCP servers
- Impact analysis: understand what breaks if a resource is removed
- Cross-resource navigation between related catalog entries

No separate graph database or dependency tracking - the runtime CRDs are the source of truth.

#### 4. Discovery and Search Without a Database

Filtering and search operate on the Kubernetes informer cache. No SQL database, no Elasticsearch - the controller's in-memory cache of CRDs is the query engine:

- Category filters by resource type, package type, framework, agent type
- Verified identity filtering (organization and publisher verification)
- Deployment status filtering (running, external, not deployed, failed)
- Tag-based search via Kubernetes labels

#### 5. Reuse Metrics from Observability Stack

Usage statistics are pulled from the observability layer (Arize Phoenix) rather than built into the registry:

- Trace-based metrics per agent and MCP server (call count, latency, error rate)
- Resource popularity as a trust signal ("used by N agents across M teams")
- Cost visibility per model from llm-d metrics

This avoids duplicating telemetry infrastructure inside the registry.

#### 6. Multi-Tenancy via Kubernetes Primitives

Access control uses existing Kubernetes and OIDC infrastructure:

- **OIDC authentication** with group-based access control
- **Namespace scoping** - teams see resources in their authorized namespaces
- **Environment isolation** - separate dev, staging, prod catalogs via DiscoveryConfig

No custom RBAC system - Kubernetes namespaces and OIDC groups are the tenancy boundary.

## Alternatives Considered

### Standalone Registry with Database

A registry backed by PostgreSQL/SQLite with its own data model, import/export workflows, and publish step.

**Rejected because:**
- Requires manual registration or sync jobs to keep data current
- Introduces data consistency problems between registry and runtime state
- Adds operational burden (database backups, migrations, connection pooling)
- Duplicates data already available in Kubernetes CRDs

### Marketplace Model (Publish/Subscribe)

A curated marketplace where teams explicitly publish resources for others to discover and consume.

**Rejected because:**
- Publish step creates friction and staleness (resources exist in runtime but not in catalog)
- Curation overhead doesn't scale with fast-moving agentic infrastructure
- "Running = available" is a better model for enterprise internal resources than "published = available"

### Hub-and-Spoke Federation

A central registry that aggregates data from per-cluster registries, each with their own database.

**Rejected because:**
- Multiplies operational complexity (N databases, N sync jobs)
- Cross-cluster queries require federation protocol
- Workload identity + DiscoveryConfig achieves multi-cluster visibility without per-cluster registries

## Consequences

### Positive

- **Zero operational overhead for catalog maintenance** - discovery is automatic
- **Single binary deployment** - controller, HTTP API, and UI in one process
- **No data consistency drift** - CRDs are both the runtime truth and the catalog
- **GitOps-native** - all governance through existing Git workflows
- **Runtime-agnostic** - works with kagent, kgateway, llm-d, and extensible to new runtimes

### Negative

- **Query capabilities limited by Kubernetes API** - no full-text search, no complex joins, no aggregation queries beyond what label selectors and informer cache provide
- **Scaling bound to etcd** - catalog size limited by etcd storage (practical limit ~10K resources per type)
- **No offline catalog** - if the cluster is unreachable, the catalog is unavailable
- **Discovery latency** - new resources appear in catalog after reconciliation cycle (seconds to minutes), not instantly

### Risks

- **etcd pressure** - large catalogs with frequent updates could stress etcd; mitigated by informer cache for reads
- **CRD schema evolution** - changes to runtime CRDs (kagent, kgateway) require corresponding updates to catalog CRDs and discovery logic
