import express, { Request, Response } from "express";

const ALERT_MANAGER_URL = process.env.ALERT_MANAGER_URL || "http://localhost:8001";
const COLLECTOR_URL = process.env.COLLECTOR_URL || "http://localhost:8002";

interface ServiceStatus {
  name: string;
  url: string;
  status: string;
  responseMs: number;
  checkedAt: string;
}

interface DashboardSummary {
  services: ServiceStatus[];
  totalServices: number;
  healthyCount: number;
  unhealthyCount: number;
  timestamp: string;
}

function log(level: string, message: string): void {
  const timestamp = new Date().toISOString();
  console.log(`${timestamp} [${level.toUpperCase()}] dashboard-api: ${message}`);
}

async function checkService(name: string, url: string): Promise<ServiceStatus> {
  const start = Date.now();
  try {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 5000);
    const resp = await fetch(`${url}/health`, { signal: controller.signal });
    clearTimeout(timeout);
    const elapsed = Date.now() - start;
    return {
      name,
      url,
      status: resp.ok ? "healthy" : "unhealthy",
      responseMs: elapsed,
      checkedAt: new Date().toISOString(),
    };
  } catch {
    return {
      name,
      url,
      status: "unreachable",
      responseMs: Date.now() - start,
      checkedAt: new Date().toISOString(),
    };
  }
}

export function createApp(): express.Application {
  const app = express();
  app.use(express.json());

  app.get("/health", (_req: Request, res: Response) => {
    log("debug", "Health check requested");
    res.json({
      status: "ok",
      service: "dashboard-api",
      timestamp: new Date().toISOString(),
    });
  });

  app.get("/api/dashboard/summary", async (_req: Request, res: Response) => {
    log("info", "Dashboard summary requested");
    const services = await Promise.all([
      checkService("alert-manager", ALERT_MANAGER_URL),
      checkService("health-collector", COLLECTOR_URL),
    ]);

    const healthyCount = services.filter((s) => s.status === "healthy").length;
    const summary: DashboardSummary = {
      services,
      totalServices: services.length,
      healthyCount,
      unhealthyCount: services.length - healthyCount,
      timestamp: new Date().toISOString(),
    };

    res.json(summary);
  });

  app.get("/api/services", (_req: Request, res: Response) => {
    log("info", "Service list requested");
    res.json({
      services: [
        { name: "alert-manager", port: 8001, description: "Manages alert rules and notifications" },
        { name: "health-collector", port: 8002, description: "Polls endpoints and records health status" },
        { name: "dashboard-api", port: 8003, description: "API gateway for the monitoring dashboard" },
      ],
    });
  });

  app.get("/api/config", (_req: Request, res: Response) => {
    log("info", "Config requested");
    res.json({
      alertManagerUrl: ALERT_MANAGER_URL,
      collectorUrl: COLLECTOR_URL,
      version: "1.0.0",
    });
  });

  return app;
}
