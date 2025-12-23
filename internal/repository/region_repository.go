package repository

import (
	"context"

	"barberpos-backend/internal/db"
	"barberpos-backend/internal/domain"
)

type RegionRepository struct {
	DB *db.Postgres
}

// List returns all active regions ordered alphabetically.
func (r RegionRepository) List(ctx context.Context) ([]domain.Region, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id, name, created_at, updated_at
		FROM regions
		WHERE deleted_at IS NULL
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Region
	for rows.Next() {
		var region domain.Region
		if err := rows.Scan(&region.ID, &region.Name, &region.CreatedAt, &region.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, region)
	}
	return items, rows.Err()
}
