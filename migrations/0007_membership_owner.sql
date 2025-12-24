-- +goose Up
ALTER TABLE employees
    ADD COLUMN IF NOT EXISTS manager_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE membership_state
    ADD COLUMN IF NOT EXISTS owner_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE membership_topups
    ADD COLUMN IF NOT EXISTS owner_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL;

-- +goose StatementBegin
DO $$
DECLARE
    owner_id BIGINT;
BEGIN
    SELECT id INTO owner_id FROM users WHERE role IN ('manager','admin') ORDER BY id ASC LIMIT 1;
    IF owner_id IS NOT NULL THEN
        UPDATE employees SET manager_user_id = owner_id WHERE manager_user_id IS NULL;
        UPDATE membership_state SET owner_user_id = owner_id WHERE owner_user_id IS NULL;
        UPDATE membership_topups SET owner_user_id = owner_id WHERE owner_user_id IS NULL;
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE membership_state DROP CONSTRAINT IF EXISTS membership_state_pkey;
ALTER TABLE membership_state ADD CONSTRAINT membership_state_owner_user_id_key UNIQUE (owner_user_id);

CREATE INDEX IF NOT EXISTS idx_membership_topups_owner ON membership_topups (owner_user_id);
CREATE INDEX IF NOT EXISTS idx_employees_manager_user ON employees (manager_user_id);

-- +goose Down
ALTER TABLE employees DROP COLUMN IF EXISTS manager_user_id;

ALTER TABLE membership_state DROP CONSTRAINT IF EXISTS membership_state_owner_user_id_key;
ALTER TABLE membership_state ADD CONSTRAINT membership_state_pkey PRIMARY KEY (id);
ALTER TABLE membership_state DROP COLUMN IF EXISTS owner_user_id;

ALTER TABLE membership_topups DROP COLUMN IF EXISTS owner_user_id;

DROP INDEX IF EXISTS idx_membership_topups_owner;
DROP INDEX IF EXISTS idx_employees_manager_user;
