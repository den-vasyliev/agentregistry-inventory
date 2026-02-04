package v1alpha1

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Repository represents the source code repository information
type Repository struct {
	// URL is the repository URL for browsing source code
	// +optional
	URL string `json:"url,omitempty"`
	// Source is the repository hosting service identifier (e.g., "github")
	// +optional
	Source string `json:"source,omitempty"`
	// ID is the repository identifier from the hosting service
	// +optional
	ID string `json:"id,omitempty"`
	// Subfolder is an optional relative path to the server location within a monorepo
	// +optional
	Subfolder string `json:"subfolder,omitempty"`
}

// Transport represents the transport protocol configuration
type Transport struct {
	// Type is the transport type (stdio or streamable-http)
	Type string `json:"type"`
	// URL is the URL for streamable-http transport
	// +optional
	URL string `json:"url,omitempty"`
	// Headers are HTTP headers for streamable-http transport
	// +optional
	Headers []KeyValueInput `json:"headers,omitempty"`
}

// KeyValueInput represents a key-value pair with optional description
type KeyValueInput struct {
	// Name is the key name
	Name string `json:"name"`
	// Description is an optional description of the input
	// +optional
	Description string `json:"description,omitempty"`
	// Value is the value (can use environment variable substitution)
	// +optional
	Value string `json:"value,omitempty"`
	// Required indicates if this input is required
	// +optional
	Required bool `json:"required,omitempty"`
}

// Argument represents a command-line argument
type Argument struct {
	// Name is the argument name
	Name string `json:"name"`
	// Type is the argument type (positional, flag)
	// +optional
	Type string `json:"type,omitempty"`
	// Description is an optional description
	// +optional
	Description string `json:"description,omitempty"`
	// Value is the default value
	// +optional
	Value string `json:"value,omitempty"`
	// Required indicates if this argument is required
	// +optional
	Required bool `json:"required,omitempty"`
	// Multiple indicates if the argument can be specified multiple times
	// +optional
	Multiple bool `json:"multiple,omitempty"`
}

// Package represents a package configuration for MCP servers
type Package struct {
	// RegistryType indicates how to download packages (e.g., "npm", "pypi", "oci", "nuget", "mcpb")
	RegistryType string `json:"registryType"`
	// RegistryBaseURL is the base URL of the package registry
	// +optional
	RegistryBaseURL string `json:"registryBaseUrl,omitempty"`
	// Identifier is the package identifier
	Identifier string `json:"identifier"`
	// Version is the package version
	// +optional
	Version string `json:"version,omitempty"`
	// FileSHA256 is the SHA-256 hash for integrity verification
	// +optional
	FileSHA256 string `json:"fileSha256,omitempty"`
	// RuntimeHint suggests the appropriate runtime for the package
	// +optional
	RuntimeHint string `json:"runtimeHint,omitempty"`
	// Transport is the transport protocol configuration
	Transport Transport `json:"transport"`
	// RuntimeArguments are passed to the package's runtime command
	// +optional
	RuntimeArguments []Argument `json:"runtimeArguments,omitempty"`
	// PackageArguments are passed to the package's binary
	// +optional
	PackageArguments []Argument `json:"packageArguments,omitempty"`
	// EnvironmentVariables are set when running the package
	// +optional
	EnvironmentVariables []KeyValueInput `json:"environmentVariables,omitempty"`
}

// CatalogStatus is the deployment status of a catalog resource
type CatalogStatus string

const (
	// CatalogStatusActive indicates the resource is active and available
	CatalogStatusActive CatalogStatus = "active"
	// CatalogStatusDeprecated indicates the resource is deprecated
	CatalogStatusDeprecated CatalogStatus = "deprecated"
	// CatalogStatusDeleted indicates the resource is marked for deletion
	CatalogStatusDeleted CatalogStatus = "deleted"
)

// CatalogConditionType represents the type of condition
type CatalogConditionType string

const (
	// CatalogConditionReady indicates whether the catalog entry is ready
	CatalogConditionReady CatalogConditionType = "Ready"
	// CatalogConditionPublished indicates whether the catalog entry is published
	CatalogConditionPublished CatalogConditionType = "Published"
)

// Common label keys used across all catalog resources
const (
	// LabelResourceUID is the unique identifier for a resource in format "name-env-ver"
	// This allows distinguishing same resources across environments and versions
	LabelResourceUID = "agentregistry.dev/resource-uid"
	// LabelResourceName is the canonical resource name
	LabelResourceName = "agentregistry.dev/resource-name"
	// LabelResourceVersion is the resource version
	LabelResourceVersion = "agentregistry.dev/resource-version"
	// LabelResourceEnvironment is the environment (dev, staging, prod, etc.)
	LabelResourceEnvironment = "agentregistry.dev/resource-environment"
	// LabelResourceSource indicates how the resource was created (discovery, manual, deployment)
	LabelResourceSource = "agentregistry.dev/resource-source"
	// LabelManagedBy indicates the managing component
	LabelManagedBy = "agentregistry.dev/managed-by"
)

// ResourceSource values for LabelResourceSource
const (
	ResourceSourceDiscovery  = "discovery"
	ResourceSourceManual     = "manual"
	ResourceSourceDeployment = "deployment"
	ResourceSourceImport     = "import"
)

// ManagementType indicates how the resource is managed
type ManagementType string

const (
	// ManagementTypeExternal indicates the resource was discovered from an external cluster
	// Status comes from the original resource, cannot be deployed
	ManagementTypeExternal ManagementType = "external"
	// ManagementTypeManaged indicates the resource was created in the catalog first
	// Status comes from RegistryDeployment, can be deployed
	ManagementTypeManaged ManagementType = "managed"
)

// GenerateResourceUID generates a unique resource identifier in format "name-env-ver"
// This ensures uniqueness across environments and versions
func GenerateResourceUID(name, environment, version string) string {
	// Sanitize inputs (replace dots with dashes in version)
	version = strings.ReplaceAll(version, ".", "-")
	uid := fmt.Sprintf("%s-%s-%s", name, environment, version)
	// Ensure lowercase and valid k8s label value
	uid = strings.ToLower(uid)
	uid = strings.ReplaceAll(uid, "_", "-")
	uid = strings.ReplaceAll(uid, "/", "-")
	// Trim if too long for label value (63 chars max)
	if len(uid) > 63 {
		uid = uid[:63]
	}
	return strings.Trim(uid, "-")
}

// CatalogCondition contains details for the current condition of this resource
type CatalogCondition struct {
	// Type is the type of condition
	Type CatalogConditionType `json:"type"`
	// Status is the status of the condition (True, False, Unknown)
	Status metav1.ConditionStatus `json:"status"`
	// LastTransitionTime is the last time the condition transitioned
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// Reason is a brief reason for the condition's last transition
	// +optional
	Reason string `json:"reason,omitempty"`
	// Message is a human-readable message indicating details about the transition
	// +optional
	Message string `json:"message,omitempty"`
}
