package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if ct := rec.Header().Get("Content-Type"); ct != contentTypeJSON {
		t.Errorf("%s %s: Content-Type = %q, want %q", method, path, ct, contentTypeJSON)
	}
	return rec
}

func TestAPI(t *testing.T) {
	resetStorage(t)
	h := NewRouter(NewHandlers())

	steps := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{"empty transactions", "GET", "/api/transactions", "", 200, "[]"},
		{"empty budgets", "GET", "/api/budgets", "", 200, "[]"},
		{"invalid budget", "POST", "/api/budgets", `{"category":"еда","limit":0}`, 400, `{"error":"limit must be positive, got 0"}`},
		{"create budget", "POST", "/api/budgets", `{"category":"еда","limit":500}`, 201, `{"category":"еда","limit":500}`},
		{"list budgets", "GET", "/api/budgets", "", 200, `[{"category":"еда","limit":500}]`},
		{"bad json", "POST", "/api/transactions", `{`, 400, ""},
		{"bad date", "POST", "/api/transactions", `{"amount":1,"category":"еда","date":"10.09.2025"}`, 400, ""},
		{"invalid amount", "POST", "/api/transactions", `{"amount":-1,"category":"еда","date":"2025-09-10"}`, 400, `{"error":"amount must be positive, got -1"}`},
		{"missing date", "POST", "/api/transactions", `{"amount":1,"category":"еда"}`, 400, `{"error":"date must be set"}`},
		{"create tx", "POST", "/api/transactions", `{"amount":450,"category":"еда","description":"ланч","date":"2025-09-10"}`, 201,
			`{"id":1,"amount":450,"category":"еда","description":"ланч","date":"2025-09-10"}`},
		{"budget exceeded", "POST", "/api/transactions", `{"amount":100,"category":"еда","date":"2025-09-11"}`, 409, `{"error":"budget exceeded"}`},
		{"list tx", "GET", "/api/transactions", "", 200,
			`[{"id":1,"amount":450,"category":"еда","description":"ланч","date":"2025-09-10"}]`},
		{"method not allowed", "DELETE", "/api/transactions", "", 405, `{"error":"method not allowed"}`},
		{"not found", "GET", "/api/unknown", "", 404, `{"error":"not found"}`},
	}

	for _, s := range steps {
		rec := do(t, h, s.method, s.path, s.body)
		if rec.Code != s.wantStatus {
			t.Fatalf("%s: status = %d, want %d (body %s)", s.name, rec.Code, s.wantStatus, rec.Body)
		}
		if s.wantBody != "" && strings.TrimSpace(rec.Body.String()) != s.wantBody {
			t.Fatalf("%s: body = %s, want %s", s.name, rec.Body, s.wantBody)
		}
	}
}
