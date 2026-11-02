package transactions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNonPositiveAmount = errors.New("amount must be positive")

func AddTransaction(tx Transaction) error {
	_, err := CreateTransaction(tx)
	return err
}

func CreateTransaction(tx Transaction) (Transaction, error) {
	if err := tx.Validate(); err != nil {
		return Transaction{}, fmt.Errorf("invalid transaction: %w", err)
	}
	db, err := conn()
	if err != nil {
		return Transaction{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	var limit int
	err = db.QueryRowContext(ctx,
		`SELECT limit_amount::bigint FROM budgets WHERE category = $1`, tx.Category).Scan(&limit)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return Transaction{}, fmt.Errorf("get budget: %w", err)
	default:
		var spent int
		if err := db.QueryRowContext(ctx,
			`SELECT COALESCE(SUM(amount), 0)::bigint FROM expenses WHERE category = $1`, tx.Category).Scan(&spent); err != nil {
			return Transaction{}, fmt.Errorf("get spent: %w", err)
		}
		if spent+tx.Amount > limit {
			return Transaction{}, fmt.Errorf("%w: category %q: spent %d + %d > limit %d",
				ErrBudgetExceeded, tx.Category, spent, tx.Amount, limit)
		}
	}

	err = db.QueryRowContext(ctx,
		`INSERT INTO expenses (amount, category, description, date) VALUES ($1, $2, $3, $4) RETURNING id`,
		tx.Amount, tx.Category, tx.Description, tx.Date).Scan(&tx.ID)
	if err != nil {
		return Transaction{}, fmt.Errorf("insert transaction: %w", err)
	}
	return tx, nil
}

func ListTransactions() ([]Transaction, error) {
	db, err := conn()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	rows, err := db.QueryContext(ctx, `
		SELECT id, amount::bigint, category, COALESCE(description, ''), date
		FROM expenses ORDER BY date DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	list := []Transaction{}
	for rows.Next() {
		var tx Transaction
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
