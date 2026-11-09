package domain

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
			assert.True(t, IsValidationError(err))
		})
	}
}

func TestIsValidationError(t *testing.T) {
	assert.True(t, IsValidationError(fmt.Errorf("wrapped: %w", ErrInvalidPeriod)))
	assert.False(t, IsValidationError(ErrBudgetExceeded))
	assert.False(t, IsValidationError(fmt.Errorf("db is down")))
	assert.False(t, IsValidationError(nil))
}
