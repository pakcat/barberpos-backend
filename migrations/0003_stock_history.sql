-- +goose Up
CREATE TABLE IF NOT EXISTS stock_history (
    id BIGSERIAL PRIMARY KEY,
    stock_id BIGINT REFERENCES stocks(id) ON DELETE CASCADE,
    product_id BIGINT REFERENCES products(id) ON DELETE SET NULL,
    change INTEGER NOT NULL,
    remaining INTEGER NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT 'adjust',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS stock_history;
