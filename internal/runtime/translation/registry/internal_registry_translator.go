package registry

import (
	"context"
	"mcp-enterprise-registry/internal/runtime/translation/api"
	"strings"

	apiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
)

// Translator is the interface for translating MCPServer objects to AgentGateway objects.
type Translator interface {
	TranslateMCPServer(
		ctx context.Context,
		registryServer *apiv0.ServerJSON,
	) (*api.MCPServer, error)
}

type registryTranslator struct{}

func (t *registryTranslator) TranslateMCPServer(
	ctx context.Context,
	registryServer *apiv0.ServerJSON,
	env map[string]string,
) (*api.MCPServer, error) {
	switch {
	case len(registryServer.Remotes) > 0:
		// create route to remote server
	case len(registryServer.Packages) > 0:
		// deploy the server either as stdio or
	}

	return &api.MCPServer{
		Name: generateInternalName(registryServer.Name),
		Deployment: api.MCPServerDeployment{
			Env: env,
		},
		TransportType: "",
		Stdio:         nil,
		HTTP:          nil,
	}, nil
}

func generateInternalName(server string) string {
	// convert the server name to a dns-1123 compliant name
	name := strings.ToLower(strings.ReplaceAll(server, " ", "-"))
	name = strings.ReplaceAll(name, ".", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, "@", "-")
	name = strings.ReplaceAll(name, "#", "-")
	name = strings.ReplaceAll(name, "$", "-")
	name = strings.ReplaceAll(name, "%", "-")
	name = strings.ReplaceAll(name, "^", "-")
	name = strings.ReplaceAll(name, "&", "-")
	name = strings.ReplaceAll(name, "*", "-")
	name = strings.ReplaceAll(name, "(", "-")
	name = strings.ReplaceAll(name, ")", "-")
	name = strings.ReplaceAll(name, "[", "-")
	name = strings.ReplaceAll(name, "]", "-")
	name = strings.ReplaceAll(name, "{", "-")
	name = strings.ReplaceAll(name, "}", "-")
	name = strings.ReplaceAll(name, "|", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, ".", "-")
	name = strings.ReplaceAll(name, ",", "-")
	name = strings.ReplaceAll(name, "!", "-")
	name = strings.ReplaceAll(name, "?", "-")
	name = strings.ReplaceAll(name, " ", "-")
	return name
}
