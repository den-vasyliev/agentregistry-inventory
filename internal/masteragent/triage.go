package masteragent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/rs/zerolog"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// EventGroup represents a set of related events grouped by the triage model
type EventGroup struct {
	// GroupID is a model-assigned identifier (e.g. "nginx-crashloop")
	GroupID string `json:"group_id"`
	// Summary is a model-generated description of the group
	Summary string `json:"summary"`
	// Priority determines processing order (1 = highest priority)
	Priority int `json:"priority"`
	// Severity is the highest severity among constituent events
	Severity string `json:"severity"`
	// EventIDs lists the IDs of events in this group
	EventIDs []string `json:"event_ids"`
	// Events holds the actual event objects (populated after parsing)
	Events []InfraEvent `json:"-"`
}

// triageResponse is the expected JSON structure from the triage model call
type triageResponse struct {
	Groups []EventGroup `json:"groups"`
}

// triageBatch sends all events to the model in one call for grouping and prioritization.
// Returns groups sorted by priority (1 = first to process).
func triageBatch(
	ctx context.Context,
	events []InfraEvent,
	r *runner.Runner,
	sessionSvc session.Service,
	state *WorldState,
	logger zerolog.Logger,
) ([]EventGroup, error) {
	prompt := BuildTriagePrompt(events, state.GetSummary(), state.GetActiveIncidentsSummary())

	createResp, err := sessionSvc.Create(ctx, &session.CreateRequest{
		AppName: "master-agent",
		UserID:  "system",
	})
	if err != nil {
		return nil, fmt.Errorf("create triage session: %w", err)
	}

	// Collect all model text output
	var responseText strings.Builder
	userContent := genai.NewContentFromText(prompt, "user")
	for ev, err := range r.Run(ctx, "system", createResp.Session.ID(), userContent, agent.RunConfig{}) {
		if err != nil {
			return nil, fmt.Errorf("triage model run: %w", err)
		}
		if ev.IsFinalResponse() && ev.Content != nil {
			for _, part := range ev.Content.Parts {
				if part.Text != "" {
					responseText.WriteString(part.Text)
				}
			}
		}
	}

	groups, err := parseTriageResponse(responseText.String(), events)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to parse triage response, falling back to individual processing")
		return fallbackGroups(events), nil
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Priority < groups[j].Priority
	})

	return groups, nil
}

// parseTriageResponse extracts JSON from the model response and maps event IDs to actual events.
func parseTriageResponse(text string, events []InfraEvent) ([]EventGroup, error) {
	// Extract JSON — model may wrap it in markdown code fences
	jsonStr := extractJSON(text)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in triage response")
	}

	var resp triageResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("unmarshal triage response: %w", err)
	}

	if len(resp.Groups) == 0 {
		return nil, fmt.Errorf("triage response contains no groups")
	}

	// Build event lookup
	eventMap := make(map[string]InfraEvent, len(events))
	for _, ev := range events {
		eventMap[ev.ID] = ev
	}

	// Map event IDs to actual events, collect any unassigned events
	assigned := make(map[string]bool)
	for i := range resp.Groups {
		for _, eid := range resp.Groups[i].EventIDs {
			if ev, ok := eventMap[eid]; ok {
				resp.Groups[i].Events = append(resp.Groups[i].Events, ev)
				assigned[eid] = true
			}
		}
	}

	// Any events not assigned by the model go into a catch-all group
	var unassigned []InfraEvent
	for _, ev := range events {
		if !assigned[ev.ID] {
			unassigned = append(unassigned, ev)
		}
	}
	if len(unassigned) > 0 {
		maxPriority := 0
		for _, g := range resp.Groups {
			if g.Priority > maxPriority {
				maxPriority = g.Priority
			}
		}
		resp.Groups = append(resp.Groups, EventGroup{
			GroupID:  "unassigned",
			Summary:  "Events not grouped by triage",
			Priority: maxPriority + 1,
			Severity: highestSeverity(unassigned),
			Events:   unassigned,
		})
	}

	return resp.Groups, nil
}

// extractJSON finds the first JSON object in text, handling optional markdown fences.
func extractJSON(text string) string {
	// Try to find ```json ... ``` block
	if start := strings.Index(text, "```json"); start != -1 {
		start += len("```json")
		if end := strings.Index(text[start:], "```"); end != -1 {
			return strings.TrimSpace(text[start : start+end])
		}
	}
	// Try to find ``` ... ``` block
	if start := strings.Index(text, "```"); start != -1 {
		start += len("```")
		if end := strings.Index(text[start:], "```"); end != -1 {
			return strings.TrimSpace(text[start : start+end])
		}
	}
	// Try raw JSON: find first { and last }
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start != -1 && end > start {
		return text[start : end+1]
	}
	return ""
}

// fallbackGroups wraps each event into its own group when triage parsing fails.
func fallbackGroups(events []InfraEvent) []EventGroup {
	groups := make([]EventGroup, len(events))
	for i, ev := range events {
		priority := 3 // default: low
		switch ev.Severity {
		case "critical":
			priority = 1
		case "warning":
			priority = 2
		}
		groups[i] = EventGroup{
			GroupID:  ev.ID,
			Summary:  ev.Message,
			Priority: priority,
			Severity: ev.Severity,
			EventIDs: []string{ev.ID},
			Events:   []InfraEvent{ev},
		}
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Priority < groups[j].Priority
	})
	return groups
}

// highestSeverity returns the highest severity from a list of events.
func highestSeverity(events []InfraEvent) string {
	severityRank := map[string]int{"critical": 3, "warning": 2, "info": 1}
	best := ""
	bestRank := 0
	for _, ev := range events {
		if r := severityRank[ev.Severity]; r > bestRank {
			bestRank = r
			best = ev.Severity
		}
	}
	if best == "" {
		return "info"
	}
	return best
}
