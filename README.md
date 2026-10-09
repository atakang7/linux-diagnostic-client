# Linux Diagnostic Client

Go API and WebSocket service for the [Linux Diagnostic Agent](https://github.com/AtakanG7/linux-diagnostic-agent).

## Local setup

Requires Go 1.21+, PostgreSQL with TimescaleDB, and the database schema in `internal/db/schema.sql`.

```bash
# Run the SQL against a PostgreSQL instance with TimescaleDB installed.
psql -h localhost -U postgres -d postgres -f internal/db/schema.sql
DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/diagnostic?sslmode=disable' go run ./cmd/api
```

The REST API binds to `127.0.0.1:8080` and the agent TCP listener to `127.0.0.1:8081` by default. Set `SERVER_ADDR` and `AGENT_ADDR` to change the bind addresses. Use a protected database credential in `DATABASE_URL` outside local development. The service does not implement authentication; do not expose it directly to untrusted networks.

## API reference

## Overview

The **Diagnostic Client API** enables real-time monitoring and analysis of system diagnostics, including file system monitoring, network traffic analysis, and log aggregation. This API provides REST endpoints and a WebSocket connection for live updates.

### Base URL
```
http://localhost:8080
```

---

## WebSocket Endpoint

### Real-time Updates Connection

```
GET /ws
```

After connecting, the WebSocket streams updates in various formats:

#### File Update Message
```json
{
  "type": "file_update",
  "payload": {
    "path": "/path/to/file",
    "parent_path": "/path/to",
    "name": "file",
    "is_directory": false,
    "size": 1024,
    "mod_time": "2024-11-02T03:18:43Z",
    "is_gzipped": false,
    "is_scraped": false
  }
}
```

#### Network Update Message
```json
{
  "type": "network",
  "payload": [
    {
      "timestamp": "2024-11-02T03:18:43Z",
      "protocol": "TCP",
      "src_ip": "192.168.1.1",
      "dst_ip": "192.168.1.2",
      "src_port": 8080,
      "dst_port": 443,
      "length": 1024,
      "payload_size": 512,
      "tcp_flags": "ACK"
    }
  ]
}
```

#### Log Update Message
```json
{
  "type": "log",
  "payload": {
    "filename": "/var/log/system.log",
    "line": "Error: Connection refused",
    "line_num": 1234,
    "timestamp": "2024-11-02T03:18:43Z",
    "level": "ERROR"
  }
}
```

---

## REST API Endpoints

### File System Operations

#### Get File Tree
```
GET /api/files
```
Retrieves a hierarchical view of the file system.

**Query Parameters:**
- `path` (string, optional) - Root path to start traversal. Default: `/`
- `depth` (integer, optional) - Depth of tree traversal. Default: 1, Max: 10

**Success Response (200 OK):**
```json
[
  {
    "path": "/path/to/file",
    "parent_path": "/path/to",
    "name": "file",
    "is_directory": false,
    "size": 1024,
    "mod_time": "2024-11-02T03:18:43Z",
    "is_gzipped": false,
    "is_scraped": false
  }
]
```

---

### Log Operations

#### Get Logs
```
GET /api/logs
```
Retrieves log entries for a specific file with pagination.

**Query Parameters:**
- `file` (string, required) - Path to the log file
- `before` (string, optional) - ISO timestamp for pagination
- `limit` (integer, optional) - Max entries to return. Default: 100, Max: 1000

**Success Response (200 OK):**
```json
[
  {
    "filename": "/var/log/system.log",
    "line": "Error: Connection refused",
    "line_num": 1234,
    "timestamp": "2024-11-02T03:18:43Z",
    "level": "ERROR"
  }
]
```

#### Search Logs
```
POST /api/logs/search
```
Searches log entries across multiple files with full-text search capabilities.

**Request Body:**
```json
{
  "query": "error connection",
  "files": ["/var/log/system.log", "/var/log/app.log"],
  "start_time": "2024-11-01T00:00:00Z",
  "end_time": "2024-11-02T00:00:00Z"
}
```

**Success Response (200 OK):**
```json
[
  {
    "filename": "/var/log/system.log",
    "line": "Error: Connection refused",
    "line_num": 1234,
    "timestamp": "2024-11-02T03:18:43Z",
    "level": "ERROR"
  }
]
```

---

### Network Operations

#### Get Network Metrics
```
GET /api/network/metrics
```
Retrieves network traffic metrics and packet information.

**Query Parameters:**
- `start` (string, optional) - Start time for metrics
- `end` (string, optional) - End time for metrics
- `protocol` (string[], optional) - Filter by protocols (e.g., TCP, UDP)

**Success Response (200 OK):**
```json
{
  "packet_count": 1000,
  "total_bytes": 1048576,
  "avg_packet_size": 1024,
  "unique_sources": 10,
  "unique_destinations": 20,
  "protocol_count": 3,
  "protocol_stats": {
    "TCP": 800,
    "UDP": 150,
    "ICMP": 50
  },
  "packets": [
    {
      "timestamp": "2024-11-02T03:18:43Z",
      "protocol": "TCP",
      "src_ip": "192.168.1.1",
      "dst_ip": "192.168.1.2",
      "src_port": 8080,
      "dst_port": 443,
      "length": 1024,
      "payload_size": 512,
      "tcp_flags": "ACK"
    }
  ]
}
```

---

## Error Responses

All endpoints use standard HTTP status codes and return errors in the following format:
```json
{
  "error": "Detailed error message",
  "code": "ERROR_CODE",
  "details": {
    "field": "Additional error context"
  }
}
```

### Common Status Codes:
- `200`: Successful operation
- `400`: Bad request (invalid parameters)
- `404`: Resource not found
- `500`: Internal server error

### Common Error Codes:
- `INVALID_PATH`: Invalid file path provided
- `FILE_NOT_FOUND`: Requested file does not exist
- `INVALID_TIME_RANGE`: Invalid time range specified
- `INVALID_PROTOCOL`: Unsupported protocol specified
- `SEARCH_ERROR`: Error during log search
- `DATABASE_ERROR`: Database operation failed
