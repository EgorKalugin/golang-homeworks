# CashCraft

- [`ledger/`](ledger/README.md): domain logic and storage. Data is stored in PostgreSQL
  (Goose migrations in `ledger/migrations/`), reports are cached in Redis for 30s.
- [`gateway/`](gateway/README.md): HTTP API on `:8080`. Same endpoints as before; it calls
  Ledger, which now keeps data in PostgreSQL instead of memory, so data survives restarts.

## Quick start

```
docker run -d --name cashcraft-pg -p 5432:5432 \
  -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=cashapp postgres:17-alpine
docker run -d --name cashcraft-redis -p 6379:6379 redis:7-alpine

export DATABASE_URL="postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable"
export REDIS_ADDR=localhost:6379 REDIS_DB=0

goose -dir ./ledger/migrations postgres "$DATABASE_URL" up     # roll back: ... down

(cd ledger && go run .)     # checks the DB connection and prints stored data
(cd gateway && go run .)    # HTTP API on :8080
```

Environment variables, migrations, cURL examples and tests are described in
[ledger/README.md](ledger/README.md) and [gateway/README.md](gateway/README.md).
