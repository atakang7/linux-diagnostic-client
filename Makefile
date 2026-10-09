GO ?= go
BUILD_DIR ?= bin
BINARY_NAME ?= diagnostic-client

.PHONY: build run test vet clean init-db dev-db

build:
	mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/api

run:
	$(GO) run ./cmd/api

test:
	$(GO) test -race -count=1 ./...

vet:
	$(GO) vet ./...

clean:
	rm -rf $(BUILD_DIR)

# Requires PostgreSQL with the TimescaleDB extension available.
init-db:
	psql -h 127.0.0.1 -U postgres -d postgres -v ON_ERROR_STOP=1 -f internal/db/schema.sql

# Local-only development database; never expose a trust-authenticated instance.
dev-db:
	docker run -d --name diagnostic-postgres \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=postgres \
		-e POSTGRES_DB=postgres \
		-p 127.0.0.1:5432:5432 \
		timescale/timescaledb:latest-pg16
