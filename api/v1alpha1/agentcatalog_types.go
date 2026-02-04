package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AgentPackage represents a package configuration for agents
type AgentPackage struct {
	// RegistryType indicates how to download packages (e.g., "oci")
	RegistryType string `json:"registryType"`
	// Identifier is the package identifier (e.g., OCI image reference)
	Identifier string `json:"identifier"`
	// Version is the package version
	// +optional
	Version string `json:"version,omitempty"`
	// Transport is the transport type configuration
	// +optional
	Transport *AgentPackageTransport `json:"transport,omitempty"`
}

// AgentPackageTransport represents the transport configuration for an agent package
type AgentPackageTransport struct {
	// Type is the transport type
	Type string `json:"type"`
}

// McpServerConfig represents an MCP server configuration for an agent
type McpServerConfig struct {
	// Type is the MCP server type (remote, command, registry)
	Type string `json:"type"`
	// Name is the MCP server name
	Name string `json:"name"`
	// Image is the container image for local MCP servers
	// +optional
	Image string `json:"image,omitempty"`
	// Build is the build path for local MCP servers
	// +optional
	Build string `json:"build,omitempty"`
	// Command is the command to run for command-type MCP servers
	// +optional
	Command string `json:"command,omitempty"`
	// Args are the command arguments
	// +optional
	Args []string `json:"args,omitempty"`
	// Env are the environment variables
	// +optional
	Env []string `json:"env,omitempty"`
	// URL is the URL for remote MCP servers
	// +optional
	URL string `json:"url,omitempty"`
	// Headers are the HTTP headers for remote MCP servers
	// +optional
	Headers map[string]string `json:"headers,omitempty"`
	// RegistryURL is the registry URL for registry-type MCP servers
	// +optional
	RegistryURL string `json:"registryURL,omitempty"`
	// RegistryServerName is the server name in the registry
	// +optional
	RegistryServerName string `json:"registryServerName,omitempty"`
	// RegistryServerVersion is the server version in the registry
	// +optional
	RegistryServerVersion string `json:"registryServerVersion,omitempty"`
	// RegistryServerPreferRemote indicates whether to prefer remote transport
	// +optional
	RegistryServerPreferRemote bool `json:"registryServerPreferRemote,omitempty"`
}

// AgentCatalogSpec defines the desired state of AgentCatalog
type AgentCatalogSpec struct {
	// Name is the canonical name of the agent
	Name string `json:"name"`
	// Version is the semantic version of the agent
	Version string `json:"version"`
	// Title is a human-readable title for the agent
	// +optional
	Title string `json:"title,omitempty"`
	// Description is a human-readable description of the agent
	// +optional
	Description string `json:"description,omitempty"`
	// Image is the container image for the agent
	Image string `json:"image"`
	// Language is the programming language of the agent
	// +optional
	Language string `json:"language,omitempty"`
	// Framework is the agent framework used (e.g., "langgraph", "autogen")
	// +optional
	Framework string `json:"framework,omitempty"`
	// ModelProvider is the LLM provider (e.g., "anthropic", "openai")
	// +optional
	ModelProvider string `json:"modelProvider,omitempty"`
	// ModelName is the specific model name
	// +optional
	ModelName string `json:"modelName,omitempty"`
	// TelemetryEndpoint is the endpoint for telemetry data
	// +optional
	TelemetryEndpoint string `json:"telemetryEndpoint,omitempty"`
	// WebsiteURL is the URL to the agent's website or documentation
	// +optional
	WebsiteURL string `json:"websiteUrl,omitempty"`
	// Repository is the source code repository information
	// +optional
	Repository *Repository `json:"repository,omitempty"`
	// Packages are the available package configurations
	// +optional
	Packages []AgentPackage `json:"packages,omitempty"`
	// Remotes are the remote transport configurations
	// +optional
	Remotes []Transport `json:"remotes,omitempty"`
	// McpServers are the MCP server configurations for the agent
	// +optional
	McpServers []McpServerConfig `json:"mcpServers,omitempty"`
	// Metadata contains additional metadata for the agent (stars, verification, etc.)
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Type=object
	Metadata *apiextensionsv1.JSON `json:"_meta,omitempty"`
}

// AgentCatalogStatus defines the observed state of AgentCatalog
type AgentCatalogStatus struct {
	// Published indicates whether this agent version is published and visible
	Published bool `json:"published"`
	// IsLatest indicates whether this is the latest version of the agent
	IsLatest bool `json:"isLatest"`
	// PublishedAt is the timestamp when this version was published
	// +optional
	PublishedAt *metav1.Time `json:"publishedAt,omitempty"`
	// Status is the lifecycle status (active, deprecated, deleted)
	// +optional
	Status CatalogStatus `json:"status,omitempty"`
	// ManagementType indicates how this resource is managed (external or managed)
	// +optional
	ManagementType ManagementType `json:"managementType,omitempty"`
	// Deployment tracks the runtime deployment info (optional, set by user or discovered)
	// For external resources: synced from SourceRef
	// For managed resources: set by RegistryDeployment
	// +optional
	Deployment *DeploymentRef `json:"deployment,omitempty"`
	// Conditions represent the latest available observations of the agent's state
	// +optional
	Conditions []CatalogCondition `json:"conditions,omitempty"`
	// ObservedGeneration is the generation last observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=agcat;agentcat
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="Published",type=boolean,JSONPath=`.status.published`
// +kubebuilder:printcolumn:name="Latest",type=boolean,JSONPath=`.status.isLatest`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// AgentCatalog is the Schema for the agentcatalogs API
type AgentCatalog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentCatalogSpec   `json:"spec,omitempty"`
	Status AgentCatalogStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AgentCatalogList contains a list of AgentCatalog
type AgentCatalogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentCatalog `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AgentCatalog{}, &AgentCatalogList{})
}
