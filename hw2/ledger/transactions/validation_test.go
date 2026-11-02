package transactions

import (
	"errors"
	"testing"
	"time"
)

func TestCheckValid(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		v       Validatable
		wantErr error
	}{
		{"valid transaction", Transaction{Amount: 100, Category: "food", Date: now}, nil},
		{"zero amount", Transaction{Amount: 0, Category: "food", Date: now}, ErrNonPositiveAmount},
		{"negative amount", Transaction{Amount: -5, Category: "food", Date: now}, ErrNonPositiveAmount},
		{"empty category", Transaction{Amount: 100, Category: "  ", Date: now}, ErrEmptyCategory},
		{"zero date", Transaction{Amount: 100, Category: "food"}, ErrZeroDate},
		{"valid budget", Budget{Category: "food", Limit: 1000}, nil},
		{"budget zero limit", Budget{Category: "food", Limit: 0}, ErrNonPositiveLimit},
		{"budget negative limit", Budget{Category: "food", Limit: -1}, ErrNonPositiveLimit},
		{"budget empty category", Budget{Category: "", Limit: 1000}, ErrEmptyCategory},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckValid(tt.v)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestAddRejectsInvalid(t *testing.T) {
	withDB(t)
	if err := AddTransaction(Transaction{Amount: 100, Date: time.Now()}); !errors.Is(err, ErrEmptyCategory) {
		t.Fatalf("want ErrEmptyCategory, got %v", err)
	}
	if got := len(mustList(t)); got != 0 {
		t.Fatalf("invalid transaction stored, have %d", got)
	}

	if err := SetBudget(Budget{Category: "food", Limit: 0}); !errors.Is(err, ErrNonPositiveLimit) {
		t.Fatalf("want ErrNonPositiveLimit, got %v", err)
	}
	if got := mustBudgets(t); len(got) != 0 {
		t.Fatalf("invalid budget stored: %v", got)
	}
}
