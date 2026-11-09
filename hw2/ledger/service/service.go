package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"ledger/domain"
)

const (
	operationTimeout = 5 * time.Second
	dateLayout       = "2006-01-02"
)

type Service interface {
	SetBudget(ctx context.Context, b domain.Budget) error
	LoadBudgets(ctx context.Context, r io.Reader) error
	ListBudgets(ctx context.Context) ([]domain.Budget, error)

	AddTransaction(ctx context.Context, tx domain.Transaction) (domain.Transaction, error)
	ListTransactions(ctx context.Context) ([]domain.Transaction, error)

	ReportSummary(ctx context.Context, from, to time.Time) (domain.ReportSummary, error)
}

type ledger struct {
	budgets  domain.BudgetRepository
	expenses domain.ExpenseRepository

	mu sync.Mutex
}

func New(budgets domain.BudgetRepository, expenses domain.ExpenseRepository) Service {
	return &ledger{budgets: budgets, expenses: expenses}
}

func (s *ledger) SetBudget(ctx context.Context, b domain.Budget) error {
	if err := b.Validate(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.budgets.Upsert(ctx, b); err != nil {
		return fmt.Errorf("set budget: %w", err)
	}
	return nil
}

func (s *ledger) LoadBudgets(ctx context.Context, r io.Reader) error {
	var loaded []domain.Budget
	if err := json.NewDecoder(r).Decode(&loaded); err != nil {
		return fmt.Errorf("load budgets: decode json: %w", err)
	}
	for i, b := range loaded {
		if err := b.Validate(); err != nil {
			return fmt.Errorf("load budgets: item %d: %w", i, err)
		}
	}
	if len(loaded) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.budgets.Upsert(ctx, loaded...); err != nil {
		return fmt.Errorf("load budgets: %w", err)
	}
	return nil
}

func (s *ledger) ListBudgets(ctx context.Context) ([]domain.Budget, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return s.budgets.List(ctx)
}

func (s *ledger) AddTransaction(ctx context.Context, tx domain.Transaction) (domain.Transaction, error) {
	if err := tx.Validate(); err != nil {
		return domain.Transaction{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkBudget(ctx, tx); err != nil {
		return domain.Transaction{}, err
	}
	created, err := s.expenses.Add(ctx, tx)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("add transaction: %w", err)
	}
	return created, nil
}

func (s *ledger) checkBudget(ctx context.Context, tx domain.Transaction) error {
	b, err := s.budgets.GetByCategory(ctx, tx.Category)
	switch {
	case errors.Is(err, domain.ErrBudgetNotFound):
		return nil
	case err != nil:
		return fmt.Errorf("get budget: %w", err)
	}
	spent, err := s.expenses.SumByCategory(ctx, tx.Category)
	if err != nil {
		return fmt.Errorf("get spent: %w", err)
	}
	if spent+tx.Amount > b.Limit {
		return fmt.Errorf("%w: category %q: spent %d + %d > limit %d",
			domain.ErrBudgetExceeded, tx.Category, spent, tx.Amount, b.Limit)
	}
	return nil
}

func (s *ledger) ListTransactions(ctx context.Context) ([]domain.Transaction, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return s.expenses.List(ctx)
}

func (s *ledger) ReportSummary(ctx context.Context, from, to time.Time) (domain.ReportSummary, error) {
	if from.After(to) {
		return domain.ReportSummary{}, domain.ErrInvalidPeriod
	}
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	totals, err := s.expenses.TotalsByCategory(ctx, from, to)
	if err != nil {
		return domain.ReportSummary{}, fmt.Errorf("report summary: %w", err)
	}
	summary := domain.ReportSummary{
		From:       from.Format(dateLayout),
		To:         to.Format(dateLayout),
		Categories: totals,
	}
	if summary.Categories == nil {
		summary.Categories = []domain.CategoryTotal{}
	}
	for _, ct := range totals {
		summary.Total += ct.Total
	}
	return summary, nil
}
