package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SkillRepository represents the repository information for a skill
type SkillRepository struct {
	// URL is the repository URL
	// +optional
	URL string `json:"url,omitempty"`
	// Source is the repository hosting service identifier
	// +optional
	Source string `json:"source,omitempty"`
}

// SkillPackage represents a package configuration for skills
type SkillPackage struct {
	// RegistryType indicates how to download packages
	RegistryType string `json:"registryType"`
	// Identifier is the package identifier
	Identifier string `json:"identifier"`
	// Version is the package version
	// +optional
	Version string `json:"version,omitempty"`
	// Transport is the transport type configuration
	// +optional
	Transport *SkillPackageTransport `json:"transport,omitempty"`
}

// SkillPackageTransport represents the transport configuration for a skill package
type SkillPackageTransport struct {
	// Type is the transport type
	Type string `json:"type"`
}

// SkillRemote represents a remote endpoint for a skill
type SkillRemote struct {
	// URL is the remote endpoint URL
	URL string `json:"url"`
}

// SkillCatalogSpec defines the desired state of SkillCatalog
type SkillCatalogSpec struct {
	// Name is the canonical name of the skill
	Name string `json:"name"`
	// Version is the semantic version of the skill
	Version string `json:"version"`
	// Title is a human-readable title for the skill
	// +optional
	Title string `json:"title,omitempty"`
	// Category is the skill category (e.g., "code-generation", "testing")
	// +optional
	Category string `json:"category,omitempty"`
	// Description is a human-readable description of the skill
	// +optional
	Description string `json:"description,omitempty"`
	// WebsiteURL is the URL to the skill's website or documentation
	// +optional
	WebsiteURL string `json:"websiteUrl,omitempty"`
	// Repository is the source code repository information
	// +optional
	Repository *SkillRepository `json:"repository,omitempty"`
	// Packages are the available package configurations
	// +optional
	Packages []SkillPackage `json:"packages,omitempty"`
	// Remotes are the remote endpoints for the skill
	// +optional
	Remotes []SkillRemote `json:"remotes,omitempty"`
}

// SkillUsageRef represents a reference to an agent using this skill
type SkillUsageRef struct {
	// Namespace of the agent
	Namespace string `json:"namespace"`
	// Name of the agent
	Name string `json:"name"`
	// Kind of the resource (e.g., Agent)
	Kind string `json:"kind,omitempty"`
}

// SkillCatalogStatus defines the observed state of SkillCatalog
type SkillCatalogStatus struct {
	// Published indicates whether this skill version is published and visible
	Published bool `json:"published"`
	// IsLatest indicates whether this is the latest version of the skill
	IsLatest bool `json:"isLatest"`
	// PublishedAt is the timestamp when this version was published
	// +optional
	PublishedAt *metav1.Time `json:"publishedAt,omitempty"`
	// Status is the lifecycle status (active, deprecated, deleted)
	// +optional
	Status CatalogStatus `json:"status,omitempty"`
	// UsedBy lists the agents that reference this skill
	// +optional
	UsedBy []SkillUsageRef `json:"usedBy,omitempty"`
	// Conditions represent the latest available observations of the skill's state
	// +optional
	Conditions []CatalogCondition `json:"conditions,omitempty"`
	// ObservedGeneration is the generation last observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=skcat;skillcat
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="Category",type=string,JSONPath=`.spec.category`
// +kubebuilder:printcolumn:name="Published",type=boolean,JSONPath=`.status.published`
// +kubebuilder:printcolumn:name="Latest",type=boolean,JSONPath=`.status.isLatest`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// SkillCatalog is the Schema for the skillcatalogs API
type SkillCatalog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SkillCatalogSpec   `json:"spec,omitempty"`
	Status SkillCatalogStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SkillCatalogList contains a list of SkillCatalog
type SkillCatalogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SkillCatalog `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SkillCatalog{}, &SkillCatalogList{})
}
