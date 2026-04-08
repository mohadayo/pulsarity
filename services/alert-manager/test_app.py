"""Tests for the Alert Manager service."""

import json
import pytest
from app import create_app, alerts


@pytest.fixture
def client():
    """Create a test client."""
    app = create_app()
    app.config["TESTING"] = True
    with app.test_client() as client:
        alerts.clear()
        yield client


def test_health_check(client):
    """Test health endpoint returns ok status."""
    resp = client.get("/health")
    assert resp.status_code == 200
    data = json.loads(resp.data)
    assert data["status"] == "ok"
    assert data["service"] == "alert-manager"
    assert "timestamp" in data


def test_list_alerts_empty(client):
    """Test listing alerts when none exist."""
    resp = client.get("/alerts")
    assert resp.status_code == 200
    data = json.loads(resp.data)
    assert data["alerts"] == []
    assert data["count"] == 0


def test_create_alert(client):
    """Test creating a new alert rule."""
    payload = {
        "name": "Test Alert",
        "target_url": "http://example.com/health",
        "threshold_ms": 3000,
        "notify_email": "test@example.com",
    }
    resp = client.post("/alerts", data=json.dumps(payload), content_type="application/json")
    assert resp.status_code == 201
    data = json.loads(resp.data)
    assert data["name"] == "Test Alert"
    assert data["target_url"] == "http://example.com/health"
    assert data["threshold_ms"] == 3000
    assert data["status"] == "active"
    assert "id" in data


def test_create_alert_missing_fields(client):
    """Test creating alert with missing required fields."""
    resp = client.post("/alerts", data=json.dumps({"name": "Incomplete"}), content_type="application/json")
    assert resp.status_code == 400


def test_create_alert_no_body(client):
    """Test creating alert with no JSON body."""
    resp = client.post("/alerts", content_type="application/json")
    assert resp.status_code == 400


def test_get_alert(client):
    """Test getting a specific alert."""
    payload = {"name": "Get Test", "target_url": "http://example.com"}
    create_resp = client.post("/alerts", data=json.dumps(payload), content_type="application/json")
    alert_id = json.loads(create_resp.data)["id"]

    resp = client.get(f"/alerts/{alert_id}")
    assert resp.status_code == 200
    data = json.loads(resp.data)
    assert data["name"] == "Get Test"


def test_get_alert_not_found(client):
    """Test getting a non-existent alert."""
    resp = client.get("/alerts/nonexistent")
    assert resp.status_code == 404


def test_delete_alert(client):
    """Test deleting an alert rule."""
    payload = {"name": "Delete Test", "target_url": "http://example.com"}
    create_resp = client.post("/alerts", data=json.dumps(payload), content_type="application/json")
    alert_id = json.loads(create_resp.data)["id"]

    resp = client.delete(f"/alerts/{alert_id}")
    assert resp.status_code == 200

    resp = client.get(f"/alerts/{alert_id}")
    assert resp.status_code == 404


def test_delete_alert_not_found(client):
    """Test deleting a non-existent alert."""
    resp = client.delete("/alerts/nonexistent")
    assert resp.status_code == 404


def test_trigger_alert(client):
    """Test triggering an alert."""
    payload = {"name": "Trigger Test", "target_url": "http://example.com"}
    create_resp = client.post("/alerts", data=json.dumps(payload), content_type="application/json")
    alert_id = json.loads(create_resp.data)["id"]

    resp = client.post(f"/alerts/{alert_id}/trigger")
    assert resp.status_code == 200
    data = json.loads(resp.data)
    assert data["alert"]["last_triggered"] is not None


def test_trigger_alert_not_found(client):
    """Test triggering a non-existent alert."""
    resp = client.post("/alerts/nonexistent/trigger")
    assert resp.status_code == 404


def test_list_alerts_after_create(client):
    """Test listing alerts after creating some."""
    for i in range(3):
        payload = {"name": f"Alert {i}", "target_url": f"http://example{i}.com"}
        client.post("/alerts", data=json.dumps(payload), content_type="application/json")

    resp = client.get("/alerts")
    data = json.loads(resp.data)
    assert data["count"] == 3
    assert len(data["alerts"]) == 3
