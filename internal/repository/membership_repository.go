package repository

import (
	"context"
	"time"

	"barberpos-backend/internal/db"
	"barberpos-backend/internal/domain"
)

type MembershipRepository struct {
	DB *db.Postgres
}

func (r MembershipRepository) GetState(ctx context.Context) (*domain.MembershipState, error) {
	row := r.DB.Pool.QueryRow(ctx, `
		SELECT used_quota, updated_at
		FROM membership_state
		WHERE id=1
	`)
	var state domain.MembershipState
	if err := row.Scan(&state.UsedQuota, &state.UpdatedAt); err != nil {
		return nil, err
	}
	return &state, nil
}

func (r MembershipRepository) SaveState(ctx context.Context, usedQuota int) (*domain.MembershipState, error) {
	var state domain.MembershipState
	err := r.DB.Pool.QueryRow(ctx, `
		INSERT INTO membership_state (id, used_quota, updated_at)
		VALUES (1, $1, now())
		ON CONFLICT (id) DO UPDATE SET used_quota=EXCLUDED.used_quota, updated_at=now()
		RETURNING used_quota, updated_at
	`, usedQuota).Scan(&state.UsedQuota, &state.UpdatedAt)
	return &state, err
}

type CreateTopupInput struct {
	Amount  int64
	Manager string
	Note    string
	Date    time.Time
}

func (r MembershipRepository) CreateTopup(ctx context.Context, in CreateTopupInput) (*domain.MembershipTopup, error) {
	var t domain.MembershipTopup
	err := r.DB.Pool.QueryRow(ctx, `
		INSERT INTO membership_topups (amount, manager, note, topup_date, created_at)
		VALUES ($1,$2,$3,$4, now())
		RETURNING id, amount, manager, note, topup_date, created_at
	`, in.Amount, in.Manager, in.Note, in.Date).Scan(&t.ID, &t.Amount.Amount, &t.Manager, &t.Note, &t.Date, &t.CreatedAt)
	return &t, err
}

func (r MembershipRepository) ListTopups(ctx context.Context, limit int) ([]domain.MembershipTopup, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id, amount, manager, note, topup_date, created_at
		FROM membership_topups
		WHERE deleted_at IS NULL
		ORDER BY topup_date DESC, id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.MembershipTopup
	for rows.Next() {
		var t domain.MembershipTopup
		if err := rows.Scan(&t.ID, &t.Amount.Amount, &t.Manager, &t.Note, &t.Date, &t.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}
