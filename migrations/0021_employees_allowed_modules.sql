-- +goose Up
ALTER TABLE employees
  ADD COLUMN IF NOT EXISTS allowed_modules TEXT[] NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE employees
  DROP COLUMN IF EXISTS allowed_modules;

