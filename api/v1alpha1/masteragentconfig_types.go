package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventSources configures which event sources the master agent consumes
type EventSources struct {
	// Kubernetes configures Kubernetes event watching
	// +optional
	Kubernetes *KubernetesEventSource `json:"kubernetes,omitempty"`
	// Webhooks enables the POST /v0/agent/events endpoint
	// +optional
	Webhooks *WebhookEventSource `json:"webhooks,omitempty"`
}

// KubernetesEventSource configures watching Kubernetes resources for events
type KubernetesEventSource struct {
	// Enabled enables Kubernetes event watching
	Enabled bool `json:"enabled"`
	// Namespaces to watch. Empty means all namespaces.
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`
	// WatchPods enables watching pod events
	// +optional
	// +kubebuilder:default=true
	WatchPods bool `json:"watchPods,omitempty"`
	// WatchNodes enables watching node events
	// +optional
	// +kubebuilder:default=true
	WatchNodes bool `json:"watchNodes,omitempty"`
	// WatchEvents enables watching Kubernetes Event resources
	// +optional
	// +kubebuilder:default=true
	WatchEvents bool `json:"watchEvents,omitempty"`
}

// WebhookEventSource configures the webhook endpoint for receiving events
type WebhookEventSource struct {
	// Enabled enables the webhook endpoint
	Enabled bool `json:"enabled"`
}

// ModelRefs maps model roles to ModelCatalog spec.name values
type ModelRefs struct {
	// Fast is the ModelCatalog spec.name for fast/cheap operations
	// +optional
	Fast string `json:"fast,omitempty"`
	// Thinking is the ModelCatalog spec.name for complex reasoning
	// +optional
	Thinking string `json:"thinking,omitempty"`
	// Default is the ModelCatalog spec.name used when no specific role is needed
	Default string `json:"default"`
}

// MCPServerRef references an MCP server the agent can connect to
type MCPServerRef struct {
	// Name is a human-readable identifier for this MCP server
	Name string `json:"name"`
	// URL is the streamable-http endpoint of the MCP server
	URL string `json:"url"`
}

// A2AConfig configures the Agent-to-Agent protocol server
type A2AConfig struct {
	// Enabled enables the A2A server
	// +optional
	Enabled bool `json:"enabled,omitempty"`
	// Port is the port to listen on
	// +optional
	// +kubebuilder:default=8084
	Port int `json:"port,omitempty"`
}

// BatchTriageConfig configures event batching and LLM-driven prioritization
type BatchTriageConfig struct {
	// Enabled activates batch triage mode. When enabled, events are accumulated
	// and sent to the model in batches for grouping and prioritization instead of
	// being processed individually.
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`
	// QueueThreshold triggers triage when this many events accumulate in the queue
	// +optional
	// +kubebuilder:default=10
	QueueThreshold int `json:"queueThreshold,omitempty"`
	// WindowSeconds is the max seconds to wait before draining whatever is queued
	// +optional
	// +kubebuilder:default=30
	WindowSeconds int `json:"windowSeconds,omitempty"`
}

// MasterAgentConfigSpec defines the desired state of MasterAgentConfig
type MasterAgentConfigSpec struct {
	// Enabled activates the master agent
	Enabled bool `json:"enabled"`
	// EventSources configures event ingestion
	// +optional
	EventSources EventSources `json:"eventSources,omitempty"`
	// Models maps roles to ModelCatalog entries
	Models ModelRefs `json:"models"`
	// MCPServers lists MCP servers the agent can use as tools
	// +optional
	MCPServers []MCPServerRef `json:"mcpServers,omitempty"`
	// A2A configures the Agent-to-Agent protocol server
	// +optional
	A2A A2AConfig `json:"a2a,omitempty"`
	// MaxConcurrentEvents is the number of parallel event processing workers
	// +optional
	// +kubebuilder:default=5
	MaxConcurrentEvents int `json:"maxConcurrentEvents,omitempty"`
	// EventRetentionMinutes is how long to keep processed events
	// +optional
	// +kubebuilder:default=60
	EventRetentionMinutes int `json:"eventRetentionMinutes,omitempty"`
	// SystemPrompt overrides the default system prompt
	// +optional
	SystemPrompt string `json:"systemPrompt,omitempty"`
	// BatchTriage configures event batching and LLM-driven prioritization.
	// When enabled, events are accumulated and triaged in batches instead of
	// being processed individually, reducing LLM calls during incident storms.
	// +optional
	BatchTriage *BatchTriageConfig `json:"batchTriage,omitempty"`
}

// WorldState represents the agent's running picture of infrastructure
type WorldState struct {
	// LastUpdated is when the world state was last modified
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
	// Summary is a human-readable summary of current infrastructure state
	// +optional
	Summary string `json:"summary,omitempty"`
	// TotalEvents is the total number of events processed
	// +optional
	TotalEvents int64 `json:"totalEvents,omitempty"`
	// PendingEvents is the number of events waiting to be processed
	// +optional
	PendingEvents int `json:"pendingEvents,omitempty"`
	// ActiveIncidents is the count of unresolved incidents
	// +optional
	ActiveIncidents int `json:"activeIncidents,omitempty"`
}

// IncidentStatus represents the lifecycle of an incident
type IncidentStatus string

const (
	IncidentStatusNew           IncidentStatus = "new"
	IncidentStatusInvestigating IncidentStatus = "investigating"
	IncidentStatusMitigated     IncidentStatus = "mitigated"
	IncidentStatusResolved      IncidentStatus = "resolved"
)

// Incident represents a detected infrastructure issue
type Incident struct {
	// ID is a unique identifier for the incident
	ID string `json:"id"`
	// Severity is the incident severity (info, warning, critical)
	Severity string `json:"severity"`
	// Source identifies the origin (e.g., "k8s/pod/namespace/name")
	Source string `json:"source"`
	// Summary is a human-readable description
	Summary string `json:"summary"`
	// FirstSeen is when the incident was first detected
	FirstSeen metav1.Time `json:"firstSeen"`
	// LastSeen is when the incident was last observed
	LastSeen metav1.Time `json:"lastSeen"`
	// Status is the incident lifecycle status
	Status IncidentStatus `json:"status"`
	// Actions lists actions taken by the agent
	// +optional
	Actions []string `json:"actions,omitempty"`
}

// MasterAgentConfigStatus defines the observed state of MasterAgentConfig
type MasterAgentConfigStatus struct {
	// WorldState is the agent's running picture of infrastructure
	// +optional
	WorldState WorldState `json:"worldState,omitempty"`
	// Incidents lists detected infrastructure issues
	// +optional
	Incidents []Incident `json:"incidents,omitempty"`
	// QueueDepth is the current number of events waiting in the queue
	// +optional
	QueueDepth int `json:"queueDepth,omitempty"`
	// LLMAvailable indicates whether the configured LLM is reachable
	// +optional
	LLMAvailable bool `json:"llmAvailable,omitempty"`
	// A2AEndpoint is the URL of the A2A server if running
	// +optional
	A2AEndpoint string `json:"a2aEndpoint,omitempty"`
	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=mac;masteragent
// +kubebuilder:printcolumn:name="Enabled",type=boolean,JSONPath=`.spec.enabled`
// +kubebuilder:printcolumn:name="LLM",type=boolean,JSONPath=`.status.llmAvailable`
// +kubebuilder:printcolumn:name="Queue",type=integer,JSONPath=`.status.queueDepth`
// +kubebuilder:printcolumn:name="Incidents",type=integer,JSONPath=`.status.worldState.activeIncidents`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// MasterAgentConfig is the Schema for the masteragentconfigs API
type MasterAgentConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MasterAgentConfigSpec   `json:"spec,omitempty"`
	Status MasterAgentConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MasterAgentConfigList contains a list of MasterAgentConfig
type MasterAgentConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MasterAgentConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MasterAgentConfig{}, &MasterAgentConfigList{})
}
