package api

import (
	"fmt"
	"ledger/domain"
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

type CategoryTotalResponse struct {
	Category string `json:"category"`
	Total    int    `json:"total"`
}

type ReportSummaryResponse struct {
	From       string                  `json:"from"`
	To         string                  `json:"to"`
	Total      int                     `json:"total"`
	Categories []CategoryTotalResponse `json:"categories"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (r CreateTransactionRequest) ToTransaction() (domain.Transaction, error) {
	tx := domain.Transaction{
		Amount:      r.Amount,
		Category:    r.Category,
		Description: r.Description,
	}
	if r.Date != "" {
		d, err := time.Parse(dateLayout, r.Date)
		if err != nil {
			return domain.Transaction{}, fmt.Errorf("date must be in format YYYY-MM-DD, got %q", r.Date)
		}
		tx.Date = d
	}
	return tx, nil
}

func NewTransactionResponse(tx domain.Transaction) TransactionResponse {
	return TransactionResponse{
		ID:          tx.ID,
		Amount:      tx.Amount,
		Category:    tx.Category,
		Description: tx.Description,
		Date:        tx.Date.Format(dateLayout),
	}
}

func NewTransactionResponses(txs []domain.Transaction) []TransactionResponse {
	resp := make([]TransactionResponse, 0, len(txs))
	for _, tx := range txs {
		resp = append(resp, NewTransactionResponse(tx))
	}
	return resp
}

func (r CreateBudgetRequest) ToBudget() domain.Budget {
	return domain.Budget{
		Category: r.Category,
		Limit:    r.Limit,
	}
}

func NewBudgetResponse(b domain.Budget) BudgetResponse {
	return BudgetResponse{
		Category: b.Category,
		Limit:    b.Limit,
	}
}

func NewBudgetResponses(bs []domain.Budget) []BudgetResponse {
	resp := make([]BudgetResponse, 0, len(bs))
	for _, b := range bs {
		resp = append(resp, NewBudgetResponse(b))
	}
	return resp
}

func NewReportSummaryResponse(s domain.ReportSummary) ReportSummaryResponse {
	resp := ReportSummaryResponse{
		From:       s.From,
		To:         s.To,
		Total:      s.Total,
		Categories: make([]CategoryTotalResponse, 0, len(s.Categories)),
	}
	for _, c := range s.Categories {
		resp.Categories = append(resp.Categories, CategoryTotalResponse{Category: c.Category, Total: c.Total})
	}
	return resp
}
