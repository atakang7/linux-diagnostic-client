# linux-diagnostic-client

Go server for collecting Linux Diagnostic Agent telemetry in TimescaleDB and exposing REST and WebSocket endpoints.

```text
+-------------+  TCP JSON  +-----------------+  SQL   +-------------+
| linux agent | ---------> | internal/tunnel | -----> | TimescaleDB |
+-------------+            +-----------------+        +-------------+
                                  |                          |
                                  v events                   v reads
                           +--------------------+     +--------------+
                           | internal/websocket |     | internal/api |
                           +--------------------+     +--------------+
                                  | WS                       | HTTP
                                  v                          v
                           +---------------+          +-------------+
                           | ws subscriber |          | rest client |
                           +---------------+          +-------------+
```

## Quick start

Requires Go 1.21, Docker, Make, and `psql`.

```sh
make dev-db
PGPASSWORD=postgres make init-db
export DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/diagnostic?sslmode=disable'
make run
```

## Layout

| Path | What it is |
| --- | --- |
| [cmd/api](cmd/api/README.md) | Executable entry point |
| [internal/api](internal/api/README.md) | HTTP handlers and listeners |
| [internal/config](internal/config/README.md) | Environment configuration |
| [internal/db](internal/db/README.md) | pgx queries and schema |
| [internal/tunnel](internal/tunnel/README.md) | Agent TCP ingestion |
| [internal/websocket](internal/websocket/README.md) | Live event fan-out |
| [pkg/models](pkg/models/README.md) | Shared telemetry types |
| [integration](integration/README.md) | End-to-end test |
| [init-scripts](init-scripts/README.md) | Database bootstrap SQL |

## Docs

[Documentation index](docs/README.md), [local setup](docs/how-to/run-locally.md), [API reference](docs/reference/http-api.md), and [architecture](docs/explanation/architecture.md).

## Limits

- HTTP, WebSocket, and agent TCP have no built-in authentication or TLS.
- Slow WebSocket subscribers may lose live events.
- File inventory is shared across agents and is identified by path.
- Database initialization requires TimescaleDB.
