-- +goose Up
-- Backfill transaction_items.product_id for old transactions created before mobile sent productId.
-- Best-effort match by product name (unique) and optional category.

UPDATE transaction_items ti
SET product_id = p.id
FROM products p
WHERE ti.deleted_at IS NULL
  AND ti.product_id IS NULL
  AND p.deleted_at IS NULL
  AND lower(ti.name) = lower(p.name)
  AND (
      ti.category IS NULL OR ti.category = '' OR
      lower(ti.category) = lower(p.category)
  );

-- +goose Down
-- No down migration (data backfill).

