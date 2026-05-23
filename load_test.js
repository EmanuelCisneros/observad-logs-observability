#!/usr/bin/env node
// load_test.js — Quick load test for the ingest endpoint.
// Usage: node load_test.js [--rps 500] [--duration 30] [--url http://localhost:8080]
//
// Sends batches of 100 logs at the requested rate and prints throughput + error stats.

const args = Object.fromEntries(
  process.argv.slice(2).reduce((acc, v, i, arr) => {
    if (v.startsWith("--")) acc.push([v.slice(2), arr[i + 1]]);
    return acc;
  }, [])
);

const URL      = (args.url      || "http://localhost:8080") + "/api/v1/logs";
const API_KEY  = args.key       || "dev-local-key";
const RPS      = parseInt(args.rps      || "200");
const DURATION = parseInt(args.duration || "20");
const BATCH    = 100;

const SERVICES = ["api", "worker", "cron", "auth", "billing"];
const LEVELS   = ["debug", "info", "info", "info", "warn", "error"];
const MESSAGES = [
  "Request handled successfully",
  "Database query completed in 12ms",
  "Cache miss — falling back to origin",
  "Retrying failed job (attempt 2/3)",
  "Connection pool at 80% capacity",
  "Unhandled exception in payment processor",
];

function randomLog() {
  return {
    service:   SERVICES[Math.floor(Math.random() * SERVICES.length)],
    level:     LEVELS[Math.floor(Math.random() * LEVELS.length)],
    message:   MESSAGES[Math.floor(Math.random() * MESSAGES.length)],
    timestamp: new Date().toISOString(),
    attributes: { region: "us-east-1", version: "1.4.2" },
  };
}

function buildBatch(n) {
  return JSON.stringify({ logs: Array.from({ length: n }, randomLog) });
}

async function sendBatch(body) {
  const t = Date.now();
  const r = await fetch(URL, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-API-Key": API_KEY,
    },
    body,
  });
  return { status: r.status, ms: Date.now() - t };
}

async function main() {
  console.log(`\nObserva load test`);
  console.log(`  Target: ${URL}`);
  console.log(`  Rate:   ${RPS} logs/s  (${RPS / BATCH} requests/s)`);
  console.log(`  Run:    ${DURATION}s\n`);

  const intervalMs = (BATCH / RPS) * 1000;
  let sent = 0, errors = 0, totalMs = 0, count = 0;
  const start = Date.now();
  const batch  = buildBatch(BATCH);

  const tick = setInterval(async () => {
    if (Date.now() - start >= DURATION * 1000) {
      clearInterval(tick);
      const elapsed = (Date.now() - start) / 1000;
      console.log(`\n── Results ─────────────────────────────`);
      console.log(`  Logs sent:       ${sent.toLocaleString()}`);
      console.log(`  Errors:          ${errors}`);
      console.log(`  Throughput:      ${Math.round(sent / elapsed).toLocaleString()} logs/s`);
      console.log(`  Avg latency:     ${count ? Math.round(totalMs / count) : 0}ms/request`);
      console.log(`  Elapsed:         ${elapsed.toFixed(1)}s`);
      return;
    }
    try {
      const { status, ms } = await sendBatch(batch);
      if (status >= 200 && status < 300) {
        sent += BATCH;
      } else {
        errors++;
        process.stdout.write(`E(${status}) `);
      }
      totalMs += ms;
      count++;
      if (count % 50 === 0) {
        process.stdout.write(
          `\r  ${sent.toLocaleString()} logs  ${errors} errors  ${Math.round(sent / ((Date.now() - start) / 1000)).toLocaleString()} logs/s  `,
        );
      }
    } catch (e) {
      errors++;
    }
  }, intervalMs);
}

main().catch(console.error);
