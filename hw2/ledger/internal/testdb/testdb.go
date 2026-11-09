package testdb

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"ledger/internal/db"
)

const (
	DefaultDSN = "postgres://postgres:postgres@localhost:5432/cashapp_test?sslmode=disable"
	lockID     = 7_000_001
)

func DSN() string {
	if dsn := os.Getenv("TEST_DATABASE_URL"); dsn != "" {
		return dsn
	}
	return DefaultDSN
}

func Open(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	conn, err := db.Open(ctx, DSN())
	if err != nil {
		t.Skipf("test database unavailable: %v (set TEST_DATABASE_URL and apply migrations to it)", err)
	}
	lock, err := conn.Conn(ctx)
	if err != nil {
		t.Fatalf("testdb: %v", err)
	}
	if _, err := lock.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, lockID); err != nil {
		t.Fatalf("testdb: lock: %v", err)
	}

	Truncate(t, conn)
	t.Cleanup(func() {
		Truncate(t, conn)
		lock.ExecContext(ctx, `SELECT pg_advisory_unlock($1)`, lockID)
		lock.Close()
		conn.Close()
	})
	return conn
}

func Truncate(t *testing.T, conn *sql.DB) {
	t.Helper()
	if _, err := conn.Exec(`TRUNCATE budgets, expenses RESTART IDENTITY`); err != nil {
		t.Fatalf("testdb: truncate: %v", err)
	}
}
