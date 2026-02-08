package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	"github.com/agentregistry-dev/agentregistry/internal/config"
	"github.com/agentregistry-dev/agentregistry/internal/masteragent"
)

// handleGetMasterAgentStatus returns the master agent's current state
func (s *MCPServer) handleGetMasterAgentStatus(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.getAgent == nil || s.getHub == nil {
		return errorResult("Master agent is not configured"), nil
	}

	agentIface := s.getAgent()
	hubIface := s.getHub()

	if agentIface == nil {
		return errorResult("Master agent is not running"), nil
	}

	// Type assert to actual types
	agent, ok := agentIface.(*masteragent.MasterAgent)
	if !ok || agent == nil {
		return errorResult("Master agent is not running"), nil
	}

	hub, ok := hubIface.(*masteragent.EventHub)
	if !ok || hub == nil {
		return errorResult("Master agent event hub is not running"), nil
	}

	state := agent.State()
	ws := state.ToStatus(hub.QueueDepth())

	type worldStateJSON struct {
		LastUpdated     string `json:"lastUpdated,omitempty"`
		Summary         string `json:"summary"`
		TotalEvents     int64  `json:"totalEvents"`
		PendingEvents   int    `json:"pendingEvents"`
		ActiveIncidents int    `json:"activeIncidents"`
	}

	type incidentJSON struct {
		ID        string    `json:"id"`
		Severity  string    `json:"severity"`
		Source    string    `json:"source"`
		Summary   string    `json:"summary"`
		FirstSeen time.Time `json:"firstSeen"`
		LastSeen  time.Time `json:"lastSeen"`
		Status    string    `json:"status"`
		Actions   []string  `json:"actions,omitempty"`
	}

	type statusJSON struct {
		Running     bool            `json:"running"`
		WorldState  *worldStateJSON `json:"worldState,omitempty"`
		Incidents   []incidentJSON  `json:"incidents,omitempty"`
		QueueDepth  int             `json:"queueDepth"`
		QueueTotal  int64           `json:"queueTotal"`
		A2AEndpoint string          `json:"a2aEndpoint,omitempty"`
	}

	status := statusJSON{
		Running:    true,
		QueueDepth: hub.QueueDepth(),
		QueueTotal: hub.TotalProcessed(),
	}

	// Get A2A endpoint from MasterAgentConfig status
	namespace := config.GetNamespace()
	macList := &agentregistryv1alpha1.MasterAgentConfigList{}
	if err := s.client.List(ctx, macList, client.InNamespace(namespace)); err == nil && len(macList.Items) > 0 {
		status.A2AEndpoint = macList.Items[0].Status.A2AEndpoint
	}

	// Convert world state
	wsJSON := &worldStateJSON{
		Summary:         ws.Summary,
		TotalEvents:     ws.TotalEvents,
		PendingEvents:   ws.PendingEvents,
		ActiveIncidents: ws.ActiveIncidents,
	}
	if ws.LastUpdated != nil {
		wsJSON.LastUpdated = ws.LastUpdated.Time.Format(time.RFC3339)
	}
	status.WorldState = wsJSON

	// Convert incidents
	incidents := state.GetIncidents()
	status.Incidents = make([]incidentJSON, 0, len(incidents))
	for _, inc := range incidents {
		status.Incidents = append(status.Incidents, incidentJSON{
			ID:        inc.ID,
			Severity:  inc.Severity,
			Source:    inc.Source,
			Summary:   inc.Summary,
			FirstSeen: inc.FirstSeen.Time,
			LastSeen:  inc.LastSeen.Time,
			Status:    string(inc.Status),
			Actions:   inc.Actions,
		})
	}

	return jsonResult(status), nil
}

// handleEmitEvent pushes an event to the master agent's queue
func (s *MCPServer) handleEmitEvent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.getHub == nil {
		return errorResult("Master agent is not configured"), nil
	}

	hubIface := s.getHub()
	if hubIface == nil {
		return errorResult("Master agent is not running"), nil
	}

	hub, ok := hubIface.(*masteragent.EventHub)
	if !ok || hub == nil {
		return errorResult("Master agent event hub is not running"), nil
	}

	args := request.GetArguments()
	eventType := getStringArg(args, "type")
	message := getStringArg(args, "message")
	severity := getStringArg(args, "severity")
	source := getStringArg(args, "source")

	if eventType == "" || message == "" {
		return errorResult("type and message are required"), nil
	}

	if severity == "" {
		severity = "info"
	}
	if source == "" {
		source = "mcp-client"
	}

	// Validate severity
	if severity != "info" && severity != "warning" && severity != "critical" {
		return errorResult("severity must be info, warning, or critical"), nil
	}

	event := masteragent.InfraEvent{
		Source:   source,
		Type:     eventType,
		Severity: severity,
		Message:  message,
	}

	if !hub.Push(event) {
		return errorResult("Event queue is full"), nil
	}

	return textResult(fmt.Sprintf("✓ Event emitted successfully\n\nEvent ID: %s\nType: %s\nSeverity: %s\nMessage: %s\n\nThe master agent will process this event and update its world state.",
		event.ID, eventType, severity, message)), nil
}

// handleGetRecentEvents returns recent events from the event hub
func (s *MCPServer) handleGetRecentEvents(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.getHub == nil {
		return errorResult("Master agent is not configured"), nil
	}

	hubIface := s.getHub()
	if hubIface == nil {
		return errorResult("Master agent is not running"), nil
	}

	hub, ok := hubIface.(*masteragent.EventHub)
	if !ok || hub == nil {
		return errorResult("Master agent event hub is not running"), nil
	}

	args := request.GetArguments()
	limit := getIntArg(args, "limit", 20)
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}

	recentEvents := hub.Recent(limit)

	type eventJSON struct {
		ID        string                 `json:"id"`
		Source    string                 `json:"source"`
		Type      string                 `json:"type"`
		Severity  string                 `json:"severity"`
		Message   string                 `json:"message"`
		Timestamp string                 `json:"timestamp"`
		Raw       map[string]interface{} `json:"raw,omitempty"`
	}

	events := make([]eventJSON, 0, len(recentEvents))
	for _, e := range recentEvents {
		events = append(events, eventJSON{
			ID:        e.ID,
			Source:    e.Source,
			Type:      e.Type,
			Severity:  e.Severity,
			Message:   e.Message,
			Timestamp: e.Timestamp.Format(time.RFC3339),
			Raw:       e.Raw,
		})
	}

	return jsonResult(map[string]interface{}{
		"count":  len(events),
		"limit":  limit,
		"events": events,
	}), nil
}
