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

// Summary aggregates today's transactions by payment method.
func (r ClosingRepository) Summary(ctx context.Context) (ClosingSummary, error) {
	var s ClosingSummary
	err := r.DB.Pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE lower(payment_method) = 'cash' AND status='paid' AND transacted_date = CURRENT_DATE),0) AS cash,
			COALESCE(SUM(amount) FILTER (WHERE lower(payment_method) <> 'cash' AND lower(payment_method) <> 'card' AND status='paid' AND transacted_date = CURRENT_DATE),0) AS noncash,
			COALESCE(SUM(amount) FILTER (WHERE lower(payment_method) = 'card' AND status='paid' AND transacted_date = CURRENT_DATE),0) AS card
		FROM transactions
		WHERE deleted_at IS NULL
	`).Scan(&s.TotalCash, &s.TotalNonCash, &s.TotalCard)
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

func (r ClosingRepository) Create(ctx context.Context, in CreateClosingInput) error {
	_, err := r.DB.Pool.Exec(ctx, `
		INSERT INTO closing_history (tanggal, shift, karyawan, shift_id, operator_name, total, status, catatan, fisik, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9, now(), now())
	`, in.Tanggal.Format("2006-01-02"), in.Shift, in.Karyawan, in.ShiftID, in.OperatorName, in.Total, in.Status, in.Catatan, in.Fisik)
	return err
}
