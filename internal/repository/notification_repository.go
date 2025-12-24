package repository

import (
	"context"
	"time"

	"barberpos-backend/internal/db"
	"barberpos-backend/internal/domain"
	"github.com/jackc/pgx/v5/pgtype"
)

type NotificationRepository struct {
	DB *db.Postgres
}

type CreateNotificationInput struct {
	UserID  int64
	Title   string
	Message string
	Type    domain.NotificationType
	Created time.Time
}

func (r NotificationRepository) Create(ctx context.Context, in CreateNotificationInput) (*domain.Notification, error) {
	var n domain.Notification
	var userID pgtype.Int8
	createdAt := in.Created
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	err := r.DB.Pool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, title, message, type, created_at)
		VALUES ($1,$2,$3,$4, $5)
		RETURNING id, user_id, title, message, type, created_at, read_at
	`, in.UserID, in.Title, in.Message, string(in.Type), createdAt).Scan(
		&n.ID, &userID, &n.Title, &n.Message, (*string)(&n.Type), &n.CreatedAt, &n.ReadAt,
	)
	if err != nil {
		return nil, err
	}
	if userID.Valid {
		n.UserID = &userID.Int64
	}
	return &n, nil
}

func (r NotificationRepository) List(ctx context.Context, userID int64, limit int) ([]domain.Notification, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id, user_id, title, message, type, created_at, read_at
		FROM notifications
		WHERE deleted_at IS NULL AND user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Notification
	for rows.Next() {
		var n domain.Notification
		var uid pgtype.Int8
		if err := rows.Scan(&n.ID, &uid, &n.Title, &n.Message, (*string)(&n.Type), &n.CreatedAt, &n.ReadAt); err != nil {
			return nil, err
		}
		if uid.Valid {
			n.UserID = &uid.Int64
		}
		items = append(items, n)
	}
	return items, rows.Err()
}
