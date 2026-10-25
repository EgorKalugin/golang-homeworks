package transactions

import (
	"errors"
	"fmt"
)

var transactions = []Transaction{}

var ErrNonPositiveAmount = errors.New("amount must be positive")

func AddTransaction(tx Transaction) error {
	if tx.Amount <= 0 {
		return ErrNonPositiveAmount
	}
	if b, ok := budgets[tx.Category]; ok {
		spent := spentInCategory(tx.Category)
		if spent+tx.Amount > b.Limit {
			return fmt.Errorf("%w: category %q: spent %d + %d > limit %d",
				ErrBudgetExceeded, tx.Category, spent, tx.Amount, b.Limit)
		}
	}
	tx.ID = len(transactions) + 1

	transactions = append(transactions, tx)
	return nil
}

func ListTransactions() []Transaction {
	return transactions
}
