# Ledger

Ledger holds the domain logic (validation, budgets, transactions, reports).
Data lives in **PostgreSQL**. Report results (and the budgets list) are cached
in **Redis** for a short TTL. Gateway imports this module and calls it directly.

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

## How it works

- `SetBudget` — `INSERT ... ON CONFLICT (category) DO UPDATE` (upsert), then `DEL budgets:all`.
- `ListBudgets` — `budgets:all` from Redis (TTL 30s), otherwise `SELECT ... ORDER BY category`.
- `AddTransaction` / `CreateTransaction` — `Validate()`, then read the category limit and
  `SUM(amount)` already spent; if `spent + amount > limit` it returns `ErrBudgetExceeded`
  and inserts nothing. Categories without a budget are not limited (same as before).
  Otherwise `INSERT ... RETURNING id`; `CreateTransaction` returns the stored transaction with its ID.
- `ListTransactions` — `SELECT ... ORDER BY date DESC, id DESC`.
- `GetReportSummary(ctx, from, to)` — key `report:summary:<from>:<to>`. On a cache hit the
  JSON from Redis is returned. On a miss it runs `SUM(amount) GROUP BY category` for the
  period and stores the result for 30 seconds. A new transaction therefore shows up in
  a cached report after the TTL expires (at most 30s).

Package layout: `internal/db` (DSN from env, `sql.Open("pgx", dsn)`, pool settings, `Ping`),
`internal/cache` (go-redis v9 client, JSON get/set with TTL), `transactions` (domain and
SQL; the `*sql.DB` is stored in a package variable for now).

## Tests

Tests that touch storage use a **separate** database because `Reset()` truncates
the tables: `TEST_DATABASE_URL` (default
`postgres://postgres:postgres@localhost:5432/cashapp_test?sslmode=disable`) and Redis DB
`TEST_REDIS_DB` (default `15`). If the test database is unreachable, those tests are skipped.

```
createdb -h localhost -U postgres cashapp_test   # or: docker exec cashcraft-pg createdb -U postgres cashapp_test
goose -dir ./ledger/migrations postgres "postgres://postgres:postgres@localhost:5432/cashapp_test?sslmode=disable" up
cd ledger && go test ./... -cover
```
