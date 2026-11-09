package app

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ledger/domain"
	"ledger/internal/cache"
	"ledger/internal/repository/cached"
	"ledger/internal/testdb"
	"ledger/service"
)

func testConfig(t *testing.T) Config {
	t.Helper()
	redisDB := os.Getenv("TEST_REDIS_DB")
	if redisDB == "" {
		redisDB = "15"
	}
	t.Setenv("DATABASE_URL", testdb.DSN())
	t.Setenv("REDIS_DB", redisDB)
	cfg, err := ConfigFromEnv()
	require.NoError(t, err)
	return cfg
}

func newTestService(t *testing.T) (service.Service, *cache.Cache) {
	t.Helper()
	cfg := testConfig(t)
	ctx := context.Background()

	testdb.Open(t)
	kv, _ := cache.Open(ctx, cfg.Redis)
	dropCache(t, kv)

	svc, closeFn, err := NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		closeFn()
		dropCache(t, kv)
		kv.Close()
	})
	return svc, kv
}

func dropCache(t *testing.T, kv *cache.Cache) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, kv.Del(ctx, cached.BudgetsKey))
	require.NoError(t, kv.DelPrefix(ctx, cached.ReportKeyPrefix))
}

func TestNewFailsOnBadDSN(t *testing.T) {
	cfg := Config{DSN: "postgres://nobody:wrong@127.0.0.1:1/none?sslmode=disable", Redis: cache.Config{Addr: "127.0.0.1:1"}}
	svc, closeFn, err := NewWithConfig(context.Background(), cfg)
	require.Error(t, err)
	assert.Nil(t, svc)
	assert.Nil(t, closeFn)
}

func TestNewRespectsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := NewWithConfig(ctx, testConfig(t))
	require.ErrorIs(t, err, context.Canceled)
}

func TestLedgerOnPostgres(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	date := time.Date(2025, 9, 10, 0, 0, 0, 0, time.UTC)

	require.NoError(t, svc.SetBudget(ctx, domain.Budget{Category: "еда", Limit: 5000}))

	first, err := svc.AddTransaction(ctx, domain.Transaction{Amount: 3000, Category: "еда", Description: "продукты", Date: date})
	require.NoError(t, err)
	assert.Equal(t, 1, first.ID)

	_, err = svc.AddTransaction(ctx, domain.Transaction{Amount: 2500, Category: "еда", Date: date})
	require.ErrorIs(t, err, domain.ErrBudgetExceeded)

	second, err := svc.AddTransaction(ctx, domain.Transaction{Amount: 2000, Category: "еда", Date: date})
	require.NoError(t, err, "exactly at limit")
	older, err := svc.AddTransaction(ctx, domain.Transaction{Amount: 10000, Category: "транспорт", Date: date.AddDate(0, 0, -1)})
	require.NoError(t, err, "category without budget is not limited")

	txs, err := svc.ListTransactions(ctx)
	require.NoError(t, err)
	assert.Equal(t, []domain.Transaction{second, first, older}, txs)

	require.ErrorContains(t, svc.LoadBudgets(ctx, strings.NewReader(`[{"category":"a","limit":1},{"category":"","limit":1}]`)),
		"item 1: category must not be empty")
	require.NoError(t, svc.LoadBudgets(ctx, strings.NewReader(`[{"category":"транспорт","limit":20000},{"category":"еда","limit":6000}]`)))
	bs, err := svc.ListBudgets(ctx)
	require.NoError(t, err)
	assert.Equal(t, []domain.Budget{{Category: "еда", Limit: 6000}, {Category: "транспорт", Limit: 20000}}, bs)
}

func TestCaching(t *testing.T) {
	svc, kv := newTestService(t)
	ctx := context.Background()
	day := func(d int) time.Time { return time.Date(2025, 9, d, 0, 0, 0, 0, time.UTC) }

	t.Run("budgets list is invalidated on update", func(t *testing.T) {
		require.NoError(t, svc.SetBudget(ctx, domain.Budget{Category: "еда", Limit: 100}))
		bs, err := svc.ListBudgets(ctx)
		require.NoError(t, err)
		assert.Equal(t, []domain.Budget{{Category: "еда", Limit: 100}}, bs)

		require.NoError(t, svc.SetBudget(ctx, domain.Budget{Category: "еда", Limit: 20000}))
		bs, err = svc.ListBudgets(ctx)
		require.NoError(t, err)
		assert.Equal(t, []domain.Budget{{Category: "еда", Limit: 20000}}, bs, "SetBudget must drop budgets:all")
	})

	t.Run("report is cached within TTL", func(t *testing.T) {
		if kv == nil {
			t.Skip("redis unavailable")
		}
		for _, tx := range []domain.Transaction{
			{Amount: 100, Category: "еда", Date: day(1)},
			{Amount: 250, Category: "еда", Date: day(10)},
			{Amount: 40, Category: "транспорт", Date: day(10)},
			{Amount: 999, Category: "еда", Date: day(20)},
		} {
			_, err := svc.AddTransaction(ctx, tx)
			require.NoError(t, err)
		}
		want := domain.ReportSummary{
			From: "2025-09-01", To: "2025-09-10", Total: 390,
			Categories: []domain.CategoryTotal{{Category: "еда", Total: 350}, {Category: "транспорт", Total: 40}},
		}
		got, err := svc.ReportSummary(ctx, day(1), day(10))
		require.NoError(t, err)
		assert.Equal(t, want, got)

		_, err = svc.AddTransaction(ctx, domain.Transaction{Amount: 1, Category: "еда", Date: day(5)})
		require.NoError(t, err)
		got, err = svc.ReportSummary(ctx, day(1), day(10))
		require.NoError(t, err)
		assert.Equal(t, want, got, "second call must be served from cache")
	})
}
