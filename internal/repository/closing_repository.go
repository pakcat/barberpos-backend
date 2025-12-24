package repository

import (
	"context"
	"time"

	"barberpos-backend/internal/db"
)

type ClosingRepository struct {
	DB *db.Postgres
}

type ClosingSummary struct {
	TotalCash    int64
	TotalNonCash int64
	TotalCard    int64
}

type ClosingHistory struct {
	ID           int64
	Tanggal      time.Time
	Shift        string
	Karyawan     string
	ShiftID      *string
	OperatorName string
	Total        int64
	Status       string
	Catatan      string
	Fisik        string
	CreatedAt    time.Time
}

// Summary aggregates today's transactions by payment method.
func (r ClosingRepository) Summary(ctx context.Context, ownerUserID int64) (ClosingSummary, error) {
	var s ClosingSummary
	err := r.DB.Pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE lower(payment_method) = 'cash' AND status='paid' AND transacted_date = CURRENT_DATE),0) AS cash,
			COALESCE(SUM(amount) FILTER (WHERE lower(payment_method) <> 'cash' AND lower(payment_method) <> 'card' AND status='paid' AND transacted_date = CURRENT_DATE),0) AS noncash,
			COALESCE(SUM(amount) FILTER (WHERE lower(payment_method) = 'card' AND status='paid' AND transacted_date = CURRENT_DATE),0) AS card
		FROM transactions
		WHERE deleted_at IS NULL AND owner_user_id=$1
	`, ownerUserID).Scan(&s.TotalCash, &s.TotalNonCash, &s.TotalCard)
	return s, err
}

type CreateClosingInput struct {
	Tanggal      time.Time
	Shift        string
	Karyawan     string
	ShiftID      *string
	OperatorName string
	Total        int64
	Status       string
	Catatan      string
	Fisik        string
}

func (r ClosingRepository) Create(ctx context.Context, ownerUserID int64, in CreateClosingInput) (int64, error) {
	var id int64
	err := r.DB.Pool.QueryRow(ctx, `
		INSERT INTO closing_history (owner_user_id, tanggal, shift, karyawan, shift_id, operator_name, total, status, catatan, fisik, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, now(), now())
		RETURNING id
	`, ownerUserID, in.Tanggal.Format("2006-01-02"), in.Shift, in.Karyawan, in.ShiftID, in.OperatorName, in.Total, in.Status, in.Catatan, in.Fisik).Scan(&id)
	return id, err
}

func (r ClosingRepository) List(ctx context.Context, ownerUserID int64, limit int) ([]ClosingHistory, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id, tanggal, shift, karyawan, shift_id, operator_name, total, status, catatan, fisik, created_at
		FROM closing_history
		WHERE deleted_at IS NULL AND owner_user_id=$1
		ORDER BY tanggal DESC, id DESC
		LIMIT $2
	`, ownerUserID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ClosingHistory
	for rows.Next() {
		var c ClosingHistory
		if err := rows.Scan(
			&c.ID,
			&c.Tanggal,
			&c.Shift,
			&c.Karyawan,
			&c.ShiftID,
			&c.OperatorName,
			&c.Total,
			&c.Status,
			&c.Catatan,
			&c.Fisik,
			&c.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}
