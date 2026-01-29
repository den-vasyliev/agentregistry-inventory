package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MCPServerCatalogSpec defines the desired state of MCPServerCatalog
type MCPServerCatalogSpec struct {
	// Name is the canonical name of the MCP server (e.g., "github/modelcontextprotocol/filesystem")
	Name string `json:"name"`
	// Version is the semantic version of the server
	Version string `json:"version"`
	// Title is a human-readable title for the server
	// +optional
	Title string `json:"title,omitempty"`
	// Description is a human-readable description of the server
	// +optional
	Description string `json:"description,omitempty"`
	// WebsiteURL is the URL to the server's website or documentation
	// +optional
	WebsiteURL string `json:"websiteUrl,omitempty"`
	// Repository is the source code repository information
	// +optional
	Repository *Repository `json:"repository,omitempty"`
	// SourceRef references a deployed MCPServer resource to sync status from
	// +optional
	SourceRef *SourceReference `json:"sourceRef,omitempty"`
	// Packages are the available package configurations
	// +optional
	Packages []Package `json:"packages,omitempty"`
	// Remotes are the remote transport configurations (streamable-http endpoints)
	// +optional
	Remotes []Transport `json:"remotes,omitempty"`
}

// SourceReference points to a deployed resource to monitor
type SourceReference struct {
	// Kind is the resource kind (e.g., MCPServer)
	Kind string `json:"kind"`
	// Name is the resource name
	Name string `json:"name"`
	// Namespace is the resource namespace
	Namespace string `json:"namespace"`
}

// MCPServerCatalogStatus defines the observed state of MCPServerCatalog
type MCPServerCatalogStatus struct {
	// Published indicates whether this server version is published and visible
	Published bool `json:"published"`
	// IsLatest indicates whether this is the latest version of the server
	IsLatest bool `json:"isLatest"`
	// PublishedAt is the timestamp when this version was published
	// +optional
	PublishedAt *metav1.Time `json:"publishedAt,omitempty"`
	// Status is the lifecycle status (active, deprecated, deleted)
	// +optional
	Status CatalogStatus `json:"status,omitempty"`
	// Deployment tracks the runtime deployment info (optional, set by user or discovered)
	// +optional
	Deployment *DeploymentRef `json:"deployment,omitempty"`
	// Conditions represent the latest available observations of the server's state
	// +optional
	Conditions []CatalogCondition `json:"conditions,omitempty"`
	// ObservedGeneration is the generation last observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// DeploymentRef references the runtime deployment (deployment-agnostic)
type DeploymentRef struct {
	// Namespace where the server is deployed
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// ServiceName is the Kubernetes Service name (if applicable)
	// +optional
	ServiceName string `json:"serviceName,omitempty"`
	// URL is the endpoint URL for health checks
	// +optional
	URL string `json:"url,omitempty"`
	// Ready indicates if the deployment is healthy
	// +optional
	Ready bool `json:"ready,omitempty"`
	// LastChecked is when health was last verified
	// +optional
	LastChecked *metav1.Time `json:"lastChecked,omitempty"`
	// Message provides additional deployment status info
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=mcscat;mcservercat
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="Published",type=boolean,JSONPath=`.status.published`
// +kubebuilder:printcolumn:name="Latest",type=boolean,JSONPath=`.status.isLatest`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// MCPServerCatalog is the Schema for the mcpservercatalogs API
type MCPServerCatalog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MCPServerCatalogSpec   `json:"spec,omitempty"`
	Status MCPServerCatalogStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MCPServerCatalogList contains a list of MCPServerCatalog
type MCPServerCatalogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MCPServerCatalog `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MCPServerCatalog{}, &MCPServerCatalogList{})
}
