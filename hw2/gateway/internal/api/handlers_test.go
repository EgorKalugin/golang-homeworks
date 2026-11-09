package api

import (
	"encoding/json"
	"ledger/app"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	return NewRouter(NewHandlers(app.NewInMemory()))
}

func send(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, contentTypeJSON, rec.Header().Get("Content-Type"))
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &v), "body: %s", rec.Body)
	return v
}

func TestBudgetsAPI(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		h := newTestRouter(t)

		rec := send(t, h, http.MethodPost, "/api/budgets", `{"category":"еда","limit":5000}`)
		require.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, BudgetResponse{Category: "еда", Limit: 5000}, decode[BudgetResponse](t, rec))

		rec = send(t, h, http.MethodGet, "/api/budgets", "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, decode[[]BudgetResponse](t, rec), BudgetResponse{Category: "еда", Limit: 5000})
	})

	t.Run("update", func(t *testing.T) {
		h := newTestRouter(t)

		send(t, h, http.MethodPost, "/api/budgets", `{"category":"еда","limit":5000}`)
		rec := send(t, h, http.MethodPost, "/api/budgets", `{"category":"еда","limit":7000}`)
		require.Equal(t, http.StatusCreated, rec.Code)

		rec = send(t, h, http.MethodGet, "/api/budgets", "")
		assert.Equal(t, []BudgetResponse{{Category: "еда", Limit: 7000}}, decode[[]BudgetResponse](t, rec))
	})

	t.Run("empty list", func(t *testing.T) {
		h := newTestRouter(t)

		rec := send(t, h, http.MethodGet, "/api/budgets", "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `[]`, rec.Body.String())
	})

	badRequests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{"bad_json", `{"category":`, "invalid JSON body"},
		{"wrong_type", `{"category":"еда","limit":"много"}`, "invalid JSON body"},
		{"zero_limit", `{"category":"еда","limit":0}`, "limit must be positive"},
		{"negative_limit", `{"category":"еда","limit":-1}`, "limit must be positive"},
		{"empty_category", `{"category":"","limit":100}`, "category must not be empty"},
	}
	for _, tt := range badRequests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRouter(t)

			rec := send(t, h, http.MethodPost, "/api/budgets", tt.body)
			require.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, decode[ErrorResponse](t, rec).Error, tt.wantErr)

			rec = send(t, h, http.MethodGet, "/api/budgets", "")
			assert.Empty(t, decode[[]BudgetResponse](t, rec), "invalid budget must not be stored")
		})
	}
}

func TestTransactionsAPI(t *testing.T) {
	const validTx = `{"amount":3000,"category":"еда","description":"продукты","date":"2025-09-10"}`
	wantTx := TransactionResponse{ID: 1, Amount: 3000, Category: "еда", Description: "продукты", Date: "2025-09-10"}

	withBudget := func(t *testing.T) http.Handler {
		t.Helper()
		h := newTestRouter(t)
		rec := send(t, h, http.MethodPost, "/api/budgets", `{"category":"еда","limit":5000}`)
		require.Equal(t, http.StatusCreated, rec.Code)
		return h
	}

	t.Run("ok", func(t *testing.T) {
		h := withBudget(t)

		rec := send(t, h, http.MethodPost, "/api/transactions", validTx)
		require.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, wantTx, decode[TransactionResponse](t, rec))
	})

	t.Run("list", func(t *testing.T) {
		h := withBudget(t)
		send(t, h, http.MethodPost, "/api/transactions", validTx)

		rec := send(t, h, http.MethodGet, "/api/transactions", "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, []TransactionResponse{wantTx}, decode[[]TransactionResponse](t, rec))
	})

	t.Run("empty list", func(t *testing.T) {
		h := newTestRouter(t)

		rec := send(t, h, http.MethodGet, "/api/transactions", "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `[]`, rec.Body.String())
	})

	t.Run("exceeded", func(t *testing.T) {
		h := withBudget(t)
		send(t, h, http.MethodPost, "/api/transactions", validTx)

		rec := send(t, h, http.MethodPost, "/api/transactions",
			`{"amount":2500,"category":"еда","date":"2025-09-11"}`)
		require.Equal(t, http.StatusConflict, rec.Code)
		assert.JSONEq(t, `{"error":"budget exceeded"}`, rec.Body.String())

		rec = send(t, h, http.MethodGet, "/api/transactions", "")
		assert.Len(t, decode[[]TransactionResponse](t, rec), 1, "rejected transaction must not be stored")
	})

	badRequests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{"bad_json", `{"amount":`, "invalid JSON body"},
		{"wrong_type", `{"amount":"сто","category":"еда","date":"2025-09-10"}`, "invalid JSON body"},
		{"bad_date", `{"amount":1,"category":"еда","date":"10.09.2025"}`, "YYYY-MM-DD"},
		{"missing_date", `{"amount":1,"category":"еда"}`, "date must be set"},
		{"zero_amount", `{"amount":0,"category":"еда","date":"2025-09-10"}`, "amount must be positive"},
		{"negative_amount", `{"amount":-5,"category":"еда","date":"2025-09-10"}`, "amount must be positive"},
		{"empty_category", `{"amount":5,"category":"","date":"2025-09-10"}`, "category must not be empty"},
	}
	for _, tt := range badRequests {
		t.Run(tt.name, func(t *testing.T) {
			h := withBudget(t)

			rec := send(t, h, http.MethodPost, "/api/transactions", tt.body)
			require.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, decode[ErrorResponse](t, rec).Error, tt.wantErr)

			rec = send(t, h, http.MethodGet, "/api/transactions", "")
			assert.Empty(t, decode[[]TransactionResponse](t, rec))
		})
	}
}

func TestReportSummaryAPI(t *testing.T) {
	h := newTestRouter(t)
	for _, body := range []string{
		`{"amount":300,"category":"еда","date":"2025-09-01"}`,
		`{"amount":150,"category":"еда","date":"2025-09-10"}`,
		`{"amount":70,"category":"транспорт","date":"2025-09-05"}`,
		`{"amount":999,"category":"еда","date":"2025-10-01"}`,
	} {
		require.Equal(t, http.StatusCreated, send(t, h, http.MethodPost, "/api/transactions", body).Code)
	}

	const wantSummary = `{"from":"2025-09-01","to":"2025-09-30","total":520,
		"categories":[{"category":"еда","total":450},{"category":"транспорт","total":70}]}`

	rec := send(t, h, http.MethodGet, "/api/reports/summary?from=2025-09-01&to=2025-09-30", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, wantSummary, rec.Body.String())

	rec = send(t, h, http.MethodGet, "/api/reports/summary?from=2025-09-01&to=2025-09-30", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, wantSummary, rec.Body.String(), "repeated request returns the same summary")

	badRequests := []struct {
		name    string
		query   string
		wantErr string
	}{
		{"missing_from", "?to=2025-09-30", `"from" is required`},
		{"missing_to", "?from=2025-09-01", `"to" is required`},
		{"bad_date", "?from=01.09.2025&to=2025-09-30", "YYYY-MM-DD"},
		{"reversed_period", "?from=2025-09-30&to=2025-09-01", "from must not be after to"},
	}
	for _, tt := range badRequests {
		t.Run(tt.name, func(t *testing.T) {
			rec := send(t, h, http.MethodGet, "/api/reports/summary"+tt.query, "")
			require.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, decode[ErrorResponse](t, rec).Error, tt.wantErr)
		})
	}
}

func TestRouting(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{"method_not_allowed_transactions", http.MethodDelete, "/api/transactions", http.StatusMethodNotAllowed, `{"error":"method not allowed"}`},
		{"method_not_allowed_budgets", http.MethodPut, "/api/budgets", http.StatusMethodNotAllowed, `{"error":"method not allowed"}`},
		{"method_not_allowed_reports", http.MethodPost, "/api/reports/summary", http.StatusMethodNotAllowed, `{"error":"method not allowed"}`},
		{"unknown_api_path", http.MethodGet, "/api/unknown", http.StatusNotFound, `{"error":"not found"}`},
		{"unknown_root_path", http.MethodGet, "/health", http.StatusNotFound, `{"error":"not found"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestRouter(t)

			rec := send(t, h, tt.method, tt.path, "")
			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.JSONEq(t, tt.wantBody, rec.Body.String())
		})
	}
}
