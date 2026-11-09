# Ledger

Ledger holds the business logic (validation, budgets, transactions, reports).
Data lives in **PostgreSQL**. Report results (and the budgets list) are cached
in **Redis** for a short TTL. Gateway gets a ready service from `app.New` and
works only through the `service.Service` interface.

## 1. Start PostgreSQL and Redis

```
docker run -d --name cashcraft-pg -p 5432:5432 \
  -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=cashapp \
  postgres:17-alpine
docker run -d --name cashcraft-redis -p 6379:6379 redis:7-alpine
```

With a local PostgreSQL install, create the database yourself:
`createdb -U postgres cashapp`.

## 2. Configure the connection

| Variable         | Default          | Purpose |
|------------------|------------------|---------|
| `DATABASE_URL`   | —                | PostgreSQL DSN; takes priority over `DB_*` |
| `DB_HOST`        | `localhost`      | used when `DATABASE_URL` is empty |
| `DB_PORT`        | `5432`           | |
| `DB_USER`        | `postgres`       | |
| `DB_PASS`        | `postgres`       | |
| `DB_NAME`        | `cashapp`        | (`sslmode=disable` is always added) |
| `REDIS_ADDR`     | `localhost:6379` | Redis address |
| `REDIS_DB`       | `0`              | Redis database number |
| `REDIS_PASSWORD` | empty            | optional |

```
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable"
export REDIS_ADDR=localhost:6379 REDIS_DB=0
```

## 3. Apply migrations (Goose)

Install Goose: `go install github.com/pressly/goose/v3/cmd/goose@latest`.

Migrations are in `ledger/migrations/`:

- `00001_create_budgets.sql` — `budgets(id, category UNIQUE, limit_amount NUMERIC(14,2) > 0)`;
- `00002_create_expenses.sql` — `expenses(id, amount NUMERIC(14,2) <> 0, category, description, date)`
  and an index on `(category, date)`.

Run from the repository root, before the first start and after every schema change:

```
goose -dir ./ledger/migrations postgres "$DATABASE_URL" up      # apply
goose -dir ./ledger/migrations postgres "$DATABASE_URL" status  # check
goose -dir ./ledger/migrations postgres "$DATABASE_URL" down    # roll back the last migration
```

Start Ledger (and then Gateway) only after `goose up` succeeds.

## 4. Run

```
cd ledger
go run .                          # connect, print stored budgets and transactions
go run . -budgets budgets.json    # also upsert budgets from the JSON file
```

On start Ledger pings PostgreSQL and logs `ledger: connected to PostgreSQL`,
then logs `ledger: connected to Redis at ...`. If PostgreSQL is unreachable it
prints the reason and exits with code 1. If Redis is unreachable it logs a
warning and works without the cache.

The pool is limited to 10 open / 5 idle connections, max lifetime 30 minutes.

## Structure

| Package | Layer | Contents |
|---------|-------|----------|
| `domain` | domain | `Transaction`, `Budget`, `CategoryTotal`, `ReportSummary`, `Validate()`, errors (`ErrBudgetExceeded`, `ErrInvalidPeriod`, ...), `IsValidationError`, repository interfaces `BudgetRepository` and `ExpenseRepository` |
| `internal/repository/pg` | repository | `BudgetRepository`, `ExpenseRepository` on `*sql.DB` |
| `internal/repository/cached` | repository | Redis decorators: `budgets:all` and `report:summary:<from>:<to>` (TTL 30s) |
| `internal/repository/memory` | repository | in-memory implementation for tests |
| `service` | service | `Service` interface and its implementation |
| `app` | assembly | `ConfigFromEnv`, `New(ctx)`, `NewWithConfig(ctx, cfg)`, `NewInMemory()` |
| `internal/db`, `internal/cache` | infrastructure | `sql.Open("pgx")` + pool + `Ping`; go-redis client |

Repository interfaces (what the service needs, not how it is stored):

```go
type BudgetRepository interface {
	Upsert(ctx, budgets ...Budget) error             // INSERT ... ON CONFLICT DO UPDATE, several in one tx
	GetByCategory(ctx, category) (Budget, error)     // ErrBudgetNotFound if missing
	List(ctx) ([]Budget, error)                      // ORDER BY category
}
type ExpenseRepository interface {
	Add(ctx, tx Transaction) (Transaction, error)    // INSERT ... RETURNING id
	List(ctx) ([]Transaction, error)                 // ORDER BY date DESC, id DESC
	SumByCategory(ctx, category) (int, error)        // COALESCE(SUM(amount), 0)
	TotalsByCategory(ctx, from, to) ([]CategoryTotal, error) // SUM ... GROUP BY category, for reports
}
```

`service.Service`: `SetBudget`, `LoadBudgets` (JSON array, all or nothing), `ListBudgets`,
`AddTransaction`, `ListTransactions`, `ReportSummary`. All methods take `context.Context`.

## How it works

- `SetBudget` — `Validate()`, then `Upsert`; the cache decorator drops `budgets:all`.
- `AddTransaction` — `Validate()`, then the budget rule: `GetByCategory` + `SumByCategory`;
  if `spent + amount > limit` it returns `ErrBudgetExceeded` and inserts nothing. Categories
  without a budget are not limited. Otherwise `Add` returns the transaction with its ID.
  The check and the insert run under a mutex in the service.
- Validation errors are returned as is (e.g. `limit must be positive, got 0`), so callers can
  show them; `domain.IsValidationError(err)` tells them apart from storage errors.
- `ReportSummary(ctx, from, to)` — `ErrInvalidPeriod` if `from > to`, then `TotalsByCategory`
  (cached in Redis for 30s, so a new transaction shows up in a cached report after at most 30s).

## Initialization

```go
svc, closeFn, err := app.New(ctx) // reads env, opens and pings PostgreSQL (with ctx), connects Redis
if err != nil { log.Fatal(err) }  // e.g. PostgreSQL is unreachable
defer closeFn()                   // closes Redis and the DB pool
```

`app.New` does not run migrations, apply them with Goose first. If Redis is unreachable,
the service is built without the cache decorators.

## Tests

- `domain` — validation; `service` — business rules on in-memory repositories (no DB needed).
- `internal/repository/pg` and `app` — integration tests on PostgreSQL + Redis through the factory.
  They use a **separate** database because they truncate the tables: `TEST_DATABASE_URL` (default
  `postgres://postgres:postgres@localhost:5432/cashapp_test?sslmode=disable`) and Redis DB
  `TEST_REDIS_DB` (default `15`). If the test database is unreachable, those tests are skipped.

```
createdb -h localhost -U postgres cashapp_test   # or: docker exec cashcraft-pg createdb -U postgres cashapp_test
goose -dir ./ledger/migrations postgres "postgres://postgres:postgres@localhost:5432/cashapp_test?sslmode=disable" up
cd ledger && go test ./... -cover
```
