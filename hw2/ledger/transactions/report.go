package transactions

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

const (
	reportCacheKeyPrefix = "report:summary:"
	reportCacheTTL       = 30 * time.Second
	dateLayout           = "2006-01-02"
)

var ErrInvalidPeriod = errors.New("from must not be after to")

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

func GetReportSummary(ctx context.Context, from, to time.Time) (ReportSummary, error) {
	if from.After(to) {
		return ReportSummary{}, ErrInvalidPeriod
	}
	db, err := conn()
	if err != nil {
		return ReportSummary{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	summary := ReportSummary{From: from.Format(dateLayout), To: to.Format(dateLayout)}
	key := reportCacheKeyPrefix + summary.From + ":" + summary.To

	if ok, err := kv.GetJSON(ctx, key, &summary); err != nil {
		log.Printf("report cache: %v", err)
	} else if ok {
		log.Printf("report cache hit: %s", key)
		return summary, nil
	}
	log.Printf("report cache miss: %s", key)

	rows, err := db.QueryContext(ctx, `
		SELECT category, SUM(amount)::bigint
		FROM expenses
		WHERE date BETWEEN $1::date AND $2::date
		GROUP BY category ORDER BY category`, summary.From, summary.To)
	if err != nil {
		return ReportSummary{}, fmt.Errorf("report summary: %w", err)
	}
	defer rows.Close()

	summary.Categories = []CategoryTotal{}
	for rows.Next() {
		var ct CategoryTotal
		if err := rows.Scan(&ct.Category, &ct.Total); err != nil {
			return ReportSummary{}, fmt.Errorf("report summary: scan: %w", err)
		}
		summary.Total += ct.Total
		summary.Categories = append(summary.Categories, ct)
	}
	if err := rows.Err(); err != nil {
		return ReportSummary{}, fmt.Errorf("report summary: %w", err)
	}

	if err := kv.SetJSON(ctx, key, summary, reportCacheTTL); err != nil {
		log.Printf("report cache: %v", err)
	}
	return summary, nil
}
