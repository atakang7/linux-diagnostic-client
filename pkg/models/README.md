Part of [linux-diagnostic-client](../../README.md)

# models

This package defines the JSON records shared by the TCP ingestion path and database queries.

## Test

```sh
go test ./pkg/models
```

## Config

There are no environment variables.

## Interfaces

| Go type | Used for |
| --- | --- |
| `FileNode` | File inventory records |
| `LogEntry` | Log lines with timestamps and severity |
| `NetworkPacket` | Packet metadata |
| `NetworkStats` | Aggregates and up to 1,000 packet records |
| `TopNetworkStats` | Top source, destination, protocol, and port counts |

JSON field names are defined in [models.go](models.go).
