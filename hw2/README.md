# CashCraft

- [`ledger/`](ledger/README.md): business logic and storage, split into layers
  (domain → repository → service) plus a factory that assembles them. Data is stored in
  PostgreSQL (Goose migrations in `ledger/migrations/`), reports are cached in Redis for 30s.
- [`gateway/`](gateway/README.md): HTTP API on `:8080`. It gets a ready `service.Service`
  from the Ledger factory and knows nothing about the database.

## Layers

```
ledger/
  domain/                     entities (Transaction, Budget, ReportSummary), validation,
                              errors, repository interfaces. Depends on nothing.
  internal/repository/pg/     PostgreSQL implementation of the domain interfaces (SQL lives here)
  internal/repository/cached/ Redis decorators over the domain interfaces (budgets list, reports)
  internal/repository/memory/ in-memory implementation (tests, NewInMemory)
  internal/db, internal/cache low-level PostgreSQL / Redis clients
  service/                    Service interface + implementation: validation, budget rule,
                              conversions. Uses only domain interfaces.
  app/                        factory: env config -> DB/Redis -> repositories -> service
gateway/
  main.go                     svc, closeFn, err := app.New(ctx); defer closeFn()
  internal/api/               HTTP handlers, DTOs; depend on service.Service and domain only
```

Dependencies point inwards: `domain` ← `repository/*`, `domain` ← `service` ← `app` → `repository/*`;
Gateway imports only `ledger/app`, `ledger/service` and `ledger/domain`.
`service`, `app` and `domain` are not under `internal/` because Go forbids importing
another module's `internal` packages, and Gateway is a separate module.

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
