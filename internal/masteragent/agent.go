package masteragent

import (
	"context"
	"fmt"

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
	agent      agent.Agent
	runner     *runner.Runner
	sessionSvc session.Service
	hub        *EventHub
	state      *WorldState
	maxWorkers int
	logger     zerolog.Logger
}

// NewMasterAgent creates a MasterAgent with ADK agent, MCP toolsets, and function tools
func NewMasterAgent(
	ctx context.Context,
	spec agentregistryv1alpha1.MasterAgentConfigSpec,
	defaultModel *GatewayModel,
	hub *EventHub,
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
		Description: "Create or update an infrastructure incident",
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
		Description: "Mark an incident as resolved",
	}, func(ctx tool.Context, args ResolveIncidentArgs) (map[string]any, error) {
		worldState.ResolveIncident(args.ID)
		return map[string]any{"status": "resolved", "id": args.ID}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("create resolve_incident tool: %w", err)
	}

	// Create the ADK LLM agent
	ag, err := llmagent.New(llmagent.Config{
		Name:        "master-agent",
		Description: "Autonomous infrastructure observer and triage agent for Agent Registry",
		Model:       defaultModel,
		Instruction: BuildSystemPrompt(spec.SystemPrompt),
		Toolsets:    toolsets,
		Tools: []tool.Tool{
			getStateTool,
			updateStateTool,
			createIncidentTool,
			resolveIncidentTool,
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
		agent:      ag,
		runner:     r,
		sessionSvc: sessionSvc,
		hub:        hub,
		state:      worldState,
		maxWorkers: maxWorkers,
		logger:     logger,
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

// Run starts the event processing loop with concurrent workers
func (m *MasterAgent) Run(ctx context.Context) {
	m.logger.Info().Int("workers", m.maxWorkers).Msg("starting master agent event processing")

	for i := range m.maxWorkers {
		go m.worker(ctx, i)
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
