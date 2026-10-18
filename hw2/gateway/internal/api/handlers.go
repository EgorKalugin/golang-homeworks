package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"ledger/transactions"
	"log"
	"net/http"
	"sync"
)

type Handlers struct {
	mu sync.Mutex
}

func NewHandlers() *Handlers {
	return &Handlers{}
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
	if err := tx.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.mu.Lock()
	err = transactions.AddTransaction(tx)
	var created transactions.Transaction
	if err == nil {
		all := transactions.ListTransactions()
		created = all[len(all)-1]
	}
	h.mu.Unlock()

	switch {
	case errors.Is(err, transactions.ErrBudgetExceeded):
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
	h.mu.Lock()
	resp := NewTransactionResponses(transactions.ListTransactions())
	h.mu.Unlock()

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handlers) CreateBudget(w http.ResponseWriter, r *http.Request) {
	var req CreateBudgetRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	b := req.ToBudget()
	if err := b.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.mu.Lock()
	err := transactions.SetBudget(b)
	h.mu.Unlock()
	if err != nil {
		log.Printf("set budget: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, NewBudgetResponse(b))
}

func (h *Handlers) ListBudgets(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	resp := NewBudgetResponses(transactions.ListBudgets())
	h.mu.Unlock()

	writeJSON(w, http.StatusOK, resp)
}

func decodeJSON(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON body: %v", err)
	}
	return nil
}
