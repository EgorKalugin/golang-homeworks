package api

import "net/http"

func NewRouter(h *Handlers) http.Handler {
	api := http.NewServeMux()

	api.HandleFunc("POST /transactions", h.CreateTransaction)
	api.HandleFunc("GET /transactions", h.ListTransactions)
	api.HandleFunc("/transactions", methodNotAllowed)

	api.HandleFunc("POST /budgets", h.CreateBudget)
	api.HandleFunc("GET /budgets", h.ListBudgets)
	api.HandleFunc("/budgets", methodNotAllowed)

	api.HandleFunc("GET /reports/summary", h.ReportSummary)
	api.HandleFunc("/reports/summary", methodNotAllowed)

	api.HandleFunc("/", notFound)

	root := http.NewServeMux()
	root.Handle("/api/", http.StripPrefix("/api", api))
	root.HandleFunc("/", notFound)

	return Logging(JSONContentType(root))
}

func notFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not found")
}

func methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}
