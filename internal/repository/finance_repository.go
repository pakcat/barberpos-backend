package repository

import (
	"context"
	"fmt"
	"time"

	"barberpos-backend/internal/db"
	"barberpos-backend/internal/domain"
)

type FinanceRepository struct {
	DB *db.Postgres
}

type CreateFinanceInput struct {
	Title    string
	Amount   int64
	Category string
	Date     time.Time
	Type     domain.FinanceEntryType
	Note     string
	Staff    *string
	Service  *string
}

func (r FinanceRepository) Create(ctx context.Context, in CreateFinanceInput) (*domain.FinanceEntry, error) {
	var fe domain.FinanceEntry
	err := r.DB.Pool.QueryRow(ctx, `
		INSERT INTO finance_entries (title, amount, category, entry_date, type, note, staff, service, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8, now())
		RETURNING id, title, amount, category, entry_date, type, note, staff, service, created_at
	`, in.Title, in.Amount, in.Category, in.Date.Format("2006-01-02"), string(in.Type), in.Note, in.Staff, in.Service).Scan(
		&fe.ID, &fe.Title, &fe.Amount.Amount, &fe.Category, &fe.Date, (*string)(&fe.Type), &fe.Note, &fe.Staff, &fe.Service, &fe.CreatedAt,
	)
	return &fe, err
}

func (r FinanceRepository) List(ctx context.Context, limit int) ([]domain.FinanceEntry, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id, title, amount, category, entry_date, type, note, staff, service, created_at
		FROM finance_entries
		WHERE deleted_at IS NULL
		ORDER BY entry_date DESC, id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.FinanceEntry
	for rows.Next() {
		var fe domain.FinanceEntry
		var t string
		if err := rows.Scan(&fe.ID, &fe.Title, &fe.Amount.Amount, &fe.Category, &fe.Date, &t, &fe.Note, &fe.Staff, &fe.Service, &fe.CreatedAt); err != nil {
			return nil, err
		}
		fe.Type = domain.FinanceEntryType(t)
		items = append(items, fe)
	}
	return items, rows.Err()
}

func (r FinanceRepository) ListFiltered(ctx context.Context, startDate, endDate *time.Time) ([]domain.FinanceEntry, error) {
	query := `
		SELECT id, title, amount, category, entry_date, type, note, staff, service, created_at
		FROM finance_entries
		WHERE deleted_at IS NULL
	`
	args := make([]any, 0, 2)
	if startDate != nil {
		query += fmt.Sprintf(" AND entry_date >= $%d", len(args)+1)
		args = append(args, startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		query += fmt.Sprintf(" AND entry_date <= $%d", len(args)+1)
		args = append(args, endDate.Format("2006-01-02"))
	}
	query += " ORDER BY entry_date DESC, id DESC"

	rows, err := r.DB.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.FinanceEntry
	for rows.Next() {
		var fe domain.FinanceEntry
		var t string
		if err := rows.Scan(&fe.ID, &fe.Title, &fe.Amount.Amount, &fe.Category, &fe.Date, &t, &fe.Note, &fe.Staff, &fe.Service, &fe.CreatedAt); err != nil {
			return nil, err
		}
		fe.Type = domain.FinanceEntryType(t)
		items = append(items, fe)
	}
	return items, rows.Err()
}
