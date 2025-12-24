-- +goose Up
ALTER TABLE products
  ADD COLUMN IF NOT EXISTS owner_user_id BIGINT;

UPDATE products
SET owner_user_id = (
  SELECT id FROM users ORDER BY id ASC LIMIT 1
)
WHERE owner_user_id IS NULL;

ALTER TABLE products
  ALTER COLUMN owner_user_id SET NOT NULL;

ALTER TABLE products
  ADD CONSTRAINT IF NOT EXISTS products_owner_user_id_fkey
  FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_products_owner_user_id
  ON products (owner_user_id);

CREATE UNIQUE INDEX IF NOT EXISTS products_owner_user_id_name_unique
  ON products (owner_user_id, name);

-- +goose Down
DROP INDEX IF EXISTS products_owner_user_id_name_unique;
DROP INDEX IF EXISTS idx_products_owner_user_id;
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_owner_user_id_fkey;
ALTER TABLE products DROP COLUMN IF EXISTS owner_user_id;
