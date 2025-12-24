package repository

import (
	"context"

	"barberpos-backend/internal/db"
	"barberpos-backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

type CustomerRepository struct {
	DB *db.Postgres
}

func (r CustomerRepository) List(ctx context.Context, ownerUserID int64, limit int) ([]domain.Customer, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id, name, phone, email, address, created_at, updated_at
		FROM customers
		WHERE deleted_at IS NULL AND owner_user_id=$1
		ORDER BY name ASC
		LIMIT $2
	`, ownerUserID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Customer
	for rows.Next() {
		var c domain.Customer
		if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &c.Email, &c.Address, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func (r CustomerRepository) Upsert(ctx context.Context, ownerUserID int64, c domain.Customer) (*domain.Customer, error) {
	var out domain.Customer
	err := r.DB.Pool.QueryRow(ctx, `
		INSERT INTO customers (id, owner_user_id, name, phone, email, address, created_at, updated_at)
		VALUES (COALESCE($1, nextval('customers_id_seq')), $2, $3, $4, $5, $6, now(), now())
		ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, phone=EXCLUDED.phone, email=EXCLUDED.email, address=EXCLUDED.address, updated_at=now(), deleted_at=NULL
		RETURNING id, name, phone, email, address, created_at, updated_at
	`, nullableID(c.ID), ownerUserID, c.Name, c.Phone, c.Email, c.Address).Scan(&out.ID, &out.Name, &out.Phone, &out.Email, &out.Address, &out.CreatedAt, &out.UpdatedAt)
	return &out, err
}

func (r CustomerRepository) Delete(ctx context.Context, ownerUserID int64, id int64) error {
	_, err := r.DB.Pool.Exec(ctx, `UPDATE customers SET deleted_at = now() WHERE id=$1 AND owner_user_id=$2`, id, ownerUserID)
	return err
}

func (r CustomerRepository) Get(ctx context.Context, ownerUserID int64, id int64) (*domain.Customer, error) {
	row := r.DB.Pool.QueryRow(ctx, `
		SELECT id, name, phone, email, address, created_at, updated_at
		FROM customers
		WHERE id=$1 AND owner_user_id=$2 AND deleted_at IS NULL
	`, id, ownerUserID)
	var c domain.Customer
	if err := row.Scan(&c.ID, &c.Name, &c.Phone, &c.Email, &c.Address, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func nullableID(id int64) *int64 {
	if id == 0 {
		return nil
	}
	return &id
}
