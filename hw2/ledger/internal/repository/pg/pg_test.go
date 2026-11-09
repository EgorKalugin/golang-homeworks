package pg

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ledger/domain"
	"ledger/internal/testdb"
)

func TestBudgetRepository(t *testing.T) {
	repo := NewBudgetRepository(testdb.Open(t))
	ctx := context.Background()

	_, err := repo.GetByCategory(ctx, "еда")
	require.ErrorIs(t, err, domain.ErrBudgetNotFound)

	require.NoError(t, repo.Upsert(ctx, domain.Budget{Category: "еда", Limit: 100}))
	require.NoError(t, repo.Upsert(ctx, domain.Budget{Category: "еда", Limit: 200}))
	b, err := repo.GetByCategory(ctx, "еда")
	require.NoError(t, err)
	assert.Equal(t, domain.Budget{Category: "еда", Limit: 200}, b)

	err = repo.Upsert(ctx, domain.Budget{Category: "кино", Limit: 1}, domain.Budget{Category: "бар", Limit: -1})
	require.Error(t, err, "CHECK limit_amount > 0 must fail")
	list, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, []domain.Budget{{Category: "еда", Limit: 200}}, list, "batch upsert must be atomic")
}

func TestExpenseRepository(t *testing.T) {
	repo := NewExpenseRepository(testdb.Open(t))
	ctx := context.Background()
	day := func(d int) time.Time { return time.Date(2025, 9, d, 0, 0, 0, 0, time.UTC) }

	sum, err := repo.SumByCategory(ctx, "еда")
	require.NoError(t, err)
	assert.Zero(t, sum)

	a, err := repo.Add(ctx, domain.Transaction{Amount: 100, Category: "еда", Description: "чай", Date: day(1)})
	require.NoError(t, err)
	assert.Equal(t, 1, a.ID)
	b, err := repo.Add(ctx, domain.Transaction{Amount: 50, Category: "еда", Date: day(5)})
	require.NoError(t, err)
	c, err := repo.Add(ctx, domain.Transaction{Amount: 7, Category: "транспорт", Date: day(30)})
	require.NoError(t, err)

	sum, err = repo.SumByCategory(ctx, "еда")
	require.NoError(t, err)
	assert.Equal(t, 150, sum)

	list, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, []domain.Transaction{c, b, a}, list)

	totals, err := repo.TotalsByCategory(ctx, day(1), day(5))
	require.NoError(t, err)
	assert.Equal(t, []domain.CategoryTotal{{Category: "еда", Total: 150}}, totals)
}
