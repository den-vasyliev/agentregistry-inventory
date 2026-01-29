package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceType represents the type of resource being deployed
type ResourceType string

const (
	// ResourceTypeMCP indicates an MCP server deployment
	ResourceTypeMCP ResourceType = "mcp"
	// ResourceTypeAgent indicates an agent deployment
	ResourceTypeAgent ResourceType = "agent"
)

// RuntimeType represents the deployment runtime
type RuntimeType string

const (
	// RuntimeTypeLocal indicates local deployment (Docker Compose)
	RuntimeTypeLocal RuntimeType = "local"
	// RuntimeTypeKubernetes indicates Kubernetes deployment
	RuntimeTypeKubernetes RuntimeType = "kubernetes"
)

// DeploymentPhase represents the current phase of a deployment
type DeploymentPhase string

const (
	// DeploymentPhasePending indicates the deployment is pending
	DeploymentPhasePending DeploymentPhase = "Pending"
	// DeploymentPhaseRunning indicates the deployment is running
	DeploymentPhaseRunning DeploymentPhase = "Running"
	// DeploymentPhaseFailed indicates the deployment has failed
	DeploymentPhaseFailed DeploymentPhase = "Failed"
)

// RegistryDeploymentSpec defines the desired state of RegistryDeployment
type RegistryDeploymentSpec struct {
	// ResourceName is the name of the resource in the catalog (matches spec.name in catalog CRs)
	ResourceName string `json:"resourceName"`
	// Version is the version of the resource to deploy
	Version string `json:"version"`
	// ResourceType is the type of resource (mcp, agent)
	ResourceType ResourceType `json:"resourceType"`
	// Runtime is the deployment runtime (local, kubernetes)
	Runtime RuntimeType `json:"runtime"`
	// PreferRemote indicates whether to prefer remote transport when available
	// +optional
	PreferRemote bool `json:"preferRemote,omitempty"`
	// Config contains deployment configuration (environment variables, etc.)
	// +optional
	Config map[string]string `json:"config,omitempty"`
	// Namespace is the target namespace for Kubernetes deployments
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// RegistryDeploymentStatus defines the observed state of RegistryDeployment
type RegistryDeploymentStatus struct {
	// Phase is the current deployment phase
	// +optional
	Phase DeploymentPhase `json:"phase,omitempty"`
	// DeployedAt is the timestamp when the deployment was created
	// +optional
	DeployedAt *metav1.Time `json:"deployedAt,omitempty"`
	// UpdatedAt is the timestamp when the deployment was last updated
	// +optional
	UpdatedAt *metav1.Time `json:"updatedAt,omitempty"`
	// Message is a human-readable message about the current status
	// +optional
	Message string `json:"message,omitempty"`
	// ManagedResources lists the Kubernetes resources created by this deployment
	// +optional
	ManagedResources []ManagedResource `json:"managedResources,omitempty"`
	// Conditions represent the latest available observations of the deployment's state
	// +optional
	Conditions []CatalogCondition `json:"conditions,omitempty"`
	// ObservedGeneration is the generation last observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// ManagedResource represents a Kubernetes resource managed by a deployment
type ManagedResource struct {
	// APIVersion is the API version of the resource
	APIVersion string `json:"apiVersion"`
	// Kind is the kind of the resource
	Kind string `json:"kind"`
	// Name is the name of the resource
	Name string `json:"name"`
	// Namespace is the namespace of the resource
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=regdeploy;rdeploy
// +kubebuilder:printcolumn:name="Resource",type=string,JSONPath=`.spec.resourceName`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.resourceType`
// +kubebuilder:printcolumn:name="Runtime",type=string,JSONPath=`.spec.runtime`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// RegistryDeployment is the Schema for the registrydeployments API
type RegistryDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RegistryDeploymentSpec   `json:"spec,omitempty"`
	Status RegistryDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RegistryDeploymentList contains a list of RegistryDeployment
type RegistryDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RegistryDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RegistryDeployment{}, &RegistryDeploymentList{})
}
