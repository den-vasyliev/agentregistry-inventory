# Agent Registry Submission Workflow

This document describes the complete workflow for submitting AI resources to the Agent Registry, from development submission through SRE review and automatic deployment.

## Overview

The workflow consists of three main phases:

1. **Developer Submission** - UI generates manifest for PR on monorepo
2. **SRE Review & CICD** - PR review, approval, and automated submission to registry
3. **Auto-Discovery & Deployment** - Registry controller discovers and optionally deploys resources

## Phase 1: Developer Submission

### Using the UI

1. **Access the Registry UI** at your Agent Registry deployment URL
2. **Click "Submit Resource"** to open the submission dialog
3. **Fill in resource details**:
   - Resource type (MCP Server, Agent, Skill, Model)
   - Name, version, description
   - Package/container information
   - Configuration requirements

4. **Generate Manifest** - Review the generated YAML manifest
5. **Open PR** - Click "Open PR on GitHub" to create the manifest file

### Manifest Structure

Resources are organized in the monorepo as:
```
resources/
├── mcp-server/
│   └── filesystem-tools/
│       └── filesystem-tools-1.0.0.yaml
├── agent/
│   └── code-reviewer/
│       └── code-reviewer-2.1.0.yaml
├── skill/
│   └── terraform-deploy/
│       └── terraform-deploy-1.5.0.yaml
└── model/
    └── claude-sonnet/
        └── claude-sonnet-3.5.yaml
```

### Example Manifest

```yaml
apiVersion: agentregistry.dev/v1alpha1
kind: MCPServerCatalog
metadata:
  name: filesystem-tools-1-0-0
  labels:
    agentregistry.dev/resource-name: filesystem-tools
    agentregistry.dev/resource-version: 1.0.0
    agentregistry.dev/resource-environment: production
    agentregistry.dev/auto-deploy: "true"  # Enable auto-deployment
spec:
  name: filesystem-tools
  version: 1.0.0
  title: Filesystem Tools MCP Server
  description: Provides file system operations for AI agents
  packages:
    - type: oci
      image: ghcr.io/your-org/mcp-filesystem:1.0.0
      transport:
        type: stdio
  repository:
    url: https://github.com/your-org/mcp-filesystem-tools
    source: github
  config:
    required:
      - name: ALLOWED_PATHS
        description: Comma-separated list of allowed paths
    optional:
      - name: MAX_FILE_SIZE
        description: Maximum file size in MB
        default: "10"
```

## Phase 2: SRE Review & CI/CD

### PR Review Process

1. **Automated Validation**: GitHub Actions validates manifest syntax and structure
2. **SRE Review**: Team reviews resource requirements and security implications
3. **Approval**: SRE approves PR for submission to registry
4. **Auto-Submission**: On merge, CI/CD automatically submits to registry

### GitHub Actions Workflow

The workflow handles:
- **Manifest Validation** - YAML syntax and required fields
- **Resource Detection** - Identifies changed resources
- **Registry Submission** - Calls webhook endpoint
- **OCI Artifact Creation** - Packages resource config
- **Auto-Approval** - For trusted automation sources

### Webhook Integration

When PR is merged, GitHub Actions calls:
```bash
POST /webhooks/github
```

With payload containing:
- Repository information
- Commit details
- Changed files
- Pusher information

### API Endpoints

#### Registry Submission
```bash
POST /admin/v0/submit
{
  "repositoryUrl": "https://github.com/your-org/monorepo"
}
```

#### OCI Artifact Creation
```bash
POST /admin/v0/oci/artifacts
{
  "resourceName": "filesystem-tools",
  "version": "1.0.0",
  "resourceType": "mcp-server",
  "environment": "production"
}
```

#### Webhook Processing
```bash
POST /webhooks/github
{
  "ref": "refs/heads/main",
  "repository": {...},
  "head_commit": {...},
  "pusher": {...}
}
```

## Phase 3: Auto-Discovery & Deployment

### Registry Controller Behavior

1. **Catalog Creation**: Webhook handler creates catalog entries
2. **Discovery**: Controller automatically indexes new resources
3. **Auto-Deployment Evaluation**: Checks for auto-deploy labels
4. **Deployment Creation**: Creates RegistryDeployment resources
5. **Runtime Deployment**: Deploys to Kubernetes clusters

### Auto-Deployment Criteria

Resources are auto-deployed if they meet ALL criteria:
- ✅ Label `agentregistry.dev/auto-deploy: "true"`
- ✅ No existing deployment for same name/version
- ✅ Has deployable packages or configuration
- ✅ Valid target environment

### Environment Configuration

Target environments are determined by:
1. `agentregistry.dev/target-environment` label
2. Environment-specific DiscoveryConfig resources
3. Default to "development" if not specified

### Deployment Labels

Auto-created deployments include:
```yaml
metadata:
  labels:
    agentregistry.dev/auto-created: "true"
    agentregistry.dev/runtime: "kubernetes"
    agentregistry.dev/source: "webhook"
  annotations:
    agentregistry.dev/created-by: "auto-deployment-controller"
    agentregistry.dev/catalog-source: "filesystem-tools-1-0-0"
```

## Configuration Options

### Environment Variables

Registry controller:
```bash
REGISTRY_API_URL=https://registry.your-domain.com
REGISTRY_API_TOKEN=your-api-token
OCI_REGISTRY=ghcr.io/your-org/agent-registry
```

### Labels for Resource Control

| Label | Description | Values |
|-------|-------------|---------|
| `agentregistry.dev/auto-deploy` | Enable auto-deployment | `"true"`, `"false"` |
| `agentregistry.dev/target-environment` | Target deployment environment | `"development"`, `"staging"`, `"production"` |
| `agentregistry.dev/managed-by` | Management system | `"registry"`, `"manual"` |
| `agentregistry.dev/verified-publisher` | Verified publisher status | `"true"`, `"false"` |

### Annotations for Metadata

| Annotation | Description |
|------------|-------------|
| `agentregistry.dev/repository` | Source repository URL |
| `agentregistry.dev/commit` | Source commit SHA |
| `agentregistry.dev/pusher` | User who pushed changes |
| `agentregistry.dev/pr-number` | Source PR number |

## Security Considerations

### Authentication

- **API Tokens**: Stored in Kubernetes secrets
- **OIDC Integration**: For web UI access
- **GitHub Tokens**: For repository access

### Access Control

- **Namespace Isolation**: Resources scoped to authorized namespaces
- **RBAC**: Kubernetes role-based access control
- **Group-based Authorization**: OIDC group membership

### Verification

- **Signed Commits**: Git commit verification
- **Image Attestation**: OCI artifact signatures
- **Publisher Verification**: Verified publisher labels

## Monitoring and Observability

### Metrics

- Resource submission rates
- Deployment success/failure rates
- Discovery latency
- API request metrics

### Logging

- Structured logging with correlation IDs
- Audit trails for resource changes
- Deployment event logging

### Alerting

- Failed submissions
- Deployment failures
- Discovery errors
- API availability

## Troubleshooting

### Common Issues

1. **Manifest Validation Failures**
   - Check YAML syntax
   - Verify required fields
   - Validate resource kind

2. **Registry Submission Errors**
   - Check webhook endpoint availability
   - Verify API token
   - Review network connectivity

3. **Auto-Deployment Not Triggered**
   - Check `auto-deploy` label
   - Verify no existing deployment
   - Review deployment criteria

4. **OCI Artifact Creation Fails**
   - Check registry credentials
   - Verify registry URL
   - Review artifact size limits

### Debug Commands

```bash
# Check catalog entries
kubectl get mcpservercatalog,agentcatalog,skillcatalog,modelcatalog -n agentregistry

# Check auto-deployments
kubectl get registrydeployment -n agentregistry -l agentregistry.dev/auto-created=true

# Review controller logs
kubectl logs -n agentregistry -l app=agentregistry-controller

# Test webhook endpoint
curl -X POST https://registry.your-domain.com/webhooks/github -H "Content-Type: application/json" -d '{...}'
```

## Best Practices

### Resource Organization

- Use semantic versioning
- Organize by resource type
- Include comprehensive descriptions
- Tag with appropriate labels

### Security

- Enable auto-deploy only for trusted resources
- Use verified publisher labels
- Implement proper RBAC
- Regularly rotate API tokens

### Operations

- Monitor deployment success rates
- Set up alerting for failures
- Regular cleanup of old versions
- Document resource dependencies

## API Reference

### Webhook Endpoints

- `POST /webhooks/github` - GitHub webhook integration
- `POST /webhooks/gitlab` - GitLab webhook integration (future)

### Admin Endpoints

- `POST /admin/v0/submit` - Manual resource submission
- `POST /admin/v0/oci/artifacts` - Create OCI artifacts
- `GET /admin/v0/stats` - Registry statistics

### Public Endpoints

- `GET /v0/servers` - List MCP servers
- `GET /v0/agents` - List agents
- `GET /v0/skills` - List skills
- `GET /v0/models` - List models
- `GET /v0/deployments` - List deployments

## Next Steps

1. **Configure Registry** - Set up Agent Registry in your cluster
2. **Set up Monorepo** - Create resources directory structure
3. **Configure CI/CD** - Add GitHub Actions workflow
4. **Test Workflow** - Submit first resource
5. **Monitor Deployment** - Verify auto-deployment works

For more information, see:
- [Agent Registry Documentation](../README.md)
- [API Documentation](../docs/API.md)
- [Deployment Guide](../docs/DEPLOYMENT.md)