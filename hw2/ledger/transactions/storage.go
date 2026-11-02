package transactions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"ledger/internal/cache"
	"ledger/internal/db"
)

const queryTimeout = 5 * time.Second

var (
	store *sql.DB
	kv    *cache.Cache
)

var ErrNotInitialized = errors.New("ledger storage is not initialized, call transactions.Init first")

func Init(ctx context.Context) error {
	conn, err := db.Open(ctx, db.DSNFromEnv())
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	store = conn
	log.Println("ledger: connected to PostgreSQL")

	cfg, err := cache.ConfigFromEnv()
	if err != nil {
		return fmt.Errorf("redis config: %w", err)
	}
	c, err := cache.Open(ctx, cfg)
	if err != nil {
		log.Printf("ledger: redis unavailable, caching disabled: %v", err)
		return nil
	}
	kv = c
	log.Printf("ledger: connected to Redis at %s (db %d)", cfg.Addr, cfg.DB)
	return nil
}

func Close() {
	if kv != nil {
		kv.Close()
		kv = nil
	}
	if store != nil {
		store.Close()
		store = nil
	}
}

func conn() (*sql.DB, error) {
	if store == nil {
		return nil, ErrNotInitialized
	}
	return store, nil
}
