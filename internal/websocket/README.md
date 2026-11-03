Part of [linux-diagnostic-client](../../README.md)

# WebSocket fan-out

One reader per tunnel stream broadcasts events to connected WebSocket clients.

## Test

```sh
go test ./internal/websocket
```

## Config

| Variable | Default | What |
| --- | --- | --- |
| `SERVER_ADDR` | `127.0.0.1:8080` | HTTP listener for `/ws` |

## Interfaces

| Interface | Details |
| --- | --- |
| `GET /ws` | WebSocket upgrade with same-origin checks |
| `view_file` | Selects one log file for log events |
| `network`, `file_update`, `log` | Outbound event types |

See [stream messages](../../docs/reference/stream.md) for payload formats. Each viewer has a 128-message queue; full queues drop events.
