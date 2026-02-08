package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/zerolog"

	"github.com/agentregistry-dev/agentregistry/internal/masteragent"
)

// MasterAgentHandler handles master agent API endpoints
type MasterAgentHandler struct {
	logger zerolog.Logger
	getHub func() *masteragent.EventHub
	getAg  func() *masteragent.MasterAgent
}

// NewMasterAgentHandler creates a new master agent handler.
// getHub and getAgent are closures that return the current hub/agent from the reconciler.
func NewMasterAgentHandler(
	logger zerolog.Logger,
	getHub func() *masteragent.EventHub,
	getAgent func() *masteragent.MasterAgent,
) *MasterAgentHandler {
	return &MasterAgentHandler{
		logger: logger.With().Str("handler", "masteragent").Logger(),
		getHub: getHub,
		getAg:  getAgent,
	}
}

// Agent status response types

type AgentStatusJSON struct {
	Running    bool            `json:"running"`
	WorldState *WorldStateJSON `json:"worldState,omitempty"`
	Incidents  []IncidentJSON  `json:"incidents,omitempty"`
	Queue      QueueStatusJSON `json:"queue"`
}

type WorldStateJSON struct {
	LastUpdated     *time.Time `json:"lastUpdated,omitempty"`
	Summary         string     `json:"summary"`
	TotalEvents     int64      `json:"totalEvents"`
	PendingEvents   int        `json:"pendingEvents"`
	ActiveIncidents int        `json:"activeIncidents"`
}

type IncidentJSON struct {
	ID        string    `json:"id"`
	Severity  string    `json:"severity"`
	Source    string    `json:"source"`
	Summary   string    `json:"summary"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
	Status    string    `json:"status"`
	Actions   []string  `json:"actions,omitempty"`
}

type QueueStatusJSON struct {
	Depth int   `json:"depth"`
	Total int64 `json:"total"`
}

// Event push request types

type PushEventInput struct {
	Body PushEventJSON
}

type PushEventJSON struct {
	Source   string         `json:"source,omitempty"`
	Type     string         `json:"type"`
	Severity string         `json:"severity,omitempty"`
	Message  string         `json:"message"`
	Raw      map[string]any `json:"raw,omitempty"`
}

type PushEventResponse struct {
	Queued bool   `json:"queued"`
	ID     string `json:"id,omitempty"`
}

// RegisterRoutes registers master agent endpoints on the given API
func (h *MasterAgentHandler) RegisterRoutes(api huma.API, pathPrefix string) {
	tags := []string{"agent"}

	// GET /v0/agent/status
	huma.Register(api, huma.Operation{
		OperationID: "agent-status",
		Method:      http.MethodGet,
		Path:        pathPrefix + "/agent/status",
		Summary:     "Get master agent status and world state",
		Tags:        tags,
	}, func(ctx context.Context, input *struct{}) (*Response[AgentStatusJSON], error) {
		return h.getStatus(ctx)
	})

	// POST /v0/agent/events
	huma.Register(api, huma.Operation{
		OperationID: "agent-push-event",
		Method:      http.MethodPost,
		Path:        pathPrefix + "/agent/events",
		Summary:     "Push an infrastructure event to the agent queue",
		Tags:        tags,
	}, func(ctx context.Context, input *PushEventInput) (*Response[PushEventResponse], error) {
		return h.pushEvent(ctx, input)
	})
}

func (h *MasterAgentHandler) getStatus(ctx context.Context) (*Response[AgentStatusJSON], error) {
	ag := h.getAg()
	hub := h.getHub()

	status := AgentStatusJSON{
		Running: ag != nil,
		Queue:   QueueStatusJSON{},
	}

	if ag != nil {
		state := ag.State()
		ws := state.ToStatus(0)
		if hub != nil {
			ws = state.ToStatus(hub.QueueDepth())
			status.Queue.Depth = hub.QueueDepth()
			status.Queue.Total = hub.TotalProcessed()
		}

		var lastUpdated *time.Time
		if ws.LastUpdated != nil {
			t := ws.LastUpdated.Time
			lastUpdated = &t
		}

		status.WorldState = &WorldStateJSON{
			LastUpdated:     lastUpdated,
			Summary:         ws.Summary,
			TotalEvents:     ws.TotalEvents,
			PendingEvents:   ws.PendingEvents,
			ActiveIncidents: ws.ActiveIncidents,
		}

		incidents := state.GetIncidents()
		for _, inc := range incidents {
			status.Incidents = append(status.Incidents, IncidentJSON{
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
	}

	return &Response[AgentStatusJSON]{Body: status}, nil
}

func (h *MasterAgentHandler) pushEvent(ctx context.Context, input *PushEventInput) (*Response[PushEventResponse], error) {
	hub := h.getHub()
	if hub == nil {
		return nil, huma.Error503ServiceUnavailable("Master agent is not running")
	}

	severity := input.Body.Severity
	if severity == "" {
		severity = "info"
	}

	event := masteragent.InfraEvent{
		Source:   input.Body.Source,
		Type:     input.Body.Type,
		Severity: severity,
		Message:  input.Body.Message,
		Raw:      input.Body.Raw,
	}

	if !hub.Push(event) {
		return nil, huma.Error429TooManyRequests("Event queue is full")
	}

	return &Response[PushEventResponse]{
		Body: PushEventResponse{
			Queued: true,
			ID:     event.ID,
		},
	}, nil
}
