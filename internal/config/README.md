Part of [linux-diagnostic-client](../../README.md)

# configuration

`Load` reads three environment variables and validates the listener ports.

## Test

```sh
go test ./internal/config
```

## Config

| Variable | Default | What |
| --- | --- | --- |
| `DATABASE_URL` | required | PostgreSQL connection string |
| `SERVER_ADDR` | `127.0.0.1:8080` | HTTP and WebSocket bind address |
| `AGENT_ADDR` | `127.0.0.1:8081` | Agent TCP bind address |

See [runtime settings](../../docs/reference/runtime.md) for the fixed buffer sizes.

## Interfaces

| Symbol | Details |
| --- | --- |
| `Load() (*Config, error)` | Returns validated listener addresses and database URL |
| `Config` | Shared settings passed to the server and tunnel |
