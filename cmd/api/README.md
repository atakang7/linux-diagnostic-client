Part of [linux-diagnostic-client](../../README.md)

# api executable

`cmd/api/main.go` opens the database and runs the HTTP and agent TCP listeners.

## Run

```sh
DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/diagnostic?sslmode=disable' make run
```

Initialize TimescaleDB first with the [local setup](../../docs/how-to/run-locally.md).

## Test

```sh
go test ./internal/api ./internal/config
```

## Config

| Variable | Default | What |
| --- | --- | --- |
| `DATABASE_URL` | required | PostgreSQL connection string |
| `SERVER_ADDR` | `127.0.0.1:8080` | HTTP listener |
| `AGENT_ADDR` | `127.0.0.1:8081` | Agent TCP listener |

## Interfaces

| Interface | Location |
| --- | --- |
| HTTP and WebSocket | [HTTP reference](../../docs/reference/http-api.md) |
| Agent TCP | [Agent protocol](../../docs/reference/agent-protocol.md) |
| Shutdown | SIGINT or SIGTERM |
