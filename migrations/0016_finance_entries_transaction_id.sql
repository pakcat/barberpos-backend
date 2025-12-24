-- +goose Up
ALTER TABLE finance_entries
ADD COLUMN IF NOT EXISTS transaction_id BIGINT REFERENCES transactions(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_finance_entries_transaction_id
ON finance_entries (transaction_id);

UPDATE finance_entries fe
SET transaction_id = t.id
FROM transactions t
WHERE fe.transaction_id IS NULL
  AND fe.transaction_code IS NOT NULL
  AND t.code = fe.transaction_code;

-- +goose Down
DROP INDEX IF EXISTS idx_finance_entries_transaction_id;

ALTER TABLE finance_entries
DROP COLUMN IF EXISTS transaction_id;
