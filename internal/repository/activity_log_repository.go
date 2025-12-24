package repository

import (
	"context"
	"time"

	"barberpos-backend/internal/db"
	"barberpos-backend/internal/domain"
)

type ActivityLogRepository struct {
	DB *db.Postgres
}

type CreateActivityLogInput struct {
	Title     string
	Message   string
	Actor     string
	Type      domain.ActivityLogType
	Timestamp time.Time
}

func (r ActivityLogRepository) Create(ctx context.Context, in CreateActivityLogInput) (int64, error) {
	var id int64
	err := r.DB.Pool.QueryRow(ctx, `
		INSERT INTO activity_logs (title, message, actor, type, logged_at, synced, created_at)
		VALUES ($1,$2,$3,$4,$5,true, now())
		RETURNING id
	`, in.Title, in.Message, in.Actor, string(in.Type), in.Timestamp).Scan(&id)
	return id, err
}

func (r ActivityLogRepository) List(ctx context.Context, limit int) ([]domain.ActivityLog, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id, title, message, actor, type, logged_at, synced
		FROM activity_logs
		WHERE deleted_at IS NULL
		ORDER BY logged_at DESC, id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.ActivityLog
	for rows.Next() {
		var l domain.ActivityLog
		var typ string
		if err := rows.Scan(&l.ID, &l.Title, &l.Message, &l.Actor, &typ, &l.LoggedAt, &l.Synced); err != nil {
			return nil, err
		}
		l.Type = domain.ActivityLogType(typ)
		out = append(out, l)
	}
	return out, rows.Err()
}
