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

func SetBudget(b Budget) {
	budgets[b.Category] = b
}

func LoadBudgets(r io.Reader) error {
	var loaded []Budget
	if err := json.NewDecoder(r).Decode(&loaded); err != nil {
		return fmt.Errorf("load budgets: decode json: %w", err)
	}

	for i, b := range loaded {
		if b.Category == "" {
			return fmt.Errorf("load budgets: item %d: empty category", i)
		}
		if b.Limit <= 0 {
			return fmt.Errorf("load budgets: item %d (%q): limit must be positive, got %d", i, b.Category, b.Limit)
		}
	}

	for _, b := range loaded {
		SetBudget(b)
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
