package repository

import (
	"context"

	"barberpos-backend/internal/db"
	"barberpos-backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

type CategoryRepository struct {
	DB *db.Postgres
}

func (r CategoryRepository) List(ctx context.Context, ownerUserID int64) ([]domain.Category, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id, name, created_at, updated_at
		FROM categories
		WHERE deleted_at IS NULL AND owner_user_id=$1
		ORDER BY name ASC
	`, ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Category
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func (r CategoryRepository) Upsert(ctx context.Context, ownerUserID int64, name string, id *int64) (*domain.Category, error) {
	var out domain.Category
	err := r.DB.Pool.QueryRow(ctx, `
		INSERT INTO categories (id, owner_user_id, name, created_at, updated_at)
		VALUES (COALESCE($1, nextval('categories_id_seq')), $2, $3, now(), now())
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, updated_at = now(), deleted_at = NULL
		RETURNING id, name, created_at, updated_at
	`, id, ownerUserID, name).Scan(&out.ID, &out.Name, &out.CreatedAt, &out.UpdatedAt)
	return &out, err
}

func (r CategoryRepository) Delete(ctx context.Context, ownerUserID int64, id int64) error {
	_, err := r.DB.Pool.Exec(ctx, `UPDATE categories SET deleted_at = now() WHERE id=$1 AND owner_user_id=$2`, id, ownerUserID)
	return err
}

func (r CategoryRepository) Get(ctx context.Context, ownerUserID int64, id int64) (*domain.Category, error) {
	row := r.DB.Pool.QueryRow(ctx, `
		SELECT id, name, created_at, updated_at
		FROM categories
		WHERE id=$1 AND owner_user_id=$2 AND deleted_at IS NULL
	`, id, ownerUserID)
	var c domain.Category
	if err := row.Scan(&c.ID, &c.Name, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}
