package pg

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"ledger/domain"
)

const dateLayout = "2006-01-02"

type ExpenseRepository struct {
	db *sql.DB
}

var _ domain.ExpenseRepository = (*ExpenseRepository)(nil)

func NewExpenseRepository(db *sql.DB) *ExpenseRepository {
	return &ExpenseRepository{db: db}
}

func (r *ExpenseRepository) Add(ctx context.Context, tx domain.Transaction) (domain.Transaction, error) {
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO expenses (amount, category, description, date) VALUES ($1, $2, $3, $4) RETURNING id`,
		tx.Amount, tx.Category, tx.Description, tx.Date).Scan(&tx.ID)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("insert transaction: %w", err)
	}
	return tx, nil
}

func (r *ExpenseRepository) List(ctx context.Context) ([]domain.Transaction, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, amount::bigint, category, COALESCE(description, ''), date
		FROM expenses ORDER BY date DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	list := []domain.Transaction{}
	for rows.Next() {
		var tx domain.Transaction
		if err := rows.Scan(&tx.ID, &tx.Amount, &tx.Category, &tx.Description, &tx.Date); err != nil {
			return nil, fmt.Errorf("list transactions: scan: %w", err)
		}
		list = append(list, tx)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	return list, nil
}

func (r *ExpenseRepository) SumByCategory(ctx context.Context, category string) (int, error) {
	var spent int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount), 0)::bigint FROM expenses WHERE category = $1`, category).Scan(&spent); err != nil {
		return 0, fmt.Errorf("sum by category: %w", err)
	}
	return spent, nil
}

func (r *ExpenseRepository) TotalsByCategory(ctx context.Context, from, to time.Time) ([]domain.CategoryTotal, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT category, SUM(amount)::bigint
		FROM expenses
		WHERE date BETWEEN $1::date AND $2::date
		GROUP BY category ORDER BY category`, from.Format(dateLayout), to.Format(dateLayout))
	if err != nil {
		return nil, fmt.Errorf("totals by category: %w", err)
	}
	defer rows.Close()

	totals := []domain.CategoryTotal{}
	for rows.Next() {
		var ct domain.CategoryTotal
		if err := rows.Scan(&ct.Category, &ct.Total); err != nil {
			return nil, fmt.Errorf("totals by category: scan: %w", err)
		}
		totals = append(totals, ct)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("totals by category: %w", err)
	}
	return totals, nil
}
