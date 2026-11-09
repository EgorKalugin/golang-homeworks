package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"ledger/domain"
)

const upsertBudgetSQL = `
	INSERT INTO budgets (category, limit_amount) VALUES ($1, $2)
	ON CONFLICT (category) DO UPDATE SET limit_amount = EXCLUDED.limit_amount`

type BudgetRepository struct {
	db *sql.DB
}

var _ domain.BudgetRepository = (*BudgetRepository)(nil)

func NewBudgetRepository(db *sql.DB) *BudgetRepository {
	return &BudgetRepository{db: db}
}

func (r *BudgetRepository) Upsert(ctx context.Context, budgets ...domain.Budget) error {
	if len(budgets) == 1 {
		if _, err := r.db.ExecContext(ctx, upsertBudgetSQL, budgets[0].Category, budgets[0].Limit); err != nil {
			return fmt.Errorf("upsert budget: %w", err)
		}
		return nil
	}

	dbTx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("upsert budgets: %w", err)
	}
	defer dbTx.Rollback()
	for _, b := range budgets {
		if _, err := dbTx.ExecContext(ctx, upsertBudgetSQL, b.Category, b.Limit); err != nil {
			return fmt.Errorf("upsert budgets: %w", err)
		}
	}
	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("upsert budgets: %w", err)
	}
	return nil
}

func (r *BudgetRepository) GetByCategory(ctx context.Context, category string) (domain.Budget, error) {
	b := domain.Budget{Category: category}
	err := r.db.QueryRowContext(ctx,
		`SELECT limit_amount::bigint FROM budgets WHERE category = $1`, category).Scan(&b.Limit)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return domain.Budget{}, domain.ErrBudgetNotFound
	case err != nil:
		return domain.Budget{}, fmt.Errorf("get budget: %w", err)
	}
	return b, nil
}

func (r *BudgetRepository) List(ctx context.Context) ([]domain.Budget, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT category, limit_amount::bigint FROM budgets ORDER BY category`)
	if err != nil {
		return nil, fmt.Errorf("list budgets: %w", err)
	}
	defer rows.Close()

	list := []domain.Budget{}
	for rows.Next() {
		var b domain.Budget
		if err := rows.Scan(&b.Category, &b.Limit); err != nil {
			return nil, fmt.Errorf("list budgets: scan: %w", err)
		}
		list = append(list, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list budgets: %w", err)
	}
	return list, nil
}
