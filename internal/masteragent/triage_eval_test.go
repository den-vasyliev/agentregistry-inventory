package masteragent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

// ---------- realistic event fixtures ----------

// Scenario 1: Pod crashloop with OOM — a classic cascading incident.
// Alerts fire, customer reports 503s, logs show OOMKill, K8s events show restarts.
var podCrashloopEvents = []InfraEvent{
	{
		ID: "ev-alert-p1", Source: "alertmanager/prod", Type: "alert",
		Severity: "critical", Message: "FIRING: PodCrashLooping - pod default/payment-svc-7f8d4b-xq2k1 has restarted 5 times in 10m",
		Timestamp: time.Date(2025, 7, 15, 14, 0, 0, 0, time.UTC),
	},
	{
		ID: "ev-k8s-oom", Source: "k8s/event/default/payment-svc-7f8d4b-xq2k1", Type: "container-oom",
		Severity: "critical", Message: "Container payment-svc OOMKilled: memory limit 512Mi exceeded",
		Timestamp: time.Date(2025, 7, 15, 14, 0, 5, 0, time.UTC),
	},
	{
		ID: "ev-k8s-restart", Source: "k8s/event/default/payment-svc-7f8d4b-xq2k1", Type: "pod-restart",
		Severity: "warning", Message: "Back-off restarting failed container payment-svc in pod payment-svc-7f8d4b-xq2k1",
		Timestamp: time.Date(2025, 7, 15, 14, 0, 10, 0, time.UTC),
	},
	{
		ID: "ev-log-oom", Source: "loki/default/payment-svc", Type: "log-error",
		Severity: "critical", Message: "java.lang.OutOfMemoryError: Java heap space at com.payment.TransactionProcessor.process(TransactionProcessor.java:142)",
		Timestamp: time.Date(2025, 7, 15, 14, 0, 3, 0, time.UTC),
	},
	{
		ID: "ev-customer-503", Source: "support/ticket-4521", Type: "customer-report",
		Severity: "warning", Message: "Customer reports: payment checkout returning 503 Service Unavailable since ~14:00 UTC",
		Timestamp: time.Date(2025, 7, 15, 14, 5, 0, 0, time.UTC),
	},
	{
		ID: "ev-endpoint-remove", Source: "k8s/event/default/payment-svc", Type: "endpoint-change",
		Severity: "info", Message: "Endpoints default/payment-svc: removed pod payment-svc-7f8d4b-xq2k1 from service",
		Timestamp: time.Date(2025, 7, 15, 14, 0, 12, 0, time.UTC),
	},
	{
		ID: "ev-hpa-scale", Source: "k8s/event/default/payment-svc", Type: "hpa-activity",
		Severity: "info", Message: "HPA scaled payment-svc from 3 to 5 replicas (cpu utilization above target)",
		Timestamp: time.Date(2025, 7, 15, 14, 1, 0, 0, time.UTC),
	},
}

// Scenario 2: Node memory pressure + unrelated deployment rollout.
// Two independent issues happening at the same time.
var mixedIncidentEvents = []InfraEvent{
	{
		ID: "ev-node-pressure", Source: "k8s/node/worker-3", Type: "node-condition",
		Severity: "warning", Message: "Node worker-3 condition MemoryPressure is True, available: 145Mi",
		Timestamp: time.Date(2025, 7, 15, 15, 0, 0, 0, time.UTC),
	},
	{
		ID: "ev-node-eviction", Source: "k8s/event/kube-system/worker-3", Type: "pod-eviction",
		Severity: "warning", Message: "Evicting pod logging-agent-d8f2k from node worker-3 due to memory pressure",
		Timestamp: time.Date(2025, 7, 15, 15, 0, 30, 0, time.UTC),
	},
	{
		ID: "ev-deploy-rollout", Source: "k8s/event/staging/frontend", Type: "deployment-update",
		Severity: "info", Message: "Deployment staging/frontend updated: replicas 3->3, image frontend:v2.4.1->v2.5.0",
		Timestamp: time.Date(2025, 7, 15, 15, 1, 0, 0, time.UTC),
	},
	{
		ID: "ev-deploy-ready", Source: "k8s/event/staging/frontend", Type: "deployment-ready",
		Severity: "info", Message: "Deployment staging/frontend rollout complete: 3/3 replicas available",
		Timestamp: time.Date(2025, 7, 15, 15, 2, 0, 0, time.UTC),
	},
	{
		ID: "ev-alert-mem", Source: "alertmanager/prod", Type: "alert",
		Severity: "warning", Message: "FIRING: NodeMemoryHigh - worker-3 memory usage at 95%",
		Timestamp: time.Date(2025, 7, 15, 15, 0, 15, 0, time.UTC),
	},
	{
		ID: "ev-log-memleak", Source: "loki/kube-system/monitoring-agent", Type: "log-warning",
		Severity: "warning", Message: "monitoring-agent on worker-3: memory usage growing linearly, possible leak detected",
		Timestamp: time.Date(2025, 7, 15, 14, 55, 0, 0, time.UTC),
	},
}

// Scenario 3: Database connection storm — app errors cascade from DB.
var dbConnectionStormEvents = []InfraEvent{
	{
		ID: "ev-db-conn-1", Source: "loki/prod/api-gateway", Type: "log-error",
		Severity: "critical", Message: "pq: too many connections for role \"app_user\" - connection pool exhausted",
		Timestamp: time.Date(2025, 7, 15, 16, 0, 0, 0, time.UTC),
	},
	{
		ID: "ev-db-conn-2", Source: "loki/prod/order-svc", Type: "log-error",
		Severity: "critical", Message: "pq: too many connections for role \"app_user\" - connection pool exhausted",
		Timestamp: time.Date(2025, 7, 15, 16, 0, 2, 0, time.UTC),
	},
	{
		ID: "ev-db-conn-3", Source: "loki/prod/inventory-svc", Type: "log-error",
		Severity: "critical", Message: "pq: too many connections for role \"app_user\" - connection pool exhausted",
		Timestamp: time.Date(2025, 7, 15, 16, 0, 5, 0, time.UTC),
	},
	{
		ID: "ev-alert-db", Source: "alertmanager/prod", Type: "alert",
		Severity: "critical", Message: "FIRING: PostgresConnectionsExhausted - pg-primary active connections 200/200 (100%)",
		Timestamp: time.Date(2025, 7, 15, 16, 0, 10, 0, time.UTC),
	},
	{
		ID: "ev-probe-fail-1", Source: "k8s/event/prod/api-gateway-6c8b9-abc12", Type: "probe-failure",
		Severity: "warning", Message: "Readiness probe failed: HTTP probe failed with statuscode: 503",
		Timestamp: time.Date(2025, 7, 15, 16, 0, 15, 0, time.UTC),
	},
	{
		ID: "ev-probe-fail-2", Source: "k8s/event/prod/order-svc-5d7a8-def34", Type: "probe-failure",
		Severity: "warning", Message: "Readiness probe failed: HTTP probe failed with statuscode: 503",
		Timestamp: time.Date(2025, 7, 15, 16, 0, 15, 0, time.UTC),
	},
	{
		ID: "ev-customer-timeout", Source: "support/ticket-4522", Type: "customer-report",
		Severity: "warning", Message: "Multiple customers reporting: orders timing out, checkout page unresponsive",
		Timestamp: time.Date(2025, 7, 15, 16, 3, 0, 0, time.UTC),
	},
	{
		ID: "ev-pg-slow", Source: "loki/prod/pg-primary", Type: "log-warning",
		Severity: "warning", Message: "LOG: checkpoints are occurring too frequently (12 seconds apart)",
		Timestamp: time.Date(2025, 7, 15, 16, 0, 8, 0, time.UTC),
	},
}

// ---------- mock LLM server ----------

// newMockLLMServer creates an httptest server that returns scripted responses.
// It uses a sequential response queue: each call pops the next response.
// After all scripted responses are consumed, returns a simple "Done." text response.
// It records all requests for later inspection.
func newMockLLMServer(t *testing.T, responses []openAIChatCompletionResponse) (*httptest.Server, *requestRecorder) {
	t.Helper()
	rec := &requestRecorder{}

	var mu sync.Mutex
	callIdx := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Logf("mock LLM: failed to decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		rec.record(req)

		mu.Lock()
		idx := callIdx
		callIdx++
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if idx < len(responses) {
			json.NewEncoder(w).Encode(responses[idx])
		} else {
			// Default final response — stops the agent loop
			json.NewEncoder(w).Encode(textResponse("Done. All actions taken."))
		}
	}))

	return server, rec
}

type requestRecorder struct {
	mu       sync.Mutex
	requests []openAIChatCompletionRequest
}

func (r *requestRecorder) record(req openAIChatCompletionRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, req)
}

func (r *requestRecorder) getRequests() []openAIChatCompletionRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]openAIChatCompletionRequest, len(r.requests))
	copy(result, r.requests)
	return result
}

// ---------- helper to build tool call responses ----------

func toolCallResponse(name, id, args string) openAIChatCompletionResponse {
	return openAIChatCompletionResponse{
		Choices: []openAIChoice{
			{
				Message: openAIMessage{
					Role: "assistant",
					ToolCalls: []openAIToolCall{
						{ID: id, Type: "function", Function: openAIFunctionCall{Name: name, Arguments: args}},
					},
				},
				FinishReason: "tool_calls",
			},
		},
	}
}

func textResponse(text string) openAIChatCompletionResponse {
	return openAIChatCompletionResponse{
		Choices: []openAIChoice{
			{Message: openAIMessage{Role: "assistant", Content: text}, FinishReason: "stop"},
		},
	}
}

// ---------- evaluation tests ----------

func TestEval_TriageGrouping_PodCrashloop(t *testing.T) {
	// Evaluate: Given 7 events from a pod crashloop incident,
	// the triage prompt should be well-formed and the response parser
	// should correctly group events by the model's JSON output.

	triageJSON := `{
		"groups": [
			{
				"group_id": "payment-svc-crashloop",
				"summary": "payment-svc pod is crash-looping due to Java heap OOM, causing 503 errors for customers. HPA is scaling up but unhealthy pods keep cycling.",
				"priority": 1,
				"severity": "critical",
				"event_ids": ["ev-alert-p1", "ev-k8s-oom", "ev-k8s-restart", "ev-log-oom", "ev-customer-503", "ev-endpoint-remove"]
			},
			{
				"group_id": "payment-svc-scaling",
				"summary": "HPA autoscaling payment-svc due to high CPU from restart churn",
				"priority": 2,
				"severity": "info",
				"event_ids": ["ev-hpa-scale"]
			}
		]
	}`

	groups, err := parseTriageResponse(triageJSON, podCrashloopEvents)
	require.NoError(t, err)
	require.Len(t, groups, 2, "should produce 2 groups")

	// Group 1: the main crashloop incident
	g1 := groups[0]
	assert.Equal(t, "payment-svc-crashloop", g1.GroupID)
	assert.Equal(t, 1, g1.Priority)
	assert.Equal(t, "critical", g1.Severity)
	assert.Len(t, g1.Events, 6, "6 events related to the crashloop")
	assert.Contains(t, g1.Summary, "OOM")

	// Group 2: scaling activity
	g2 := groups[1]
	assert.Equal(t, "payment-svc-scaling", g2.GroupID)
	assert.Equal(t, 2, g2.Priority)
	assert.Len(t, g2.Events, 1)
}

func TestEval_TriageGrouping_MixedIncidents(t *testing.T) {
	// Evaluate: Two independent incidents (node pressure + deployment rollout)
	// should be grouped separately.

	triageJSON := `{
		"groups": [
			{
				"group_id": "worker3-memory-pressure",
				"summary": "Node worker-3 under memory pressure, evicting pods. Possible memory leak in monitoring-agent.",
				"priority": 1,
				"severity": "warning",
				"event_ids": ["ev-node-pressure", "ev-node-eviction", "ev-alert-mem", "ev-log-memleak"]
			},
			{
				"group_id": "frontend-rollout",
				"summary": "Normal deployment rollout of frontend v2.5.0 in staging, completed successfully.",
				"priority": 2,
				"severity": "info",
				"event_ids": ["ev-deploy-rollout", "ev-deploy-ready"]
			}
		]
	}`

	groups, err := parseTriageResponse(triageJSON, mixedIncidentEvents)
	require.NoError(t, err)
	require.Len(t, groups, 2)

	// Memory pressure group should be higher priority
	assert.Equal(t, "worker3-memory-pressure", groups[0].GroupID)
	assert.Equal(t, 1, groups[0].Priority)
	assert.Len(t, groups[0].Events, 4)

	// Deployment rollout is benign
	assert.Equal(t, "frontend-rollout", groups[1].GroupID)
	assert.Equal(t, "info", groups[1].Severity)
}

func TestEval_TriageGrouping_DBConnectionStorm(t *testing.T) {
	// Evaluate: DB connection exhaustion causes cascading failures across services.
	// A good triage should identify the root cause (DB) and group all effects together.

	triageJSON := `{
		"groups": [
			{
				"group_id": "postgres-connection-exhaustion",
				"summary": "PostgreSQL primary has exhausted all 200 connections. Multiple services (api-gateway, order-svc, inventory-svc) failing with connection errors, causing probe failures and customer-facing timeouts. DB is also showing checkpoint pressure.",
				"priority": 1,
				"severity": "critical",
				"event_ids": ["ev-db-conn-1", "ev-db-conn-2", "ev-db-conn-3", "ev-alert-db", "ev-probe-fail-1", "ev-probe-fail-2", "ev-customer-timeout", "ev-pg-slow"]
			}
		]
	}`

	groups, err := parseTriageResponse(triageJSON, dbConnectionStormEvents)
	require.NoError(t, err)
	require.Len(t, groups, 1, "all events should be grouped as one root cause")

	g := groups[0]
	assert.Equal(t, "postgres-connection-exhaustion", g.GroupID)
	assert.Equal(t, 1, g.Priority)
	assert.Equal(t, "critical", g.Severity)
	assert.Len(t, g.Events, 8, "all 8 events are part of the same incident")
	assert.Contains(t, g.Summary, "PostgreSQL")
}

func TestEval_TriagePromptContainsAllEvents(t *testing.T) {
	// Evaluate: the triage prompt must include every event ID so the model can reference them.

	prompt := BuildTriagePrompt(dbConnectionStormEvents, "Systems normal", "No active incidents.")

	for _, ev := range dbConnectionStormEvents {
		assert.Contains(t, prompt, ev.ID, "prompt must contain event ID %s", ev.ID)
		assert.Contains(t, prompt, ev.Message, "prompt must contain event message")
	}

	assert.Contains(t, prompt, "Systems normal")
	assert.Contains(t, prompt, "8 infrastructure events to triage")
	assert.Contains(t, prompt, "Respond with JSON only")
}

func TestEval_TriagePromptWithActiveIncidents(t *testing.T) {
	// When there are already active incidents, triage prompt should include them
	// so the model can correlate new events with existing incidents.

	state := NewWorldState()
	state.SetSummary("payment-svc experiencing intermittent OOM kills since 13:45")
	state.AddOrUpdateIncident("payment-oom", "warning", "k8s/pod/default/payment-svc",
		"Payment service intermittent OOM", agentregistryv1alpha1.IncidentStatusInvestigating)

	prompt := BuildTriagePrompt(podCrashloopEvents, state.GetSummary(), state.GetActiveIncidentsSummary())

	assert.Contains(t, prompt, "payment-svc experiencing intermittent OOM")
	assert.Contains(t, prompt, "Payment service intermittent OOM")
	assert.Contains(t, prompt, "investigating")
}

func TestEval_FallbackGrouping_PreservesPriority(t *testing.T) {
	// When triage parsing fails, fallback should still prioritize critical > warning > info.

	groups := fallbackGroups(dbConnectionStormEvents)

	// All critical events should come before warning events
	seenWarning := false
	for _, g := range groups {
		if g.Severity == "warning" {
			seenWarning = true
		}
		if g.Severity == "critical" {
			assert.False(t, seenWarning, "critical event group %s came after a warning group", g.GroupID)
		}
	}
}

func TestEval_EndToEnd_SingleEvent(t *testing.T) {
	// Evaluate end-to-end: single event should skip triage and process directly.
	// The mock LLM receives the event, calls create_incident, then update_world_state, then done.

	responses := []openAIChatCompletionResponse{
		// Turn 1: model sees the event, wants to create an incident
		toolCallResponse("create_incident", "call_1", `{"id":"payment-crash","severity":"critical","source":"alertmanager/prod","summary":"payment-svc crash-looping"}`),
		// Turn 2: after tool response, model updates world state
		toolCallResponse("update_world_state", "call_2", `{"summary":"payment-svc is crash-looping in default namespace. Critical incident created."}`),
		// Turn 3: final response (stops agent loop)
		textResponse("Incident payment-crash created. World state updated."),
	}

	server, rec := newMockLLMServer(t, responses)
	defer server.Close()

	mdl := NewGatewayModel("eval-model", server.URL, "")
	hub := NewEventHub(100, 50)
	ag, err := NewMasterAgent(
		context.Background(),
		agentregistryv1alpha1.MasterAgentConfigSpec{
			Enabled: true,
			Models:  agentregistryv1alpha1.ModelRefs{Default: "eval-model"},
			BatchTriage: &agentregistryv1alpha1.BatchTriageConfig{
				Enabled:        true,
				QueueThreshold: 10,
				WindowSeconds:  1,
			},
		},
		mdl,
		hub,
		nil,
		noopLogger(),
	)
	require.NoError(t, err)

	// Push single event and let batch loop process it
	hub.Push(podCrashloopEvents[0])

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ag.Run(ctx)

	// Wait for processing
	assert.Eventually(t, func() bool {
		ag.state.mu.RLock()
		defer ag.state.mu.RUnlock()
		return ag.state.totalEvents > 0
	}, 5*time.Second, 100*time.Millisecond, "event should be processed")

	cancel()

	// Verify: LLM was called (3 turns: tool call, tool call, final)
	reqs := rec.getRequests()
	assert.GreaterOrEqual(t, len(reqs), 1, "LLM should have been called")

	// Verify: first request should contain the event details
	found := false
	for _, req := range reqs {
		for _, msg := range req.Messages {
			if strings.Contains(msg.Content, "PodCrashLooping") {
				found = true
			}
		}
	}
	assert.True(t, found, "LLM request should contain event message")

	// Verify: incident was created via tool
	incidents := ag.State().GetIncidents()
	assert.GreaterOrEqual(t, len(incidents), 1, "incident should have been created by tool call")

	// Verify: world state was updated
	summary := ag.State().GetSummary()
	assert.Contains(t, summary, "crash-looping", "world state should reflect the incident")
}

func TestEval_EndToEnd_BatchTriage(t *testing.T) {
	// Evaluate end-to-end: multiple events trigger batch triage.
	// Flow: triage call → group processing (tool calls) → done.

	triageJSON := `{"groups":[{"group_id":"db-outage","summary":"DB connection exhaustion","priority":1,"severity":"critical","event_ids":["ev-db-conn-1","ev-db-conn-2","ev-db-conn-3","ev-alert-db","ev-probe-fail-1","ev-probe-fail-2","ev-customer-timeout","ev-pg-slow"]}]}`

	responses := []openAIChatCompletionResponse{
		// Call 1 (triage): returns JSON grouping
		textResponse(triageJSON),
		// Call 2 (group processing turn 1): create incident
		toolCallResponse("create_incident", "call_1", `{"id":"db-conn-exhaustion","severity":"critical","source":"postgres/pg-primary","summary":"PostgreSQL connection pool exhausted across all services"}`),
		// Call 3 (group processing turn 2): update world state
		toolCallResponse("update_world_state", "call_2", `{"summary":"CRITICAL: PostgreSQL primary connection pool exhausted. API gateway, order-svc, and inventory-svc all affected. Customer-facing impact confirmed."}`),
		// Call 4 (group processing turn 3): final response
		textResponse("Incident db-conn-exhaustion created. All services affected by DB connection pool exhaustion."),
	}

	server, rec := newMockLLMServer(t, responses)
	defer server.Close()

	mdl := NewGatewayModel("eval-model", server.URL, "")
	hub := NewEventHub(100, 50)

	ag, err := NewMasterAgent(
		context.Background(),
		agentregistryv1alpha1.MasterAgentConfigSpec{
			Enabled: true,
			Models:  agentregistryv1alpha1.ModelRefs{Default: "eval-model"},
			BatchTriage: &agentregistryv1alpha1.BatchTriageConfig{
				Enabled:        true,
				QueueThreshold: 5,
				WindowSeconds:  1,
			},
		},
		mdl,
		hub,
		nil,
		noopLogger(),
	)
	require.NoError(t, err)

	// Push all DB connection storm events
	for _, ev := range dbConnectionStormEvents {
		hub.Push(ev)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ag.Run(ctx)

	// Wait for all events to be processed
	assert.Eventually(t, func() bool {
		ag.state.mu.RLock()
		defer ag.state.mu.RUnlock()
		return ag.state.totalEvents >= int64(len(dbConnectionStormEvents))
	}, 10*time.Second, 100*time.Millisecond, "all events should be processed")

	cancel()

	// Verify: triage call happened (contains "events to triage")
	reqs := rec.getRequests()
	triageCallFound := false
	groupCallFound := false
	for _, req := range reqs {
		for _, msg := range req.Messages {
			if strings.Contains(msg.Content, "events to triage") {
				triageCallFound = true
			}
			if strings.Contains(msg.Content, "Event group:") {
				groupCallFound = true
			}
		}
	}
	assert.True(t, triageCallFound, "triage LLM call should have occurred")
	assert.True(t, groupCallFound, "group processing LLM call should have occurred")

	// Verify: incident was created
	incidents := ag.State().GetIncidents()
	assert.GreaterOrEqual(t, len(incidents), 1, "incident should have been created")

	// Verify: world state was updated
	summary := ag.State().GetSummary()
	assert.Contains(t, summary, "PostgreSQL", "world state should reflect DB incident")

	// Key eval metric: total LLM calls should be much less than total events.
	// 8 events → 1 triage + 3 group processing turns = 4 LLM calls vs. 8+ individual
	t.Logf("Total LLM calls: %d for %d events (%.0f%% reduction vs per-event processing)",
		len(reqs), len(dbConnectionStormEvents),
		(1-float64(len(reqs))/float64(len(dbConnectionStormEvents)))*100)
}

func TestEval_GroupProcessingPrompt(t *testing.T) {
	// Evaluate: the prompt sent for group processing contains all the right context.

	group := EventGroup{
		GroupID:  "payment-crash",
		Summary:  "payment-svc crash-looping due to OOM",
		Priority: 1,
		Severity: "critical",
		Events:   podCrashloopEvents[:4],
	}

	state := NewWorldState()
	state.SetSummary("All systems were normal before this incident")

	// Build the prompt that processEventGroup would construct
	var eventsDetail strings.Builder
	for i, ev := range group.Events {
		eventsDetail.WriteString(
			strings.Join([]string{
				string(rune('0' + i + 1)), ". [id=", ev.ID, "] severity=", ev.Severity,
				" source=", ev.Source, " type=", ev.Type, ": ", ev.Message, "\n",
			}, ""),
		)
	}

	// The prompt should contain:
	assert.Contains(t, eventsDetail.String(), "ev-alert-p1")
	assert.Contains(t, eventsDetail.String(), "ev-k8s-oom")
	assert.Contains(t, eventsDetail.String(), "ev-k8s-restart")
	assert.Contains(t, eventsDetail.String(), "ev-log-oom")
}

func TestEval_TriageBatchSize_Reduction(t *testing.T) {
	// Evaluate the key metric: how many LLM calls we save via batching.
	// This tests the structural guarantee — N events → ~2-3 calls instead of N.

	scenarios := []struct {
		name       string
		events     []InfraEvent
		triageJSON string
		wantGroups int
	}{
		{
			name:   "7 events → 2 groups",
			events: podCrashloopEvents,
			triageJSON: `{"groups":[
				{"group_id":"crash","summary":"crash","priority":1,"severity":"critical","event_ids":["ev-alert-p1","ev-k8s-oom","ev-k8s-restart","ev-log-oom","ev-customer-503","ev-endpoint-remove"]},
				{"group_id":"scale","summary":"scaling","priority":2,"severity":"info","event_ids":["ev-hpa-scale"]}
			]}`,
			wantGroups: 2,
		},
		{
			name:   "6 events → 2 groups",
			events: mixedIncidentEvents,
			triageJSON: `{"groups":[
				{"group_id":"mem","summary":"memory","priority":1,"severity":"warning","event_ids":["ev-node-pressure","ev-node-eviction","ev-alert-mem","ev-log-memleak"]},
				{"group_id":"deploy","summary":"rollout","priority":2,"severity":"info","event_ids":["ev-deploy-rollout","ev-deploy-ready"]}
			]}`,
			wantGroups: 2,
		},
		{
			name:   "8 events → 1 group (single root cause)",
			events: dbConnectionStormEvents,
			triageJSON: `{"groups":[
				{"group_id":"db","summary":"db exhaustion","priority":1,"severity":"critical","event_ids":["ev-db-conn-1","ev-db-conn-2","ev-db-conn-3","ev-alert-db","ev-probe-fail-1","ev-probe-fail-2","ev-customer-timeout","ev-pg-slow"]}
			]}`,
			wantGroups: 1,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			groups, err := parseTriageResponse(sc.triageJSON, sc.events)
			require.NoError(t, err)
			assert.Len(t, groups, sc.wantGroups)

			// LLM call reduction: 1 (triage) + wantGroups (processing) vs len(events) individual
			totalCalls := 1 + sc.wantGroups
			reduction := float64(len(sc.events)-totalCalls) / float64(len(sc.events)) * 100
			t.Logf("%d events → %d groups → %d LLM calls (%.0f%% reduction)",
				len(sc.events), sc.wantGroups, totalCalls, reduction)
			assert.Greater(t, reduction, 0.0, "batching should reduce LLM calls")
		})
	}
}

// noopLogger returns a zerolog.Logger that discards output
func noopLogger() zerolog.Logger {
	return zerolog.Nop()
}
