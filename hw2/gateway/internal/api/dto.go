package api

import (
	"fmt"
	"ledger/transactions"
	"time"
)

const dateLayout = "2006-01-02"

type CreateTransactionRequest struct {
	Amount      int    `json:"amount"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Date        string `json:"date"`
}

type TransactionResponse struct {
	ID          int    `json:"id"`
	Amount      int    `json:"amount"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Date        string `json:"date"`
}

type CreateBudgetRequest struct {
	Category string `json:"category"`
	Limit    int    `json:"limit"`
}

type BudgetResponse struct {
	Category string `json:"category"`
	Limit    int    `json:"limit"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// ToTransaction converts the request into a domain transaction.
// An empty date is left as zero time so that tx.Validate() reports it.
func (r CreateTransactionRequest) ToTransaction() (transactions.Transaction, error) {
	tx := transactions.Transaction{
		Amount:      r.Amount,
		Category:    r.Category,
		Description: r.Description,
	}
	if r.Date != "" {
		d, err := time.Parse(dateLayout, r.Date)
		if err != nil {
			return transactions.Transaction{}, fmt.Errorf("date must be in format YYYY-MM-DD, got %q", r.Date)
		}
		tx.Date = d
	}
	return tx, nil
}

func NewTransactionResponse(tx transactions.Transaction) TransactionResponse {
	return TransactionResponse{
		ID:          tx.ID,
		Amount:      tx.Amount,
		Category:    tx.Category,
		Description: tx.Description,
		Date:        tx.Date.Format(dateLayout),
	}
}

func NewTransactionResponses(txs []transactions.Transaction) []TransactionResponse {
	resp := make([]TransactionResponse, 0, len(txs))
	for _, tx := range txs {
		resp = append(resp, NewTransactionResponse(tx))
	}
	return resp
}

func (r CreateBudgetRequest) ToBudget() transactions.Budget {
	return transactions.Budget{
		Category: r.Category,
		Limit:    r.Limit,
	}
}

func NewBudgetResponse(b transactions.Budget) BudgetResponse {
	return BudgetResponse{
		Category: b.Category,
		Limit:    b.Limit,
	}
}

func NewBudgetResponses(bs []transactions.Budget) []BudgetResponse {
	resp := make([]BudgetResponse, 0, len(bs))
	for _, b := range bs {
		resp = append(resp, NewBudgetResponse(b))
	}
	return resp
}
