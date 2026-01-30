# Add MCP Server Workflow

## Overview

The Agent Registry now uses a GitOps-style workflow for adding MCP servers. Instead of directly creating servers via API, users generate Kubernetes manifests that can be committed to their repository and applied via kubectl.

## Workflow Steps

### 1. Fill the Form

Navigate to the Admin UI and click "Add" → "Add Server". Fill in the MCP server details:

- **Server Name** (required): Must be in format `namespace/name` (e.g., `io.example/my-server`)
- **Version** (required): Semantic version (e.g., `1.0.0`)
- **Description** (required): What the server does
- **Title** (optional): Display name
- **Website URL** (optional): Documentation or homepage
- **Repository** (optional): Source code repository
- **Packages** (optional): Package configurations (npm, pypi, docker, etc.)
- **Remotes** (optional): Remote transport endpoints

### 2. Generate Manifest

Click **"Generate Manifest"** to create a `MCPServerCatalog` Kubernetes resource manifest.

### 3. Review and Download

The UI will display the generated YAML manifest with:
- Instructions for applying it
- A preview of the manifest content
- Options to **Copy to Clipboard** or **Download YAML**

### 4. Commit to Repository

Save the manifest file (e.g., `io-example-my-server.yaml`) and commit it to your Git repository:

```bash
git add io-example-my-server.yaml
git commit -m "Add MCP server: io.example/my-server"
git push
```

### 5. Apply to Kubernetes

Apply the manifest to your Kubernetes cluster:

```bash
kubectl apply -f io-example-my-server.yaml
```

### 6. Automatic Registration

The Agent Registry controller watches for `MCPServerCatalog` resources and automatically:
- Validates the manifest
- Registers the server in the registry database
- Syncs the status
- Makes it available via the public API

## Example Manifest

```yaml
apiVersion: agentregistry.dev/v1alpha1
kind: MCPServerCatalog
metadata:
  name: io-example-my-server
spec:
  name: io.example/my-server
  version: 1.0.0
  title: My Example Server
  description: A sample MCP server for demonstration
  websiteUrl: https://example.com
  repository:
    url: https://github.com/example/my-server
    source: github
  packages:
    - identifier: example/my-server
      version: 1.0.0
      registryType: npm
      transport:
        type: stdio
```

## Benefits of GitOps Workflow

1. **Source Control**: All server definitions are version-controlled
2. **Audit Trail**: Git history provides complete audit trail
3. **Review Process**: Changes can go through pull request reviews
4. **Declarative**: Infrastructure-as-code approach
5. **Reproducible**: Easy to recreate environments from Git

## Controller Integration

The `MCPServerCatalogController` in `/internal/controller/mcpservercatalog_controller.go` handles:
- Watching for new/updated `MCPServerCatalog` resources
- Validating the spec
- Creating/updating entries in the registry database
- Managing versioning and "latest" flag
- Syncing deployment status (if applicable)

## Next Steps

After the server is registered:
1. Check status: `kubectl get mcpservercatalog`
2. View in UI: Navigate to the Servers tab
3. Publish: Mark as published to make it visible in public listings
4. Deploy: Optionally deploy the server to Kubernetes
