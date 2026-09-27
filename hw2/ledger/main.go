package main

import (
	"fmt"
	"ledger/transactions"
	"time"
)

func main() {
	fmt.Println("Ledger service started")

	err := transactions.AddTransaction(transactions.Transaction{
		Amount:      1500,
		Category:    "food",
		Description: "groceries",
		Date:        time.Now(),
	})
	if err != nil {
		fmt.Println("error:", err)
	}
	err = transactions.AddTransaction(transactions.Transaction{
		Amount:      1500,
		Category:    "food",
		Description: "groceries",
		Date:        time.Now(),
	})
	if err != nil {
		fmt.Println("error:", err)
	}

	for _, tx := range transactions.ListTransactions() {
		fmt.Printf("#%d %s %d (%s) %s\n",
			tx.ID, tx.Category, tx.Amount, tx.Description, tx.Date.Format("2006-01-02"))
	}
}
