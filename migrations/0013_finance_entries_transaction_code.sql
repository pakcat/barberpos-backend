-- +goose Up
ALTER TABLE finance_entries
ADD COLUMN IF NOT EXISTS transaction_code TEXT;

CREATE INDEX IF NOT EXISTS idx_finance_entries_transaction_code
ON finance_entries (transaction_code);

-- +goose Down
DROP INDEX IF EXISTS idx_finance_entries_transaction_code;
ALTER TABLE finance_entries
DROP COLUMN IF EXISTS transaction_code;
