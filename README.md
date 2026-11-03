# Linux Diagnostic Client

Receives JSON telemetry from [Linux Diagnostic Agent](https://github.com/atakang7/linux-diagnostic-agent), stores it in TimescaleDB, and serves HTTP queries and WebSocket events.

```text
+------------------------+ JSON/TCP +----------------+ events +-------------------+
| Linux Diagnostic Agent | -------> | tunnel.Handler | -----> | websocket.Handler |
+------------------------+          +----------------+        +-------------------+
                                           | writes
                                           v
                                    +---------------+ results +-------------+
                                    | db.DB         | ------> | api.Handler |
                                    | (TimescaleDB) |         +-------------+
                                    +---------------+
```

## Run

Requires Go 1.21+, Docker, `make`, and `psql`.

```sh
git clone https://github.com/atakang7/linux-diagnostic-client.git
cd linux-diagnostic-client
make dev-db
until docker exec diagnostic-postgres pg_isready -U postgres -d postgres; do sleep 1; done
PGPASSWORD=postgres make init-db
set -a
. ./.env.example
set +a
make build
./bin/diagnostic-client
```

## Config

The process reads environment variables; `.env.example` contains loopback-only development settings.

| Var | Default | What |
| --- | --- | --- |
| `DATABASE_URL` | Required | PostgreSQL/TimescaleDB connection string |
| `SERVER_ADDR` | `127.0.0.1:8080` | HTTP and WebSocket listener |
| `AGENT_ADDR` | `127.0.0.1:8081` | Agent TCP listener |

## CI

```text
+----------------+ triggers +------------+ passes +-----------------+
| PR / main push | -------> | verify     | -----> | integration     |
+----------------+          +------------+        +-----------------+
                                                        | starts
                                                        v
                                                  +-------------+
                                                  | TimescaleDB |
                                                  +-------------+
```

## Limits

- No built-in authentication or TLS; restrict network access.
- Slow WebSocket subscribers can miss events.
- File inventory is shared between agents; overlapping agents can overwrite records.
- The database schema requires the TimescaleDB extension.
