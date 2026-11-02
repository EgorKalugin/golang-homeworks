package api

import (
	"context"
	"fmt"
	"ledger/transactions"
	"os"
	"testing"
)

const defaultTestDSN = "postgres://postgres:postgres@localhost:5432/cashapp_test?sslmode=disable"

var initErr error

func TestMain(m *testing.M) {
	initErr = initForTests()
	code := m.Run()
	transactions.Close()
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
	if err := transactions.Init(context.Background()); err != nil {
		return fmt.Errorf("%w (set TEST_DATABASE_URL and apply migrations to it)", err)
	}
	return nil
}

func resetStorage(t *testing.T) {
	t.Helper()
	if initErr != nil {
		t.Skipf("test database unavailable: %v", initErr)
	}
	transactions.Reset()
	t.Cleanup(transactions.Reset)
}
