package memory

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"time"

	"ledger/domain"
)

type BudgetRepository struct {
	mu      sync.RWMutex
	budgets map[string]int
}

var _ domain.BudgetRepository = (*BudgetRepository)(nil)

func NewBudgetRepository() *BudgetRepository {
	return &BudgetRepository{budgets: map[string]int{}}
}

func (r *BudgetRepository) Upsert(_ context.Context, budgets ...domain.Budget) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, b := range budgets {
		r.budgets[b.Category] = b.Limit
	}
	return nil
}

func (r *BudgetRepository) GetByCategory(_ context.Context, category string) (domain.Budget, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	limit, ok := r.budgets[category]
	if !ok {
		return domain.Budget{}, domain.ErrBudgetNotFound
	}
	return domain.Budget{Category: category, Limit: limit}, nil
}

func (r *BudgetRepository) List(_ context.Context) ([]domain.Budget, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]domain.Budget, 0, len(r.budgets))
	for c, l := range r.budgets {
		list = append(list, domain.Budget{Category: c, Limit: l})
	}
	slices.SortFunc(list, func(a, b domain.Budget) int { return cmp.Compare(a.Category, b.Category) })
	return list, nil
}

type ExpenseRepository struct {
	mu     sync.RWMutex
	txs    []domain.Transaction
	nextID int
}

var _ domain.ExpenseRepository = (*ExpenseRepository)(nil)

func NewExpenseRepository() *ExpenseRepository {
	return &ExpenseRepository{nextID: 1}
}

func (r *ExpenseRepository) Add(_ context.Context, tx domain.Transaction) (domain.Transaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tx.ID = r.nextID
	r.nextID++
	r.txs = append(r.txs, tx)
	return tx, nil
}

func (r *ExpenseRepository) List(_ context.Context) ([]domain.Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := slices.Clone(r.txs)
	if list == nil {
		list = []domain.Transaction{}
	}
	slices.SortFunc(list, func(a, b domain.Transaction) int {
		if c := b.Date.Compare(a.Date); c != 0 {
			return c
		}
		return cmp.Compare(b.ID, a.ID)
	})
	return list, nil
}

func (r *ExpenseRepository) SumByCategory(_ context.Context, category string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sum := 0
	for _, tx := range r.txs {
		if tx.Category == category {
			sum += tx.Amount
		}
	}
	return sum, nil
}

func (r *ExpenseRepository) TotalsByCategory(_ context.Context, from, to time.Time) ([]domain.CategoryTotal, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	from, to = truncateDay(from), truncateDay(to)
	byCategory := map[string]int{}
	for _, tx := range r.txs {
		if d := truncateDay(tx.Date); !d.Before(from) && !d.After(to) {
			byCategory[tx.Category] += tx.Amount
		}
	}
	totals := make([]domain.CategoryTotal, 0, len(byCategory))
	for c, t := range byCategory {
		totals = append(totals, domain.CategoryTotal{Category: c, Total: t})
	}
	slices.SortFunc(totals, func(a, b domain.CategoryTotal) int { return cmp.Compare(a.Category, b.Category) })
	return totals, nil
}

func truncateDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
