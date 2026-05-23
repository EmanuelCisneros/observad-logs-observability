# Observa

Self-hosted log viewer. You POST logs over HTTP, they land in ClickHouse, and you browse them in a Next.js UI with search, charts, and live tail.

Stack: Go backend (`observad`), Redis as a write buffer, ClickHouse for storage, Next.js frontend.

```
  Next.js UI  ──HTTP/WS──▶  observad  ──Redis Stream──▶  workers  ──▶  ClickHouse
```

What works today:

- HTTP ingest (up to 10k lines per request)
- Search with time range, level, service, and message filters
- Dashboard with counts, histogram, top services
- Live tail over WebSocket
- API key on all `/api/v1/logs/*` routes
- Docker Compose for local dev
- Tests in Go and TypeScript

---

## Run it locally

You need Docker. Node 20+ if you want the frontend with hot reload. Go 1.22+ only if you're building the backend outside Docker.

**Backend + infra**

```bash
cd deploy
docker compose up --build
```

That starts ClickHouse (8123/9000), Redis (6379), observad (8080), Prometheus (9090), and Grafana (3001). Schema goes in on first boot from `backend/migrations/001_init.sql`.

**Frontend**

```bash
cd frontend
cp .env.example .env.local
npm install
npm run dev
```

Open http://localhost:3000.

Important: put env vars in `frontend/.env.local`, not a `.env` at the repo root. Next.js won't pick that up. `NEXT_PUBLIC_API_KEY` has to match `OBSERVA_INGEST_API_KEY` on the backend (`dev-local-key` in the default compose file). Restart `npm run dev` after changing env vars.

**Send a test batch**

```bash
curl -X POST http://localhost:8080/api/v1/logs \
  -H "Content-Type: application/json" \
  -H "X-API-Key: dev-local-key" \
  -d '{
    "logs": [
      { "service": "api",    "level": "info",  "message": "Server started" },
      { "service": "api",    "level": "error", "message": "Connection timeout" },
      { "service": "worker", "level": "warn",  "message": "Queue depth high" }
    ]
  }'
```

The dashboard polls every 15s. For instant output, open Live Tail.

**Tests**

```bash
cd backend && go test ./...
cd frontend && npm test
```

---

## Config

### Backend

For local runs without Docker, copy `backend/.env.example` to `backend/.env`. With Compose, edit `deploy/docker-compose.yml` or drop a `.env` next to it.

| Variable | Default | Notes |
|----------|---------|-------|
| `OBSERVA_HTTP_ADDR` | `:8080` | |
| `OBSERVA_LOG_LEVEL` | `info` | |
| `OBSERVA_INGEST_API_KEY` | `dev-local-key` | Used on every `/api/v1/logs/*` route |
| `OBSERVA_CORS_ORIGINS` | `*` | Comma-separated |
| `OBSERVA_INGEST_RATE_LIMIT` | `1000` | Ingest POSTs/sec, backed by Redis |
| `OBSERVA_CH_ADDR` | `127.0.0.1:9000` | ClickHouse native port |
| `OBSERVA_CH_DB` | `observa` | |
| `OBSERVA_CH_USER` | `default` | |
| `OBSERVA_CH_PASSWORD` | | |
| `OBSERVA_REDIS_ADDR` | `127.0.0.1:6379` | |
| `OBSERVA_REDIS_PASSWORD` | | |
| `OBSERVA_REDIS_DB` | `0` | |
| `OBSERVA_REDIS_STREAM` | `observa:ingest` | |
| `OBSERVA_BATCH_SIZE` | `10000` | Rows per insert |
| `OBSERVA_FLUSH_INTERVAL` | `1s` | |
| `OBSERVA_WORKERS` | `4` | Ingest workers |
| `OBSERVA_RETENTION` | `720h` | TTL, applied on startup |
| `OBSERVA_ALLOW_MEMSTORE` | | Set `1` to use in-memory store if CH is down (dev only, no persistence) |

### Frontend

Copy `frontend/.env.example` to `frontend/.env.local`:

| Variable | Default | Notes |
|----------|---------|-------|
| `NEXT_PUBLIC_API_BASE` | `http://localhost:8080` | No trailing slash |
| `NEXT_PUBLIC_API_KEY` | | Same as `OBSERVA_INGEST_API_KEY` |

---

## API

Everything under `/api/v1/logs` needs auth: header `X-API-Key`, or `Authorization: Bearer <key>`. WebSocket tail accepts `?api_key=` because browsers can't set headers on the upgrade.

`/healthz` and `/metrics` are open.

**POST /api/v1/logs** — ingest. Max 10k lines, 64 KiB/message. Rate limited. Returns 202 or 207 (partial rejections in `errors`).

```json
{
  "logs": [
    {
      "service": "my-service",
      "message": "something happened",
      "level": "info",
      "timestamp": "2024-06-01T12:00:00Z",
      "trace_id": "abc123",
      "host": "prod-1",
      "attributes": { "user_id": "42" }
    }
  ]
}
```

Only `service` and `message` are required. `level` defaults to `info`, `timestamp` to now.

**GET /api/v1/logs/search** — paginated logs.

| Param | Example |
|-------|---------|
| `from`, `to` | `1h`, `30m`, RFC3339 |
| `levels` | `error,warn` |
| `services` | `api,worker` |
| `search` | substring on message |
| `limit` | max 1000 |
| `offset` | |
| `order` | `asc` / `desc` |

**GET /api/v1/logs/stats** — same filters. Histogram comes from the `logs_by_minute` view unless you pass `search`.

**WS /api/v1/logs/tail** — one JSON log per message. Optional `service`, `level`, `api_key`.

---

## How writes flow

1. POST validates the batch and XADDs to Redis (~1M maxlen)
2. Workers read via consumer group: reclaim stale pending entries, then new ones
3. Batch insert into ClickHouse (`ReplacingMergeTree`, dedupe by log id)
4. XACK on success, broadcast to tail subscribers

Useful metrics: `observa_ingest_queue_depth`, `observa_ingest_queue_pending`, `observa_ingest_to_tail_seconds`.

Redis Streams instead of Kafka: good enough well below ~100k logs/sec, much less ops overhead. Kafka adapter is on the wishlist.

ClickHouse instead of Elasticsearch: cheaper storage, fast aggregations, weaker fuzzy search. Fine for substring search at this stage.

---

## When something breaks

**401 / missing API key in the UI**

`frontend/.env.local` with the right `NEXT_PUBLIC_API_KEY`. Restart the dev server.

**Sent logs but dashboard is empty**

Wait 15s or hit stats directly:

```bash
curl -H "X-API-Key: dev-local-key" \
  "http://localhost:8080/api/v1/logs/stats?from=1h"
```

**Old ClickHouse volume from before ReplacingMergeTree**

```bash
cd deploy
docker compose down -v
docker compose up --build
```

---

## Repo layout

```
observa/
├── backend/
│   ├── cmd/observad/       entrypoint
│   ├── internal/
│   │   ├── config/
│   │   ├── httpapi/        routes, auth, websocket
│   │   ├── ingest/         redis buffer + workers
│   │   ├── model/
│   │   ├── query/
│   │   ├── realtime/       tail fan-out
│   │   └── store/          clickhouse + mem fallback
│   ├── migrations/
│   └── pkg/clog/
├── frontend/
│   └── src/                next.js app
└── deploy/
    ├── docker-compose.yml
    └── prometheus.yml
```

---

## Later

- Multi-tenant auth (Postgres)
- Kafka buffer backend
- Alerts
- OpenTelemetry traces
- Helm chart

---

Author: [Emanuel Cisneros](https://github.com/EmanuelCisneros) — April 25, 2026
