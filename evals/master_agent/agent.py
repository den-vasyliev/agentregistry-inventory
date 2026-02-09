"""Remote A2A agent wrapper for the Go master agent."""

import os

from google.adk.agents.remote_a2a_agent import A2AClientConfig, A2AClientFactory, RemoteA2aAgent

A2A_URL = os.getenv("MASTER_AGENT_A2A_URL", "http://localhost:8084")

root_agent = RemoteA2aAgent(
    name="master_agent",
    description="Infrastructure observer and triage agent that processes events, creates incidents, and maintains world state.",
    agent_card=f"{A2A_URL}/.well-known/agent-card.json",
    a2a_client_factory=A2AClientFactory(
        config=A2AClientConfig(streaming=True),
    ),
)
