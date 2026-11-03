Part of [linux-diagnostic-client](../../README.md)

# agent tunnel

The TCP listener decodes agent messages, persists records through `internal/db`, and sends events to the WebSocket fan-out.

## Test

```sh
go test ./internal/tunnel
```

The [integration test](../../integration/README.md) exercises TCP ingestion.

## Config

| Variable | Default | What |
| --- | --- | --- |
| `AGENT_ADDR` | `127.0.0.1:8081` | TCP listener |

## Interfaces

| Message type | Action |
| --- | --- |
| `log_list` | Reconcile the shared file inventory |
| `log_data` | Store logs and emit log events |
| `metrics` | Batch network packets and emit network events |

Agent framing and payloads are in the [protocol reference](../../docs/reference/agent-protocol.md). Network batches flush at 10,000 packets or every 5 seconds.
