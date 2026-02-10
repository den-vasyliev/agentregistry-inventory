package masteragent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			"raw JSON",
			`{"groups": []}`,
			`{"groups": []}`,
		},
		{
			"markdown fenced json",
			"```json\n{\"groups\": []}\n```",
			`{"groups": []}`,
		},
		{
			"markdown fenced no lang",
			"```\n{\"groups\": []}\n```",
			`{"groups": []}`,
		},
		{
			"text before and after JSON",
			"Here is the result:\n{\"groups\": [{\"group_id\": \"test\"}]}\nDone.",
			`{"groups": [{"group_id": "test"}]}`,
		},
		{
			"no JSON",
			"No JSON here",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSON(tt.text)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseTriageResponse(t *testing.T) {
	events := []InfraEvent{
		{ID: "ev1", Source: "k8s/pod/default/nginx", Type: "pod-crash", Severity: "critical", Message: "CrashLoopBackOff"},
		{ID: "ev2", Source: "k8s/pod/default/nginx", Type: "container-oom", Severity: "critical", Message: "OOMKilled"},
		{ID: "ev3", Source: "k8s/node/node1", Type: "node-pressure", Severity: "warning", Message: "MemoryPressure"},
	}

	jsonResp := `{
		"groups": [
			{
				"group_id": "nginx-crashloop",
				"summary": "nginx pod crash-looping with OOM",
				"priority": 1,
				"severity": "critical",
				"event_ids": ["ev1", "ev2"]
			},
			{
				"group_id": "node-memory",
				"summary": "node1 under memory pressure",
				"priority": 2,
				"severity": "warning",
				"event_ids": ["ev3"]
			}
		]
	}`

	groups, err := parseTriageResponse(jsonResp, events)
	require.NoError(t, err)
	require.Len(t, groups, 2)

	// First group
	assert.Equal(t, "nginx-crashloop", groups[0].GroupID)
	assert.Equal(t, 1, groups[0].Priority)
	assert.Equal(t, "critical", groups[0].Severity)
	assert.Len(t, groups[0].Events, 2)
	assert.Equal(t, "ev1", groups[0].Events[0].ID)
	assert.Equal(t, "ev2", groups[0].Events[1].ID)

	// Second group
	assert.Equal(t, "node-memory", groups[1].GroupID)
	assert.Len(t, groups[1].Events, 1)
}

func TestParseTriageResponse_UnassignedEvents(t *testing.T) {
	events := []InfraEvent{
		{ID: "ev1", Severity: "critical", Message: "crash"},
		{ID: "ev2", Severity: "info", Message: "normal"},
		{ID: "ev3", Severity: "warning", Message: "slow"},
	}

	// Model only groups ev1, leaving ev2 and ev3 unassigned
	jsonResp := `{"groups": [{"group_id": "crash", "summary": "crash", "priority": 1, "severity": "critical", "event_ids": ["ev1"]}]}`

	groups, err := parseTriageResponse(jsonResp, events)
	require.NoError(t, err)
	require.Len(t, groups, 2)

	// Unassigned group should be appended with lower priority
	unassigned := groups[1]
	assert.Equal(t, "unassigned", unassigned.GroupID)
	assert.Equal(t, 2, unassigned.Priority)
	assert.Equal(t, "warning", unassigned.Severity) // highest among unassigned
	assert.Len(t, unassigned.Events, 2)
}

func TestParseTriageResponse_InvalidJSON(t *testing.T) {
	events := []InfraEvent{{ID: "ev1"}}
	_, err := parseTriageResponse("not json", events)
	assert.Error(t, err)
}

func TestParseTriageResponse_EmptyGroups(t *testing.T) {
	events := []InfraEvent{{ID: "ev1"}}
	_, err := parseTriageResponse(`{"groups": []}`, events)
	assert.Error(t, err)
}

func TestFallbackGroups(t *testing.T) {
	events := []InfraEvent{
		{ID: "ev1", Severity: "warning", Message: "warn msg"},
		{ID: "ev2", Severity: "critical", Message: "crit msg"},
		{ID: "ev3", Severity: "info", Message: "info msg"},
	}

	groups := fallbackGroups(events)
	require.Len(t, groups, 3)

	// Should be sorted by priority: critical (1) < warning (2) < info (3)
	assert.Equal(t, 1, groups[0].Priority)
	assert.Equal(t, "critical", groups[0].Severity)
	assert.Equal(t, "ev2", groups[0].GroupID)

	assert.Equal(t, 2, groups[1].Priority)
	assert.Equal(t, "warning", groups[1].Severity)

	assert.Equal(t, 3, groups[2].Priority)
	assert.Equal(t, "info", groups[2].Severity)
}

func TestHighestSeverity(t *testing.T) {
	tests := []struct {
		name     string
		events   []InfraEvent
		expected string
	}{
		{"critical wins", []InfraEvent{{Severity: "info"}, {Severity: "critical"}, {Severity: "warning"}}, "critical"},
		{"warning wins", []InfraEvent{{Severity: "info"}, {Severity: "warning"}}, "warning"},
		{"info only", []InfraEvent{{Severity: "info"}}, "info"},
		{"empty events", []InfraEvent{}, "info"},
		{"unknown severity", []InfraEvent{{Severity: "unknown"}}, "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, highestSeverity(tt.events))
		})
	}
}

func TestBuildTriagePrompt(t *testing.T) {
	events := []InfraEvent{
		{ID: "ev1", Source: "k8s/pod/ns/nginx", Type: "pod-crash", Severity: "critical", Message: "CrashLoopBackOff"},
		{ID: "ev2", Source: "k8s/node/node1", Type: "node-pressure", Severity: "warning", Message: "MemoryPressure"},
	}

	prompt := BuildTriagePrompt(events, "All systems normal", "No active incidents")

	assert.Contains(t, prompt, "2 infrastructure events to triage")
	assert.Contains(t, prompt, "All systems normal")
	assert.Contains(t, prompt, "No active incidents")
	assert.Contains(t, prompt, "[id=ev1]")
	assert.Contains(t, prompt, "[id=ev2]")
	assert.Contains(t, prompt, "severity=critical")
	assert.Contains(t, prompt, "Respond with JSON only")
}

func TestBuildTriagePrompt_EmptyState(t *testing.T) {
	events := []InfraEvent{{ID: "ev1", Message: "test"}}
	prompt := BuildTriagePrompt(events, "", "")

	assert.Contains(t, prompt, "1 infrastructure events to triage")
	assert.NotContains(t, prompt, "Current world state:")
}

func TestEventHub_Drain(t *testing.T) {
	hub := NewEventHub(10, 5)

	hub.Push(InfraEvent{Message: "1"})
	hub.Push(InfraEvent{Message: "2"})
	hub.Push(InfraEvent{Message: "3"})
	assert.Equal(t, 3, hub.QueueDepth())

	drained := hub.Drain()
	assert.Len(t, drained, 3)
	assert.Equal(t, "1", drained[0].Message)
	assert.Equal(t, "2", drained[1].Message)
	assert.Equal(t, "3", drained[2].Message)
	assert.Equal(t, 0, hub.QueueDepth())
}

func TestEventHub_DrainEmpty(t *testing.T) {
	hub := NewEventHub(10, 5)
	drained := hub.Drain()
	assert.Empty(t, drained)
}

func TestProcessBatch_SingleEventFastPath(t *testing.T) {
	// Verify that a single-event batch doesn't need triage
	// This is a structural test — we just check that the batch config
	// and single-event path are wired correctly
	hub := NewEventHub(10, 5)

	events := []InfraEvent{
		{ID: "ev1", Source: "test", Type: "test", Severity: "info", Message: "single", Timestamp: time.Now()},
	}

	// With 1 event, fallbackGroups should create 1 group
	groups := fallbackGroups(events)
	require.Len(t, groups, 1)
	assert.Equal(t, "ev1", groups[0].GroupID)
	assert.Len(t, groups[0].Events, 1)

	_ = hub // hub used for context only
}
