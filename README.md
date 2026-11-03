# Linux Diagnostic Client

A Go receiver for telemetry sent by [Linux Diagnostic Agent](https://github.com/atakang7/linux-diagnostic-agent). It saves log, file, and network records in PostgreSQL with TimescaleDB, then serves them over HTTP and WebSocket.

## Quick start

Requires Go 1.21 or newer, Docker, `make`, and the PostgreSQL client (`psql`).

```sh
git clone https://github.com/atakang7/linux-diagnostic-client.git
cd linux-diagnostic-client
make dev-db
# Once the database is ready:
PGPASSWORD=postgres make init-db
set -a
. ./.env.example
set +a
make build
./bin/diagnostic-client
```

The development database is published only on `127.0.0.1:5432`. Use `make run` instead of `make build` and the binary if you prefer to run from source.

## Configuration

The service reads environment variables at startup. It does not read `.env` files itself.

| Variable | Required | Default |
| --- | --- | --- |
| `DATABASE_URL` | Yes | None |
| `SERVER_ADDR` | No | `127.0.0.1:8080` |
| `AGENT_ADDR` | No | `127.0.0.1:8081` |

`.env.example` contains local-only database credentials. Supply a separate `DATABASE_URL` with suitable credentials and TLS settings outside development. Keep secrets out of version control.

## How it works

The agent sends JSON messages on TCP port 8081. The receiver writes file metadata and logs to PostgreSQL, batches network packets, and forwards new events to WebSocket viewers. The database schema requires TimescaleDB.

```text
Agent -> TCP :8081 -> receiver -> PostgreSQL / TimescaleDB
                            |
                            +-> HTTP :8080
                            +-> WebSocket :8080/ws
```

## Usage

| Method | Endpoint | Result |
| --- | --- | --- |
| GET | `/healthz` | Process liveness |
| GET | `/readyz` | Database readiness |
| GET | `/api/files?path=/&depth=1` | File tree, depth capped at 10 |
| GET | `/api/logs?file=/var/log/app.log` | Latest 100 entries; optional RFC3339 `before` |
| POST | `/api/logs/search` | Log search, JSON `query`; optional `files`, `start_time`, `end_time` |
| GET | `/api/network/metrics` | Packet statistics and up to 1,000 packets; optional `start`, `end`, `protocol` |
| GET | `/ws` | Live `network`, `file_update`, and `log` events |

```sh
curl -fsS http://127.0.0.1:8080/readyz
curl -fsS 'http://127.0.0.1:8080/api/files?path=/&depth=1'
curl -fsS -H 'Content-Type: application/json' \
  -d '{"query":"connection refused"}' \
  http://127.0.0.1:8080/api/logs/search
```

WebSocket events have `type` and `payload` fields. To follow logs for one file, send `{"type":"view_file","payload":"/var/log/app.log"}` after connecting to `/ws`.

## Running outside development

Provision PostgreSQL with TimescaleDB, apply `internal/db/schema.sql` once to a fresh instance, set `DATABASE_URL`, and run the built binary. Configure listener addresses through `SERVER_ADDR` and `AGENT_ADDR`. Put access controls and TLS at the network boundary before accepting traffic from other machines.

```sh
make build
make vet
make test
```

CI runs formatting checks, dependency verification, `go build`, `go vet`, and race-enabled tests. Its integration job starts TimescaleDB and tests ingestion, persistence, HTTP reads, WebSocket delivery, and shutdown.

## Limitations

- Neither TCP ingestion nor HTTP/WebSocket endpoints provide authentication or TLS.
- Subscriber queues are bounded; slow WebSocket clients can lose events.
- File inventory is shared across agents, so overlapping sources can overwrite one another.
- PostgreSQL without the TimescaleDB extension cannot run this schema.

Licensed under MIT. See [LICENSE](LICENSE).
