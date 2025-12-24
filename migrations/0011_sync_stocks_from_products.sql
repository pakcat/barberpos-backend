-- +goose Up
-- Ensure 1 stock row per product_id and sync from products.track_stock.

-- Remove duplicates (keep smallest id) before adding unique constraint.
-- +goose StatementBegin
WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY product_id ORDER BY id ASC) AS rn
    FROM stocks
    WHERE product_id IS NOT NULL
)
DELETE FROM stocks
WHERE id IN (SELECT id FROM ranked WHERE rn > 1);
-- +goose StatementEnd

-- Goose splits statements on semicolons unless wrapped in StatementBegin/End.
-- Use a unique index with IF NOT EXISTS (works well for Postgres and is idempotent).
CREATE UNIQUE INDEX IF NOT EXISTS stocks_product_id_unique
ON stocks (product_id)
WHERE product_id IS NOT NULL;

-- Create missing stock rows for products that track stock.
INSERT INTO stocks (product_id, name, category, image, stock, transactions, created_at, updated_at)
SELECT p.id, p.name, p.category, p.image, p.stock, 0, now(), now()
FROM products p
WHERE p.deleted_at IS NULL AND p.track_stock = TRUE
  AND NOT EXISTS (
      SELECT 1 FROM stocks s WHERE s.product_id = p.id AND s.deleted_at IS NULL
  );

-- Keep stocks in sync with product fields and soft-delete when a product stops tracking stock.
UPDATE stocks s
SET name = p.name,
    category = p.category,
    image = p.image,
    stock = p.stock,
    updated_at = now(),
    deleted_at = NULL
FROM products p
WHERE s.product_id = p.id
  AND p.deleted_at IS NULL
  AND p.track_stock = TRUE;

UPDATE stocks s
SET deleted_at = now(),
    updated_at = now()
FROM products p
WHERE s.product_id = p.id
  AND (p.deleted_at IS NOT NULL OR p.track_stock = FALSE)
  AND s.deleted_at IS NULL;

-- +goose Down
-- Keep constraint; do not drop in down migration.
