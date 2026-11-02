package transactions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"time"
)

type Budget struct {
	Category string `json:"category"`
	Limit    int    `json:"limit"`
}

const (
	budgetsCacheKey = "budgets:all"
	budgetsCacheTTL = 30 * time.Second
)

var ErrBudgetExceeded = errors.New("budget exceeded")

const upsertBudgetSQL = `
	INSERT INTO budgets (category, limit_amount) VALUES ($1, $2)
	ON CONFLICT (category) DO UPDATE SET limit_amount = EXCLUDED.limit_amount`

func SetBudget(b Budget) error {
	if err := b.Validate(); err != nil {
		return fmt.Errorf("invalid budget: %w", err)
	}
	db, err := conn()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	if _, err := db.ExecContext(ctx, upsertBudgetSQL, b.Category, b.Limit); err != nil {
		return fmt.Errorf("set budget: %w", err)
	}
	invalidateBudgets(ctx)
	return nil
}

func ListBudgets() ([]Budget, error) {
	db, err := conn()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	var list []Budget
	if ok, err := kv.GetJSON(ctx, budgetsCacheKey, &list); err != nil {
		log.Printf("budgets cache: %v", err)
	} else if ok {
		return list, nil
	}

	rows, err := db.QueryContext(ctx, `SELECT category, limit_amount::bigint FROM budgets ORDER BY category`)
	if err != nil {
		return nil, fmt.Errorf("list budgets: %w", err)
	}
	defer rows.Close()

	list = []Budget{}
	for rows.Next() {
		var b Budget
		if err := rows.Scan(&b.Category, &b.Limit); err != nil {
			return nil, fmt.Errorf("list budgets: scan: %w", err)
		}
		list = append(list, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list budgets: %w", err)
	}

	if err := kv.SetJSON(ctx, budgetsCacheKey, list, budgetsCacheTTL); err != nil {
		log.Printf("budgets cache: %v", err)
	}
	return list, nil
}

func LoadBudgets(r io.Reader) error {
	var loaded []Budget
	if err := json.NewDecoder(r).Decode(&loaded); err != nil {
		return fmt.Errorf("load budgets: decode json: %w", err)
	}

	for i, b := range loaded {
		if err := b.Validate(); err != nil {
			return fmt.Errorf("load budgets: item %d: %w", i, err)
		}
	}

	db, err := conn()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	dbTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("load budgets: %w", err)
	}
	defer dbTx.Rollback()
	for _, b := range loaded {
		if _, err := dbTx.ExecContext(ctx, upsertBudgetSQL, b.Category, b.Limit); err != nil {
			return fmt.Errorf("load budgets: %w", err)
		}
	}
	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("load budgets: %w", err)
	}
	invalidateBudgets(ctx)
	return nil
}

func invalidateBudgets(ctx context.Context) {
	if err := kv.Del(ctx, budgetsCacheKey); err != nil {
		log.Printf("budgets cache: %v", err)
	}
}
