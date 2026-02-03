package controller

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

// Index field names for cache queries
const (
	// MCPServerCatalog indexes
	IndexMCPServerName      = "spec.name"
	IndexMCPServerPublished = "status.published"
	IndexMCPServerIsLatest  = "status.isLatest"

	// AgentCatalog indexes
	IndexAgentName      = "spec.name"
	IndexAgentPublished = "status.published"
	IndexAgentIsLatest  = "status.isLatest"

	// SkillCatalog indexes
	IndexSkillName      = "spec.name"
	IndexSkillPublished = "status.published"
	IndexSkillIsLatest  = "status.isLatest"

	// ModelCatalog indexes
	IndexModelName      = "spec.name"
	IndexModelPublished = "status.published"

	// RegistryDeployment indexes
	IndexDeploymentResourceName = "spec.resourceName"
	IndexDeploymentResourceType = "spec.resourceType"
	IndexDeploymentRuntime      = "spec.runtime"
)

// SetupIndexes configures cache indexes for efficient queries
func SetupIndexes(mgr ctrl.Manager) error {
	// MCPServerCatalog indexes
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&agentregistryv1alpha1.MCPServerCatalog{},
		IndexMCPServerName,
		func(obj client.Object) []string {
			server := obj.(*agentregistryv1alpha1.MCPServerCatalog)
			return []string{server.Spec.Name}
		},
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&agentregistryv1alpha1.MCPServerCatalog{},
		IndexMCPServerPublished,
		func(obj client.Object) []string {
			server := obj.(*agentregistryv1alpha1.MCPServerCatalog)
			if server.Status.Published {
				return []string{"true"}
			}
			return []string{"false"}
		},
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&agentregistryv1alpha1.MCPServerCatalog{},
		IndexMCPServerIsLatest,
		func(obj client.Object) []string {
			server := obj.(*agentregistryv1alpha1.MCPServerCatalog)
			if server.Status.IsLatest {
				return []string{"true"}
			}
			return []string{"false"}
		},
	); err != nil {
		return err
	}

	// AgentCatalog indexes
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&agentregistryv1alpha1.AgentCatalog{},
		IndexAgentName,
		func(obj client.Object) []string {
			agent := obj.(*agentregistryv1alpha1.AgentCatalog)
			return []string{agent.Spec.Name}
		},
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&agentregistryv1alpha1.AgentCatalog{},
		IndexAgentPublished,
		func(obj client.Object) []string {
			agent := obj.(*agentregistryv1alpha1.AgentCatalog)
			if agent.Status.Published {
				return []string{"true"}
			}
			return []string{"false"}
		},
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&agentregistryv1alpha1.AgentCatalog{},
		IndexAgentIsLatest,
		func(obj client.Object) []string {
			agent := obj.(*agentregistryv1alpha1.AgentCatalog)
			if agent.Status.IsLatest {
				return []string{"true"}
			}
			return []string{"false"}
		},
	); err != nil {
		return err
	}

	// SkillCatalog indexes
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&agentregistryv1alpha1.SkillCatalog{},
		IndexSkillName,
		func(obj client.Object) []string {
			skill := obj.(*agentregistryv1alpha1.SkillCatalog)
			return []string{skill.Spec.Name}
		},
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&agentregistryv1alpha1.SkillCatalog{},
		IndexSkillPublished,
		func(obj client.Object) []string {
			skill := obj.(*agentregistryv1alpha1.SkillCatalog)
			if skill.Status.Published {
				return []string{"true"}
			}
			return []string{"false"}
		},
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&agentregistryv1alpha1.SkillCatalog{},
		IndexSkillIsLatest,
		func(obj client.Object) []string {
			skill := obj.(*agentregistryv1alpha1.SkillCatalog)
			if skill.Status.IsLatest {
				return []string{"true"}
			}
			return []string{"false"}
		},
	); err != nil {
		return err
	}

	// ModelCatalog indexes
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&agentregistryv1alpha1.ModelCatalog{},
		IndexModelName,
		func(obj client.Object) []string {
			model := obj.(*agentregistryv1alpha1.ModelCatalog)
			return []string{model.Spec.Name}
		},
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&agentregistryv1alpha1.ModelCatalog{},
		IndexModelPublished,
		func(obj client.Object) []string {
			model := obj.(*agentregistryv1alpha1.ModelCatalog)
			if model.Status.Published {
				return []string{"true"}
			}
			return []string{"false"}
		},
	); err != nil {
		return err
	}

	// RegistryDeployment indexes
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&agentregistryv1alpha1.RegistryDeployment{},
		IndexDeploymentResourceName,
		func(obj client.Object) []string {
			deploy := obj.(*agentregistryv1alpha1.RegistryDeployment)
			return []string{deploy.Spec.ResourceName}
		},
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&agentregistryv1alpha1.RegistryDeployment{},
		IndexDeploymentResourceType,
		func(obj client.Object) []string {
			deploy := obj.(*agentregistryv1alpha1.RegistryDeployment)
			return []string{string(deploy.Spec.ResourceType)}
		},
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&agentregistryv1alpha1.RegistryDeployment{},
		IndexDeploymentRuntime,
		func(obj client.Object) []string {
			deploy := obj.(*agentregistryv1alpha1.RegistryDeployment)
			return []string{string(deploy.Spec.Runtime)}
		},
	); err != nil {
		return err
	}

	return nil
}
