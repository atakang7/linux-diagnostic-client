# HTTP API

The HTTP listener defaults to `127.0.0.1:8080`.

| Method | Path | Request | Response |
| --- | --- | --- | --- |
| GET | `/healthz` | none | `ok` |
| GET | `/readyz` | none | `ready` or HTTP 503 |
| GET | `/api/files` | `path` default `/`; `depth` default 1, capped at 10 | Array of file nodes |
| GET | `/api/logs` | `file` required; `before` optional RFC3339 timestamp | Up to 100 log entries |
| POST | `/api/logs/search` | JSON `query` required; optional `files`, `start_time`, `end_time` | Up to 1,000 log entries |
| GET | `/api/network/metrics` | Optional RFC3339 `start`, `end`, repeated `protocol` | Aggregates plus up to 1,000 packets |
| GET | `/ws` | WebSocket upgrade | [Event stream](stream.md) |

`start_time` and `end_time` are JSON timestamps; `start`, `end`, and `before` are URL query parameters.

The log search request body is bounded to 1 MiB. Invalid parameters return HTTP 400 and database errors return HTTP 500.
