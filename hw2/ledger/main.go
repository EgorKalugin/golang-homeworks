package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"ledger/app"
	"ledger/domain"
	"ledger/service"
	"log"
	"os"
	"time"
)

func main() {
	budgetsFile := flag.String("budgets", "", "JSON file with budgets to upsert on start (e.g. budgets.json)")
	flag.Parse()

	ctx := context.Background()
	svc, closeFn, err := app.New(ctx)
	if err != nil {
		log.Fatalf("ledger: cannot start: %v\n"+
			"check that PostgreSQL is running, DATABASE_URL (or DB_HOST/DB_PORT/DB_USER/DB_PASS/DB_NAME) is correct "+
			"and migrations are applied: goose -dir ./migrations postgres \"$DATABASE_URL\" up", err)
	}
	defer closeFn()
	log.Println("Ledger service started")

	fmt.Println("--- CheckValid demo ---")
	checks := []domain.Validatable{
		domain.Transaction{Amount: 100, Category: "food", Date: time.Now()},
		domain.Transaction{Amount: -50, Category: "food", Date: time.Now()},
		domain.Transaction{Amount: 100, Category: "", Date: time.Now()},
		domain.Transaction{Amount: 100, Category: "food"},
		domain.Budget{Category: "food", Limit: 1000},
		domain.Budget{Category: "food", Limit: 0},
		domain.Budget{Category: "", Limit: 1000},
	}
	for _, v := range checks {
		if err := domain.CheckValid(v); err != nil {
			fmt.Printf("invalid %T %+v: %v\n", v, v, err)
			continue
		}
		fmt.Printf("valid   %T %+v\n", v, v)
	}
	fmt.Println("------------------------")

	if err := run(ctx, svc, *budgetsFile); err != nil {
		log.Printf("ledger: %v", err)
		closeFn()
		os.Exit(1)
	}
}

func run(ctx context.Context, svc service.Service, budgetsFile string) error {
	if budgetsFile != "" {
		if err := loadBudgetsFromFile(ctx, svc, budgetsFile); err != nil {
			return err
		}
		log.Printf("budgets loaded from %s", budgetsFile)
	}

	budgets, err := svc.ListBudgets(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("budgets in database: %d\n", len(budgets))
	for _, b := range budgets {
		fmt.Printf("  %s: %d\n", b.Category, b.Limit)
	}

	txs, err := svc.ListTransactions(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("transactions in database: %d\n", len(txs))
	for _, tx := range txs {
		fmt.Printf("  #%d %s %d (%s) %s\n",
			tx.ID, tx.Category, tx.Amount, tx.Description, tx.Date.Format("2006-01-02"))
	}
	return nil
}

func loadBudgetsFromFile(ctx context.Context, svc service.Service, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open budgets file: %w", err)
	}
	defer f.Close()

	return svc.LoadBudgets(ctx, bufio.NewReader(f))
}
