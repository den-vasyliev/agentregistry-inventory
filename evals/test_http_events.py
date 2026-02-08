"""HTTP event pipeline tests: push events and verify status/incidents."""

import time
import uuid

import pytest


PROCESS_WAIT_SECONDS = 15


class TestPushEventAndCheckStatus:
    """Push infrastructure events via HTTP API and verify agent processing."""

    def test_push_critical_event(self, http_client):
        """Push a critical pod crash event and verify it is queued."""
        event_id = f"eval-{uuid.uuid4().hex[:8]}"
        resp = http_client.post(
            "/v0/agent/events",
            json={
                "source": "k8s/pod/production/nginx-eval",
                "type": "pod-crash",
                "severity": "critical",
                "message": f"Pod nginx-eval in production is CrashLoopBackOff (eval {event_id})",
            },
        )
        assert resp.status_code == 200
        body = resp.json()
        assert body["queued"] is True
        assert "id" in body

    def test_push_event_updates_status(self, http_client):
        """Push an event and verify the agent processes it into world state."""
        resp = http_client.post(
            "/v0/agent/events",
            json={
                "source": "k8s/node/worker-eval-1",
                "type": "node-pressure",
                "severity": "warning",
                "message": "Node worker-eval-1 is experiencing memory pressure",
            },
        )
        assert resp.status_code == 200
        assert resp.json()["queued"] is True

        # Wait for the agent to process the event
        time.sleep(PROCESS_WAIT_SECONDS)

        status = http_client.get("/v0/agent/status").json()
        assert status["running"] is True
        assert status["worldState"]["totalEvents"] >= 1

    def test_push_critical_creates_incident(self, http_client):
        """Push a critical event and verify an incident is created."""
        source = f"k8s/pod/staging/api-eval-{uuid.uuid4().hex[:6]}"
        resp = http_client.post(
            "/v0/agent/events",
            json={
                "source": source,
                "type": "pod-crash",
                "severity": "critical",
                "message": f"Pod {source.split('/')[-1]} in staging is OOMKilled",
            },
        )
        assert resp.status_code == 200

        # Wait for the agent to process and create an incident
        time.sleep(PROCESS_WAIT_SECONDS)

        status = http_client.get("/v0/agent/status").json()
        assert status["worldState"]["activeIncidents"] >= 1

        # Verify at least one incident exists
        incidents = status.get("incidents", [])
        assert len(incidents) >= 1

    def test_queue_depth_decreases(self, http_client):
        """Verify the event queue drains after processing."""
        # Push an event
        http_client.post(
            "/v0/agent/events",
            json={
                "source": "k8s/pod/default/drain-test",
                "type": "pod-restart",
                "severity": "info",
                "message": "Pod drain-test restarted",
            },
        )

        # Wait for processing
        time.sleep(PROCESS_WAIT_SECONDS)

        status = http_client.get("/v0/agent/status").json()
        # Queue should have drained (depth 0 or low)
        assert status["queue"]["depth"] <= 1
        assert status["queue"]["total"] >= 1


class TestEventValidation:
    """Verify API validation for event payloads."""

    def test_missing_required_fields(self, http_client):
        """Push event with missing required fields should fail."""
        resp = http_client.post(
            "/v0/agent/events",
            json={
                "source": "test",
                # missing type and message
            },
        )
        assert resp.status_code >= 400

    def test_empty_message(self, http_client):
        """Push event with empty message should fail."""
        resp = http_client.post(
            "/v0/agent/events",
            json={
                "source": "test",
                "type": "test",
                "message": "",
            },
        )
        assert resp.status_code >= 400
