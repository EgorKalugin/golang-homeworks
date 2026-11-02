package transactions

import (
	"context"
	"fmt"
	"os"
	"testing"
)

const defaultTestDSN = "postgres://postgres:postgres@localhost:5432/cashapp_test?sslmode=disable"

var initErr error

func TestMain(m *testing.M) {
	initErr = initForTests()
	code := m.Run()
	Close()
	os.Exit(code)
}

func initForTests() error {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = defaultTestDSN
	}
	redisDB := os.Getenv("TEST_REDIS_DB")
	if redisDB == "" {
		redisDB = "15"
	}
	os.Setenv("DATABASE_URL", dsn)
	os.Setenv("REDIS_DB", redisDB)
	if err := Init(context.Background()); err != nil {
		return fmt.Errorf("%w (set TEST_DATABASE_URL and apply migrations to it)", err)
	}
	return nil
}

func withDB(t *testing.T) {
	t.Helper()
	if initErr != nil {
		t.Skipf("test database unavailable: %v", initErr)
	}
	Reset()
	t.Cleanup(Reset)
}

func mustList(t *testing.T) []Transaction {
	t.Helper()
	txs, err := ListTransactions()
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	return txs
}

func mustBudgets(t *testing.T) []Budget {
	t.Helper()
	bs, err := ListBudgets()
	if err != nil {
		t.Fatalf("list budgets: %v", err)
	}
	return bs
}
