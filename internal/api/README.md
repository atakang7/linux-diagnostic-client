Part of [linux-diagnostic-client](../../README.md)

# HTTP API

`Server` binds the HTTP and agent listeners; `Handler` reads files, logs, and network statistics from `internal/db`.

## Test

```sh
go test ./internal/api
```

## Config

| Variable | Default | What |
| --- | --- | --- |
| `SERVER_ADDR` | `127.0.0.1:8080` | HTTP and WebSocket bind address |

## Interfaces

| Interface | Details |
| --- | --- |
| REST | [Routes and parameters](../../docs/reference/http-api.md) |
| `GET /healthz` | Process liveness |
| `GET /readyz` | Database ping with a 2-second deadline |
| `GET /ws` | Handled by `internal/websocket` |

The HTTP server uses 15-second read and write timeouts and a 60-second idle timeout.
