package masteragent

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

func TestNewWorldState(t *testing.T) {
	ws := NewWorldState()
	require.NotNil(t, ws)

	assert.Equal(t, "No events processed yet. Infrastructure state unknown.", ws.GetSummary())
	assert.Empty(t, ws.GetIncidents())
	assert.Equal(t, "No active incidents.", ws.GetActiveIncidentsSummary())
}

func TestWorldState_SetGetSummary(t *testing.T) {
	ws := NewWorldState()

	ws.SetSummary("All systems operational")
	assert.Equal(t, "All systems operational", ws.GetSummary())

	ws.SetSummary("Degraded performance in us-east-1")
	assert.Equal(t, "Degraded performance in us-east-1", ws.GetSummary())
}

func TestWorldState_IncrementEvents(t *testing.T) {
	ws := NewWorldState()

	ws.IncrementEvents()
	ws.IncrementEvents()
	ws.IncrementEvents()

	status := ws.ToStatus(0)
	assert.Equal(t, int64(3), status.TotalEvents)
	assert.NotNil(t, status.LastUpdated)
}

func TestWorldState_AddOrUpdateIncident(t *testing.T) {
	ws := NewWorldState()

	ws.AddOrUpdateIncident("inc-1", "critical", "k8s/node/worker-1", "Node has MemoryPressure",
		agentregistryv1alpha1.IncidentStatusInvestigating)

	incidents := ws.GetIncidents()
	require.Len(t, incidents, 1)

	inc := incidents[0]
	assert.Equal(t, "inc-1", inc.ID)
	assert.Equal(t, "critical", inc.Severity)
	assert.Equal(t, "k8s/node/worker-1", inc.Source)
	assert.Equal(t, "Node has MemoryPressure", inc.Summary)
	assert.Equal(t, agentregistryv1alpha1.IncidentStatusInvestigating, inc.Status)
	assert.False(t, inc.FirstSeen.IsZero())
	assert.False(t, inc.LastSeen.IsZero())
}

func TestWorldState_UpdateExistingIncident(t *testing.T) {
	ws := NewWorldState()

	ws.AddOrUpdateIncident("inc-1", "warning", "k8s/pod/default/nginx", "Pod restarting",
		agentregistryv1alpha1.IncidentStatusInvestigating)

	incidents := ws.GetIncidents()
	require.Len(t, incidents, 1)
	firstSeen := incidents[0].FirstSeen

	// Update the same incident
	ws.AddOrUpdateIncident("inc-1", "critical", "k8s/pod/default/nginx", "Pod in CrashLoopBackOff",
		agentregistryv1alpha1.IncidentStatusInvestigating)

	incidents = ws.GetIncidents()
	require.Len(t, incidents, 1)

	inc := incidents[0]
	assert.Equal(t, "critical", inc.Severity, "severity should be updated")
	assert.Equal(t, "Pod in CrashLoopBackOff", inc.Summary, "summary should be updated")
	assert.Equal(t, "k8s/pod/default/nginx", inc.Source, "source should be preserved")
	assert.Equal(t, firstSeen, inc.FirstSeen, "firstSeen should not change")
	assert.True(t, !inc.LastSeen.Before(&firstSeen), "lastSeen should be >= firstSeen")
}

func TestWorldState_SeverityNeverDowngrades(t *testing.T) {
	ws := NewWorldState()

	// Create as critical
	ws.AddOrUpdateIncident("inc-1", "critical", "k8s/pod/default/nginx", "OOMKilled",
		agentregistryv1alpha1.IncidentStatusInvestigating)

	// Update with info severity — should NOT downgrade
	ws.AddOrUpdateIncident("inc-1", "info", "k8s/pod/default/nginx", "Endpoint removed",
		agentregistryv1alpha1.IncidentStatusInvestigating)

	incidents := ws.GetIncidents()
	require.Len(t, incidents, 1)
	assert.Equal(t, "critical", incidents[0].Severity, "severity should not downgrade from critical to info")
	assert.Equal(t, "Endpoint removed", incidents[0].Summary, "summary should still update")

	// Update with warning — should NOT downgrade
	ws.AddOrUpdateIncident("inc-1", "warning", "k8s/pod/default/nginx", "Back-off restarting",
		agentregistryv1alpha1.IncidentStatusInvestigating)

	incidents = ws.GetIncidents()
	assert.Equal(t, "critical", incidents[0].Severity, "severity should not downgrade from critical to warning")
}

func TestWorldState_ResolveIncident(t *testing.T) {
	ws := NewWorldState()

	ws.AddOrUpdateIncident("inc-1", "warning", "test", "Test incident",
		agentregistryv1alpha1.IncidentStatusInvestigating)

	ws.ResolveIncident("inc-1")

	incidents := ws.GetIncidents()
	require.Len(t, incidents, 1)
	assert.Equal(t, agentregistryv1alpha1.IncidentStatusResolved, incidents[0].Status)
}

func TestWorldState_ResolveNonExistent(t *testing.T) {
	ws := NewWorldState()
	// Should not panic
	ws.ResolveIncident("does-not-exist")
}

func TestWorldState_AddIncidentAction(t *testing.T) {
	ws := NewWorldState()

	ws.AddOrUpdateIncident("inc-1", "critical", "test", "Test",
		agentregistryv1alpha1.IncidentStatusInvestigating)

	ws.AddIncidentAction("inc-1", "Scaled up replicas to 3")
	ws.AddIncidentAction("inc-1", "Notified on-call team")

	incidents := ws.GetIncidents()
	require.Len(t, incidents, 1)
	assert.Equal(t, []string{"Scaled up replicas to 3", "Notified on-call team"}, incidents[0].Actions)
}

func TestWorldState_AddIncidentAction_NonExistent(t *testing.T) {
	ws := NewWorldState()
	// Should not panic
	ws.AddIncidentAction("does-not-exist", "some action")
}

func TestWorldState_GetIncidents(t *testing.T) {
	ws := NewWorldState()

	ws.AddOrUpdateIncident("inc-1", "warning", "src-1", "Incident 1",
		agentregistryv1alpha1.IncidentStatusInvestigating)
	ws.AddOrUpdateIncident("inc-2", "critical", "src-2", "Incident 2",
		agentregistryv1alpha1.IncidentStatusInvestigating)

	incidents := ws.GetIncidents()
	assert.Len(t, incidents, 2)

	// Verify returned slice is a copy (modifying it doesn't affect internal state)
	incidents[0].Summary = "modified"
	original := ws.GetIncidents()
	for _, inc := range original {
		assert.NotEqual(t, "modified", inc.Summary)
	}
}

func TestWorldState_GetActiveIncidentsSummary(t *testing.T) {
	ws := NewWorldState()

	// No incidents
	assert.Equal(t, "No active incidents.", ws.GetActiveIncidentsSummary())

	// Add active and resolved incidents
	ws.AddOrUpdateIncident("inc-1", "critical", "k8s/node/w1", "Node pressure",
		agentregistryv1alpha1.IncidentStatusInvestigating)
	ws.AddOrUpdateIncident("inc-2", "warning", "k8s/pod/nginx", "Pod restart",
		agentregistryv1alpha1.IncidentStatusInvestigating)
	ws.ResolveIncident("inc-2")

	summary := ws.GetActiveIncidentsSummary()
	assert.True(t, strings.HasPrefix(summary, "Active incidents:"))
	assert.Contains(t, summary, "critical")
	assert.Contains(t, summary, "Node pressure")
	// Resolved incident should not appear
	assert.NotContains(t, summary, "Pod restart")
}

func TestWorldState_ToStatus(t *testing.T) {
	ws := NewWorldState()

	// Empty state
	status := ws.ToStatus(0)
	assert.Equal(t, "", status.Summary)
	assert.Equal(t, int64(0), status.TotalEvents)
	assert.Equal(t, 0, status.PendingEvents)
	assert.Equal(t, 0, status.ActiveIncidents)
	assert.Nil(t, status.LastUpdated)

	// With data
	ws.SetSummary("All good")
	ws.IncrementEvents()
	ws.IncrementEvents()
	ws.AddOrUpdateIncident("inc-1", "warning", "src", "test",
		agentregistryv1alpha1.IncidentStatusInvestigating)
	ws.AddOrUpdateIncident("inc-2", "info", "src", "resolved",
		agentregistryv1alpha1.IncidentStatusResolved) // use AddOrUpdate then resolve
	ws.ResolveIncident("inc-2")

	status = ws.ToStatus(5)
	assert.Equal(t, "All good", status.Summary)
	assert.Equal(t, int64(2), status.TotalEvents)
	assert.Equal(t, 5, status.PendingEvents)
	assert.Equal(t, 1, status.ActiveIncidents, "only non-resolved incidents count")
	assert.NotNil(t, status.LastUpdated)
}

func TestWorldState_Concurrent(t *testing.T) {
	ws := NewWorldState()
	var wg sync.WaitGroup

	// Writers
	for i := range 10 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range 20 {
				ws.SetSummary("summary update")
				ws.IncrementEvents()
				ws.AddOrUpdateIncident("inc", "warning", "src", "test",
					agentregistryv1alpha1.IncidentStatusInvestigating)
				if j%5 == 0 {
					ws.ResolveIncident("inc")
				}
				ws.AddIncidentAction("inc", "action")
			}
		}(i)
	}

	// Readers
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				ws.GetSummary()
				ws.GetIncidents()
				ws.GetActiveIncidentsSummary()
				ws.ToStatus(0)
			}
		}()
	}

	wg.Wait()
	// No race conditions = success
}
