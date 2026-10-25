package transactions

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionValidate(t *testing.T) {
	t.Parallel()
	date := time.Date(2025, 9, 10, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		tx      Transaction
		wantErr error
	}{
		{"valid", Transaction{Amount: 100, Category: "еда", Description: "ланч", Date: date}, nil},
		{"valid without description", Transaction{Amount: 1, Category: "еда", Date: date}, nil},
		{"zero amount", Transaction{Amount: 0, Category: "еда", Date: date}, ErrNonPositiveAmount},
		{"negative amount", Transaction{Amount: -100, Category: "еда", Date: date}, ErrNonPositiveAmount},
		{"empty category", Transaction{Amount: 100, Category: "", Date: date}, ErrEmptyCategory},
		{"blank category", Transaction{Amount: 100, Category: " \t", Date: date}, ErrEmptyCategory},
		{"zero date", Transaction{Amount: 100, Category: "еда"}, ErrZeroDate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.tx.Validate()
			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestBudgetValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		budget  Budget
		wantErr error
	}{
		{"positive limit", Budget{Category: "еда", Limit: 5000}, nil},
		{"minimal limit", Budget{Category: "еда", Limit: 1}, nil},
		{"zero limit", Budget{Category: "еда", Limit: 0}, ErrNonPositiveLimit},
		{"negative limit", Budget{Category: "еда", Limit: -10}, ErrNonPositiveLimit},
		{"empty category", Budget{Category: "", Limit: 5000}, ErrEmptyCategory},
		{"blank category", Budget{Category: "   ", Limit: 5000}, ErrEmptyCategory},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.budget.Validate()
			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// TestBudgetFlow touches package-level state, so it is not parallel.
func TestBudgetFlow(t *testing.T) {
	date := time.Date(2025, 9, 10, 0, 0, 0, 0, time.UTC)

	t.Run("within limit then exceeded", func(t *testing.T) {
		Reset()
		t.Cleanup(Reset)

		require.NoError(t, SetBudget(Budget{Category: "еда", Limit: 5000}))

		err := AddTransaction(Transaction{Amount: 3000, Category: "еда", Date: date})
		require.NoError(t, err)
		require.Len(t, ListTransactions(), 1)

		err = AddTransaction(Transaction{Amount: 2500, Category: "еда", Date: date})
		assert.ErrorIs(t, err, ErrBudgetExceeded)
		assert.ErrorContains(t, err, "budget exceeded")
		assert.Len(t, ListTransactions(), 1, "rejected transaction must not be stored")
	})

	t.Run("exactly at limit", func(t *testing.T) {
		Reset()
		t.Cleanup(Reset)

		require.NoError(t, SetBudget(Budget{Category: "еда", Limit: 5000}))
		assert.NoError(t, AddTransaction(Transaction{Amount: 5000, Category: "еда", Date: date}))
		assert.ErrorIs(t, AddTransaction(Transaction{Amount: 1, Category: "еда", Date: date}), ErrBudgetExceeded)
		assert.Len(t, ListTransactions(), 1)
	})

	t.Run("other category is not limited", func(t *testing.T) {
		Reset()
		t.Cleanup(Reset)

		require.NoError(t, SetBudget(Budget{Category: "еда", Limit: 5000}))
		assert.NoError(t, AddTransaction(Transaction{Amount: 10000, Category: "транспорт", Date: date}))
		assert.Len(t, ListTransactions(), 1)
	})

	t.Run("invalid budget is rejected", func(t *testing.T) {
		Reset()
		t.Cleanup(Reset)

		assert.ErrorIs(t, SetBudget(Budget{Category: "еда", Limit: 0}), ErrNonPositiveLimit)
		assert.Empty(t, ListBudgets())
	})

	t.Run("ids are sequential", func(t *testing.T) {
		Reset()
		t.Cleanup(Reset)

		require.NoError(t, AddTransaction(Transaction{Amount: 1, Category: "еда", Date: date}))
		require.NoError(t, AddTransaction(Transaction{Amount: 2, Category: "еда", Date: date}))
		txs := ListTransactions()
		require.Len(t, txs, 2)
		assert.Equal(t, 1, txs[0].ID)
		assert.Equal(t, 2, txs[1].ID)
	})

	t.Run("reset clears state", func(t *testing.T) {
		require.NoError(t, SetBudget(Budget{Category: "еда", Limit: 1}))
		require.NoError(t, AddTransaction(Transaction{Amount: 1, Category: "еда", Date: date}))
		Reset()
		assert.Empty(t, ListTransactions())
		assert.Empty(t, ListBudgets())
	})
}
