# Agent TCP protocol

The agent connects to `AGENT_ADDR`, which defaults to `127.0.0.1:8081`. The server decodes a stream of JSON objects.

| `type` | `payload` | Storage |
| --- | --- | --- |
| `log_list` | Array of `FileNode` objects | Upserts, updates, and deletions in `files` |
| `log_data` | Array of `LogEntry` objects | Inserts into `logs` |
| `metrics` | Object with `timestamp` string and `packets` array | Inserts into `network_packets` |

```json
{"type":"metrics","payload":{"timestamp":"2026-01-01T00:00:00Z","packets":[]}}
```

Record fields are defined by [Go models](../../pkg/models/models.go). The listener has no built-in authentication or TLS.
