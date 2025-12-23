-- +goose Up
ALTER TABLE membership_state
    ADD COLUMN IF NOT EXISTS free_used INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS free_period_start DATE NOT NULL DEFAULT date_trunc('month', now())::date,
    ADD COLUMN IF NOT EXISTS topup_balance INTEGER NOT NULL DEFAULT 0;

-- Attempt to reconcile existing data into new columns
DO $$
DECLARE
    total_topups INTEGER := (SELECT COALESCE(SUM(amount), 0) FROM membership_topups WHERE deleted_at IS NULL);
    prev_used INTEGER := (SELECT COALESCE(used_quota, 0) FROM membership_state WHERE id = 1);
    free_cap INTEGER := 1000;
    new_free_used INTEGER;
    topup_used INTEGER;
    new_topup_balance INTEGER;
BEGIN
    new_free_used := LEAST(prev_used, free_cap);
    topup_used := GREATEST(prev_used - free_cap, 0);
    new_topup_balance := GREATEST(total_topups - topup_used, 0);

    INSERT INTO membership_state (id, used_quota, free_used, free_period_start, topup_balance, updated_at)
    VALUES (1, new_free_used + topup_used, new_free_used, date_trunc('month', now())::date, new_topup_balance, now())
    ON CONFLICT (id) DO UPDATE
        SET used_quota = EXCLUDED.used_quota,
            free_used = EXCLUDED.free_used,
            free_period_start = EXCLUDED.free_period_start,
            topup_balance = EXCLUDED.topup_balance,
            updated_at = now();
END $$;

ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS stylist_id BIGINT REFERENCES employees(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE membership_state
    DROP COLUMN IF EXISTS free_used,
    DROP COLUMN IF EXISTS free_period_start,
    DROP COLUMN IF EXISTS topup_balance;

ALTER TABLE transactions
    DROP COLUMN IF EXISTS stylist_id;
