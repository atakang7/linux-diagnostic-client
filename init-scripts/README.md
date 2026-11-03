Part of [linux-diagnostic-client](../README.md)

# database initialization

`init.sql` creates the `diagnostic` database and its TimescaleDB tables.

## Run

```sh
PGPASSWORD=postgres make init-db
```

The Makefile applies `internal/db/schema.sql`; `init.sql` is an identical copy.

## Test

```sh
cmp internal/db/schema.sql init-scripts/init.sql
```

## Config

| Variable | Default | What |
| --- | --- | --- |
| `PGPASSWORD` | unset | Password for the local PostgreSQL role |

## Interfaces

| File | Details |
| --- | --- |
| [init.sql](init.sql) | Schema bootstrap for `psql` |
| [Database reference](../docs/reference/database.md) | Tables and indexes |
