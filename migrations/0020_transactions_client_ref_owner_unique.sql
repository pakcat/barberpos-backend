-- +goose Up
-- client_ref must be unique per owner_user_id (multi-tenant), not globally.
DROP INDEX IF EXISTS idx_transactions_client_ref_unique;

CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_owner_client_ref_unique
ON transactions (owner_user_id, client_ref)
WHERE client_ref IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_transactions_owner_client_ref_unique;

CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_client_ref_unique
ON transactions (client_ref)
WHERE client_ref IS NOT NULL;

