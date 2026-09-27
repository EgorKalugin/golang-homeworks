package transactions

import "time"

type Transaction struct {
	ID          int
	Amount      int
	Category    string
	Description string
	Date        time.Time
}
