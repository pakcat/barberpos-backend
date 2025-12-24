package repository

import (
	"context"
	"time"

	"barberpos-backend/internal/db"
	"barberpos-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type MembershipRepository struct {
	DB *db.Postgres
}

// pgxQuerier is satisfied by both pgxpool.Pool and pgx.Tx.
type pgxQuerier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type SaveMembershipStateParams struct {
	OwnerUserID     int64
	UsedQuota       int
	FreeUsed        int
	FreePeriodStart time.Time
	TopupBalance    int
}

func (r MembershipRepository) GetState(ctx context.Context, ownerUserID int64) (*domain.MembershipState, error) {
	return r.getStateWith(ctx, r.DB.Pool, ownerUserID)
}

func (r MembershipRepository) SaveState(ctx context.Context, p SaveMembershipStateParams) (*domain.MembershipState, error) {
	return r.saveStateWith(ctx, r.DB.Pool, p)
}

func (r MembershipRepository) SumTopups(ctx context.Context, ownerUserID int64) (int64, error) {
	return r.sumTopupsWith(ctx, r.DB.Pool, ownerUserID)
}

func (r MembershipRepository) IncrementTopupBalance(ctx context.Context, ownerUserID int64, delta int64) (*domain.MembershipState, error) {
	return r.incrementTopupBalanceWith(ctx, r.DB.Pool, ownerUserID, delta)
}

type CreateTopupInput struct {
	OwnerUserID int64
	Amount      int64
	Manager     string
	Note        string
	Date        time.Time
}

func (r MembershipRepository) CreateTopup(ctx context.Context, in CreateTopupInput) (*domain.MembershipTopup, error) {
	tx, err := r.DB.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var t domain.MembershipTopup
	row := tx.QueryRow(ctx, `
		INSERT INTO membership_topups (owner_user_id, amount, manager, note, topup_date, created_at)
		VALUES ($1,$2,$3,$4,$5, now())
		RETURNING id, owner_user_id, amount, manager, note, topup_date, created_at
	`, in.OwnerUserID, in.Amount, in.Manager, in.Note, in.Date)
	var ownerID pgtype.Int8
	if err := row.Scan(&t.ID, &ownerID, &t.Amount.Amount, &t.Manager, &t.Note, &t.Date, &t.CreatedAt); err != nil {
		return nil, err
	}
	if ownerID.Valid {
		t.OwnerID = &ownerID.Int64
	}

	if _, err := r.incrementTopupBalanceWith(ctx, tx, in.OwnerUserID, in.Amount); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r MembershipRepository) ListTopups(ctx context.Context, ownerUserID int64, limit int) ([]domain.MembershipTopup, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id, owner_user_id, amount, manager, note, topup_date, created_at
		FROM membership_topups
		WHERE deleted_at IS NULL AND owner_user_id = $1
		ORDER BY topup_date DESC, id DESC
		LIMIT $2
	`, ownerUserID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.MembershipTopup
	for rows.Next() {
		var t domain.MembershipTopup
		var ownerID pgtype.Int8
		if err := rows.Scan(&t.ID, &ownerID, &t.Amount.Amount, &t.Manager, &t.Note, &t.Date, &t.CreatedAt); err != nil {
			return nil, err
		}
		if ownerID.Valid {
			t.OwnerID = &ownerID.Int64
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

// Transaction-scoped helpers
func (r MembershipRepository) GetStateWithTx(ctx context.Context, tx pgx.Tx, ownerUserID int64) (*domain.MembershipState, error) {
	return r.getStateWith(ctx, tx, ownerUserID)
}

func (r MembershipRepository) SaveStateWithTx(ctx context.Context, tx pgx.Tx, p SaveMembershipStateParams) (*domain.MembershipState, error) {
	return r.saveStateWith(ctx, tx, p)
}

func (r MembershipRepository) SumTopupsWithTx(ctx context.Context, tx pgx.Tx, ownerUserID int64) (int64, error) {
	return r.sumTopupsWith(ctx, tx, ownerUserID)
}

func (r MembershipRepository) IncrementTopupBalanceWithTx(ctx context.Context, tx pgx.Tx, ownerUserID int64, delta int64) (*domain.MembershipState, error) {
	return r.incrementTopupBalanceWith(ctx, tx, ownerUserID, delta)
}

func (r MembershipRepository) getStateWith(ctx context.Context, q pgxQuerier, ownerUserID int64) (*domain.MembershipState, error) {
	var state domain.MembershipState
	row := q.QueryRow(ctx, `
		SELECT owner_user_id, used_quota, free_used, free_period_start, topup_balance, updated_at
		FROM membership_state
		WHERE owner_user_id = $1
	`, ownerUserID)
	var ownerID pgtype.Int8
	if err := row.Scan(&ownerID, &state.UsedQuota, &state.FreeUsed, &state.FreeStart, &state.TopupBal, &state.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			now := time.Now()
			start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
			return r.saveStateWith(ctx, q, SaveMembershipStateParams{
				OwnerUserID:     ownerUserID,
				UsedQuota:       0,
				FreeUsed:        0,
				FreePeriodStart: start,
				TopupBalance:    0,
			})
		}
		return nil, err
	}
	if ownerID.Valid {
		state.OwnerID = &ownerID.Int64
	}
	return &state, nil
}

func (r MembershipRepository) saveStateWith(ctx context.Context, q pgxQuerier, p SaveMembershipStateParams) (*domain.MembershipState, error) {
	var state domain.MembershipState
	row := q.QueryRow(ctx, `
		INSERT INTO membership_state (owner_user_id, used_quota, free_used, free_period_start, topup_balance, updated_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (owner_user_id) DO UPDATE SET
			used_quota=EXCLUDED.used_quota,
			free_used=EXCLUDED.free_used,
			free_period_start=EXCLUDED.free_period_start,
			topup_balance=EXCLUDED.topup_balance,
			updated_at=now()
		RETURNING owner_user_id, used_quota, free_used, free_period_start, topup_balance, updated_at
	`, p.OwnerUserID, p.UsedQuota, p.FreeUsed, p.FreePeriodStart, p.TopupBalance)
	var ownerID pgtype.Int8
	err := row.Scan(&ownerID, &state.UsedQuota, &state.FreeUsed, &state.FreeStart, &state.TopupBal, &state.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if ownerID.Valid {
		state.OwnerID = &ownerID.Int64
	}
	return &state, nil
}

func (r MembershipRepository) sumTopupsWith(ctx context.Context, q pgxQuerier, ownerUserID int64) (int64, error) {
	var total int64
	if err := q.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM membership_topups
		WHERE deleted_at IS NULL AND owner_user_id = $1
	`, ownerUserID).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r MembershipRepository) incrementTopupBalanceWith(ctx context.Context, q pgxQuerier, ownerUserID int64, delta int64) (*domain.MembershipState, error) {
	var state domain.MembershipState
	row := q.QueryRow(ctx, `
		INSERT INTO membership_state (owner_user_id, topup_balance, used_quota, free_used, free_period_start, updated_at)
		VALUES ($1, $2, 0, 0, date_trunc('month', now())::date, now())
		ON CONFLICT (owner_user_id) DO UPDATE SET
			topup_balance = membership_state.topup_balance + EXCLUDED.topup_balance,
			updated_at = now()
		RETURNING owner_user_id, used_quota, free_used, free_period_start, topup_balance, updated_at
	`, ownerUserID, delta)
	var ownerID pgtype.Int8
	err := row.Scan(&ownerID, &state.UsedQuota, &state.FreeUsed, &state.FreeStart, &state.TopupBal, &state.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if ownerID.Valid {
		state.OwnerID = &ownerID.Int64
	}
	return &state, nil
}
