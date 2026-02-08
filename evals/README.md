# Master Agent Evaluation Suite

Python-based eval suite for testing the master agent via A2A protocol and HTTP API.

## Prerequisites

1. Controller running with a `MasterAgentConfig` resource applied
2. A `ModelCatalog` configured so the agent has a working LLM backend
3. Python 3.11+

## Setup

```bash
cd evals
pip install -e .
```

## Running Evals

### All tests

```bash
pytest -v
```

### A2A evals (via AgentEvaluator)

```bash
pytest test_a2a_eval.py -v
```

### HTTP event pipeline tests

```bash
pytest test_http_events.py -v
```

## Configuration

| Environment Variable | Default | Description |
|---|---|---|
| `MASTER_AGENT_API_URL` | `http://localhost:8080` | HTTP API base URL |
| `MASTER_AGENT_A2A_URL` | `http://localhost:8084` | A2A server base URL |

## Eval Datasets

- `eval_datasets/triage_basic.test.json` - Basic triage: infrastructure event triggers incident creation
- `eval_datasets/multi_event.test.json` - Multi-event lifecycle: create then resolve incidents
- `eval_datasets/test_config.json` - Criteria thresholds for eval scoring

## Adding New Evals

1. Create a new `.test.json` file in `eval_datasets/`
2. Each entry needs `query`, `expected_tool_use`, and `reference` fields
3. Add a test function in `test_a2a_eval.py` pointing to the new dataset
