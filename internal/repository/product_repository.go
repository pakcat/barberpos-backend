package repository

import (
	"context"
	"errors"

	"barberpos-backend/internal/db"
	"barberpos-backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

type ProductRepository struct {
	DB *db.Postgres
}

func (r ProductRepository) List(ctx context.Context, ownerUserID int64) ([]domain.Product, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id, name, category, price, image, track_stock, stock, min_stock
		FROM products
		WHERE deleted_at IS NULL AND owner_user_id=$1
		ORDER BY id ASC
	`, ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Category, &p.Price.Amount, &p.Image, &p.TrackStock, &p.Stock, &p.MinStock); err != nil {
			return nil, err
		}
		// currency stored globally; set per config elsewhere if needed
		items = append(items, p)
	}
	return items, rows.Err()
}

func (r ProductRepository) GetByID(ctx context.Context, ownerUserID int64, id int64) (*domain.Product, error) {
	row := r.DB.Pool.QueryRow(ctx, `
		SELECT id, name, category, price, image, track_stock, stock, min_stock
		FROM products
		WHERE id=$1 AND owner_user_id=$2 AND deleted_at IS NULL
	`, id, ownerUserID)

	var p domain.Product
	if err := row.Scan(&p.ID, &p.Name, &p.Category, &p.Price.Amount, &p.Image, &p.TrackStock, &p.Stock, &p.MinStock); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r ProductRepository) Save(ctx context.Context, ownerUserID int64, p domain.Product) (*domain.Product, error) {
	if p.ID == 0 {
		err := r.DB.Pool.QueryRow(ctx, `
			INSERT INTO products (owner_user_id, name, category, price, image, track_stock, stock, min_stock, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8, now(), now())
			RETURNING id, name, category, price, image, track_stock, stock, min_stock, created_at, updated_at
		`, ownerUserID, p.Name, p.Category, p.Price.Amount, p.Image, p.TrackStock, p.Stock, p.MinStock).
			Scan(&p.ID, &p.Name, &p.Category, &p.Price.Amount, &p.Image, &p.TrackStock, &p.Stock, &p.MinStock, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
	} else {
		err := r.DB.Pool.QueryRow(ctx, `
			UPDATE products
			SET name=$1,
				category=$2,
				price=$3,
				image=$4,
				track_stock=$5,
				stock=$6,
				min_stock=$7,
				updated_at=now(),
				deleted_at=NULL
			WHERE id=$8 AND owner_user_id=$9
			RETURNING id, name, category, price, image, track_stock, stock, min_stock, created_at, updated_at
		`, p.Name, p.Category, p.Price.Amount, p.Image, p.TrackStock, p.Stock, p.MinStock, p.ID, ownerUserID).
			Scan(&p.ID, &p.Name, &p.Category, &p.Price.Amount, &p.Image, &p.TrackStock, &p.Stock, &p.MinStock, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrNotFound
			}
			return nil, err
		}
	}

	// Keep stocks table in sync for tracked products (used by /stock endpoints).
	if p.TrackStock {
		_, _ = r.DB.Pool.Exec(ctx, `
			INSERT INTO stocks (product_id, name, category, image, stock, transactions, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5, 0, now(), now())
			ON CONFLICT (product_id) DO UPDATE SET
				name=EXCLUDED.name,
				category=EXCLUDED.category,
				image=EXCLUDED.image,
				stock=EXCLUDED.stock,
				updated_at=now(),
				deleted_at=NULL
		`, p.ID, p.Name, p.Category, p.Image, p.Stock)
	} else {
		_, _ = r.DB.Pool.Exec(ctx, `
			UPDATE stocks
			SET deleted_at=now(), updated_at=now()
			WHERE product_id=$1 AND deleted_at IS NULL
		`, p.ID)
	}
	return &p, nil
}

func (r ProductRepository) Delete(ctx context.Context, ownerUserID int64, id int64) error {
	_, err := r.DB.Pool.Exec(ctx, `UPDATE products SET deleted_at = now() WHERE id=$1 AND owner_user_id=$2`, id, ownerUserID)
	return err
}

func (r ProductRepository) UpdateImage(ctx context.Context, ownerUserID int64, id int64, image string) error {
	ct, err := r.DB.Pool.Exec(ctx, `
		UPDATE products
		SET image=$1, updated_at=now()
		WHERE id=$2 AND owner_user_id=$3 AND deleted_at IS NULL
	`, image, id, ownerUserID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}

	// Keep stocks table aligned (if exists) so stock UI shows latest image.
	_, _ = r.DB.Pool.Exec(ctx, `
		UPDATE stocks
		SET image=$1, updated_at=now()
		WHERE product_id=$2 AND deleted_at IS NULL
	`, image, id)
	return nil
}
