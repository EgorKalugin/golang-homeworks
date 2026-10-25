package transactions

func Reset() {
	transactions = []Transaction{}
	budgets = map[string]Budget{}
}
