# Run locally

Requires Go 1.21, Docker, Make, and `psql`.

1. Start TimescaleDB on loopback.

   ```sh
   make dev-db
   ```

2. Create the database and tables.

   ```sh
   PGPASSWORD=postgres make init-db
   ```

3. Set the connection URL and start both listeners.

   ```sh
   export DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/diagnostic?sslmode=disable'
   make run
   ```

4. Check the HTTP listener.

   ```sh
   curl -fsS http://127.0.0.1:8080/healthz
   curl -fsS http://127.0.0.1:8080/readyz
   ```

For an existing TimescaleDB server, skip the container and set `DATABASE_URL` to its initialized `diagnostic` database.
