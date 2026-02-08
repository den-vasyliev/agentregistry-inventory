"""A2A-based evaluations using ADK AgentEvaluator."""

import os

import pytest
from google.adk.evaluation.agent_evaluator import AgentEvaluator

EVAL_DIR = os.path.join(os.path.dirname(__file__), "eval_datasets")


@pytest.mark.asyncio
async def test_triage_basic():
    """Evaluate basic infrastructure event triage via A2A."""
    await AgentEvaluator.evaluate(
        agent_module="master_agent",
        eval_dataset_file_path_or_dir=os.path.join(EVAL_DIR, "triage_basic.test.json"),
        num_runs=1,
    )


@pytest.mark.asyncio
async def test_multi_event():
    """Evaluate multi-event processing and incident lifecycle via A2A."""
    await AgentEvaluator.evaluate(
        agent_module="master_agent",
        eval_dataset_file_path_or_dir=os.path.join(EVAL_DIR, "multi_event.test.json"),
        num_runs=1,
    )
