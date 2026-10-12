package transactions

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

type Budget struct {
	Category string `json:"category"`
	Limit    int    `json:"limit"`
}

var budgets = map[string]Budget{}

var ErrBudgetExceeded = errors.New("budget exceeded")

func SetBudget(b Budget) error {
	if err := b.Validate(); err != nil {
		return fmt.Errorf("invalid budget: %w", err)
	}
	budgets[b.Category] = b
	return nil
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

	for _, b := range loaded {
		budgets[b.Category] = b
	}
	return nil
}

func spentInCategory(category string) int {
	total := 0
	for _, tx := range transactions {
		if tx.Category == category {
			total += tx.Amount
		}
	}
	return total
}
