Part of [linux-diagnostic-client](../../README.md)

# database

`DB` uses a pgx connection pool for file inventory, log entries, and network packets.

## Run

```sh
PGPASSWORD=postgres make init-db
```

The command requires a TimescaleDB-enabled PostgreSQL server on `127.0.0.1:5432`.

## Test

```sh
make test
```

Database integration tests run in [CI](../../.github/workflows/go.yml).

## Config

| Variable | Default | What |
| --- | --- | --- |
| `DATABASE_URL` | required | Connection URL passed to `db.New` |

## Interfaces

| Interface | Details |
| --- | --- |
| `New(ctx, url)` | Creates and pings a pgx connection pool |
| `SaveFiles`, `SaveLogs`, `SaveNetworkPackets` | Persist agent data |
| `GetFileTree`, `GetLogs`, `SearchLogs` | REST query methods |
| `internal/db/schema.sql` | [Database schema](../../docs/reference/database.md) |
