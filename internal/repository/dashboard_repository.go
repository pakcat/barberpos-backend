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
	TodayTransactions int64
	TodayCustomers    int64
	ServicesSold      int64
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

func (r DashboardRepository) Summary(ctx context.Context, ownerUserID int64) (DashboardSummary, error) {
	var s DashboardSummary
	err := r.DB.Pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE status = 'paid'),0) AS total_revenue,
			COUNT(*) FILTER (WHERE status = 'paid') AS total_tx,
			COALESCE(SUM(amount) FILTER (WHERE status = 'paid' AND transacted_date = CURRENT_DATE),0) AS today_revenue,
			COUNT(*) FILTER (WHERE status = 'paid' AND transacted_date = CURRENT_DATE) AS today_tx,
			COALESCE((
				SELECT COUNT(DISTINCT NULLIF(customer_name, ''))
				FROM transactions
				WHERE deleted_at IS NULL AND status = 'paid' AND transacted_date = CURRENT_DATE AND owner_user_id=$1
			),0) AS today_customers,
			COALESCE((
				SELECT SUM(ti.qty)
				FROM transaction_items ti
				JOIN transactions t ON t.id = ti.transaction_id
				WHERE t.deleted_at IS NULL AND t.status = 'paid' AND t.transacted_date = CURRENT_DATE AND t.owner_user_id=$1
			),0) AS services_sold
		FROM transactions
		WHERE deleted_at IS NULL AND owner_user_id=$1
	`, ownerUserID).Scan(&s.TotalRevenue, &s.TotalTransactions, &s.TodayRevenue, &s.TodayTransactions, &s.TodayCustomers, &s.ServicesSold)
	return s, err
}

func (r DashboardRepository) TopServices(ctx context.Context, ownerUserID int64, limit int) ([]DashboardItem, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT name, COALESCE(SUM(price*qty),0) AS amount, SUM(qty) AS qty
		FROM transaction_items
		WHERE transaction_id IN (
			SELECT id FROM transactions WHERE deleted_at IS NULL AND owner_user_id=$1
		)
		GROUP BY name
		ORDER BY amount DESC
		LIMIT $2
	`, ownerUserID, limit)
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

func (r DashboardRepository) TopStaff(ctx context.Context, ownerUserID int64, limit int) ([]DashboardItem, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT stylist, COALESCE(SUM(amount),0) AS amount, COUNT(*) AS cnt
		FROM transactions
		WHERE stylist <> '' AND deleted_at IS NULL AND owner_user_id=$1
		GROUP BY stylist
		ORDER BY amount DESC
		LIMIT $2
	`, ownerUserID, limit)
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

func (r DashboardRepository) SalesSeries(ctx context.Context, ownerUserID int64, days int) ([]SalesPoint, error) {
	start := time.Now().AddDate(0, 0, -days+1).Format("2006-01-02")
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT transacted_date, COALESCE(SUM(amount),0) AS amount
		FROM transactions
		WHERE deleted_at IS NULL
		  AND owner_user_id=$2
		  AND transacted_date >= $1::date
		GROUP BY transacted_date
		ORDER BY transacted_date ASC
	`, start, ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var points []SalesPoint
	for rows.Next() {
		var p SalesPoint
		var date time.Time
		if err := rows.Scan(&date, &p.Amount); err != nil {
			return nil, err
		}
		p.Label = date.Format("2006-01-02")
		points = append(points, p)
	}
	return points, rows.Err()
}
