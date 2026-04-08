"""Pulsarity Alert Manager Service - Manages alert rules and notifications."""

import logging
import os
import uuid
from datetime import datetime, timezone
from flask import Flask, jsonify, request

app = Flask(__name__)

LOG_LEVEL = os.environ.get("LOG_LEVEL", "INFO").upper()
logging.basicConfig(
    level=getattr(logging, LOG_LEVEL, logging.INFO),
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("alert-manager")

PORT = int(os.environ.get("ALERT_MANAGER_PORT", 8001))
COLLECTOR_URL = os.environ.get("COLLECTOR_URL", "http://localhost:8002")

# In-memory alert store
alerts: dict[str, dict] = {}


@app.route("/health")
def health():
    """Health check endpoint."""
    logger.debug("Health check requested")
    return jsonify({"status": "ok", "service": "alert-manager", "timestamp": datetime.now(timezone.utc).isoformat()})


@app.route("/alerts", methods=["GET"])
def list_alerts():
    """List all alert rules."""
    logger.info("Listing %d alert rules", len(alerts))
    return jsonify({"alerts": list(alerts.values()), "count": len(alerts)})


@app.route("/alerts", methods=["POST"])
def create_alert():
    """Create a new alert rule."""
    data = request.get_json()
    if not data:
        logger.warning("Create alert called with no JSON body")
        return jsonify({"error": "Request body must be JSON"}), 400

    name = data.get("name")
    target_url = data.get("target_url")
    threshold_ms = data.get("threshold_ms", 5000)
    notify_email = data.get("notify_email")

    if not name or not target_url:
        logger.warning("Create alert missing required fields: name=%s, target_url=%s", name, target_url)
        return jsonify({"error": "Fields 'name' and 'target_url' are required"}), 400

    alert_id = str(uuid.uuid4())[:8]
    alert = {
        "id": alert_id,
        "name": name,
        "target_url": target_url,
        "threshold_ms": threshold_ms,
        "notify_email": notify_email,
        "status": "active",
        "created_at": datetime.now(timezone.utc).isoformat(),
        "last_triggered": None,
    }
    alerts[alert_id] = alert
    logger.info("Created alert rule: id=%s name=%s target=%s", alert_id, name, target_url)
    return jsonify(alert), 201


@app.route("/alerts/<alert_id>", methods=["GET"])
def get_alert(alert_id):
    """Get a specific alert rule."""
    alert = alerts.get(alert_id)
    if not alert:
        logger.warning("Alert not found: %s", alert_id)
        return jsonify({"error": "Alert not found"}), 404
    return jsonify(alert)


@app.route("/alerts/<alert_id>", methods=["DELETE"])
def delete_alert(alert_id):
    """Delete an alert rule."""
    if alert_id not in alerts:
        logger.warning("Delete failed - alert not found: %s", alert_id)
        return jsonify({"error": "Alert not found"}), 404
    del alerts[alert_id]
    logger.info("Deleted alert rule: %s", alert_id)
    return jsonify({"message": "Alert deleted"}), 200


@app.route("/alerts/<alert_id>/trigger", methods=["POST"])
def trigger_alert(alert_id):
    """Manually trigger an alert (simulate notification)."""
    alert = alerts.get(alert_id)
    if not alert:
        return jsonify({"error": "Alert not found"}), 404
    alert["last_triggered"] = datetime.now(timezone.utc).isoformat()
    logger.info("Alert triggered: id=%s name=%s", alert_id, alert["name"])
    return jsonify({"message": "Alert triggered", "alert": alert})


def create_app():
    """Factory function for creating the Flask app."""
    return app


if __name__ == "__main__":
    logger.info("Starting Alert Manager on port %d", PORT)
    app.run(host="0.0.0.0", port=PORT)
