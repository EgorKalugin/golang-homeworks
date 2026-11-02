package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"ledger/transactions"
	"log"
	"os"
	"time"
)

func main() {
	budgetsFile := flag.String("budgets", "", "JSON file with budgets to upsert on start (e.g. budgets.json)")
	flag.Parse()

	ctx := context.Background()
	if err := transactions.Init(ctx); err != nil {
		log.Fatalf("ledger: cannot start: %v\n"+
			"check that PostgreSQL is running, DATABASE_URL (or DB_HOST/DB_PORT/DB_USER/DB_PASS/DB_NAME) is correct "+
			"and migrations are applied: goose -dir ./migrations postgres \"$DATABASE_URL\" up", err)
	}
	defer transactions.Close()
	log.Println("Ledger service started")

	fmt.Println("--- CheckValid demo ---")
	checks := []transactions.Validatable{
		transactions.Transaction{Amount: 100, Category: "food", Date: time.Now()},
		transactions.Transaction{Amount: -50, Category: "food", Date: time.Now()},
		transactions.Transaction{Amount: 100, Category: "", Date: time.Now()},
		transactions.Transaction{Amount: 100, Category: "food"},
		transactions.Budget{Category: "food", Limit: 1000},
		transactions.Budget{Category: "food", Limit: 0},
		transactions.Budget{Category: "", Limit: 1000},
	}
	for _, v := range checks {
		if err := transactions.CheckValid(v); err != nil {
			fmt.Printf("invalid %T %+v: %v\n", v, v, err)
			continue
		}
		fmt.Printf("valid   %T %+v\n", v, v)
	}
	fmt.Println("------------------------")

	if *budgetsFile != "" {
		if err := loadBudgetsFromFile(*budgetsFile); err != nil {
			log.Fatalf("ledger: %v", err)
		}
		log.Printf("budgets loaded from %s", *budgetsFile)
	}

	budgets, err := transactions.ListBudgets()
	if err != nil {
		log.Fatalf("ledger: %v", err)
	}
	fmt.Printf("budgets in database: %d\n", len(budgets))
	for _, b := range budgets {
		fmt.Printf("  %s: %d\n", b.Category, b.Limit)
	}

	txs, err := transactions.ListTransactions()
	if err != nil {
		log.Fatalf("ledger: %v", err)
	}
	fmt.Printf("transactions in database: %d\n", len(txs))
	for _, tx := range txs {
		fmt.Printf("  #%d %s %d (%s) %s\n",
			tx.ID, tx.Category, tx.Amount, tx.Description, tx.Date.Format("2006-01-02"))
	}
}

func loadBudgetsFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open budgets file: %w", err)
	}
	defer f.Close()

	return transactions.LoadBudgets(bufio.NewReader(f))
}
