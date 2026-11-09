package domain

import "time"

type Transaction struct {
	ID          int
	Amount      int
	Category    string
	Description string
	Date        time.Time
}

type Budget struct {
	Category string `json:"category"`
	Limit    int    `json:"limit"`
}

type CategoryTotal struct {
	Category string `json:"category"`
	Total    int    `json:"total"`
}

type ReportSummary struct {
	From       string          `json:"from"`
	To         string          `json:"to"`
	Total      int             `json:"total"`
	Categories []CategoryTotal `json:"categories"`
}
