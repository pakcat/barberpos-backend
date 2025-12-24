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

-- Postgres doesn't support `ADD CONSTRAINT IF NOT EXISTS`, so guard manually.
-- +goose StatementBegin
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint c
    WHERE c.conname = 'products_owner_user_id_fkey'
  ) THEN
    ALTER TABLE products
      ADD CONSTRAINT products_owner_user_id_fkey
      FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE;
  END IF;
END $$;
-- +goose StatementEnd

CREATE INDEX IF NOT EXISTS idx_products_owner_user_id
  ON products (owner_user_id);

CREATE UNIQUE INDEX IF NOT EXISTS products_owner_user_id_name_unique
  ON products (owner_user_id, name);

-- +goose Down
DROP INDEX IF EXISTS products_owner_user_id_name_unique;
DROP INDEX IF EXISTS idx_products_owner_user_id;
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_owner_user_id_fkey;
ALTER TABLE products DROP COLUMN IF EXISTS owner_user_id;
