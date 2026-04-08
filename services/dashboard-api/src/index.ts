import { createApp } from "./app";

const PORT = parseInt(process.env.DASHBOARD_PORT || "8003", 10);
const LOG_LEVEL = process.env.LOG_LEVEL || "info";

const app = createApp();

if (LOG_LEVEL === "debug") {
  console.log("[dashboard-api] Debug logging enabled");
}

app.listen(PORT, () => {
  console.log(`[dashboard-api] Starting Dashboard API on port ${PORT}`);
});
