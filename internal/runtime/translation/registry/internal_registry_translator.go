package registry

import (
	"context"
	"fmt"
	"github.com/modelcontextprotocol/registry/pkg/model"
	"mcp-enterprise-registry/internal/runtime/translation/api"
	"strings"

	apiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
)

// Translator is the interface for translating MCPServer objects to AgentGateway objects.
type Translator interface {
	TranslateMCPServer(
		ctx context.Context,
		registryServer *apiv0.ServerJSON,
		preferRemote bool,
		envValues map[string]string,
		argValues map[string]string,
		headerValues map[string]string,
	) (*api.MCPServer, error)
}

type registryTranslator struct{}

func (t *registryTranslator) TranslateMCPServer(
	ctx context.Context,
	registryServer *apiv0.ServerJSON,
	preferRemote bool,
	envValues map[string]string,
	argValues map[string]string,
	headerValues map[string]string,
) (*api.MCPServer, error) {
	useRemote := len(registryServer.Remotes) > 0 && (preferRemote || len(registryServer.Packages) == 0)
	usePackage := len(registryServer.Packages) > 0 && (!preferRemote || len(registryServer.Remotes) == 0)

	switch {
	case useRemote:
		return translateRemoteMCPServer(
			ctx,
			registryServer,
			headerValues,
		)
	case usePackage:
		return translateLocalMCPServer(
			ctx,
			registryServer,
			envValues,
			argValues,
		)
	}

	return nil, fmt.Errorf("no valid deployment method found for server: %s", registryServer.Name)
}

func translateRemoteMCPServer(
	ctx context.Context,
	registryServer *apiv0.ServerJSON,
	headerValues map[string]string,
) (*api.MCPServer, error) {
	remoteInfo := registryServer.Remotes[0]

	var headers []api.HeaderValue
	for _, h := range remoteInfo.Headers {
		k := h.Name
		v := h.Value
		if v == "" {
			v = h.Default
		}
		if headerValues != nil {
			if override, exists := headerValues[k]; exists {
				v = override
			}
		}
		if h.IsRequired && v == "" {
			return nil, fmt.Errorf("missing required header value for header: %s", k)
		}
		headers = append(headers, api.HeaderValue{
			Name:  k,
			Value: v,
		})
	}

	return &api.MCPServer{
		Name:          generateInternalName(registryServer.Name),
		MCPServerType: api.MCPServerTypeRemote,
		Remote: &api.RemoteMCPServer{
			URL:     remoteInfo.URL,
			Headers: headers,
		},
	}, nil
}

func translateLocalMCPServer(
	ctx context.Context,
	registryServer *apiv0.ServerJSON,
	envValues map[string]string,
	argValues map[string]string,
) (*api.MCPServer, error) {
	var (
		stdioConfig *api.StdioTransport
		httpConfig  *api.HTTPTransport

		image string
		port  uint16
		cmd   string
		args  []string
	)

	// deploy the server either as stdio or http
	packageInfo := registryServer.Packages[0]

	cmd = packageInfo.RunTimeHint

	switch packageInfo.RegistryType {
	case "npm":
		image = "node:24-alpine3.21"
		if cmd == "" {
			cmd = "npx"
		}
		args = []string{packageInfo.Identifier}
	case "pypi":
		image = "ghcr.io/astral-sh/uv:debian"
		if cmd == "" {
			cmd = "uvx"
		}
		args = []string{packageInfo.Identifier}
	case "oci":
		image = packageInfo.Identifier
	default:
		return nil, fmt.Errorf("unsupported package registry type: %s", packageInfo.RegistryType)
	}

	getArgValue := func(arg model.Argument) string {
		if v, exists := argValues[arg.Name]; exists {
			return v
		}
		return arg.Value
	}

	for _, arg := range packageInfo.RuntimeArguments {
		switch arg.Type {
		case model.ArgumentTypePositional:
			args = append(args, getArgValue(arg))
		}
	}
	for _, arg := range packageInfo.RuntimeArguments {
		switch arg.Type {
		case model.ArgumentTypeNamed:
			args = append(args, arg.Name, getArgValue(arg))
		}
	}

	for _, envVar := range packageInfo.EnvironmentVariables {
		if _, exists := envValues[envVar.Name]; !exists {
			if envVar.IsRequired {
				return nil, fmt.Errorf("missing required environment variable: %s", envVar.Name)
			} else if envVar.Default != "" {
				envValues[envVar.Name] = envVar.Default
			}
		}
	}

	switch packageInfo.Transport.Type {
	case "stdio":
		stdioConfig = &api.StdioTransport{}
	default:
		httpConfig = &api.HTTPTransport{}
	}

	return &api.MCPServer{
		Name:          generateInternalName(registryServer.Name),
		MCPServerType: api.MCPServerTypeLocal,
		Local: &api.LocalMCPServer{
			Deployment: api.MCPServerDeployment{
				Image: image,
				Port:  port,
				Cmd:   cmd,
				Args:  args,
				Env:   envValues,
			},
			TransportType: "",
			Stdio:         stdioConfig,
			HTTP:          httpConfig,
		},
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
