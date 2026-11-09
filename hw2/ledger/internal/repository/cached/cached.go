package cached

import (
	"context"
	"log"
	"time"

	"ledger/domain"
	"ledger/internal/cache"
)

const (
	BudgetsKey      = "budgets:all"
	budgetsTTL      = 30 * time.Second
	ReportKeyPrefix = "report:summary:"
	reportTTL       = 30 * time.Second
	dateLayout      = "2006-01-02"
)

type BudgetRepository struct {
	domain.BudgetRepository
	kv *cache.Cache
}

func NewBudgetRepository(next domain.BudgetRepository, kv *cache.Cache) *BudgetRepository {
	return &BudgetRepository{BudgetRepository: next, kv: kv}
}

func (r *BudgetRepository) Upsert(ctx context.Context, budgets ...domain.Budget) error {
	if err := r.BudgetRepository.Upsert(ctx, budgets...); err != nil {
		return err
	}
	if err := r.kv.Del(ctx, BudgetsKey); err != nil {
		log.Printf("budgets cache: %v", err)
	}
	return nil
}

func (r *BudgetRepository) List(ctx context.Context) ([]domain.Budget, error) {
	var list []domain.Budget
	if ok, err := r.kv.GetJSON(ctx, BudgetsKey, &list); err != nil {
		log.Printf("budgets cache: %v", err)
	} else if ok {
		return list, nil
	}

	list, err := r.BudgetRepository.List(ctx)
	if err != nil {
		return nil, err
	}
	if err := r.kv.SetJSON(ctx, BudgetsKey, list, budgetsTTL); err != nil {
		log.Printf("budgets cache: %v", err)
	}
	return list, nil
}

type ExpenseRepository struct {
	domain.ExpenseRepository
	kv *cache.Cache
}

func NewExpenseRepository(next domain.ExpenseRepository, kv *cache.Cache) *ExpenseRepository {
	return &ExpenseRepository{ExpenseRepository: next, kv: kv}
}

func (r *ExpenseRepository) TotalsByCategory(ctx context.Context, from, to time.Time) ([]domain.CategoryTotal, error) {
	key := ReportKeyPrefix + from.Format(dateLayout) + ":" + to.Format(dateLayout)

	var totals []domain.CategoryTotal
	if ok, err := r.kv.GetJSON(ctx, key, &totals); err != nil {
		log.Printf("report cache: %v", err)
	} else if ok {
		log.Printf("report cache hit: %s", key)
		return totals, nil
	}
	log.Printf("report cache miss: %s", key)

	totals, err := r.ExpenseRepository.TotalsByCategory(ctx, from, to)
	if err != nil {
		return nil, err
	}
	if err := r.kv.SetJSON(ctx, key, totals, reportTTL); err != nil {
		log.Printf("report cache: %v", err)
	}
	return totals, nil
}
