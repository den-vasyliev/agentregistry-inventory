package masteragent

import (
	"fmt"
	"strings"
	"sync"
	"time"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorldState maintains a thread-safe running picture of infrastructure state
type WorldState struct {
	mu          sync.RWMutex
	summary     string
	totalEvents int64
	incidents   map[string]*agentregistryv1alpha1.Incident
	lastUpdated time.Time
}

// NewWorldState creates an empty WorldState
func NewWorldState() *WorldState {
	return &WorldState{
		incidents: make(map[string]*agentregistryv1alpha1.Incident),
	}
}

// GetSummary returns the current summary
func (w *WorldState) GetSummary() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.summary == "" {
		return "No events processed yet. Infrastructure state unknown."
	}
	return w.summary
}

// SetSummary updates the world state summary
func (w *WorldState) SetSummary(summary string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.summary = summary
	w.lastUpdated = time.Now()
}

// IncrementEvents increments the total event count
func (w *WorldState) IncrementEvents() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.totalEvents++
	w.lastUpdated = time.Now()
}

// AddOrUpdateIncident creates or updates an incident
func (w *WorldState) AddOrUpdateIncident(id, severity, source, summary string, status agentregistryv1alpha1.IncidentStatus) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := metav1.Now()
	if existing, ok := w.incidents[id]; ok {
		existing.LastSeen = now
		existing.Summary = summary
		existing.Status = status
		existing.Severity = severity
	} else {
		w.incidents[id] = &agentregistryv1alpha1.Incident{
			ID:        id,
			Severity:  severity,
			Source:    source,
			Summary:   summary,
			FirstSeen: now,
			LastSeen:  now,
			Status:    status,
		}
	}
	w.lastUpdated = time.Now()
}

// AddIncidentAction appends an action to an incident
func (w *WorldState) AddIncidentAction(id, action string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if inc, ok := w.incidents[id]; ok {
		inc.Actions = append(inc.Actions, action)
	}
}

// ResolveIncident marks an incident as resolved
func (w *WorldState) ResolveIncident(id string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if inc, ok := w.incidents[id]; ok {
		inc.Status = agentregistryv1alpha1.IncidentStatusResolved
		inc.LastSeen = metav1.Now()
	}
}

// ToStatus converts the world state to a CRD status WorldState
func (w *WorldState) ToStatus(pendingEvents int) agentregistryv1alpha1.WorldState {
	w.mu.RLock()
	defer w.mu.RUnlock()

	activeCount := 0
	for _, inc := range w.incidents {
		if inc.Status != agentregistryv1alpha1.IncidentStatusResolved {
			activeCount++
		}
	}

	var lastUpdated *metav1.Time
	if !w.lastUpdated.IsZero() {
		t := metav1.NewTime(w.lastUpdated)
		lastUpdated = &t
	}

	return agentregistryv1alpha1.WorldState{
		LastUpdated:     lastUpdated,
		Summary:         w.summary,
		TotalEvents:     w.totalEvents,
		PendingEvents:   pendingEvents,
		ActiveIncidents: activeCount,
	}
}

// GetIncidents returns all incidents as a slice
func (w *WorldState) GetIncidents() []agentregistryv1alpha1.Incident {
	w.mu.RLock()
	defer w.mu.RUnlock()

	incidents := make([]agentregistryv1alpha1.Incident, 0, len(w.incidents))
	for _, inc := range w.incidents {
		incidents = append(incidents, *inc)
	}
	return incidents
}

// GetActiveIncidentsSummary returns a formatted summary of active incidents
func (w *WorldState) GetActiveIncidentsSummary() string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var active []string
	for _, inc := range w.incidents {
		if inc.Status != agentregistryv1alpha1.IncidentStatusResolved {
			active = append(active, fmt.Sprintf("- [%s] %s: %s (status: %s)",
				inc.Severity, inc.Source, inc.Summary, inc.Status))
		}
	}
	if len(active) == 0 {
		return "No active incidents."
	}
	return "Active incidents:\n" + strings.Join(active, "\n")
}
