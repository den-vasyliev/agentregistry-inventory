# AI Registry and Runtime - arrt

A CLI tool for managing MCP (Model Context Protocol) servers, skills, and registries with an embedded web UI.

## Quick Start

```bash
# Build the project
make build

# Run the CLI
./bin/arrt --help

# Launch the web UI
./bin/arrt ui
```

Then open http://localhost:8080 in your browser.


## CLI Commands

```
# Connects an existing public or private registry to arrt
# this will fetch the data from the registry and store it locally
arrt connect <registry-url> <registry-name>

# removes the cached data and the registry from the config
arrt disconnect <registry-name>

# List resources -- this lists the resources across the connected registries
arrt list <resource-type>

# Lists MCP servers from all connected registries
arrt list mcp

# Lists skills from all connected registries
arrt list skill

# Lists all connected registries
arrt list registry

# Search for resources from the connected registries
arrt search <resource-type> <search-term>

# Updates/fetches the new data from the connected registries
arrt refresh

# Shows details of a resource
arrt show <resource-type> <resource-name>
arrt show mcp <mcp-server-name>
arrt show skill <skill-name>
arrt show registry <registry-name>

# Install/uninstall resources 
arrt install mcp <mcp-server-name> <version> <config>
arrt install skill <skill-name> <version>

arrt uninstall mcp <mcp-server-name>
arrt uninstall skill <skill-name>

# Client configuration - creates the .json configuration for each client, so it can connect to the arrt
arrt configure <client-name>

# Starts/restarts the arrt with the existing configuration
arrt start

# Launches the UI
arrt ui

# shows the version
arrt version
```
