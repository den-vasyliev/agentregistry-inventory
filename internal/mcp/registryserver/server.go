package registryserver

import (
	"github.com/agentregistry-dev/agentregistry/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer constructs an MCP server that exposes read-only discovery tools backed by Kubernetes CRDs.
// TODO: Implement with Kubernetes client to query Application CRDs
func NewServer(_ interface{}) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "agentregistry-mcp",
		Version: version.Version,
	}, &mcp.ServerOptions{
		HasTools: true,
	})

	// TODO: Add tools that query Kubernetes Application CRDs
	// - list_agents
	// - get_agent
	// - list_servers
	// - get_server
	// - list_skills
	// - get_skill

	return server
}
