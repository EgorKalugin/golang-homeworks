# Gateway

Gateway is the HTTP API on `:8080`. It uses the `ledger` module
(`replace ledger => ../ledger`) only through the `service.Service` interface and
the `app.New` factory; it does not import `database/sql`, pgx, Redis or
`ledger/internal/repository/*`. Database and cache are invisible to it.

## Initialization

```go
// main.go
svc, closeFn, err := app.New(ctx) // Ledger reads env, connects and pings PostgreSQL, builds the service
if err != nil {
	log.Fatalf("gateway: cannot start ledger: %v", err)
}
defer closeFn()

srv := &http.Server{Addr: ":8080", Handler: api.NewRouter(api.NewHandlers(svc))}
```

Handlers call `svc.SetBudget`, `svc.AddTransaction`, `svc.ReportSummary` and so on, and map
errors to HTTP codes: `domain.IsValidationError` → `400`, `domain.ErrBudgetExceeded` → `409`,
anything else → `500`. On `Ctrl+C`/`SIGTERM` the server shuts down gracefully and `closeFn`
releases the connections.

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
and `Gateway service started on :8080`. If PostgreSQL is unreachable it logs
`gateway: cannot start ledger: ...` and exits with code 1. Data survives a restart.

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

`internal/api` tests run the HTTP layer against the real service built by
`app.NewInMemory()` (in-memory repositories), so they need no database. Storage
is covered by the Ledger integration tests, see [ledger/README.md](../ledger/README.md#tests).

```
cd gateway && go test ./... -cover
```
