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
		// Idempotent: products.name is unique.
		_, err := r.DB.Pool.Exec(ctx, `
			INSERT INTO products (name, category, price, image, track_stock, stock, min_stock, created_at, updated_at)
			VALUES ($1,$2,$3,'',$4,$5,$6, now(), now())
			ON CONFLICT (name) DO NOTHING
		`, p.Name, p.Category, p.Price.Amount, p.TrackStock, p.Stock, p.MinStock)
		if err != nil {
			return err
		}
	}
	return nil
}
