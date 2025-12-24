package repository

import "context"

func (r RegionRepository) SeedDefaults(ctx context.Context) error {
	// 38 provinces (as of current Indonesian administrative divisions).
	provinces := []string{
		"Aceh",
		"Sumatera Utara",
		"Sumatera Barat",
		"Riau",
		"Kepulauan Riau",
		"Jambi",
		"Sumatera Selatan",
		"Bangka Belitung",
		"Bengkulu",
		"Lampung",
		"DKI Jakarta",
		"Jawa Barat",
		"Banten",
		"Jawa Tengah",
		"DI Yogyakarta",
		"Jawa Timur",
		"Bali",
		"Nusa Tenggara Barat",
		"Nusa Tenggara Timur",
		"Kalimantan Barat",
		"Kalimantan Tengah",
		"Kalimantan Selatan",
		"Kalimantan Timur",
		"Kalimantan Utara",
		"Sulawesi Utara",
		"Gorontalo",
		"Sulawesi Tengah",
		"Sulawesi Barat",
		"Sulawesi Selatan",
		"Sulawesi Tenggara",
		"Maluku",
		"Maluku Utara",
		"Papua",
		"Papua Barat",
		"Papua Barat Daya",
		"Papua Selatan",
		"Papua Tengah",
		"Papua Pegunungan",
	}

	for _, name := range provinces {
		_, err := r.DB.Pool.Exec(ctx, `
			INSERT INTO regions (name, created_at, updated_at)
			VALUES ($1, now(), now())
			ON CONFLICT (name) DO NOTHING
		`, name)
		if err != nil {
			return err
		}
	}
	return nil
}
