package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"ledger/domain"
	"ledger/service"
	"log"
	"net/http"
	"time"
)

type Handlers struct {
	svc service.Service
}

func NewHandlers(svc service.Service) *Handlers {
	return &Handlers{svc: svc}
}

func (h *Handlers) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req CreateTransactionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := req.ToTransaction()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	created, err := h.svc.AddTransaction(r.Context(), tx)
	switch {
	case domain.IsValidationError(err):
		writeError(w, http.StatusBadRequest, err.Error())
		return
	case errors.Is(err, domain.ErrBudgetExceeded):
		writeError(w, http.StatusConflict, "budget exceeded")
		return
	case err != nil:
		log.Printf("add transaction: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, NewTransactionResponse(created))
}

func (h *Handlers) ListTransactions(w http.ResponseWriter, r *http.Request) {
	txs, err := h.svc.ListTransactions(r.Context())
	if err != nil {
		log.Printf("list transactions: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, NewTransactionResponses(txs))
}

func (h *Handlers) CreateBudget(w http.ResponseWriter, r *http.Request) {
	var req CreateBudgetRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	b := req.ToBudget()
	err := h.svc.SetBudget(r.Context(), b)
	switch {
	case domain.IsValidationError(err):
		writeError(w, http.StatusBadRequest, err.Error())
		return
	case err != nil:
		log.Printf("set budget: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, NewBudgetResponse(b))
}

func (h *Handlers) ListBudgets(w http.ResponseWriter, r *http.Request) {
	bs, err := h.svc.ListBudgets(r.Context())
	if err != nil {
		log.Printf("list budgets: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, NewBudgetResponses(bs))
}

func (h *Handlers) ReportSummary(w http.ResponseWriter, r *http.Request) {
	from, err := parseDateParam(r, "from")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	to, err := parseDateParam(r, "to")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	summary, err := h.svc.ReportSummary(r.Context(), from, to)
	switch {
	case domain.IsValidationError(err):
		writeError(w, http.StatusBadRequest, err.Error())
		return
	case err != nil:
		log.Printf("report summary: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, NewReportSummaryResponse(summary))
}

func decodeJSON(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON body: %v", err)
	}
	return nil
}

func parseDateParam(r *http.Request, name string) (time.Time, error) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return time.Time{}, fmt.Errorf("query parameter %q is required (YYYY-MM-DD)", name)
	}
	d, err := time.Parse(dateLayout, v)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be in format YYYY-MM-DD, got %q", name, v)
	}
	return d, nil
}
