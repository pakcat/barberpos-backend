package repository

import (
	"context"
	"time"

	"barberpos-backend/internal/db"
)

type FCMRepository struct {
	DB *db.Postgres
}

type RegisterTokenInput struct {
	UserID   *int64
	Token    string
	Platform string
}

func (r FCMRepository) Register(ctx context.Context, in RegisterTokenInput) error {
	_, err := r.DB.Pool.Exec(ctx, `
		INSERT INTO fcm_tokens (user_id, token, platform, created_at)
		VALUES ($1,$2,$3, now())
		ON CONFLICT (token) DO UPDATE SET user_id=EXCLUDED.user_id, platform=EXCLUDED.platform
	`, in.UserID, in.Token, in.Platform)
	return err
}

func (r FCMRepository) LastUpdated(ctx context.Context, token string) (time.Time, error) {
	var ts time.Time
	err := r.DB.Pool.QueryRow(ctx, `SELECT created_at FROM fcm_tokens WHERE token=$1`, token).Scan(&ts)
	return ts, err
}
