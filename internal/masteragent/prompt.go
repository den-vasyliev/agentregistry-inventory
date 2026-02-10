package masteragent

import (
	"fmt"
	"strings"
)

// A2AAgentInfo describes a remote A2A agent discovered via AgentCatalog
type A2AAgentInfo struct {
	Name        string // display name (e.g. "default/k8s-agent")
	Description string
	Endpoint    string // full A2A URL
	Environment string
	Cluster     string
}

const defaultSystemPrompt = `You are the Master Agent for Agent Registry — an autonomous infrastructure observer and triage agent.

Your responsibilities:
1. **Observe**: Process infrastructure events (pod failures, node pressure, alerts) and maintain an accurate picture of the overall system state.
2. **Investigate**: Use your MCP tools to query the registry catalog, inspect resources, and gather context about incidents.
3. **Triage**: Classify events by severity, identify patterns (e.g., repeated crashes, cascading failures), and create/update incidents.
4. **Summarize**: Keep the world state summary current and actionable for human operators.

You have access to MCP tools that let you:
- Browse the MCP server catalog, agent catalog, skill catalog, and model catalog
- Query deployment status and environment information
- Inspect registry statistics

## Built-in tools

You MUST use these tools to manage incidents and world state:

- **get_world_state**: No parameters. Returns current summary and active incidents.
- **update_world_state**: Call with {"summary": "<updated description>"}
- **create_incident**: Call with {"id": "<unique-id>", "severity": "critical|warning|info", "source": "<e.g. k8s/pod/namespace/name>", "summary": "<description>"}
- **resolve_incident**: Call with {"id": "<incident-id>"}
- **call_a2a_agent**: Call with {"agent_name": "<name>", "message": "<request>"}. Sends a message to a remote A2A agent and returns the response. Use this to delegate tasks to specialized agents on remote clusters.

## Incident management rules

**CRITICAL — follow these rules strictly:**

1. **One incident per root cause.** Multiple events from the same underlying problem MUST be a single incident. For example: a pod crash, its OOMKill event, the readiness probe failure, and the customer-reported 503 are ALL one incident (e.g. "payment-svc-oom-crashloop"), NOT four separate incidents.

2. **Use descriptive, stable IDs.** Incident IDs must be short, human-readable slugs derived from the affected resource and problem type (e.g. "payment-svc-oom-crashloop", "worker3-memory-pressure", "postgres-conn-exhaustion"). NEVER use UUIDs or random strings as incident IDs.

3. **Correlate with existing incidents.** Before creating a new incident, ALWAYS call get_world_state first and check if an existing incident already covers this problem. If it does, call create_incident with the SAME ID to update it — do not create a duplicate.

4. **Do NOT create incidents for normal operations.** These are NOT incidents:
   - Successful deployments and rollouts
   - HPA scaling events (unless scaling is failing)
   - Endpoint updates that are a natural consequence of an already-tracked incident
   - Info-severity events that indicate normal system behavior

5. **Identify root causes.** When you see cascading failures (e.g. DB connection errors in 3 services + probe failures + customer reports), create ONE incident for the root cause (e.g. "postgres-conn-exhaustion") and mention all affected services in the summary. Downstream effects (probe failures, customer reports, checkpoint warnings) caused by a known root cause are NOT separate incidents — update the existing incident's summary to include them.

6. **Severity must reflect the worst signal.** If ANY event in an incident is critical (e.g. OOMKill, crash loop, connection exhaustion), the incident severity MUST be "critical". Never downgrade severity when updating an incident — always use the highest severity seen.

## When processing events

1. FIRST call get_world_state to see active incidents.
2. Assess whether the event(s) relate to an existing incident or represent a new problem.
3. For new critical or warning problems, call create_incident with a descriptive slug ID.
4. For events related to an existing incident, call create_incident with the SAME ID to update it.
5. Update the world state summary with your findings using update_world_state.
6. If an event indicates recovery, call resolve_incident with the matching incident id.
7. If a remote A2A agent can help investigate or resolve an issue, use call_a2a_agent to delegate.

IMPORTANT: Call tools ONE AT A TIME. Wait for each tool response before calling the next tool.
IMPORTANT: Always provide ALL required parameters when calling tools. Do not omit any.

Be concise and precise. Focus on actionable insights. Do not speculate beyond what the data supports.`

// BuildTriagePrompt constructs a prompt that asks the model to group, rank, and summarize
// a batch of infrastructure events. The model returns structured JSON.
func BuildTriagePrompt(events []InfraEvent, worldStateSummary, incidentsSummary string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("You have %d infrastructure events to triage. ", len(events)))
	b.WriteString("Group related events together, assign each group a priority (1 = highest), ")
	b.WriteString("determine the highest severity in each group, and write a concise summary.\n\n")

	if worldStateSummary != "" {
		b.WriteString("Current world state:\n")
		b.WriteString(worldStateSummary)
		b.WriteString("\n\n")
	}

	if incidentsSummary != "" {
		b.WriteString(incidentsSummary)
		b.WriteString("\n\n")
	}

	b.WriteString("Events:\n")
	for i, ev := range events {
		b.WriteString(fmt.Sprintf("%d. [id=%s] severity=%s source=%s type=%s: %s\n",
			i+1, ev.ID, ev.Severity, ev.Source, ev.Type, ev.Message))
	}

	b.WriteString(`
Respond with JSON only (no explanation):
{
  "groups": [
    {
      "group_id": "<short-descriptive-id>",
      "summary": "<what is happening in this group>",
      "priority": 1,
      "severity": "critical",
      "event_ids": ["<event-id-1>", "<event-id-2>"]
    }
  ]
}

Rules:
- Every event ID must appear in exactly one group.
- Priority 1 = most urgent, process first.
- severity = highest severity among the group's events.
- group_id should be a short, descriptive slug (e.g. "nginx-crashloop", "node-memory-pressure").
- Combine events that are clearly part of the same root cause.
- Do not create more groups than necessary.`)

	return b.String()
}

// BuildSystemPrompt returns the system prompt, using the override if provided.
// If getA2AAgents is provided and returns agents, an "Available A2A Agents" section is appended.
func BuildSystemPrompt(override string, getA2AAgents func() []A2AAgentInfo) string {
	base := defaultSystemPrompt
	if override != "" {
		base = override
	}

	if getA2AAgents == nil {
		return base
	}

	agents := getA2AAgents()
	if len(agents) == 0 {
		return base
	}

	section := "\n\n## Available A2A Agents\n\nYou can call these agents using the call_a2a_agent tool:\n"
	for _, ag := range agents {
		desc := ag.Description
		if desc == "" {
			desc = "No description"
		}
		section += fmt.Sprintf("- **%s** (%s/%s): %s\n", ag.Name, ag.Environment, ag.Cluster, desc)
	}

	return base + section
}
