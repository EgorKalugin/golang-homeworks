run gateway:
```
cd gateway
go run main.go
```

run ledger:
```
cd ledger
go run main.go
```

## Gateway HTTP API

Gateway imports the `ledger` module (via `replace ledger => ../ledger` in `gateway/go.mod`)
and listens on `:8080`. Storage is in memory, so data is lost on restart.

All responses use `Content-Type: application/json; charset=utf-8`.
Errors have the form `{"error":"..."}`. Every request is logged to the console
as method, path and duration.

| Method | Path                | Body                                                      | Success            |
|--------|---------------------|-----------------------------------------------------------|--------------------|
| POST   | `/api/transactions` | `{"amount","category","description","date":"YYYY-MM-DD"}` | `201` transaction  |
| GET    | `/api/transactions` | —                                                         | `200` array (`[]` if empty) |
| POST   | `/api/budgets`      | `{"category","limit"}` (creates or updates)               | `201` budget       |
| GET    | `/api/budgets`      | —                                                         | `200` array (`[]` if empty) |

Error statuses:

- `400 Bad Request` — invalid JSON or failed validation (`amount <= 0`, empty category, missing/invalid date, `limit <= 0`);
- `409 Conflict` — the transaction exceeds the category budget (`{"error":"budget exceeded"}`);
- `500 Internal Server Error` — any other unexpected error;
- `404` / `405` — unknown path / unsupported method.

### cURL examples

Create a budget:
```
curl -X POST http://localhost:8080/api/budgets \
  -H "Content-Type: application/json" \
  -d '{"category":"еда","limit":1000}'
```

List budgets:
```
curl http://localhost:8080/api/budgets
```

Create a transaction:
```
curl -X POST http://localhost:8080/api/transactions \
  -H "Content-Type: application/json" \
  -d '{"amount":450,"category":"еда","description":"ланч","date":"2025-09-10"}'
```

Exceed the budget (returns `409`):
```
curl -i -X POST http://localhost:8080/api/transactions \
  -H "Content-Type: application/json" \
  -d '{"amount":600,"category":"еда","description":"ужин","date":"2025-09-11"}'
```

Validation error (returns `400`):
```
curl -i -X POST http://localhost:8080/api/transactions \
  -H "Content-Type: application/json" \
  -d '{"amount":-5,"category":"еда","date":"2025-09-10"}'
```

List transactions:
```
curl http://localhost:8080/api/transactions
```

## Tests

- `ledger/transactions/ledger_test.go` — unit tests for the domain: table-driven
  `Transaction.Validate()` / `Budget.Validate()` (parallel) and the budget flow
  (`SetBudget` → `AddTransaction` within/over limit → `ListTransactions`).
- `gateway/internal/api/handlers_test.go` — HTTP integration tests via `httptest`
  (`POST/GET /api/budgets`, `POST/GET /api/transactions`, 400/409/404/405 mapping,
  `Content-Type` header). Subtests: `ok`, `list`, `exceeded`, `bad_json`, ...

Ledger keeps data in memory, so `transactions.Reset()` clears transactions and
budgets; stateful tests call it at start and in `t.Cleanup()` and are not parallel.

Run tests with coverage (in each module, `ledger` and `gateway`):
```
cd ledger   # or: cd gateway
go test ./... -cover -coverprofile=cover.out
go tool cover -html=cover.out -o cover.html
```

Coverage:

| Package                | Coverage |
|------------------------|----------|
| `ledger/transactions`  | ~96%     |
| `gateway/internal/api` | ~96%     |
| `ledger` module total  | ~47% (`main.go` demo is not tested) |
| `gateway` module total | ~93%     |
