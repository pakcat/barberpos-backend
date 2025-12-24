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

func (r ProductRepository) List(ctx context.Context) ([]domain.Product, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id, name, category, price, image, track_stock, stock, min_stock
		FROM products
		WHERE deleted_at IS NULL
		ORDER BY id ASC
	`)
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

func (r ProductRepository) GetByID(ctx context.Context, id int64) (*domain.Product, error) {
	row := r.DB.Pool.QueryRow(ctx, `
		SELECT id, name, category, price, image, track_stock, stock, min_stock
		FROM products
		WHERE id=$1 AND deleted_at IS NULL
	`, id)

	var p domain.Product
	if err := row.Scan(&p.ID, &p.Name, &p.Category, &p.Price.Amount, &p.Image, &p.TrackStock, &p.Stock, &p.MinStock); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r ProductRepository) Upsert(ctx context.Context, p domain.Product) (*domain.Product, error) {
	err := r.DB.Pool.QueryRow(ctx, `
		INSERT INTO products (id, name, category, price, image, track_stock, stock, min_stock, created_at, updated_at)
		VALUES (COALESCE($1, nextval('products_id_seq')), $2,$3,$4,$5,$6,$7,$8, now(), now())
		ON CONFLICT (id) DO UPDATE SET
			name=EXCLUDED.name,
			category=EXCLUDED.category,
			price=EXCLUDED.price,
			image=EXCLUDED.image,
			track_stock=EXCLUDED.track_stock,
			stock=EXCLUDED.stock,
			min_stock=EXCLUDED.min_stock,
			updated_at=now(),
			deleted_at=NULL
		RETURNING id, name, category, price, image, track_stock, stock, min_stock, created_at, updated_at
	`, nullableID(p.ID), p.Name, p.Category, p.Price.Amount, p.Image, p.TrackStock, p.Stock, p.MinStock).
		Scan(&p.ID, &p.Name, &p.Category, &p.Price.Amount, &p.Image, &p.TrackStock, &p.Stock, &p.MinStock, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
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

func (r ProductRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.DB.Pool.Exec(ctx, `UPDATE products SET deleted_at = now() WHERE id=$1`, id)
	return err
}
