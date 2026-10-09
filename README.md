# Linux Diagnostic Client

Go service for receiving telemetry from the [Linux Diagnostic Agent](https://github.com/AtakanG7/linux-diagnostic-agent). It stores log and network data in PostgreSQL/TimescaleDB and exposes a REST API and live WebSocket stream.

## Architecture

```text
Linux Diagnostic Agent
         | TCP :8081 (newline-delimited JSON)
         v
   Tunnel handler -----> PostgreSQL/TimescaleDB -----> REST :8080
         |
         +-------------> WebSocket fan-out :8080/ws
```

## Run locally

Requires Go 1.21+, Docker, and the PostgreSQL CLI (`psql`).

```sh
make dev-db        # localhost-only TimescaleDB container
make init-db       # create database and schema
make build
./bin/diagnostic-client
```

The database container is optional when a TimescaleDB-enabled PostgreSQL instance is already available. `make test` runs race-enabled unit tests; CI also launches a real database and executes integration tests.

| Variable | Default | Purpose |
| --- | --- | --- |
| `DATABASE_URL` | `postgres://postgres:postgres@127.0.0.1:5432/diagnostic?sslmode=disable` | Database connection |
| `SERVER_ADDR` | `127.0.0.1:8080` | HTTP and WebSocket listener |
| `AGENT_ADDR` | `127.0.0.1:8081` | Agent TCP listener |

These defaults use demo credentials and bind locally. Do not expose the service to untrusted networks.

## HTTP and WebSocket

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/files?path=/&depth=1` | Discovered files; depth 1–10 |
| GET | `/api/logs?file=/var/log/app.log` | Up to 100 log entries; optional RFC3339 `before` |
| POST | `/api/logs/search` | Full-text search with JSON `query`, optional `files`, `start_time`, `end_time` |
| GET | `/api/network/metrics` | Aggregated statistics and latest 1,000 matching packets; optional `start`, `end`, `protocol` |
| GET | `/ws` | Live network, file, and selected log events |

```sh
curl -sS http://127.0.0.1:8080/api/files
curl -sS -H 'Content-Type: application/json' -d '{"query":"connection refused"}' \
  http://127.0.0.1:8080/api/logs/search
```

WebSocket messages have `type` and `payload`. Event types are `network`, `file_update`, and `log`. To receive events for a particular log file, send:

```json
{"type":"view_file","payload":"/var/log/app.log"}
```

Validation errors return HTTP 400; unexpected database failures return HTTP 500 without exposing internal details.

## Verification

```sh
make build
make vet
make test
```

The CI integration job creates a real TimescaleDB instance, compiles and launches the client, and verifies TCP ingestion, database persistence, REST reads and two simultaneous WebSocket subscribers.

## Operational boundaries

- This project has **no built-in authentication or TLS** for REST, WebSocket, or agent TCP. Keep it on loopback or inside an appropriately protected network.
- WebSocket subscribers have bounded queues; slow consumers can miss updates.
- File inventory is global, not partitioned by agent identity. Avoid running independent overlapping agents.
- Plain PostgreSQL is insufficient: the schema uses TimescaleDB.
