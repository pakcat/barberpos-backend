package repository

import "context"

func (r CategoryRepository) SeedDefaults(ctx context.Context, ownerUserID int64) error {
	defaults := []string{"Layanan", "Produk"}
	for _, name := range defaults {
		_, err := r.DB.Pool.Exec(ctx, `
			INSERT INTO categories (owner_user_id, name, created_at, updated_at)
			VALUES ($1, $2, now(), now())
			ON CONFLICT (owner_user_id, name) DO NOTHING
		`, ownerUserID, name)
		if err != nil {
			return err
		}
	}
	return nil
}
