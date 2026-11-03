Part of [linux-diagnostic-client](../README.md)

# integration tests

The end-to-end test starts the compiled client and checks TCP ingestion, persistence, REST reads, WebSocket fan-out, and SIGINT shutdown.

## Run

Initialize TimescaleDB using the [local setup](../docs/how-to/run-locally.md), then run:

```sh
make build
CLIENT_BINARY="$PWD/bin/diagnostic-client" DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/diagnostic?sslmode=disable' go test -tags=integration -v -count=1 ./integration
```

## Config

| Variable | Default | What |
| --- | --- | --- |
| `CLIENT_BINARY` | required | Path to the compiled executable |
| `DATABASE_URL` | required | Existing initialized TimescaleDB database |

## Interfaces

| File | Details |
| --- | --- |
| `e2e_test.go` | `TestTCPIngestionRESTAndWebSocket` |
| [CI workflow](../.github/workflows/go.yml) | Runs the test against a TimescaleDB service |
