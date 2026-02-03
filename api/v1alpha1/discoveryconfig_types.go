package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DiscoveryConfigSpec defines the desired state of DiscoveryConfig
type DiscoveryConfigSpec struct {
	// Environments is a list of environments to discover resources from
	Environments []Environment `json:"environments"`
}

// Environment represents a target cluster/namespace for resource discovery
type Environment struct {
	// Name is a unique identifier for this environment
	Name string `json:"name"`

	// Cluster contains cluster connection information
	Cluster ClusterConfig `json:"cluster"`

	// Provider is the cloud provider (gcp, aws, azure)
	// +optional
	Provider string `json:"provider,omitempty"`

	// Registry contains container registry information for this environment
	// +optional
	Registry RegistryConfig `json:"registry,omitempty"`

	// DiscoveryEnabled enables/disables discovery for this environment
	// +optional
	// +kubebuilder:default=true
	DiscoveryEnabled bool `json:"discoveryEnabled,omitempty"`

	// Namespaces is a list of namespaces to discover in. Empty means all namespaces.
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`

	// ResourceTypes specifies which types to discover (MCPServer, Agent, Skill, ModelConfig)
	// Empty means all types
	// +optional
	ResourceTypes []string `json:"resourceTypes,omitempty"`

	// Labels are additional labels to apply to discovered resources
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// ClusterConfig contains cluster connection information
type ClusterConfig struct {
	// Name is the cluster name
	Name string `json:"name"`

	// Namespace is the default namespace for this cluster
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// ProjectID is the GCP project ID (for GCP workload identity)
	// +optional
	ProjectID string `json:"projectId,omitempty"`

	// Zone is the cluster zone/region
	// +optional
	Zone string `json:"zone,omitempty"`

	// Region is the cluster region (alternative to Zone)
	// +optional
	Region string `json:"region,omitempty"`

	// Endpoint is the cluster API server endpoint
	// If not provided, will be auto-discovered using provider APIs
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// CAData is the base64-encoded certificate authority data
	// If not provided, will use workload identity for authentication
	// +optional
	CAData string `json:"caData,omitempty"`

	// UseWorkloadIdentity enables workload identity for cluster authentication
	// +optional
	// +kubebuilder:default=true
	UseWorkloadIdentity bool `json:"useWorkloadIdentity,omitempty"`

	// ServiceAccount is the service account to use for workload identity
	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`
}

// RegistryConfig contains container registry information
type RegistryConfig struct {
	// URL is the registry URL (e.g., "europe-docker.pkg.dev/project-id")
	URL string `json:"url"`

	// Prefix is the registry path prefix for images
	// +optional
	Prefix string `json:"prefix,omitempty"`

	// UseWorkloadIdentity enables workload identity for registry authentication
	// +optional
	// +kubebuilder:default=true
	UseWorkloadIdentity bool `json:"useWorkloadIdentity,omitempty"`
}

// DiscoveryConfigStatus defines the observed state of DiscoveryConfig
type DiscoveryConfigStatus struct {
	// Environments contains the status of each environment
	// +optional
	Environments []EnvironmentStatus `json:"environments,omitempty"`

	// Conditions represent the latest available observations of the DiscoveryConfig's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastSyncTime is the last time discovery was synced
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// EnvironmentStatus represents the status of discovery for a specific environment
type EnvironmentStatus struct {
	// Name is the environment name
	Name string `json:"name"`

	// Connected indicates if connection to the cluster is successful
	Connected bool `json:"connected"`

	// LastSyncTime is the last time this environment was synced
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// DiscoveredResources contains counts of discovered resources
	// +optional
	DiscoveredResources DiscoveredResourceCounts `json:"discoveredResources,omitempty"`

	// Message contains human-readable status information
	// +optional
	Message string `json:"message,omitempty"`

	// Error contains error information if connection failed
	// +optional
	Error string `json:"error,omitempty"`
}

// DiscoveredResourceCounts tracks the number of resources discovered
type DiscoveredResourceCounts struct {
	// MCPServers is the count of discovered MCP servers
	// +optional
	MCPServers int `json:"mcpServers,omitempty"`

	// Agents is the count of discovered agents
	// +optional
	Agents int `json:"agents,omitempty"`

	// Skills is the count of discovered skills
	// +optional
	Skills int `json:"skills,omitempty"`

	// Models is the count of discovered models
	// +optional
	Models int `json:"models,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Environments",type=integer,JSONPath=`.spec.environments`
// +kubebuilder:printcolumn:name="Connected",type=integer,JSONPath=`.status.conditions[?(@.type=="Connected")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DiscoveryConfig is the Schema for the discoveryconfigs API
type DiscoveryConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DiscoveryConfigSpec   `json:"spec,omitempty"`
	Status DiscoveryConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DiscoveryConfigList contains a list of DiscoveryConfig
type DiscoveryConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DiscoveryConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DiscoveryConfig{}, &DiscoveryConfigList{})
}
