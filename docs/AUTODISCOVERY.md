# Multi-Cluster Autodiscovery

The Agent Registry controller supports automatic discovery of resources across multiple Kubernetes clusters using the `DiscoveryConfig` CRD.

## Overview

Autodiscovery allows a centralized Agent Registry controller to:

- **Monitor multiple clusters** - Discover resources across dev, staging, and production clusters
- **Automatic catalog creation** - Automatically create catalog entries for discovered resources
- **Workload Identity support** - Use cloud provider workload identity for authentication
- **Namespace filtering** - Discover resources in specific namespaces or all namespaces
- **Resource type filtering** - Choose which resource types to discover
- **Custom labeling** - Apply custom labels to discovered catalog entries

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Agent Registry Controller (Central Cluster)                │
│                                                              │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ DiscoveryConfig Reconciler                            │  │
│  │  - Connects to remote clusters                        │  │
│  │  - Discovers resources (MCPServer, Agent, ModelConfig)│  │
│  │  - Creates catalog entries                            │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ Catalog Storage (CRDs)                                │  │
│  │  - MCPServerCatalog                                   │  │
│  │  - AgentCatalog                                       │  │
│  │  - ModelCatalog                                       │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                    │           │           │
        ┌───────────┘           │           └───────────┐
        │                       │                       │
        ▼                       ▼                       ▼
┌───────────────┐      ┌───────────────┐      ┌───────────────┐
│ Dev Cluster   │      │ Staging       │      │ Production    │
│               │      │ Cluster       │      │ Cluster       │
│ MCPServers    │      │ MCPServers    │      │ MCPServers    │
│ Agents        │      │ Agents        │      │ Agents        │
│ ModelConfigs  │      │ ModelConfigs  │      │ ModelConfigs  │
└───────────────┘      └───────────────┘      └───────────────┘
```

## Configuration

### DiscoveryConfig CRD

The `DiscoveryConfig` CRD defines which clusters and namespaces to discover resources from:

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
        namespace: ed210
        projectId: gfk-eco-dev-green
        zone: europe-west3
        useWorkloadIdentity: true
      provider: gcp
      registry:
        url: europe-docker.pkg.dev/gfk-eco-shared-blue
        prefix: newron-dev
        useWorkloadIdentity: true
      discoveryEnabled: true
      namespaces:
        - ed210
        - default
      resourceTypes:
        - MCPServer
        - Agent
        - ModelConfig
      labels:
        environment: dev
        managed-by: agentregistry

    - name: prod
      cluster:
        name: prod
        namespace: production
        projectId: gcp-demo-project
        zone: europe-west3
        useWorkloadIdentity: true
      provider: gcp
      discoveryEnabled: true
      namespaces:
        - production
      resourceTypes:
        - MCPServer
        - Agent
      labels:
        environment: prod
        managed-by: agentregistry
```

### Field Reference

#### Environment

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique identifier for this environment |
| `cluster` | ClusterConfig | Yes | Cluster connection information |
| `provider` | string | No | Cloud provider (gcp, aws, azure) |
| `registry` | RegistryConfig | No | Container registry information |
| `discoveryEnabled` | bool | No | Enable/disable discovery (default: true) |
| `namespaces` | []string | No | Namespaces to discover in (empty = all) |
| `resourceTypes` | []string | No | Resource types to discover (empty = all) |
| `labels` | map[string]string | No | Labels to apply to discovered resources |

#### ClusterConfig

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Cluster name |
| `namespace` | string | No | Default namespace |
| `projectId` | string | No | GCP project ID (for workload identity) |
| `zone` | string | No | Cluster zone |
| `region` | string | No | Cluster region (alternative to zone) |
| `endpoint` | string | No | Cluster API endpoint (auto-discovered if not provided) |
| `caData` | string | No | Base64-encoded CA certificate |
| `useWorkloadIdentity` | bool | No | Use workload identity (default: true) |
| `serviceAccount` | string | No | Service account for workload identity |

#### RegistryConfig

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | Yes | Registry URL |
| `prefix` | string | No | Image path prefix |
| `useWorkloadIdentity` | bool | No | Use workload identity (default: true) |

## Authentication Methods

### Workload Identity (Recommended)

For GCP GKE clusters, use workload identity for secure, credential-less authentication:

```yaml
cluster:
  name: prod
  projectId: my-project-id
  zone: us-central1-a
  useWorkloadIdentity: true
  serviceAccount: agentregistry@my-project-id.iam.gserviceaccount.com
```

**Prerequisites:**
1. Enable Workload Identity on the GKE cluster where the controller runs
2. Grant the service account `container.clusters.get` permission
3. Bind the Kubernetes service account to the GCP service account

### Static Credentials

For clusters without workload identity, provide explicit credentials:

```yaml
cluster:
  name: external-cluster
  endpoint: https://1.2.3.4
  caData: LS0tLS1CRUdJTi... # base64-encoded CA cert
  useWorkloadIdentity: false
```

## Resource Discovery

### Supported Resource Types

| Resource Type | CRD | Catalog Type | Status |
|--------------|-----|--------------|--------|
| `MCPServer` | `kmcp.agentregistry.dev/v1alpha1` | MCPServerCatalog | ✅ Implemented |
| `Agent` | `kagent.dev/v1alpha2` | AgentCatalog | 🚧 Partial |
| `ModelConfig` | `kagent.dev/v1alpha2` | ModelCatalog | 🚧 Partial |
| `Skill` | Extracted from Agents | SkillCatalog | ⏱️ Planned |

### Discovery Process

1. **Connect** - Controller establishes connection to remote cluster using workload identity or provided credentials
2. **List Resources** - Query for resources in configured namespaces
3. **Create Catalogs** - Create or update catalog entries in the central cluster
4. **Label** - Apply environment and discovery labels
5. **Status Update** - Update DiscoveryConfig status with counts and connection state
6. **Repeat** - Re-sync every 5 minutes

### Catalog Naming

Discovered catalog entries are named using the pattern:

```
{environment}-{namespace}-{resource-name}
```

Example: `dev-ed210-filesystem-mcp-server`

### Catalog Labels

All discovered catalog entries receive these labels:

| Label | Description | Example |
|-------|-------------|---------|
| `agentregistry.dev/discovered` | Marks as discovered | `true` |
| `agentregistry.dev/source-kind` | Source resource type | `MCPServer` |
| `agentregistry.dev/source-name` | Source resource name | `filesystem-mcp-server` |
| `agentregistry.dev/source-namespace` | Source namespace | `ed210` |
| `agentregistry.dev/environment` | Environment name | `dev` |
| `agentregistry.dev/cluster` | Cluster name | `dev` |
| Custom labels | From environment config | `managed-by: agentregistry` |

## Deployment

### Prerequisites

1. **Controller Deployment** - Agent Registry controller running in a central cluster
2. **Network Access** - Network connectivity from controller to remote clusters
3. **RBAC Permissions** - Controller needs permissions to create catalog CRDs
4. **Workload Identity** (GCP) - Service account with appropriate permissions

### GCP Workload Identity Setup

```bash
# 1. Create GCP service account
gcloud iam service-accounts create agentregistry-discovery \
  --project=my-project-id

# 2. Grant container.clusters.get permission
gcloud projects add-iam-policy-binding my-project-id \
  --member="serviceAccount:agentregistry-discovery@my-project-id.iam.gserviceaccount.com" \
  --role="roles/container.clusterViewer"

# 3. Bind Kubernetes service account
gcloud iam service-accounts add-iam-policy-binding \
  agentregistry-discovery@my-project-id.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:my-project-id.svc.id.goog[agentregistry/agentregistry-controller]"

# 4. Annotate Kubernetes service account
kubectl annotate serviceaccount agentregistry-controller \
  -n agentregistry \
  iam.gke.io/gcp-service-account=agentregistry-discovery@my-project-id.iam.gserviceaccount.com
```

### Apply DiscoveryConfig

```bash
# Apply the configuration
kubectl apply -f config/samples/discoveryconfig_example.yaml

# Check status
kubectl get discoveryconfig multi-cluster-discovery -o yaml

# View discovered catalogs
kubectl get mcpservercatalog -l agentregistry.dev/discovered=true
```

## Monitoring

### Check DiscoveryConfig Status

```bash
kubectl get discoveryconfig multi-cluster-discovery -o jsonpath='{.status}' | jq
```

Example output:

```json
{
  "conditions": [
    {
      "lastTransitionTime": "2025-02-02T12:00:00Z",
      "message": "All environments are connected",
      "reason": "AllEnvironmentsConnected",
      "status": "True",
      "type": "Connected"
    }
  ],
  "environments": [
    {
      "name": "dev",
      "connected": true,
      "message": "Connected",
      "lastSyncTime": "2025-02-02T12:00:00Z",
      "discoveredResources": {
        "mcpServers": 5,
        "agents": 3,
        "models": 2
      }
    }
  ],
  "lastSyncTime": "2025-02-02T12:00:00Z"
}
```

### View Discovered Resources

```bash
# List all discovered MCPServer catalogs
kubectl get mcpservercatalog -l agentregistry.dev/discovered=true

# List catalogs from specific environment
kubectl get mcpservercatalog -l agentregistry.dev/environment=dev

# List catalogs from specific cluster
kubectl get mcpservercatalog -l agentregistry.dev/cluster=prod
```

### Controller Logs

```bash
# View discovery logs
kubectl logs -n agentregistry -l app=agentregistry-controller \
  | grep "controller=discoveryconfig"
```

## Troubleshooting

### Connection Failures

**Symptom:** Environment shows `connected: false`

**Solutions:**
1. Check workload identity configuration
2. Verify network connectivity to remote cluster
3. Ensure service account has necessary permissions
4. Check controller logs for detailed errors

```bash
kubectl logs -n agentregistry -l app=agentregistry-controller | grep ERROR
```

### Missing Resources

**Symptom:** Expected resources not appearing in catalog

**Checks:**
1. Verify resource exists in remote cluster
2. Check namespace filter configuration
3. Verify resourceTypes includes the resource type
4. Check RBAC permissions in remote cluster

### Stale Catalog Entries

**Symptom:** Catalog entries for deleted resources

**Solution:** Catalog entries are preserved when source resources are deleted. To clean up:

```bash
# Delete specific catalog entry
kubectl delete mcpservercatalog dev-ed210-old-server

# Delete all catalogs from an environment
kubectl delete mcpservercatalog -l agentregistry.dev/environment=dev
```

## Limitations

### Current Limitations

1. **GKE Endpoint Discovery** - Automatic GKE cluster endpoint discovery not yet implemented. Must provide `cluster.endpoint` explicitly.
2. **Agent/Model Discovery** - Agent and ModelConfig catalog creation is partial implementation
3. **Cross-Cloud** - Only GCP workload identity is implemented. AWS/Azure support planned.
4. **Deletion Sync** - Deleted resources don't automatically remove catalog entries (by design for history)

### Planned Enhancements

- [ ] Automatic GKE cluster endpoint discovery using GCP Container API
- [ ] AWS IAM Roles for Service Accounts (IRSA) support
- [ ] Azure AD Workload Identity support
- [ ] Agent and ModelConfig catalog creation
- [ ] Configurable sync interval
- [ ] Webhook for immediate sync on resource changes
- [ ] Metrics and alerting integration

## Security Considerations

1. **Least Privilege** - Grant only necessary permissions to the discovery service account
2. **Network Policies** - Use network policies to restrict controller access
3. **Credential Rotation** - Use workload identity to avoid credential management
4. **Audit Logging** - Enable audit logs for discovery operations
5. **Resource Isolation** - Consider separate DiscoveryConfig per environment for isolation

## Examples

See [config/samples/discoveryconfig_example.yaml](../config/samples/discoveryconfig_example.yaml) for a complete example configuration.

## API Reference

For detailed API specifications, see:
- [DiscoveryConfig CRD](../api/v1alpha1/discoveryconfig_types.go)
- [Controller Implementation](../internal/controller/discoveryconfig_controller.go)
