package repository

import "context"

func (r CategoryRepository) SeedDefaults(ctx context.Context) error {
	defaults := []string{"Layanan", "Produk"}
	for _, name := range defaults {
		_, err := r.DB.Pool.Exec(ctx, `
			INSERT INTO categories (name, created_at, updated_at)
			VALUES ($1, now(), now())
			ON CONFLICT (name) DO NOTHING
		`, name)
		if err != nil {
			return err
		}
	}
	return nil
}
