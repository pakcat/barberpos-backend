-- +goose Up
ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS refunded_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS refunded_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS refund_note TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_transactions_refunded_at ON transactions (refunded_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_transactions_refunded_at;

ALTER TABLE transactions
    DROP COLUMN IF EXISTS refund_note,
    DROP COLUMN IF EXISTS refunded_by,
    DROP COLUMN IF EXISTS refunded_at;

