package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ledger/domain"
	"ledger/internal/repository/memory"
	"ledger/service"
)

var ctx = context.Background()

func newService() service.Service {
	return service.New(memory.NewBudgetRepository(), memory.NewExpenseRepository())
}

func mustList(t *testing.T, svc service.Service) []domain.Transaction {
	t.Helper()
	txs, err := svc.ListTransactions(ctx)
	require.NoError(t, err)
	return txs
}

func mustBudgets(t *testing.T, svc service.Service) []domain.Budget {
	t.Helper()
	bs, err := svc.ListBudgets(ctx)
	require.NoError(t, err)
	return bs
}

func TestAddTransactionBudget(t *testing.T) {
	tests := []struct {
		name    string
		budget  *domain.Budget
		amounts []int
		wantErr []bool
	}{
		{"within limit", &domain.Budget{Category: "food", Limit: 5000}, []int{1500, 2000}, []bool{false, false}},
		{"exactly at limit", &domain.Budget{Category: "food", Limit: 5000}, []int{2000, 3000}, []bool{false, false}},
		{"exceeds limit", &domain.Budget{Category: "food", Limit: 5000}, []int{4000, 1001, 1000}, []bool{false, true, false}},
		{"no budget", nil, []int{100000}, []bool{false}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newService()
			if tt.budget != nil {
				require.NoError(t, svc.SetBudget(ctx, *tt.budget))
			}

			accepted := 0
			for i, amount := range tt.amounts {
				_, err := svc.AddTransaction(ctx, domain.Transaction{Amount: amount, Category: "food", Date: time.Now()})
				if tt.wantErr[i] {
					require.ErrorIs(t, err, domain.ErrBudgetExceeded, "tx %d", i)
					continue
				}
				require.NoError(t, err, "tx %d", i)
				accepted++
			}
			assert.Len(t, mustList(t, svc), accepted)
		})
	}
}

func TestSetBudgetUpdatesLimit(t *testing.T) {
	svc := newService()
	tx := domain.Transaction{Amount: 1500, Category: "food", Date: time.Now()}

	require.NoError(t, svc.SetBudget(ctx, domain.Budget{Category: "food", Limit: 1000}))
	_, err := svc.AddTransaction(ctx, tx)
	require.ErrorIs(t, err, domain.ErrBudgetExceeded)

	require.NoError(t, svc.SetBudget(ctx, domain.Budget{Category: "food", Limit: 2000}))
	_, err = svc.AddTransaction(ctx, tx)
	require.NoError(t, err, "after raising limit")
}

func TestLoadBudgets(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
		want    []domain.Budget
	}{
		{
			name:  "valid",
			input: `[{"category":"food","limit":5000},{"category":"transport","limit":3000}]`,
			want:  []domain.Budget{{Category: "food", Limit: 5000}, {Category: "transport", Limit: 3000}},
		},
		{name: "broken json", input: `[{"category":"food",`, wantErr: "decode json"},
		{name: "not an array", input: `{"category":"food","limit":5000}`, wantErr: "decode json"},
		{name: "empty category", input: `[{"category":"food","limit":1},{"category":"","limit":5}]`, wantErr: "item 1: category must not be empty"},
		{name: "non-positive limit", input: `[{"category":"food","limit":0}]`, wantErr: "limit must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newService()
			err := svc.LoadBudgets(ctx, strings.NewReader(tt.input))

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Empty(t, mustBudgets(t, svc), "budgets must stay untouched on error")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, mustBudgets(t, svc))
		})
	}
}

func TestBudgetFlow(t *testing.T) {
	date := time.Date(2025, 9, 10, 0, 0, 0, 0, time.UTC)
	add := func(svc service.Service, amount int, category string) error {
		_, err := svc.AddTransaction(ctx, domain.Transaction{Amount: amount, Category: category, Date: date})
		return err
	}

	t.Run("within limit then exceeded", func(t *testing.T) {
		svc := newService()
		require.NoError(t, svc.SetBudget(ctx, domain.Budget{Category: "еда", Limit: 5000}))

		require.NoError(t, add(svc, 3000, "еда"))
		require.Len(t, mustList(t, svc), 1)

		err := add(svc, 2500, "еда")
		assert.ErrorIs(t, err, domain.ErrBudgetExceeded)
		assert.ErrorContains(t, err, "budget exceeded")
		assert.Len(t, mustList(t, svc), 1, "rejected transaction must not be stored")
	})

	t.Run("exactly at limit", func(t *testing.T) {
		svc := newService()
		require.NoError(t, svc.SetBudget(ctx, domain.Budget{Category: "еда", Limit: 5000}))
		assert.NoError(t, add(svc, 5000, "еда"))
		assert.ErrorIs(t, add(svc, 1, "еда"), domain.ErrBudgetExceeded)
		assert.Len(t, mustList(t, svc), 1)
	})

	t.Run("other category is not limited", func(t *testing.T) {
		svc := newService()
		require.NoError(t, svc.SetBudget(ctx, domain.Budget{Category: "еда", Limit: 5000}))
		assert.NoError(t, add(svc, 10000, "транспорт"))
		assert.Len(t, mustList(t, svc), 1)
	})

	t.Run("invalid input is rejected", func(t *testing.T) {
		svc := newService()
		assert.ErrorIs(t, svc.SetBudget(ctx, domain.Budget{Category: "еда", Limit: 0}), domain.ErrNonPositiveLimit)
		assert.Empty(t, mustBudgets(t, svc))

		_, err := svc.AddTransaction(ctx, domain.Transaction{Amount: 100, Date: date})
		assert.ErrorIs(t, err, domain.ErrEmptyCategory)
		assert.Empty(t, mustList(t, svc))
	})

	t.Run("ids are generated and list is newest first", func(t *testing.T) {
		svc := newService()
		first, err := svc.AddTransaction(ctx, domain.Transaction{Amount: 1, Category: "еда", Description: "чай", Date: date})
		require.NoError(t, err)
		assert.Equal(t, 1, first.ID)
		second, err := svc.AddTransaction(ctx, domain.Transaction{Amount: 2, Category: "еда", Date: date})
		require.NoError(t, err)
		assert.Equal(t, 2, second.ID)
		older, err := svc.AddTransaction(ctx, domain.Transaction{Amount: 3, Category: "еда", Date: date.AddDate(0, 0, -1)})
		require.NoError(t, err)

		assert.Equal(t, []domain.Transaction{second, first, older}, mustList(t, svc))
	})
}

func TestReportSummary(t *testing.T) {
	svc := newService()
	day := func(d int) time.Time { return time.Date(2025, 9, d, 0, 0, 0, 0, time.UTC) }
	for _, tx := range []domain.Transaction{
		{Amount: 100, Category: "еда", Date: day(1)},
		{Amount: 250, Category: "еда", Date: day(10)},
		{Amount: 40, Category: "транспорт", Date: day(10)},
		{Amount: 999, Category: "еда", Date: day(20)},
	} {
		_, err := svc.AddTransaction(ctx, tx)
		require.NoError(t, err)
	}

	got, err := svc.ReportSummary(ctx, day(1), day(10))
	require.NoError(t, err)
	assert.Equal(t, domain.ReportSummary{
		From: "2025-09-01", To: "2025-09-10", Total: 390,
		Categories: []domain.CategoryTotal{{Category: "еда", Total: 350}, {Category: "транспорт", Total: 40}},
	}, got)

	got, err = svc.ReportSummary(ctx, day(25), day(30))
	require.NoError(t, err)
	assert.Equal(t, domain.ReportSummary{From: "2025-09-25", To: "2025-09-30", Categories: []domain.CategoryTotal{}}, got)

	_, err = svc.ReportSummary(ctx, day(10), day(1))
	assert.ErrorIs(t, err, domain.ErrInvalidPeriod)
}

type failingRepo struct{ domain.ExpenseRepository }

func (failingRepo) SumByCategory(context.Context, string) (int, error) {
	return 0, errors.New("connection refused")
}

func TestStorageErrorIsNotBusinessError(t *testing.T) {
	budgets := memory.NewBudgetRepository()
	svc := service.New(budgets, failingRepo{memory.NewExpenseRepository()})
	require.NoError(t, svc.SetBudget(ctx, domain.Budget{Category: "еда", Limit: 10}))

	_, err := svc.AddTransaction(ctx, domain.Transaction{Amount: 1, Category: "еда", Date: time.Now()})
	require.ErrorContains(t, err, "connection refused")
	assert.NotErrorIs(t, err, domain.ErrBudgetExceeded)
	assert.False(t, domain.IsValidationError(err))
}
