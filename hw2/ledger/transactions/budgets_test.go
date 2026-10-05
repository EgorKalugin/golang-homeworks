package transactions

import (
	"errors"
	"strings"
	"testing"
)

func reset() {
	transactions = []Transaction{}
	budgets = map[string]Budget{}
}

func TestAddTransactionBudget(t *testing.T) {
	tests := []struct {
		name    string
		budget  *Budget
		amounts []int
		wantErr []bool
	}{
		{"within limit", &Budget{"food", 5000}, []int{1500, 2000}, []bool{false, false}},
		{"exactly at limit", &Budget{"food", 5000}, []int{2000, 3000}, []bool{false, false}},
		{"exceeds limit", &Budget{"food", 5000}, []int{4000, 1001, 1000}, []bool{false, true, false}},
		{"no budget", nil, []int{100000}, []bool{false}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reset()
			if tt.budget != nil {
				SetBudget(*tt.budget)
			}

			accepted := 0
			for i, amount := range tt.amounts {
				err := AddTransaction(Transaction{Amount: amount, Category: "food"})
				if tt.wantErr[i] {
					if !errors.Is(err, ErrBudgetExceeded) {
						t.Fatalf("tx %d: want ErrBudgetExceeded, got %v", i, err)
					}
					continue
				}
				if err != nil {
					t.Fatalf("tx %d: unexpected error: %v", i, err)
				}
				accepted++
			}

			if got := len(ListTransactions()); got != accepted {
				t.Fatalf("stored %d transactions, want %d", got, accepted)
			}
		})
	}
}

func TestSetBudgetUpdatesLimit(t *testing.T) {
	reset()
	SetBudget(Budget{"food", 1000})
	if err := AddTransaction(Transaction{Amount: 1500, Category: "food"}); !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("want ErrBudgetExceeded, got %v", err)
	}

	SetBudget(Budget{"food", 2000})
	if err := AddTransaction(Transaction{Amount: 1500, Category: "food"}); err != nil {
		t.Fatalf("unexpected error after raising limit: %v", err)
	}
}

func TestLoadBudgets(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
		want    map[string]Budget
	}{
		{
			name:  "valid",
			input: `[{"category":"food","limit":5000},{"category":"transport","limit":3000}]`,
			want:  map[string]Budget{"food": {"food", 5000}, "transport": {"transport", 3000}},
		},
		{name: "broken json", input: `[{"category":"food",`, wantErr: "decode json"},
		{name: "not an array", input: `{"category":"food","limit":5000}`, wantErr: "decode json"},
		{name: "empty category", input: `[{"category":"food","limit":1},{"category":"","limit":5}]`, wantErr: "item 1: empty category"},
		{name: "non-positive limit", input: `[{"category":"food","limit":0}]`, wantErr: "limit must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reset()
			err := LoadBudgets(strings.NewReader(tt.input))

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
				}
				if len(budgets) != 0 {
					t.Fatalf("budgets must stay untouched on error, got %v", budgets)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(budgets) != len(tt.want) {
				t.Fatalf("got %v, want %v", budgets, tt.want)
			}
			for k, v := range tt.want {
				if budgets[k] != v {
					t.Fatalf("budget %q: got %v, want %v", k, budgets[k], v)
				}
			}
		})
	}
}
