package masteragent_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Live evaluation tests — run against a real controller with LLM connection.
//
// Usage:
//   LIVE_EVAL=1 go test ./internal/masteragent/ -run TestLiveEval -v -timeout 5m
//
// Run a single scenario:
//   LIVE_EVAL=1 go test ./internal/masteragent/ -run TestLiveEval_PodCrashloop -v -timeout 3m
//
// Prerequisites:
//   - Controller running (make dev-controller)
//   - LLM reachable via configured gateway
//
// These tests are skipped by default (no LIVE_EVAL env var).

const defaultBaseURL = "http://localhost:8080"

func baseURL() string {
	if u := os.Getenv("LIVE_EVAL_URL"); u != "" {
		return u
	}
	return defaultBaseURL
}

func skipUnlessLive(t *testing.T) {
	t.Helper()
	if os.Getenv("LIVE_EVAL") == "" {
		t.Skip("skipping live eval (set LIVE_EVAL=1 to run)")
	}
}

// ---------- API helpers ----------

type pushEventReq struct {
	Source   string `json:"source"`
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type pushEventResp struct {
	Queued bool   `json:"queued"`
	ID     string `json:"id"`
}

type agentStatus struct {
	Running    bool            `json:"running"`
	WorldState *worldStateJSON `json:"worldState,omitempty"`
	Incidents  []incidentJSON  `json:"incidents,omitempty"`
	Queue      queueStatusJSON `json:"queue"`
}

type worldStateJSON struct {
	Summary         string `json:"summary"`
	TotalEvents     int64  `json:"totalEvents"`
	PendingEvents   int    `json:"pendingEvents"`
	ActiveIncidents int    `json:"activeIncidents"`
}

type queueStatusJSON struct {
	Depth int   `json:"depth"`
	Total int64 `json:"total"`
}

type incidentJSON struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Source   string `json:"source"`
	Summary  string `json:"summary"`
	Status   string `json:"status"`
}

func pushEvent(t *testing.T, ev pushEventReq) pushEventResp {
	t.Helper()
	body, _ := json.Marshal(ev)
	resp, err := http.Post(baseURL()+"/v0/agent/events", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "event push should succeed")

	var result pushEventResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result
}

func getStatus(t *testing.T) agentStatus {
	t.Helper()
	resp, err := http.Get(baseURL() + "/v0/agent/status")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var status agentStatus
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&status))
	return status
}

// waitForQueueDrain waits until the event queue is empty and the event count
// has stabilized, meaning all pushed events have been fully processed.
// With concurrent workers, the queue can be momentarily empty while workers are
// still making LLM calls, so we wait for the event count to stop changing.
func waitForQueueDrain(t *testing.T, baselineEvents int64, pushedCount int, timeout time.Duration) agentStatus {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var status agentStatus
	stableCount := 0
	lastEvents := int64(-1)

	for time.Now().Before(deadline) {
		status = getStatus(t)
		if status.WorldState == nil {
			time.Sleep(3 * time.Second)
			continue
		}

		currentEvents := status.WorldState.TotalEvents
		queueEmpty := status.Queue.Depth == 0

		if queueEmpty && currentEvents > baselineEvents {
			// Queue is empty and we've processed something.
			// Check if event count is stable (not changing between polls).
			if currentEvents == lastEvents {
				stableCount++
			} else {
				stableCount = 0
			}
			// Consider done after event count is stable for 3 consecutive polls (~9s)
			if stableCount >= 3 {
				return status
			}
		} else {
			stableCount = 0
		}

		lastEvents = currentEvents
		time.Sleep(3 * time.Second)
	}

	eventsProcessed := int64(0)
	queueDepth := 0
	if status.WorldState != nil {
		eventsProcessed = status.WorldState.TotalEvents
	}
	queueDepth = status.Queue.Depth
	t.Fatalf("timed out waiting for processing to complete (baseline: %d, pushed: %d, processed: %d, queue depth: %d)",
		baselineEvents, pushedCount, eventsProcessed, queueDepth)
	return status
}

// ---------- live eval tests ----------

func TestLiveEval_PodCrashloop(t *testing.T) {
	skipUnlessLive(t)

	// Record baseline
	before := getStatus(t)
	require.True(t, before.Running, "agent must be running")
	baselineEvents := int64(0)
	if before.WorldState != nil {
		baselineEvents = before.WorldState.TotalEvents
	}

	// Push a realistic pod crashloop incident
	events := []pushEventReq{
		{Source: "alertmanager/prod", Type: "alert", Severity: "critical",
			Message: "FIRING: PodCrashLooping - pod default/payment-svc-7f8d4b-xq2k1 has restarted 5 times in 10m"},
		{Source: "k8s/event/default/payment-svc-7f8d4b-xq2k1", Type: "container-oom", Severity: "critical",
			Message: "Container payment-svc OOMKilled: memory limit 512Mi exceeded"},
		{Source: "k8s/event/default/payment-svc-7f8d4b-xq2k1", Type: "pod-restart", Severity: "warning",
			Message: "Back-off restarting failed container payment-svc in pod payment-svc-7f8d4b-xq2k1"},
		{Source: "loki/default/payment-svc", Type: "log-error", Severity: "critical",
			Message: "java.lang.OutOfMemoryError: Java heap space at com.payment.TransactionProcessor.process(TransactionProcessor.java:142)"},
		{Source: "support/ticket-4521", Type: "customer-report", Severity: "warning",
			Message: "Customer reports: payment checkout returning 503 Service Unavailable since ~14:00 UTC"},
		{Source: "k8s/event/default/payment-svc", Type: "endpoint-change", Severity: "info",
			Message: "Endpoints default/payment-svc: removed pod payment-svc-7f8d4b-xq2k1 from service"},
		{Source: "k8s/event/default/payment-svc", Type: "hpa-activity", Severity: "info",
			Message: "HPA scaled payment-svc from 3 to 5 replicas (cpu utilization above target)"},
	}

	t.Log("Pushing 7 pod crashloop events...")
	start := time.Now()
	for _, ev := range events {
		result := pushEvent(t, ev)
		assert.True(t, result.Queued)
	}

	t.Log("Waiting for agent to process events...")
	status := waitForQueueDrain(t, baselineEvents, len(events), 2*time.Minute)
	elapsed := time.Since(start)

	// Evaluate results
	t.Log("--- EVALUATION RESULTS ---")
	t.Logf("Processing time: %s", elapsed.Round(time.Millisecond))
	t.Logf("Total events processed: %d", status.WorldState.TotalEvents)
	t.Logf("Active incidents: %d", status.WorldState.ActiveIncidents)
	t.Logf("World state summary: %s", status.WorldState.Summary)

	for _, inc := range status.Incidents {
		t.Logf("Incident [%s] %s: %s (status: %s)", inc.Severity, inc.ID, inc.Summary, inc.Status)
	}

	// Assertions: the agent should have understood this is a critical incident
	assert.NotEmpty(t, status.WorldState.Summary, "world state should be updated")
	assert.GreaterOrEqual(t, len(status.Incidents), 1, "should create at least one incident")

	hasCriticalOrWarning := false
	mentionsOOM := false
	mentionsPayment := false
	for _, inc := range status.Incidents {
		if inc.Severity == "critical" || inc.Severity == "warning" {
			hasCriticalOrWarning = true
		}
		lower := strings.ToLower(inc.Summary)
		if strings.Contains(lower, "oom") || strings.Contains(lower, "memory") {
			mentionsOOM = true
		}
		if strings.Contains(lower, "payment") || strings.Contains(lower, "crashloop") || strings.Contains(lower, "crash") {
			mentionsPayment = true
		}
	}
	assert.True(t, hasCriticalOrWarning, "should identify this as at least a warning/critical incident")
	assert.True(t, mentionsOOM || mentionsPayment,
		"incident summary should mention OOM/memory or payment/crash")
}

func TestLiveEval_MixedIncidents(t *testing.T) {
	skipUnlessLive(t)

	before := getStatus(t)
	require.True(t, before.Running)
	baselineEvents := int64(0)
	if before.WorldState != nil {
		baselineEvents = before.WorldState.TotalEvents
	}

	events := []pushEventReq{
		{Source: "k8s/node/worker-3", Type: "node-condition", Severity: "warning",
			Message: "Node worker-3 condition MemoryPressure is True, available: 145Mi"},
		{Source: "k8s/event/kube-system/worker-3", Type: "pod-eviction", Severity: "warning",
			Message: "Evicting pod logging-agent-d8f2k from node worker-3 due to memory pressure"},
		{Source: "k8s/event/staging/frontend", Type: "deployment-update", Severity: "info",
			Message: "Deployment staging/frontend updated: replicas 3->3, image frontend:v2.4.1->v2.5.0"},
		{Source: "k8s/event/staging/frontend", Type: "deployment-ready", Severity: "info",
			Message: "Deployment staging/frontend rollout complete: 3/3 replicas available"},
		{Source: "alertmanager/prod", Type: "alert", Severity: "warning",
			Message: "FIRING: NodeMemoryHigh - worker-3 memory usage at 95%"},
		{Source: "loki/kube-system/monitoring-agent", Type: "log-warning", Severity: "warning",
			Message: "monitoring-agent on worker-3: memory usage growing linearly, possible leak detected"},
	}

	t.Log("Pushing 6 mixed-incident events...")
	start := time.Now()
	for _, ev := range events {
		pushEvent(t, ev)
	}

	t.Log("Waiting for agent to process events...")
	status := waitForQueueDrain(t, baselineEvents, len(events), 2*time.Minute)
	elapsed := time.Since(start)

	t.Log("--- EVALUATION RESULTS ---")
	t.Logf("Processing time: %s", elapsed.Round(time.Millisecond))
	t.Logf("Total events processed: %d", status.WorldState.TotalEvents)
	t.Logf("Active incidents: %d", status.WorldState.ActiveIncidents)
	t.Logf("World state summary: %s", status.WorldState.Summary)
	for _, inc := range status.Incidents {
		t.Logf("Incident [%s] %s: %s (status: %s)", inc.Severity, inc.ID, inc.Summary, inc.Status)
	}

	assert.NotEmpty(t, status.WorldState.Summary)

	hasMemoryIncident := false
	for _, inc := range status.Incidents {
		lower := strings.ToLower(inc.Summary)
		if strings.Contains(lower, "memory") || strings.Contains(lower, "worker-3") {
			hasMemoryIncident = true
		}
	}
	assert.True(t, hasMemoryIncident, "should create an incident for node memory pressure")
}

func TestLiveEval_DBConnectionStorm(t *testing.T) {
	skipUnlessLive(t)

	before := getStatus(t)
	require.True(t, before.Running)
	baselineEvents := int64(0)
	if before.WorldState != nil {
		baselineEvents = before.WorldState.TotalEvents
	}

	events := []pushEventReq{
		{Source: "loki/prod/api-gateway", Type: "log-error", Severity: "critical",
			Message: "pq: too many connections for role \"app_user\" - connection pool exhausted"},
		{Source: "loki/prod/order-svc", Type: "log-error", Severity: "critical",
			Message: "pq: too many connections for role \"app_user\" - connection pool exhausted"},
		{Source: "loki/prod/inventory-svc", Type: "log-error", Severity: "critical",
			Message: "pq: too many connections for role \"app_user\" - connection pool exhausted"},
		{Source: "alertmanager/prod", Type: "alert", Severity: "critical",
			Message: "FIRING: PostgresConnectionsExhausted - pg-primary active connections 200/200 (100%)"},
		{Source: "k8s/event/prod/api-gateway-6c8b9-abc12", Type: "probe-failure", Severity: "warning",
			Message: "Readiness probe failed: HTTP probe failed with statuscode: 503"},
		{Source: "k8s/event/prod/order-svc-5d7a8-def34", Type: "probe-failure", Severity: "warning",
			Message: "Readiness probe failed: HTTP probe failed with statuscode: 503"},
		{Source: "support/ticket-4522", Type: "customer-report", Severity: "warning",
			Message: "Multiple customers reporting: orders timing out, checkout page unresponsive"},
		{Source: "loki/prod/pg-primary", Type: "log-warning", Severity: "warning",
			Message: "LOG: checkpoints are occurring too frequently (12 seconds apart)"},
	}

	t.Log("Pushing 8 DB connection storm events...")
	start := time.Now()
	for _, ev := range events {
		pushEvent(t, ev)
	}

	t.Log("Waiting for agent to process events...")
	status := waitForQueueDrain(t, baselineEvents, len(events), 2*time.Minute)
	elapsed := time.Since(start)

	t.Log("--- EVALUATION RESULTS ---")
	t.Logf("Processing time: %s", elapsed.Round(time.Millisecond))
	t.Logf("Total events processed: %d", status.WorldState.TotalEvents)
	t.Logf("Active incidents: %d", status.WorldState.ActiveIncidents)
	t.Logf("World state summary: %s", status.WorldState.Summary)
	for _, inc := range status.Incidents {
		t.Logf("Incident [%s] %s: %s (status: %s)", inc.Severity, inc.ID, inc.Summary, inc.Status)
	}

	assert.NotEmpty(t, status.WorldState.Summary)
	assert.GreaterOrEqual(t, len(status.Incidents), 1)

	hasCritical := false
	mentionsDB := false
	for _, inc := range status.Incidents {
		if inc.Severity == "critical" {
			hasCritical = true
		}
		lower := strings.ToLower(inc.Summary)
		if strings.Contains(lower, "connection") || strings.Contains(lower, "postgres") ||
			strings.Contains(lower, "database") || strings.Contains(lower, "pg") {
			mentionsDB = true
		}
	}
	assert.True(t, hasCritical, "should flag as critical")
	assert.True(t, mentionsDB, "should identify database/connection issue")
}

func TestLiveEval_LLMCallReduction(t *testing.T) {
	skipUnlessLive(t)

	before := getStatus(t)
	require.True(t, before.Running)
	baselineEvents := int64(0)
	if before.WorldState != nil {
		baselineEvents = before.WorldState.TotalEvents
	}

	events := []pushEventReq{
		{Source: "k8s/pod/ns/app-1", Type: "pod-crash", Severity: "critical", Message: "CrashLoopBackOff pod-1"},
		{Source: "k8s/pod/ns/app-1", Type: "container-oom", Severity: "critical", Message: "OOMKilled pod-1"},
		{Source: "k8s/pod/ns/app-1", Type: "pod-restart", Severity: "warning", Message: "Back-off restart pod-1"},
		{Source: "k8s/pod/ns/app-1", Type: "probe-failure", Severity: "warning", Message: "Readiness failed pod-1"},
		{Source: "alertmanager", Type: "alert", Severity: "critical", Message: "FIRING: PodCrashLoop app-1"},
		{Source: "loki/ns/app-1", Type: "log-error", Severity: "critical", Message: "panic: nil pointer dereference"},
		{Source: "k8s/endpoints/ns/app-1", Type: "endpoint-change", Severity: "info", Message: "Removed pod from endpoints"},
		{Source: "support/ticket", Type: "customer-report", Severity: "warning", Message: "App unresponsive"},
	}

	start := time.Now()

	t.Log("Pushing 8 events for efficiency measurement...")
	for _, ev := range events {
		pushEvent(t, ev)
	}

	status := waitForQueueDrain(t, baselineEvents, len(events), 2*time.Minute)
	elapsed := time.Since(start)

	t.Log("--- EFFICIENCY RESULTS ---")
	t.Logf("8 events processed in %s", elapsed.Round(time.Millisecond))
	t.Logf("Queue total: %d", status.Queue.Total)
	t.Logf("Incidents created: %d", len(status.Incidents))

	fmt.Printf("\n=== LIVE EVAL: LLM CALL REDUCTION ===\n")
	fmt.Printf("Events: 8 | Processing time: %s | Incidents: %d\n",
		elapsed.Round(time.Millisecond), len(status.Incidents))
	fmt.Printf("With per-event processing, this would be 8+ sequential LLM calls.\n")
	fmt.Printf("With batch triage, this should be ~2-3 LLM calls.\n")
	fmt.Printf("=====================================\n\n")
}
