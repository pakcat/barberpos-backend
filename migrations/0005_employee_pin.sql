-- +goose Up
ALTER TABLE employees
    ADD COLUMN IF NOT EXISTS pin_hash TEXT;

-- +goose Down
ALTER TABLE employees
    DROP COLUMN IF EXISTS pin_hash;
