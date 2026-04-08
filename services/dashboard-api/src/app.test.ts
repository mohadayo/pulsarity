import request from "supertest";
import { createApp } from "./app";

const app = createApp();

describe("Dashboard API", () => {
  describe("GET /health", () => {
    it("should return ok status", async () => {
      const res = await request(app).get("/health");
      expect(res.status).toBe(200);
      expect(res.body.status).toBe("ok");
      expect(res.body.service).toBe("dashboard-api");
      expect(res.body.timestamp).toBeDefined();
    });
  });

  describe("GET /api/services", () => {
    it("should return the list of services", async () => {
      const res = await request(app).get("/api/services");
      expect(res.status).toBe(200);
      expect(res.body.services).toHaveLength(3);
      expect(res.body.services[0].name).toBe("alert-manager");
      expect(res.body.services[1].name).toBe("health-collector");
      expect(res.body.services[2].name).toBe("dashboard-api");
    });

    it("should include port and description for each service", async () => {
      const res = await request(app).get("/api/services");
      for (const svc of res.body.services) {
        expect(svc.port).toBeDefined();
        expect(svc.description).toBeDefined();
      }
    });
  });

  describe("GET /api/config", () => {
    it("should return configuration", async () => {
      const res = await request(app).get("/api/config");
      expect(res.status).toBe(200);
      expect(res.body.alertManagerUrl).toBeDefined();
      expect(res.body.collectorUrl).toBeDefined();
      expect(res.body.version).toBe("1.0.0");
    });
  });

  describe("GET /api/dashboard/summary", () => {
    it("should return dashboard summary with service statuses", async () => {
      const res = await request(app).get("/api/dashboard/summary");
      expect(res.status).toBe(200);
      expect(res.body.services).toBeDefined();
      expect(res.body.totalServices).toBe(2);
      expect(res.body.timestamp).toBeDefined();
      expect(typeof res.body.healthyCount).toBe("number");
      expect(typeof res.body.unhealthyCount).toBe("number");
    });

    it("should mark unreachable services correctly", async () => {
      const res = await request(app).get("/api/dashboard/summary");
      for (const svc of res.body.services) {
        expect(["healthy", "unhealthy", "unreachable"]).toContain(svc.status);
        expect(svc.responseMs).toBeDefined();
        expect(svc.checkedAt).toBeDefined();
      }
    });
  });
});
