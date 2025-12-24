-- +goose Up
ALTER TABLE transactions
ADD COLUMN IF NOT EXISTS client_ref TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_client_ref_unique
ON transactions (client_ref)
WHERE client_ref IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_transactions_client_ref_unique;
ALTER TABLE transactions
DROP COLUMN IF EXISTS client_ref;

