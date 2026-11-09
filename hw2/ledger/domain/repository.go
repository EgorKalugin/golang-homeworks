package domain

import (
	"context"
	"time"
)

type BudgetRepository interface {
	Upsert(ctx context.Context, budgets ...Budget) error
	GetByCategory(ctx context.Context, category string) (Budget, error)
	List(ctx context.Context) ([]Budget, error)
}

type ExpenseRepository interface {
	Add(ctx context.Context, tx Transaction) (Transaction, error)
	List(ctx context.Context) ([]Transaction, error)
	SumByCategory(ctx context.Context, category string) (int, error)
	TotalsByCategory(ctx context.Context, from, to time.Time) ([]CategoryTotal, error)
}
