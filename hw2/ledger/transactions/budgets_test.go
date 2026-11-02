package transactions

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

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
			withDB(t)
			if tt.budget != nil {
				SetBudget(*tt.budget)
			}

			accepted := 0
			for i, amount := range tt.amounts {
				err := AddTransaction(Transaction{Amount: amount, Category: "food", Date: time.Now()})
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

			if got := len(mustList(t)); got != accepted {
				t.Fatalf("stored %d transactions, want %d", got, accepted)
			}
		})
	}
}

func TestSetBudgetUpdatesLimit(t *testing.T) {
	withDB(t)
	SetBudget(Budget{"food", 1000})
	if err := AddTransaction(Transaction{Amount: 1500, Category: "food", Date: time.Now()}); !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("want ErrBudgetExceeded, got %v", err)
	}

	SetBudget(Budget{"food", 2000})
	if err := AddTransaction(Transaction{Amount: 1500, Category: "food", Date: time.Now()}); err != nil {
		t.Fatalf("unexpected error after raising limit: %v", err)
	}
}

func TestLoadBudgets(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
		want    []Budget
	}{
		{
			name:  "valid",
			input: `[{"category":"food","limit":5000},{"category":"transport","limit":3000}]`,
			want:  []Budget{{"food", 5000}, {"transport", 3000}},
		},
		{name: "broken json", input: `[{"category":"food",`, wantErr: "decode json"},
		{name: "not an array", input: `{"category":"food","limit":5000}`, wantErr: "decode json"},
		{name: "empty category", input: `[{"category":"food","limit":1},{"category":"","limit":5}]`, wantErr: "item 1: category must not be empty"},
		{name: "non-positive limit", input: `[{"category":"food","limit":0}]`, wantErr: "limit must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withDB(t)
			err := LoadBudgets(strings.NewReader(tt.input))

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
				}
				if got := mustBudgets(t); len(got) != 0 {
					t.Fatalf("budgets must stay untouched on error, got %v", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := mustBudgets(t); !slices.Equal(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
