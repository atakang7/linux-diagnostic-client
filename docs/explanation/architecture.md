# Architecture

The agent sends JSON records over a dedicated TCP listener. The HTTP handlers read PostgreSQL, so queries do not depend on an active agent connection.

File lists are reconciled against an in-memory path cache. The database schema identifies files by path, so overlapping agents can overwrite each other's inventory.

Logs persist on receipt. Network packets accumulate in memory until the batch threshold or periodic flush, which reduces the number of SQL inserts and delays packet visibility until a flush.

The WebSocket handler consumes the tunnel channels once and copies each event to subscriber queues. Slow subscribers lose live events; the database holds persisted records for later REST queries.

TimescaleDB stores packets as a time-partitioned hypertable. PostgreSQL full-text search handles log queries.
