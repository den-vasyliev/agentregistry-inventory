# Multi-Cluster Autodiscovery

Agent Registry automatically discovers resources across multiple Kubernetes clusters using the `DiscoveryConfig` CRD.

## Features

- Monitor multiple clusters (dev, staging, prod)
- Auto-create catalog entries for discovered resources
- Workload identity authentication (GCP)
- Namespace and resource type filtering
- Custom labels on discovered resources

## Quick Start

1. **Apply DiscoveryConfig:**
   ```bash
   kubectl apply -f config/samples/discoveryconfig_example.yaml
   ```

2. **View discovered resources:**
   ```bash
   kubectl get mcpservercatalog -l agentregistry.dev/discovered=true
   ```

## Architecture

```
┌─────────────────────────────────────────┐
│  Agent Registry Controller              │
│  ┌─────────────────────────────────┐   │
│  │ DiscoveryConfig Reconciler      │   │
│  │  - Connects to remote clusters  │   │
│  │  - Discovers resources          │   │
│  │  - Creates catalog entries      │   │
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘
           │           │           │
           ▼           ▼           ▼
    ┌──────────┐ ┌──────────┐ ┌──────────┐
    │   Dev    │ │ Staging  │ │   Prod   │
    │ Cluster  │ │ Cluster  │ │ Cluster  │
    └──────────┘ └──────────┘ └──────────┘
```

## Configuration Example

```yaml
apiVersion: agentregistry.dev/v1alpha1
kind: DiscoveryConfig
metadata:
  name: multi-cluster-discovery
spec:
  environments:
    - name: dev
      cluster:
        name: dev
        projectId: my-project
        zone: us-central1-a
        useWorkloadIdentity: true
      provider: gcp
      discoveryEnabled: true
      namespaces: [default, dev]
      resourceTypes: [MCPServer, Agent, ModelConfig]
      labels:
        environment: dev
```

### Key Fields

- **environments**: List of clusters to discover from
- **cluster**: GCP cluster info (name, projectId, zone)
- **namespaces**: Namespaces to scan (empty = all)
- **resourceTypes**: Resource types to discover
- **labels**: Custom labels for catalog entries

## Setup (GKE Workload Identity)

**1. Grant GKE permissions:**
```bash
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:GSA@PROJECT.iam.gserviceaccount.com" \
  --role="roles/container.clusterViewer"
```

**2. Bind workload identity:**
```bash
gcloud iam service-accounts add-iam-policy-binding GSA@PROJECT.iam.gserviceaccount.com \
  --role=roles/iam.workloadIdentityUser \
  --member="serviceAccount:PROJECT.svc.id.goog[agentregistry/agentregistry-inventory]"
```

**3. Annotate K8s service account:**
```bash
kubectl annotate serviceaccount agentregistry-inventory -n agentregistry \
  iam.gke.io/gcp-service-account=GSA@PROJECT.iam.gserviceaccount.com
```

**4. Configure RBAC on remote clusters:**
```bash
# Apply on each remote cluster to allow discovery
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: agentregistry-discovery
rules:
- apiGroups: ["kagent.dev"]
  resources: [modelconfigs, agents, remotemcpservers, mcpservers]
  verbs: ["get", "list", "watch"]
- apiGroups: ["kmcp.agentregistry.dev"]
  resources: [mcpservers]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: agentregistry-discovery-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: agentregistry-discovery
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: User
  name: GSA@PROJECT.iam.gserviceaccount.com
EOF
```

## How It Works

1. Controller connects to remote clusters via workload identity
2. Lists resources (MCPServer, Agent, ModelConfig) in configured namespaces
3. Creates catalog entries with labels: `agentregistry.dev/discovered=true`, `agentregistry.dev/environment`, etc.
4. Re-syncs every 5 minutes

Catalog naming: `{environment}-{namespace}-{resource-name}` (e.g., `dev-default-filesystem-mcp`)

## Reference

- [Example config](../config/samples/discoveryconfig_example.yaml)
- [DiscoveryConfig CRD](../api/v1alpha1/discoveryconfig_types.go)
- [Controller implementation](../internal/controller/discoveryconfig_controller.go)
