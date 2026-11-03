# Database schema

The bootstrap SQL is [internal/db/schema.sql](../../internal/db/schema.sql). It creates a database named `diagnostic` with the `timescaledb` and `btree_gin` extensions.

| Table | Primary data | Indexes |
| --- | --- | --- |
| `files` | Path, parent, metadata, scrape state | Primary path, parent, directory, scrape state |
| `logs` | File path, line, time, level, English text search vector | File and line, time, level, GIN search |
| `network_packets` | Time, protocol, addresses, ports, lengths, flags | Protocol and time, source and destination IP |

`logs.file_path` references `files.path` with `ON DELETE CASCADE`. `network_packets` is a TimescaleDB hypertable with one-hour chunks.

The same SQL is stored in [init-scripts/init.sql](../../init-scripts/init.sql); CI compares the copies byte for byte.
