package main

import (
	"bufio"
	"fmt"
	"ledger/transactions"
	"os"
	"time"
)

func main() {
	fmt.Println("Ledger service started")

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

	for _, b := range []transactions.Budget{
		{Category: "food", Limit: 3000},
		{Category: "transport", Limit: 3000},
		{Category: "", Limit: 500},
	} {
		if err := transactions.SetBudget(b); err != nil {
			fmt.Printf("budget rejected: %+v: %v\n", b, err)
		}
	}

	if err := loadBudgetsFromFile("budgets.json"); err != nil {
		fmt.Println("error:", err)
	}

	add := func(amount int, category, description string) {
		err := transactions.AddTransaction(transactions.Transaction{
			Amount:      amount,
			Category:    category,
			Description: description,
			Date:        time.Now(),
		})
		if err != nil {
			fmt.Printf("rejected: %s %d (%s): %v\n", category, amount, description, err)
			return
		}
		fmt.Printf("accepted: %s %d (%s)\n", category, amount, description)
	}

	add(1500, "food", "groceries")
	add(3500, "food", "restaurant")
	add(1, "food", "gum")
	add(3500, "transport", "taxi")
	add(10000, "rent", "monthly rent")
	add(0, "food", "free sample")
	add(200, "", "unknown")

	for _, tx := range transactions.ListTransactions() {
		fmt.Printf("#%d %s %d (%s) %s\n",
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
