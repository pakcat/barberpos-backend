-- +goose Up
ALTER TABLE activity_logs
  ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- +goose Down
ALTER TABLE activity_logs
  DROP COLUMN IF EXISTS created_at;
