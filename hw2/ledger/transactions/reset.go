package transactions

import (
	"context"
	"log"
)

func Reset() {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	if store != nil {
		if _, err := store.ExecContext(ctx, `TRUNCATE budgets, expenses RESTART IDENTITY`); err != nil {
			log.Printf("reset: truncate: %v", err)
		}
	}
	if err := kv.Del(ctx, budgetsCacheKey); err != nil {
		log.Printf("reset: %v", err)
	}
	if err := kv.DelPrefix(ctx, reportCacheKeyPrefix); err != nil {
		log.Printf("reset: %v", err)
	}
}
