package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ModelProvider represents the model provider type
type ModelProvider string

const (
	ModelProviderAnthropic         ModelProvider = "Anthropic"
	ModelProviderOpenAI            ModelProvider = "OpenAI"
	ModelProviderAzureOpenAI       ModelProvider = "AzureOpenAI"
	ModelProviderOllama            ModelProvider = "Ollama"
	ModelProviderGemini            ModelProvider = "Gemini"
	ModelProviderGeminiVertexAI    ModelProvider = "GeminiVertexAI"
	ModelProviderAnthropicVertexAI ModelProvider = "AnthropicVertexAI"
)

// ModelUsageRef represents a reference to an agent using this model
type ModelUsageRef struct {
	// Namespace of the agent
	Namespace string `json:"namespace"`
	// Name of the agent
	Name string `json:"name"`
	// Kind of the resource (e.g., Agent)
	Kind string `json:"kind,omitempty"`
}

// ModelCatalogSpec defines the desired state of ModelCatalog
type ModelCatalogSpec struct {
	// Name is the canonical name of the model config
	Name string `json:"name"`
	// Provider is the model provider (OpenAI, Anthropic, Ollama, etc.)
	Provider string `json:"provider"`
	// Model is the model identifier
	Model string `json:"model"`
	// BaseURL is the API endpoint URL
	// +optional
	BaseURL string `json:"baseUrl,omitempty"`
	// Description of the model configuration
	// +optional
	Description string `json:"description,omitempty"`
	// SourceRef references the deployed ModelConfig resource
	// +optional
	SourceRef *SourceReference `json:"sourceRef,omitempty"`
}

// ModelCatalogStatus defines the observed state of ModelCatalog
type ModelCatalogStatus struct {
	// Published indicates whether this model config is published and visible
	Published bool `json:"published"`
	// PublishedAt is the timestamp when this was published
	// +optional
	PublishedAt *metav1.Time `json:"publishedAt,omitempty"`
	// Status is the lifecycle status (active, deprecated, deleted)
	// +optional
	Status CatalogStatus `json:"status,omitempty"`
	// ManagementType indicates how this resource is managed (external or managed)
	// +optional
	ManagementType ManagementType `json:"managementType,omitempty"`
	// UsedBy lists the agents that reference this model config
	// +optional
	UsedBy []ModelUsageRef `json:"usedBy,omitempty"`
	// Ready indicates if the model config is ready for use
	// For external resources: synced from SourceRef
	// For managed resources: set by RegistryDeployment
	// +optional
	Ready bool `json:"ready,omitempty"`
	// Message provides additional status information
	// +optional
	Message string `json:"message,omitempty"`
	// Conditions represent the latest available observations
	// +optional
	Conditions []CatalogCondition `json:"conditions,omitempty"`
	// ObservedGeneration is the generation last observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=mcat;modelcat
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="Model",type=string,JSONPath=`.spec.model`
// +kubebuilder:printcolumn:name="Published",type=boolean,JSONPath=`.status.published`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ModelCatalog is the Schema for the modelcatalogs API
type ModelCatalog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ModelCatalogSpec   `json:"spec,omitempty"`
	Status ModelCatalogStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ModelCatalogList contains a list of ModelCatalog
type ModelCatalogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ModelCatalog `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ModelCatalog{}, &ModelCatalogList{})
}
