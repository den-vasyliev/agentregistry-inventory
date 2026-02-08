"""Shared fixtures for master agent evals."""

import os

import httpx
import pytest


API_URL = os.getenv("MASTER_AGENT_API_URL", "http://localhost:8080")
A2A_URL = os.getenv("MASTER_AGENT_A2A_URL", "http://localhost:8084")


@pytest.fixture(scope="session")
def api_url():
    return API_URL


@pytest.fixture(scope="session")
def a2a_url():
    return A2A_URL


@pytest.fixture(scope="session", autouse=True)
def check_agent_health():
    """Skip all tests if the master agent is not reachable."""
    try:
        resp = httpx.get(f"{API_URL}/v0/agent/status", timeout=5)
        resp.raise_for_status()
    except (httpx.ConnectError, httpx.TimeoutException, httpx.HTTPStatusError) as exc:
        pytest.skip(f"Master agent not reachable at {API_URL}: {exc}")


@pytest.fixture(scope="session")
def http_client():
    """Shared httpx client for the test session."""
    with httpx.Client(base_url=API_URL, timeout=30) as client:
        yield client
