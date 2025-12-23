package repository

import (
	"context"
	"time"

	"barberpos-backend/internal/db"
)

type DashboardRepository struct {
	DB *db.Postgres
}

type DashboardSummary struct {
	TotalRevenue      int64
	TotalTransactions int64
	TodayRevenue      int64
}

type DashboardItem struct {
	Name   string
	Amount int64
	Count  int64
}

type SalesPoint struct {
	Label  string
	Amount int64
}

func (r DashboardRepository) Summary(ctx context.Context) (DashboardSummary, error) {
	var s DashboardSummary
	err := r.DB.Pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(amount),0) AS total_revenue,
			COUNT(*) AS total_tx,
			COALESCE(SUM(amount) FILTER (WHERE transacted_date = CURRENT_DATE),0) AS today_revenue
		FROM transactions
		WHERE deleted_at IS NULL AND status = 'paid'
	`).Scan(&s.TotalRevenue, &s.TotalTransactions, &s.TodayRevenue)
	return s, err
}

func (r DashboardRepository) TopServices(ctx context.Context, limit int) ([]DashboardItem, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT name, COALESCE(SUM(price*qty),0) AS amount, SUM(qty) AS qty
		FROM transaction_items
		GROUP BY name
		ORDER BY amount DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DashboardItem
	for rows.Next() {
		var it DashboardItem
		if err := rows.Scan(&it.Name, &it.Amount, &it.Count); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (r DashboardRepository) TopStaff(ctx context.Context, limit int) ([]DashboardItem, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT stylist, COALESCE(SUM(amount),0) AS amount, COUNT(*) AS cnt
		FROM transactions
		WHERE stylist <> ''
		GROUP BY stylist
		ORDER BY amount DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DashboardItem
	for rows.Next() {
		var it DashboardItem
		if err := rows.Scan(&it.Name, &it.Amount, &it.Count); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (r DashboardRepository) SalesSeries(ctx context.Context, days int) ([]SalesPoint, error) {
	start := time.Now().AddDate(0, 0, -days+1).Format("2006-01-02")
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT transacted_date, COALESCE(SUM(amount),0) AS amount
		FROM transactions
		WHERE deleted_at IS NULL
		  AND transacted_date >= $1::date
		GROUP BY transacted_date
		ORDER BY transacted_date ASC
	`, start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var points []SalesPoint
	for rows.Next() {
		var p SalesPoint
		if err := rows.Scan(&p.Label, &p.Amount); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}
