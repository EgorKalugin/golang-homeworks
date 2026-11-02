# Gateway

Gateway is the HTTP API on `:8080`. It exposes the same endpoints as before
(plus `GET /api/reports/summary`) and calls the `ledger` module directly
(`replace ledger => ../ledger`). Ledger now stores data in PostgreSQL instead of
memory, and caches reports in Redis. Gateway itself does not know about the
cache, so caching is invisible to API clients.

## Run

1. Start PostgreSQL and Redis and apply migrations, see [ledger/README.md](../ledger/README.md).
2. Set the same environment variables as for Ledger and start Gateway:

```
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable"
export REDIS_ADDR=localhost:6379 REDIS_DB=0   # REDIS_PASSWORD if needed
cd gateway
go run .
```

On start it logs `ledger: connected to PostgreSQL`, `ledger: connected to Redis ...`
and `Gateway service started on :8080`. Data survives a Gateway/Ledger restart.

## Endpoints

| Method | Path                   | Body / query                                              | Success |
|--------|------------------------|-----------------------------------------------------------|---------|
| POST   | `/api/transactions`    | `{"amount","category","description","date":"YYYY-MM-DD"}` | `201` transaction with DB `id` |
| GET    | `/api/transactions`    | —                                                         | `200` array, newest first |
| POST   | `/api/budgets`         | `{"category","limit"}` (creates or updates)               | `201` budget |
| GET    | `/api/budgets`         | —                                                         | `200` array sorted by category |
| GET    | `/api/reports/summary` | `?from=YYYY-MM-DD&to=YYYY-MM-DD`                          | `200` `{"from","to","total","categories":[{"category","total"}]}` |

Errors: `400` (invalid JSON/validation/dates), `409` budget exceeded (nothing is
inserted), `404`/`405`, `500` (e.g. database error, details only in the log).

## cURL check

```
# 201, row appears in budgets
curl -i -X POST http://localhost:8080/api/budgets \
  -H "Content-Type: application/json" -d '{"category":"еда","limit":1000}'

# 201 within the limit, row appears in expenses
curl -i -X POST http://localhost:8080/api/transactions \
  -H "Content-Type: application/json" \
  -d '{"amount":450,"category":"еда","description":"ланч","date":"2025-09-10"}'

# 409 over the limit, nothing is inserted
curl -i -X POST http://localhost:8080/api/transactions \
  -H "Content-Type: application/json" \
  -d '{"amount":600,"category":"еда","description":"ужин","date":"2025-09-11"}'

# lists come from the database
curl http://localhost:8080/api/transactions
curl http://localhost:8080/api/budgets

# report twice in a row: the log shows "report cache miss" then "report cache hit"
curl "http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30"
curl "http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30"
```

Check the tables and cache directly:

```
psql "$DATABASE_URL" -c "SELECT * FROM budgets" -c "SELECT * FROM expenses"
docker exec cashcraft-redis redis-cli KEYS '*'
docker exec cashcraft-redis redis-cli TTL report:summary:2025-09-01:2025-09-30
```

Restart Gateway and repeat the `GET` requests: the data is still there.

## Tests

`internal/api` tests use the same test database as Ledger (`TEST_DATABASE_URL`,
default `cashapp_test`; Redis DB `TEST_REDIS_DB`, default `15`) and are skipped
if it is unreachable. See [ledger/README.md](../ledger/README.md#tests).

```
cd gateway && go test ./... -cover
```
