package repository

import (
	"context"

	"barberpos-backend/internal/domain"
)

func (r ProductRepository) SeedDefaults(ctx context.Context) error {
	defaults := []domain.Product{
		{Name: "Potong Rambut", Category: "Layanan", Price: domain.Money{Amount: 30000}, TrackStock: false, Stock: 0, MinStock: 0},
		{Name: "Cukur Jenggot", Category: "Layanan", Price: domain.Money{Amount: 20000}, TrackStock: false, Stock: 0, MinStock: 0},
		{Name: "Cuci + Pijat", Category: "Layanan", Price: domain.Money{Amount: 15000}, TrackStock: false, Stock: 0, MinStock: 0},
		{Name: "Creambath", Category: "Layanan", Price: domain.Money{Amount: 50000}, TrackStock: false, Stock: 0, MinStock: 0},
		{Name: "Pomade", Category: "Produk", Price: domain.Money{Amount: 50000}, TrackStock: true, Stock: 10, MinStock: 2},
		{Name: "Shampoo", Category: "Produk", Price: domain.Money{Amount: 35000}, TrackStock: true, Stock: 20, MinStock: 5},
	}

	for _, p := range defaults {
		var (
			productID   int64
			name        string
			category    string
			image       string
			trackStock  bool
			stock       int
		)

		// Idempotent: products.name is unique. Also restores soft-deleted defaults.
		err := r.DB.Pool.QueryRow(ctx, `
			INSERT INTO products (name, category, price, image, track_stock, stock, min_stock, created_at, updated_at)
			VALUES ($1,$2,$3,'',$4,$5,$6, now(), now())
			ON CONFLICT (name) DO UPDATE SET
				category=EXCLUDED.category,
				price=EXCLUDED.price,
				image=EXCLUDED.image,
				track_stock=EXCLUDED.track_stock,
				stock=EXCLUDED.stock,
				min_stock=EXCLUDED.min_stock,
				updated_at=now(),
				deleted_at=NULL
			RETURNING id, name, category, image, track_stock, stock
		`, p.Name, p.Category, p.Price.Amount, p.TrackStock, p.Stock, p.MinStock).Scan(
			&productID, &name, &category, &image, &trackStock, &stock,
		)
		if err != nil {
			return err
		}

		// Keep stocks table in sync for tracked defaults.
		if trackStock {
			// Insert if missing, then update to desired values (works even without a unique index).
			_, _ = r.DB.Pool.Exec(ctx, `
				INSERT INTO stocks (product_id, name, category, image, stock, transactions, created_at, updated_at)
				SELECT $1,$2,$3,$4,$5,0, now(), now()
				WHERE NOT EXISTS (
					SELECT 1 FROM stocks s WHERE s.product_id=$1 AND s.deleted_at IS NULL
				)
			`, productID, name, category, image, stock)
			_, _ = r.DB.Pool.Exec(ctx, `
				UPDATE stocks
				SET name=$2, category=$3, image=$4, stock=$5, updated_at=now(), deleted_at=NULL
				WHERE product_id=$1
			`, productID, name, category, image, stock)
		} else {
			_, _ = r.DB.Pool.Exec(ctx, `
				UPDATE stocks
				SET deleted_at=now(), updated_at=now()
				WHERE product_id=$1 AND deleted_at IS NULL
			`, productID)
		}
	}
	return nil
}
