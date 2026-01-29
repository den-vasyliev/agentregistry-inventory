package models

import (
	"time"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	apiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
	"github.com/modelcontextprotocol/registry/pkg/model"
)

// ServerJSONFromCatalog converts an MCPServerCatalog CR to ServerJSON format
// This is used by the HTTP API to return data in the expected format
func ServerJSONFromCatalog(catalog *agentregistryv1alpha1.MCPServerCatalog) apiv0.ServerJSON {
	server := apiv0.ServerJSON{
		Name:        catalog.Spec.Name,
		Version:     catalog.Spec.Version,
		Title:       catalog.Spec.Title,
		Description: catalog.Spec.Description,
		WebsiteURL:  catalog.Spec.WebsiteURL,
	}

	if catalog.Spec.Repository != nil {
		server.Repository = &model.Repository{
			URL:       catalog.Spec.Repository.URL,
			Source:    catalog.Spec.Repository.Source,
			ID:        catalog.Spec.Repository.ID,
			Subfolder: catalog.Spec.Repository.Subfolder,
		}
	}

	for _, p := range catalog.Spec.Packages {
		pkg := model.Package{
			RegistryType:    p.RegistryType,
			RegistryBaseURL: p.RegistryBaseURL,
			Identifier:      p.Identifier,
			Version:         p.Version,
			FileSHA256:      p.FileSHA256,
			RunTimeHint:     p.RuntimeHint,
			Transport: model.Transport{
				Type: p.Transport.Type,
				URL:  p.Transport.URL,
			},
		}

		for _, h := range p.Transport.Headers {
			pkg.Transport.Headers = append(pkg.Transport.Headers, model.KeyValueInput{
				Name: h.Name,
			})
		}

		for _, a := range p.RuntimeArguments {
			pkg.RuntimeArguments = append(pkg.RuntimeArguments, model.Argument{
				Name: a.Name,
				Type: model.ArgumentType(a.Type),
			})
		}

		for _, a := range p.PackageArguments {
			pkg.PackageArguments = append(pkg.PackageArguments, model.Argument{
				Name: a.Name,
				Type: model.ArgumentType(a.Type),
			})
		}

		for _, e := range p.EnvironmentVariables {
			pkg.EnvironmentVariables = append(pkg.EnvironmentVariables, model.KeyValueInput{
				Name: e.Name,
			})
		}

		server.Packages = append(server.Packages, pkg)
	}

	for _, r := range catalog.Spec.Remotes {
		remote := model.Transport{
			Type: r.Type,
			URL:  r.URL,
		}
		for _, h := range r.Headers {
			remote.Headers = append(remote.Headers, model.KeyValueInput{
				Name: h.Name,
			})
		}
		server.Remotes = append(server.Remotes, remote)
	}

	return server
}

// ServerResponseFromCatalog converts an MCPServerCatalog CR to ServerResponse format
func ServerResponseFromCatalog(catalog *agentregistryv1alpha1.MCPServerCatalog) ServerResponse {
	server := ServerJSONFromCatalog(catalog)

	var publishedAt time.Time
	if catalog.Status.PublishedAt != nil {
		publishedAt = catalog.Status.PublishedAt.Time
	}

	return ServerResponse{
		Server: server,
		Meta: ServerResponseMeta{
			Official: &apiv0.RegistryExtensions{
				Status:      model.Status(catalog.Status.Status),
				PublishedAt: publishedAt,
				UpdatedAt:   catalog.CreationTimestamp.Time,
				IsLatest:    catalog.Status.IsLatest,
			},
		},
	}
}

// AgentJSONFromCatalog converts an AgentCatalog CR to AgentJSON format
func AgentJSONFromCatalog(catalog *agentregistryv1alpha1.AgentCatalog) AgentJSON {
	agent := AgentJSON{
		AgentManifest: AgentManifest{
			Name:              catalog.Spec.Name,
			Image:             catalog.Spec.Image,
			Language:          catalog.Spec.Language,
			Framework:         catalog.Spec.Framework,
			ModelProvider:     catalog.Spec.ModelProvider,
			ModelName:         catalog.Spec.ModelName,
			Description:       catalog.Spec.Description,
			Version:           catalog.Spec.Version,
			TelemetryEndpoint: catalog.Spec.TelemetryEndpoint,
			UpdatedAt:         catalog.CreationTimestamp.Time,
		},
		Title:      catalog.Spec.Title,
		Version:    catalog.Spec.Version,
		Status:     string(catalog.Status.Status),
		WebsiteURL: catalog.Spec.WebsiteURL,
	}

	if catalog.Spec.Repository != nil {
		agent.Repository = &model.Repository{
			URL:       catalog.Spec.Repository.URL,
			Source:    catalog.Spec.Repository.Source,
			ID:        catalog.Spec.Repository.ID,
			Subfolder: catalog.Spec.Repository.Subfolder,
		}
	}

	for _, p := range catalog.Spec.Packages {
		pkg := AgentPackageInfo{
			RegistryType: p.RegistryType,
			Identifier:   p.Identifier,
			Version:      p.Version,
		}
		if p.Transport != nil {
			pkg.Transport.Type = p.Transport.Type
		}
		agent.Packages = append(agent.Packages, pkg)
	}

	for _, r := range catalog.Spec.Remotes {
		remote := model.Transport{
			Type: r.Type,
			URL:  r.URL,
		}
		for _, h := range r.Headers {
			remote.Headers = append(remote.Headers, model.KeyValueInput{
				Name: h.Name,
			})
		}
		agent.Remotes = append(agent.Remotes, remote)
	}

	for _, m := range catalog.Spec.McpServers {
		agent.McpServers = append(agent.McpServers, McpServerType{
			Type:                       m.Type,
			Name:                       m.Name,
			Image:                      m.Image,
			Build:                      m.Build,
			Command:                    m.Command,
			Args:                       m.Args,
			Env:                        m.Env,
			URL:                        m.URL,
			Headers:                    m.Headers,
			RegistryURL:                m.RegistryURL,
			RegistryServerName:         m.RegistryServerName,
			RegistryServerVersion:      m.RegistryServerVersion,
			RegistryServerPreferRemote: m.RegistryServerPreferRemote,
		})
	}

	return agent
}

// AgentResponseFromCatalog converts an AgentCatalog CR to AgentResponse format
func AgentResponseFromCatalog(catalog *agentregistryv1alpha1.AgentCatalog) AgentResponse {
	agent := AgentJSONFromCatalog(catalog)

	var publishedAt time.Time
	if catalog.Status.PublishedAt != nil {
		publishedAt = catalog.Status.PublishedAt.Time
	}

	return AgentResponse{
		Agent: agent,
		Meta: AgentResponseMeta{
			Official: &AgentRegistryExtensions{
				Status:      string(catalog.Status.Status),
				PublishedAt: publishedAt,
				UpdatedAt:   catalog.CreationTimestamp.Time,
				IsLatest:    catalog.Status.IsLatest,
				Published:   catalog.Status.Published,
			},
		},
	}
}

// SkillJSONFromCatalog converts a SkillCatalog CR to SkillJSON format
func SkillJSONFromCatalog(catalog *agentregistryv1alpha1.SkillCatalog) SkillJSON {
	skill := SkillJSON{
		Name:        catalog.Spec.Name,
		Title:       catalog.Spec.Title,
		Category:    catalog.Spec.Category,
		Description: catalog.Spec.Description,
		Version:     catalog.Spec.Version,
		Status:      string(catalog.Status.Status),
		WebsiteURL:  catalog.Spec.WebsiteURL,
	}

	if catalog.Spec.Repository != nil {
		skill.Repository = &SkillRepository{
			URL:    catalog.Spec.Repository.URL,
			Source: catalog.Spec.Repository.Source,
		}
	}

	for _, p := range catalog.Spec.Packages {
		pkg := SkillPackageInfo{
			RegistryType: p.RegistryType,
			Identifier:   p.Identifier,
			Version:      p.Version,
		}
		if p.Transport != nil {
			pkg.Transport.Type = p.Transport.Type
		}
		skill.Packages = append(skill.Packages, pkg)
	}

	for _, r := range catalog.Spec.Remotes {
		skill.Remotes = append(skill.Remotes, SkillRemoteInfo{
			URL: r.URL,
		})
	}

	return skill
}

// SkillResponseFromCatalog converts a SkillCatalog CR to SkillResponse format
func SkillResponseFromCatalog(catalog *agentregistryv1alpha1.SkillCatalog) SkillResponse {
	skill := SkillJSONFromCatalog(catalog)

	var publishedAt time.Time
	if catalog.Status.PublishedAt != nil {
		publishedAt = catalog.Status.PublishedAt.Time
	}

	return SkillResponse{
		Skill: skill,
		Meta: SkillResponseMeta{
			Official: &SkillRegistryExtensions{
				Status:      string(catalog.Status.Status),
				PublishedAt: publishedAt,
				UpdatedAt:   catalog.CreationTimestamp.Time,
				IsLatest:    catalog.Status.IsLatest,
				Published:   catalog.Status.Published,
			},
		},
	}
}

// DeploymentFromCR converts a RegistryDeployment CR to Deployment model
func DeploymentFromCR(cr *agentregistryv1alpha1.RegistryDeployment) Deployment {
	deployment := Deployment{
		ServerName:   cr.Spec.ResourceName,
		Version:      cr.Spec.Version,
		Status:       string(cr.Status.Phase),
		Config:       cr.Spec.Config,
		PreferRemote: cr.Spec.PreferRemote,
		ResourceType: string(cr.Spec.ResourceType),
		Runtime:      string(cr.Spec.Runtime),
	}

	if cr.Status.DeployedAt != nil {
		deployment.DeployedAt = cr.Status.DeployedAt.Time
	}
	if cr.Status.UpdatedAt != nil {
		deployment.UpdatedAt = cr.Status.UpdatedAt.Time
	}

	return deployment
}
