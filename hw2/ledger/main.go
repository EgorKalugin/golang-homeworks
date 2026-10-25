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

	transactions.SetBudget(transactions.Budget{Category: "food", Limit: 3000})
	transactions.SetBudget(transactions.Budget{Category: "transport", Limit: 3000})

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

	add(1500, "food", "groceries")     // within limit
	add(3500, "food", "restaurant")    // exactly reaches the limit 5000
	add(1, "food", "gum")              // exceeds the limit
	add(3500, "transport", "taxi")     // exceeds the limit at once
	add(10000, "rent", "monthly rent") // no budget for category

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
