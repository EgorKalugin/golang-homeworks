package transactions

import "errors"

var transactions = []Transaction{}

var ErrNonPositiveAmount = errors.New("amount must be positive")

func AddTransaction(tx Transaction) error {
	if tx.Amount <= 0 {
		return ErrNonPositiveAmount
	}
	tx.ID = len(transactions) + 1

	transactions = append(transactions, tx)
	return nil
}

func ListTransactions() []Transaction {
	return transactions
}
