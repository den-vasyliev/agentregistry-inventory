package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	"github.com/agentregistry-dev/agentregistry/internal/controller"
)

func (s *MCPServer) registerResources() {
	// Static resource
	s.mcpServer.AddResource(
		mcp.NewResource("registry://stats", "Registry statistics"),
		s.handleResourceStats,
	)

	// Resource templates
	s.mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate("registry://servers", "All MCP servers"),
		s.handleResourceServers,
	)
	s.mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate("registry://servers/{name}", "MCP server details"),
		s.handleResourceServerByName,
	)
	s.mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate("registry://agents", "All agents"),
		s.handleResourceAgents,
	)
	s.mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate("registry://agents/{name}", "Agent details"),
		s.handleResourceAgentByName,
	)
	s.mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate("registry://skills", "All skills"),
		s.handleResourceSkills,
	)
	s.mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate("registry://skills/{name}", "Skill details"),
		s.handleResourceSkillByName,
	)
	s.mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate("registry://models", "All models"),
		s.handleResourceModels,
	)
	s.mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate("registry://models/{name}", "Model details"),
		s.handleResourceModelByName,
	)
	s.mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate("registry://deployments", "All deployments"),
		s.handleResourceDeployments,
	)
	s.mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate("registry://deployments/{name}", "Deployment details"),
		s.handleResourceDeploymentByName,
	)
	s.mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate("registry://environments", "All environments"),
		s.handleResourceEnvironments,
	)
}

// extractNameFromURI extracts the last path segment from a resource URI.
// e.g., "registry://servers/my-server" -> "my-server"
func extractNameFromURI(uri, prefix string) string {
	name, _ := strings.CutPrefix(uri, prefix)
	return name
}

func marshalToResourceContents(uri string, v any) ([]mcp.ResourceContents, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

func (s *MCPServer) handleResourceStats(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	stats, err := s.getStats(ctx)
	if err != nil {
		return nil, err
	}
	return marshalToResourceContents(request.Params.URI, stats)
}

func (s *MCPServer) handleResourceServers(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	var list agentregistryv1alpha1.MCPServerCatalogList
	if err := s.cache.List(ctx, &list); err != nil {
		return nil, err
	}
	return marshalToResourceContents(request.Params.URI, summarizeServers(list.Items))
}

func (s *MCPServer) handleResourceServerByName(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	name := extractNameFromURI(request.Params.URI, "registry://servers/")
	if name == "" {
		return nil, fmt.Errorf("name parameter required")
	}

	var list agentregistryv1alpha1.MCPServerCatalogList
	if err := s.cache.List(ctx, &list, client.MatchingFields{
		controller.IndexMCPServerName:     name,
		controller.IndexMCPServerIsLatest: "true",
	}); err != nil {
		return nil, err
	}
	if len(list.Items) == 0 {
		return nil, fmt.Errorf("server '%s' not found", name)
	}
	return marshalToResourceContents(request.Params.URI, list.Items[0].Spec)
}

func (s *MCPServer) handleResourceAgents(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	var list agentregistryv1alpha1.AgentCatalogList
	if err := s.cache.List(ctx, &list); err != nil {
		return nil, err
	}
	return marshalToResourceContents(request.Params.URI, summarizeAgents(list.Items))
}

func (s *MCPServer) handleResourceAgentByName(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	name := extractNameFromURI(request.Params.URI, "registry://agents/")
	if name == "" {
		return nil, fmt.Errorf("name parameter required")
	}

	var list agentregistryv1alpha1.AgentCatalogList
	if err := s.cache.List(ctx, &list, client.MatchingFields{
		controller.IndexAgentName:     name,
		controller.IndexAgentIsLatest: "true",
	}); err != nil {
		return nil, err
	}
	if len(list.Items) == 0 {
		return nil, fmt.Errorf("agent '%s' not found", name)
	}
	return marshalToResourceContents(request.Params.URI, list.Items[0].Spec)
}

func (s *MCPServer) handleResourceSkills(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	var list agentregistryv1alpha1.SkillCatalogList
	if err := s.cache.List(ctx, &list); err != nil {
		return nil, err
	}

	type skillBrief struct {
		Name     string `json:"name"`
		Version  string `json:"version"`
		Title    string `json:"title,omitempty"`
		Category string `json:"category,omitempty"`
	}
	items := make([]skillBrief, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, skillBrief{
			Name:     item.Spec.Name,
			Version:  item.Spec.Version,
			Title:    item.Spec.Title,
			Category: item.Spec.Category,
		})
	}
	return marshalToResourceContents(request.Params.URI, items)
}

func (s *MCPServer) handleResourceSkillByName(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	name := extractNameFromURI(request.Params.URI, "registry://skills/")
	if name == "" {
		return nil, fmt.Errorf("name parameter required")
	}

	var list agentregistryv1alpha1.SkillCatalogList
	if err := s.cache.List(ctx, &list, client.MatchingFields{
		controller.IndexSkillName:     name,
		controller.IndexSkillIsLatest: "true",
	}); err != nil {
		return nil, err
	}
	if len(list.Items) == 0 {
		return nil, fmt.Errorf("skill '%s' not found", name)
	}
	return marshalToResourceContents(request.Params.URI, list.Items[0].Spec)
}

func (s *MCPServer) handleResourceModels(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	var list agentregistryv1alpha1.ModelCatalogList
	if err := s.cache.List(ctx, &list); err != nil {
		return nil, err
	}
	return marshalToResourceContents(request.Params.URI, summarizeModels(list.Items))
}

func (s *MCPServer) handleResourceModelByName(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	name := extractNameFromURI(request.Params.URI, "registry://models/")
	if name == "" {
		return nil, fmt.Errorf("name parameter required")
	}

	var list agentregistryv1alpha1.ModelCatalogList
	if err := s.cache.List(ctx, &list, client.MatchingFields{
		controller.IndexModelName: name,
	}); err != nil {
		return nil, err
	}
	if len(list.Items) == 0 {
		return nil, fmt.Errorf("model '%s' not found", name)
	}
	return marshalToResourceContents(request.Params.URI, list.Items[0].Spec)
}

func (s *MCPServer) handleResourceDeployments(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	var list agentregistryv1alpha1.RegistryDeploymentList
	if err := s.cache.List(ctx, &list); err != nil {
		return nil, err
	}
	return marshalToResourceContents(request.Params.URI, summarizeDeployments(list.Items))
}

func (s *MCPServer) handleResourceDeploymentByName(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	name := extractNameFromURI(request.Params.URI, "registry://deployments/")
	if name == "" {
		return nil, fmt.Errorf("name parameter required")
	}

	var deployment agentregistryv1alpha1.RegistryDeployment
	if err := s.cache.Get(ctx, client.ObjectKey{Namespace: "agentregistry", Name: name}, &deployment); err != nil {
		return nil, fmt.Errorf("deployment '%s' not found", name)
	}
	return marshalToResourceContents(request.Params.URI, deployment.Spec)
}

func (s *MCPServer) handleResourceEnvironments(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	var list agentregistryv1alpha1.DiscoveryConfigList
	if err := s.client.List(ctx, &list, client.InNamespace("agentregistry")); err != nil {
		return nil, err
	}

	type envBrief struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}
	envs := make([]envBrief, 0)
	for _, dc := range list.Items {
		for _, env := range dc.Spec.Environments {
			ns := env.Cluster.Namespace
			if ns == "" && len(env.Namespaces) > 0 {
				ns = env.Namespaces[0]
			}
			envs = append(envs, envBrief{
				Name:      env.Name,
				Namespace: ns,
			})
		}
	}
	return marshalToResourceContents(request.Params.URI, envs)
}
