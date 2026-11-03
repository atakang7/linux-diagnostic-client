# Runtime settings

Environment variables are listed in [internal/config](../../internal/config/README.md). The following values are fixed in the current code.

| Setting | Value | Source |
| --- | --- | --- |
| Log event channel capacity | 10,000 entries | `internal/config/config.go` |
| Network event channel capacity | 50,000 batches | `internal/config/config.go` |
| Network batch threshold | 10,000 packets | `internal/config/config.go` |
| Network flush interval | 5 seconds | `internal/tunnel/handler.go` |
| WebSocket client queue | 128 messages | `internal/websocket/handler.go` |
| WebSocket ping interval | 30 seconds | `internal/websocket/handler.go` |
| WebSocket read deadline | 90 seconds, refreshed by pong | `internal/websocket/handler.go` |
| HTTP read and write timeouts | 15 seconds each | `internal/api/server.go` |
| HTTP idle timeout | 60 seconds | `internal/api/server.go` |
| Database pool | 5 minimum, 20 maximum connections | `internal/db/db.go` |
