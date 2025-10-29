package dockercompose

import (
	"context"
	"fmt"
	"github.com/compose-spec/compose-go/v2/types"
	"log"
	"mcp-enterprise-registry/internal/runtime/translation/api"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

type DockerComposeConfig = types.Project

const (
	agentGatewayRepository     = "ghcr.io/agentgateway/agentgateway"
	defaultAgentGatewayVersion = "0.9.0"
)

// versionRegex validates that version strings contain only allowed characters
// (alphanumeric, dots, hyphens) to prevent potential image injection attacks
var versionRegex = regexp.MustCompile(`^[a-zA-Z0-9.\-]+$`)

type AiRuntimeConfig struct {
	DockerCompose *DockerComposeConfig
	AgentGateway  *AgentGatewayConfig
}

// Translator is the interface for translating MCPServer objects to AgentGateway objects.
type Translator interface {
	TranslateRuntimeConfig(
		ctx context.Context,
		desired *api.DesiredState,
	) (*AiRuntimeConfig, error)
}

type agentGatewayTranslator struct {
	composeWorkingDir string
	agentGatewayPort  uint16
}

func NewAgentGatewayTranslator(composeWorkingDir string, agentGatewayPort uint16) Translator {
	return &agentGatewayTranslator{
		composeWorkingDir: composeWorkingDir,
		agentGatewayPort:  agentGatewayPort,
	}
}

func (t *agentGatewayTranslator) TranslateRuntimeConfig(
	ctx context.Context,
	desired *api.DesiredState,
) (*AiRuntimeConfig, error) {

	agentGatewayService, err := t.translateAgentGatewayService()
	if err != nil {
		return nil, fmt.Errorf("failed to translate agent gateway service: %w", err)
	}

	dockerComposeServices := map[string]types.ServiceConfig{
		"agent_gateway": *agentGatewayService,
	}

	for _, mcpServer := range desired.MCPServers {
		// only need to create services for local servers
		if mcpServer.MCPServerType != api.MCPServerTypeLocal {
			continue
		}
		// error if MCPServer name is not unique
		if _, exists := dockerComposeServices[mcpServer.Name]; exists {
			return nil, fmt.Errorf("duplicate MCPServer name found: %s", mcpServer.Name)
		}

		serviceConfig, err := t.translateMCPServerToServiceConfig(ctx, mcpServer)
		if err != nil {
			return nil, fmt.Errorf("failed to translate MCPServer %s to service config: %w", mcpServer.Name, err)
		}
		dockerComposeServices[mcpServer.Name] = *serviceConfig
	}

	dockerCompose := &DockerComposeConfig{
		Name:       "ai_registry",
		WorkingDir: t.composeWorkingDir,
		Services:   dockerComposeServices,
		//Networks:         nil,
		//Volumes:          nil,
		//Secrets:          nil,
		//Configs:          nil,
		//Models:           nil,
		//Extensions:       nil,
		//ComposeFiles:     nil,
		//Environment:      nil,
		//DisabledServices: nil,
		//Profiles:         nil,
	}

	gwConfig, err := t.translateAgentGatewayConfig(desired.MCPServers)
	if err != nil {
		return nil, fmt.Errorf("failed to translate agent gateway config: %w", err)
	}

	return &AiRuntimeConfig{
		DockerCompose: dockerCompose,
		AgentGateway:  gwConfig,
	}, nil
}

func (t *agentGatewayTranslator) translateAgentGatewayService() (*types.ServiceConfig, error) {
	port := t.agentGatewayPort
	if port == 0 {
		return nil, fmt.Errorf("agent gateway port must be specified")
	}
	return &types.ServiceConfig{
		Name:    "agent_gateway",
		Image:   getAgentGatewayImage(),
		Command: []string{"-f", "/config/local.yaml"},
		Ports: []types.ServicePortConfig{{
			Name:      "http",
			Target:    uint32(port),
			Published: fmt.Sprintf("%d", port),
		}},
		Volumes: []types.ServiceVolumeConfig{{
			Type:   "volume",
			Source: filepath.Join(t.composeWorkingDir, "agent_gateway"),
			Target: "/config",
		}},
	}, nil
}

func (t *agentGatewayTranslator) translateMCPServerToServiceConfig(ctx context.Context, server api.MCPServer) (*types.ServiceConfig, error) {
	image := server.Local.Deployment.Image
	if image == "" && server.Local.Deployment.Cmd == "uvx" {
		image = "ghcr.io/astral-sh/uv:debian"
	}
	if image == "" && server.Local.Deployment.Cmd == "npx" {
		image = "node:24-alpine3.21"
	}
	if image == "" {
		return nil, fmt.Errorf("image must be specified for MCPServer %s or the command must be 'uvx' or 'npx'", server.Name)
	}
	return &types.ServiceConfig{
		Name:    server.Name,
		Image:   getAgentGatewayImage(),
		Command: []string{"-f", "/config/local.yaml"},
		Volumes: []types.ServiceVolumeConfig{{
			Type:   "volume",
			Source: filepath.Join(t.composeWorkingDir, "agent_gateway"),
			Target: "/config",
		}},
	}, nil
}

func (t *agentGatewayTranslator) translateAgentGatewayConfig(servers []api.MCPServer) (*AgentGatewayConfig, error) {
	var routes []LocalRoute

	for _, server := range servers {
		pathPrefix := fmt.Sprintf("%s/mcp", server.Name)

		mcpTarget := MCPTarget{
			Name: server.Name,
		}

		switch server.MCPServerType {
		case api.MCPServerTypeRemote:
			mcpTarget.SSE = &SSETargetSpec{
				Host: server.Remote.Host,
				Port: server.Remote.Port,
				Path: server.Remote.Path,
			}
		case api.MCPServerTypeLocal:
			switch server.Local.TransportType {
			case api.TransportTypeStdio:
				mcpTarget.Stdio = &StdioTargetSpec{
					Cmd:  server.Local.Deployment.Cmd,
					Args: server.Local.Deployment.Args,
					Env:  server.Local.Deployment.Env,
				}
			case api.TransportTypeHTTP:
				httpTransportConfig := server.Local.HTTP
				if httpTransportConfig == nil || httpTransportConfig.Port == 0 {
					return nil, fmt.Errorf("HTTP transport requires a target port")
				}
				mcpTarget.SSE = &SSETargetSpec{
					Host: server.Name,
					Port: httpTransportConfig.Port,
					Path: httpTransportConfig.Path,
				}
			default:
				return nil, fmt.Errorf("unsupported transport type: %s", server.Local.TransportType)
			}
		}

		routes = append(routes, LocalRoute{
			RouteName: server.Name,
			Matches: []RouteMatch{
				{
					Path: PathMatch{
						PathPrefix: pathPrefix,
					},
				},
			},
			Backends: []RouteBackend{{
				Weight: 100,
				MCP: &MCPBackend{
					Targets: []MCPTarget{mcpTarget},
				},
			}},
		})
	}

	// sort for idepmpotence
	sort.SliceStable(routes, func(i, j int) bool {
		return routes[i].RouteName < routes[j].RouteName
	})

	return &AgentGatewayConfig{
		Config: struct{}{},
		Binds: []LocalBind{
			{
				Port: t.agentGatewayPort,
				Listeners: []LocalListener{
					{
						Name:     "default",
						Protocol: "HTTP",
						Routes:   routes,
					},
				},
			},
		},
	}, nil
}

// getAgentGatewayImage returns the agent gateway container image,
// using the environment variable if provided and valid, otherwise using the default
func getAgentGatewayImage() string {
	agentGatewayVersion := os.Getenv("TRANSPORT_ADAPTER_VERSION")
	if agentGatewayVersion == "" {
		return fmt.Sprintf("%s:%s-musl", agentGatewayRepository, defaultAgentGatewayVersion)
	}

	if err := validateVersion(agentGatewayVersion); err != nil {
		log.Printf("WARN: Invalid TRANSPORT_ADAPTER_VERSION: %v, fallback to %s", err, defaultAgentGatewayVersion)
		return fmt.Sprintf("%s:%s-musl", agentGatewayRepository, defaultAgentGatewayVersion)
	}

	return fmt.Sprintf("%s:%s-musl", agentGatewayRepository, agentGatewayVersion)
}

// validateVersion validates that a version string contains only allowed characters
// to prevent potential image injection attacks
func validateVersion(version string) error {
	if !versionRegex.MatchString(version) {
		return fmt.Errorf("invalid version format: %s (only alphanumeric characters, dots, and hyphens are allowed)", version)
	}
	return nil
}
