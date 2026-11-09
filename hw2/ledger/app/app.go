package app

import (
	"context"
	"fmt"
	"log"

	"ledger/domain"
	"ledger/internal/cache"
	"ledger/internal/db"
	"ledger/internal/repository/cached"
	"ledger/internal/repository/memory"
	"ledger/internal/repository/pg"
	"ledger/service"
)

func New(ctx context.Context) (service.Service, func(), error) {
	cfg, err := ConfigFromEnv()
	if err != nil {
		return nil, nil, err
	}
	return NewWithConfig(ctx, cfg)
}

func NewWithConfig(ctx context.Context, cfg Config) (service.Service, func(), error) {
	conn, err := db.Open(ctx, cfg.DSN)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to database: %w", err)
	}
	log.Println("ledger: connected to PostgreSQL")

	var (
		budgets  domain.BudgetRepository  = pg.NewBudgetRepository(conn)
		expenses domain.ExpenseRepository = pg.NewExpenseRepository(conn)
	)

	kv, err := cache.Open(ctx, cfg.Redis)
	if err != nil {
		log.Printf("ledger: redis unavailable, caching disabled: %v", err)
	} else {
		log.Printf("ledger: connected to Redis at %s (db %d)", cfg.Redis.Addr, cfg.Redis.DB)
		budgets = cached.NewBudgetRepository(budgets, kv)
		expenses = cached.NewExpenseRepository(expenses, kv)
	}

	closeFn := func() {
		if err := kv.Close(); err != nil {
			log.Printf("ledger: close redis: %v", err)
		}
		if err := conn.Close(); err != nil {
			log.Printf("ledger: close postgres: %v", err)
		}
	}
	return service.New(budgets, expenses), closeFn, nil
}

func NewInMemory() service.Service {
	return service.New(memory.NewBudgetRepository(), memory.NewExpenseRepository())
}
