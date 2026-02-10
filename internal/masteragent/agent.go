package masteragent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2aclient"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"google.golang.org/genai"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/adk/tool/mcptoolset"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

// MasterAgent wraps an ADK agent with event processing capabilities
type MasterAgent struct {
	agent       agent.Agent
	runner      *runner.Runner
	sessionSvc  session.Service
	hub         *EventHub
	state       *WorldState
	maxWorkers  int
	batchConfig *agentregistryv1alpha1.BatchTriageConfig
	logger      zerolog.Logger
}

// NewMasterAgent creates a MasterAgent with ADK agent, MCP toolsets, and function tools.
// getA2AAgents, if non-nil, provides discovered remote A2A agents for the call_a2a_agent tool.
func NewMasterAgent(
	ctx context.Context,
	spec agentregistryv1alpha1.MasterAgentConfigSpec,
	defaultModel *GatewayModel,
	hub *EventHub,
	getA2AAgents func() []A2AAgentInfo,
	logger zerolog.Logger,
) (*MasterAgent, error) {
	// Build MCP toolsets — auto-discover tools from each MCP server
	var toolsets []tool.Toolset
	for _, srv := range spec.MCPServers {
		ts, err := mcptoolset.New(mcptoolset.Config{
			Transport: &mcp.StreamableClientTransport{
				Endpoint: srv.URL,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("mcp toolset %s: %w", srv.Name, err)
		}
		toolsets = append(toolsets, ts)
	}

	worldState := NewWorldState()

	// Function tools for world state management
	getStateTool, err := functiontool.New(functiontool.Config{
		Name:        "get_world_state",
		Description: "Get the current world state summary including active incidents and recent events",
	}, func(ctx tool.Context, args struct{}) (map[string]any, error) {
		return map[string]any{
			"summary":          worldState.GetSummary(),
			"active_incidents": worldState.GetActiveIncidentsSummary(),
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("create get_world_state tool: %w", err)
	}

	type UpdateStateArgs struct {
		Summary string `json:"summary"`
	}
	updateStateTool, err := functiontool.New(functiontool.Config{
		Name:        "update_world_state",
		Description: "Update the world state summary with new information about infrastructure status. REQUIRED parameter 'summary' (string): the updated state description.",
	}, func(ctx tool.Context, args UpdateStateArgs) (map[string]any, error) {
		worldState.SetSummary(args.Summary)
		return map[string]any{"status": "updated"}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("create update_world_state tool: %w", err)
	}

	type CreateIncidentArgs struct {
		ID       string `json:"id"`
		Severity string `json:"severity"`
		Source   string `json:"source"`
		Summary  string `json:"summary"`
	}
	createIncidentTool, err := functiontool.New(functiontool.Config{
		Name:        "create_incident",
		Description: "Create or update an infrastructure incident. Parameters: 'id' (string, unique identifier e.g. 'nginx-abc123-crashloop'), 'severity' (string, one of: info, warning, critical), 'source' (string, origin e.g. 'k8s/pod/namespace/name'), 'summary' (string, human-readable description of the incident).",
	}, func(ctx tool.Context, args CreateIncidentArgs) (map[string]any, error) {
		worldState.AddOrUpdateIncident(
			args.ID, args.Severity, args.Source, args.Summary,
			agentregistryv1alpha1.IncidentStatusInvestigating,
		)
		return map[string]any{"status": "created", "id": args.ID}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("create create_incident tool: %w", err)
	}

	type ResolveIncidentArgs struct {
		ID string `json:"id"`
	}
	resolveIncidentTool, err := functiontool.New(functiontool.Config{
		Name:        "resolve_incident",
		Description: "Mark an incident as resolved. Parameter: 'id' (string, the incident identifier to resolve).",
	}, func(ctx tool.Context, args ResolveIncidentArgs) (map[string]any, error) {
		worldState.ResolveIncident(args.ID)
		return map[string]any{"status": "resolved", "id": args.ID}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("create resolve_incident tool: %w", err)
	}

	// call_a2a_agent tool for delegating to remote A2A agents
	type CallA2AAgentArgs struct {
		AgentName string `json:"agent_name"`
		Message   string `json:"message"`
	}
	callA2ATool, err := functiontool.New(functiontool.Config{
		Name:        "call_a2a_agent",
		Description: "Send a message to a remote A2A agent and return its response. Parameters: 'agent_name' (string, the agent name as shown in Available A2A Agents), 'message' (string, the request to send).",
	}, func(ctx tool.Context, args CallA2AAgentArgs) (map[string]any, error) {
		if getA2AAgents == nil {
			return map[string]any{"error": "no A2A agents configured"}, nil
		}
		agents := getA2AAgents()
		var target *A2AAgentInfo
		for i := range agents {
			if agents[i].Name == args.AgentName {
				target = &agents[i]
				break
			}
		}
		if target == nil {
			names := make([]string, len(agents))
			for i, a := range agents {
				names[i] = a.Name
			}
			return map[string]any{"error": fmt.Sprintf("agent %q not found, available: %s", args.AgentName, strings.Join(names, ", "))}, nil
		}
		result, err := callA2AAgent(ctx, target.Endpoint, args.Message)
		if err != nil {
			return map[string]any{"error": err.Error()}, nil
		}
		return result, nil
	})
	if err != nil {
		return nil, fmt.Errorf("create call_a2a_agent tool: %w", err)
	}

	// Create the ADK LLM agent
	ag, err := llmagent.New(llmagent.Config{
		Name:        "master-agent",
		Description: "Autonomous infrastructure observer and triage agent for Agent Registry",
		Model:       defaultModel,
		Instruction: BuildSystemPrompt(spec.SystemPrompt, getA2AAgents),
		Toolsets:    toolsets,
		Tools: []tool.Tool{
			getStateTool,
			updateStateTool,
			createIncidentTool,
			resolveIncidentTool,
			callA2ATool,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create ADK agent: %w", err)
	}

	sessionSvc := session.InMemoryService()

	r, err := runner.New(runner.Config{
		AppName:        "master-agent",
		Agent:          ag,
		SessionService: sessionSvc,
	})
	if err != nil {
		return nil, fmt.Errorf("create runner: %w", err)
	}

	maxWorkers := spec.MaxConcurrentEvents
	if maxWorkers <= 0 {
		maxWorkers = 5
	}

	return &MasterAgent{
		agent:       ag,
		runner:      r,
		sessionSvc:  sessionSvc,
		hub:         hub,
		state:       worldState,
		maxWorkers:  maxWorkers,
		batchConfig: spec.BatchTriage,
		logger:      logger,
	}, nil
}

// Agent returns the underlying ADK agent
func (m *MasterAgent) Agent() agent.Agent {
	return m.agent
}

// State returns the world state
func (m *MasterAgent) State() *WorldState {
	return m.state
}

// Hub returns the event hub
func (m *MasterAgent) Hub() *EventHub {
	return m.hub
}

// SessionService returns the session service
func (m *MasterAgent) SessionService() session.Service {
	return m.sessionSvc
}

// Run starts the event processing loop. When batch triage is enabled, a single
// batch collector goroutine accumulates events and sends them through LLM-driven
// triage before processing. Otherwise, falls back to per-event concurrent workers.
func (m *MasterAgent) Run(ctx context.Context) {
	if m.batchConfig != nil && m.batchConfig.Enabled {
		m.logger.Info().
			Int("threshold", m.batchConfig.QueueThreshold).
			Int("window_seconds", m.batchConfig.WindowSeconds).
			Msg("starting master agent with batch triage")
		go m.batchLoop(ctx)
	} else {
		m.logger.Info().Int("workers", m.maxWorkers).Msg("starting master agent event processing")
		for i := range m.maxWorkers {
			go m.worker(ctx, i)
		}
	}
}

func (m *MasterAgent) worker(ctx context.Context, id int) {
	logger := m.logger.With().Int("worker", id).Logger()
	logger.Debug().Msg("worker started")

	for {
		event, ok := m.hub.Pop(ctx)
		if !ok {
			logger.Debug().Msg("worker stopping (context cancelled)")
			return
		}

		logger.Info().
			Str("event_id", event.ID).
			Str("source", event.Source).
			Str("type", event.Type).
			Str("severity", event.Severity).
			Msg("processing event")

		m.processEvent(ctx, event, logger)
		m.state.IncrementEvents()
	}
}

// batchLoop accumulates events and processes them in prioritized groups.
// It drains the queue when either the threshold is reached or the timer fires.
func (m *MasterAgent) batchLoop(ctx context.Context) {
	threshold := m.batchConfig.QueueThreshold
	if threshold <= 0 {
		threshold = 10
	}
	windowSec := m.batchConfig.WindowSeconds
	if windowSec <= 0 {
		windowSec = 30
	}
	window := time.Duration(windowSec) * time.Second

	var pending []InfraEvent
	timer := time.NewTimer(window)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			m.logger.Debug().Msg("batch loop stopping (context cancelled)")
			return

		case ev, ok := <-m.hub.queue:
			if !ok {
				return
			}
			pending = append(pending, ev)

			// Check threshold
			if len(pending) >= threshold {
				// Also drain anything else available
				pending = append(pending, m.hub.Drain()...)
				m.processBatch(ctx, pending)
				pending = nil
				timer.Reset(window)
			}

		case <-timer.C:
			// Timer fired — drain whatever is queued
			pending = append(pending, m.hub.Drain()...)
			if len(pending) > 0 {
				m.processBatch(ctx, pending)
				pending = nil
			}
			timer.Reset(window)
		}
	}
}

// processBatch handles a batch of accumulated events. Single events skip triage
// and are processed directly. Multiple events go through LLM triage first.
func (m *MasterAgent) processBatch(ctx context.Context, events []InfraEvent) {
	logger := m.logger.With().Int("batch_size", len(events)).Logger()
	logger.Info().Msg("processing event batch")

	if len(events) == 1 {
		// Fast path: single event, no triage needed
		logger.Debug().Str("event_id", events[0].ID).Msg("single event, skipping triage")
		m.processEvent(ctx, events[0], logger)
		m.state.IncrementEvents()
		return
	}

	// Triage the batch
	groups, err := triageBatch(ctx, events, m.runner, m.sessionSvc, m.state, logger)
	if err != nil {
		logger.Error().Err(err).Msg("batch triage failed, processing events individually")
		for _, ev := range events {
			m.processEvent(ctx, ev, logger)
			m.state.IncrementEvents()
		}
		return
	}

	logger.Info().Int("groups", len(groups)).Msg("triage complete, processing groups sequentially")

	// Process groups sequentially by priority
	for _, group := range groups {
		logger.Info().
			Str("group_id", group.GroupID).
			Int("priority", group.Priority).
			Str("severity", group.Severity).
			Int("events", len(group.Events)).
			Msg("processing event group")

		m.processEventGroup(ctx, group, logger)

		// Count all constituent events
		for range group.Events {
			m.state.IncrementEvents()
		}
	}
}

// processEventGroup processes a triaged group of related events in a single LLM call.
func (m *MasterAgent) processEventGroup(ctx context.Context, group EventGroup, logger zerolog.Logger) {
	var eventsDetail strings.Builder
	for i, ev := range group.Events {
		eventsDetail.WriteString(fmt.Sprintf(
			"%d. [id=%s] severity=%s source=%s type=%s time=%s: %s\n",
			i+1, ev.ID, ev.Severity, ev.Source, ev.Type,
			ev.Timestamp.Format("2006-01-02T15:04:05Z"), ev.Message,
		))
	}

	prompt := fmt.Sprintf(
		"Current world state:\n%s\n\n%s\n\n"+
			"Event group: %s (priority %d, severity %s)\n"+
			"Summary: %s\n\n"+
			"Constituent events (%d):\n%s\n"+
			"Triage this event group. Use available MCP tools to investigate if needed. "+
			"Update the world state summary and create/update incidents as appropriate.",
		m.state.GetSummary(),
		m.state.GetActiveIncidentsSummary(),
		group.GroupID, group.Priority, group.Severity,
		group.Summary,
		len(group.Events), eventsDetail.String(),
	)

	createResp, err := m.sessionSvc.Create(ctx, &session.CreateRequest{
		AppName: "master-agent",
		UserID:  "system",
	})
	if err != nil {
		logger.Error().Err(err).Str("group_id", group.GroupID).Msg("failed to create session for group")
		return
	}

	userContent := genai.NewContentFromText(prompt, "user")
	for ev, err := range m.runner.Run(ctx, "system", createResp.Session.ID(), userContent, agent.RunConfig{}) {
		if err != nil {
			logger.Error().Err(err).Str("group_id", group.GroupID).Msg("agent run error for group")
			break
		}
		if ev.IsFinalResponse() {
			logger.Debug().
				Str("group_id", group.GroupID).
				Str("author", ev.Author).
				Msg("agent completed group processing")
		}
	}
}

func (m *MasterAgent) processEvent(ctx context.Context, event InfraEvent, logger zerolog.Logger) {
	prompt := fmt.Sprintf(
		"Current world state:\n%s\n\n%s\n\nNew infrastructure event:\nID: %s\nSource: %s\nType: %s\nSeverity: %s\nMessage: %s\nTimestamp: %s\n\n"+
			"Triage this event. Use available MCP tools to investigate if needed. "+
			"Update the world state summary and create/update incidents as appropriate.",
		m.state.GetSummary(),
		m.state.GetActiveIncidentsSummary(),
		event.ID, event.Source, event.Type, event.Severity, event.Message, event.Timestamp.Format("2006-01-02T15:04:05Z"),
	)

	// Create a session for this event
	createResp, err := m.sessionSvc.Create(ctx, &session.CreateRequest{
		AppName: "master-agent",
		UserID:  "system",
	})
	if err != nil {
		logger.Error().Err(err).Str("event_id", event.ID).Msg("failed to create session")
		return
	}

	// Run the ADK agent loop
	userContent := genai.NewContentFromText(prompt, "user")
	for ev, err := range m.runner.Run(ctx, "system", createResp.Session.ID(), userContent, agent.RunConfig{}) {
		if err != nil {
			logger.Error().Err(err).Str("event_id", event.ID).Msg("agent run error")
			break
		}
		if ev.IsFinalResponse() {
			logger.Debug().
				Str("event_id", event.ID).
				Str("author", ev.Author).
				Msg("agent completed event processing")
		}
	}
}

// callA2AAgent sends a message to a remote A2A agent and extracts the text response
func callA2AAgent(ctx context.Context, endpoint, message string) (map[string]any, error) {
	client, err := a2aclient.NewFromEndpoints(ctx, []a2a.AgentInterface{
		{URL: endpoint, Transport: a2a.TransportProtocolJSONRPC},
	})
	if err != nil {
		return nil, fmt.Errorf("create A2A client: %w", err)
	}
	defer client.Destroy()

	callCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.TextPart{Text: message})
	result, err := client.SendMessage(callCtx, &a2a.MessageSendParams{Message: msg})
	if err != nil {
		return nil, fmt.Errorf("send A2A message: %w", err)
	}

	return extractA2AResult(result), nil
}

// extractA2AResult extracts text from a SendMessageResult (either *Message or *Task)
func extractA2AResult(result a2a.SendMessageResult) map[string]any {
	switch r := result.(type) {
	case *a2a.Message:
		return map[string]any{"text": extractPartsText(r.Parts)}
	case *a2a.Task:
		state := string(r.Status.State)
		text := ""
		if r.Status.Message != nil {
			text = extractPartsText(r.Status.Message.Parts)
		}
		if text == "" {
			for _, art := range r.Artifacts {
				t := extractPartsText(art.Parts)
				if t != "" {
					text = t
					break
				}
			}
		}
		return map[string]any{"state": state, "text": text}
	default:
		return map[string]any{"text": fmt.Sprintf("%+v", result)}
	}
}

// extractPartsText concatenates text parts from A2A content parts
func extractPartsText(parts a2a.ContentParts) string {
	var texts []string
	for _, p := range parts {
		if tp, ok := p.(a2a.TextPart); ok {
			texts = append(texts, tp.Text)
		}
	}
	return strings.Join(texts, "\n")
}
