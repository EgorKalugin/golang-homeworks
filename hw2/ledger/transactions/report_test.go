package transactions

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetReportSummary(t *testing.T) {
	withDB(t)
	ctx := context.Background()
	day := func(d int) time.Time { return time.Date(2025, 9, d, 0, 0, 0, 0, time.UTC) }

	require.NoError(t, AddTransaction(Transaction{Amount: 100, Category: "еда", Date: day(1)}))
	require.NoError(t, AddTransaction(Transaction{Amount: 250, Category: "еда", Date: day(10)}))
	require.NoError(t, AddTransaction(Transaction{Amount: 40, Category: "транспорт", Date: day(10)}))
	require.NoError(t, AddTransaction(Transaction{Amount: 999, Category: "еда", Date: day(20)}))

	want := ReportSummary{
		From:  "2025-09-01",
		To:    "2025-09-10",
		Total: 390,
		Categories: []CategoryTotal{
			{Category: "еда", Total: 350},
			{Category: "транспорт", Total: 40},
		},
	}
	got, err := GetReportSummary(ctx, day(1), day(10))
	require.NoError(t, err)
	assert.Equal(t, want, got)

	t.Run("cached result is returned within TTL", func(t *testing.T) {
		if kv == nil {
			t.Skip("redis unavailable")
		}
		require.NoError(t, AddTransaction(Transaction{Amount: 1, Category: "еда", Date: day(5)}))
		got, err := GetReportSummary(ctx, day(1), day(10))
		require.NoError(t, err)
		assert.Equal(t, want, got, "second call must be served from cache")
	})

	t.Run("empty period", func(t *testing.T) {
		got, err := GetReportSummary(ctx, day(25), day(30))
		require.NoError(t, err)
		assert.Equal(t, ReportSummary{From: "2025-09-25", To: "2025-09-30", Categories: []CategoryTotal{}}, got)
	})

	t.Run("invalid period", func(t *testing.T) {
		_, err := GetReportSummary(ctx, day(10), day(1))
		assert.ErrorIs(t, err, ErrInvalidPeriod)
	})
}

func TestListBudgetsCacheInvalidation(t *testing.T) {
	withDB(t)

	require.NoError(t, SetBudget(Budget{Category: "еда", Limit: 100}))
	assert.Equal(t, []Budget{{"еда", 100}}, mustBudgets(t))

	require.NoError(t, SetBudget(Budget{Category: "еда", Limit: 200}))
	assert.Equal(t, []Budget{{"еда", 200}}, mustBudgets(t), "SetBudget must drop budgets:all")
}
