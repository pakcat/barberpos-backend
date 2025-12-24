-- +goose Up
-- Seed default products/services so a fresh install isn't empty.
-- Note: products.name is UNIQUE, so this is idempotent.
INSERT INTO products (name, category, price, image, track_stock, stock, min_stock, created_at, updated_at)
VALUES
  ('Potong Rambut', 'Layanan', 30000, '', false, 0, 0, now(), now()),
  ('Cukur Jenggot', 'Layanan', 20000, '', false, 0, 0, now(), now()),
  ('Cuci + Pijat', 'Layanan', 15000, '', false, 0, 0, now(), now()),
  ('Creambath', 'Layanan', 50000, '', false, 0, 0, now(), now()),
  ('Pomade', 'Produk', 50000, '', true, 10, 2, now(), now()),
  ('Shampoo', 'Produk', 35000, '', true, 20, 5, now(), now())
ON CONFLICT (name) DO NOTHING;

-- +goose Down
-- No down migration for seed
