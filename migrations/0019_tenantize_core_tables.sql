-- +goose Up
-- Tenantize core operational tables so data is isolated per manager/user.

-- Resolve a fallback owner (first manager/admin). Used only for backfilling legacy rows.
-- +goose StatementBegin
DO $$
DECLARE
    owner_id BIGINT;
BEGIN
    SELECT id INTO owner_id FROM users WHERE role IN ('manager','admin') ORDER BY id ASC LIMIT 1;

    -- Categories
    ALTER TABLE categories ADD COLUMN IF NOT EXISTS owner_user_id BIGINT;
    IF owner_id IS NOT NULL THEN
        UPDATE categories SET owner_user_id = owner_id WHERE owner_user_id IS NULL;
    END IF;
    IF EXISTS (SELECT 1 FROM categories WHERE owner_user_id IS NULL) THEN
        RAISE EXCEPTION 'categories.owner_user_id backfill failed: create a manager/admin user first';
    END IF;
    ALTER TABLE categories
        ALTER COLUMN owner_user_id SET NOT NULL;
    ALTER TABLE categories
        DROP CONSTRAINT IF EXISTS categories_name_key;
    CREATE INDEX IF NOT EXISTS idx_categories_owner_user_id ON categories (owner_user_id);
    CREATE UNIQUE INDEX IF NOT EXISTS categories_owner_name_unique
        ON categories (owner_user_id, name);

    -- Customers
    ALTER TABLE customers ADD COLUMN IF NOT EXISTS owner_user_id BIGINT;
    IF owner_id IS NOT NULL THEN
        UPDATE customers SET owner_user_id = owner_id WHERE owner_user_id IS NULL;
    END IF;
    IF EXISTS (SELECT 1 FROM customers WHERE owner_user_id IS NULL) THEN
        RAISE EXCEPTION 'customers.owner_user_id backfill failed: create a manager/admin user first';
    END IF;
    ALTER TABLE customers
        ALTER COLUMN owner_user_id SET NOT NULL;
    ALTER TABLE customers
        DROP CONSTRAINT IF EXISTS customers_phone_key;
    CREATE INDEX IF NOT EXISTS idx_customers_owner_user_id ON customers (owner_user_id);
    CREATE UNIQUE INDEX IF NOT EXISTS customers_owner_phone_unique
        ON customers (owner_user_id, phone);

    -- Transactions
    ALTER TABLE transactions ADD COLUMN IF NOT EXISTS owner_user_id BIGINT;
    IF owner_id IS NOT NULL THEN
        UPDATE transactions SET owner_user_id = owner_id WHERE owner_user_id IS NULL;
    END IF;
    IF EXISTS (SELECT 1 FROM transactions WHERE owner_user_id IS NULL) THEN
        RAISE EXCEPTION 'transactions.owner_user_id backfill failed: create a manager/admin user first';
    END IF;
    ALTER TABLE transactions
        ALTER COLUMN owner_user_id SET NOT NULL;
    CREATE INDEX IF NOT EXISTS idx_transactions_owner_user_id ON transactions (owner_user_id);

    -- Finance entries
    ALTER TABLE finance_entries ADD COLUMN IF NOT EXISTS owner_user_id BIGINT;
    IF owner_id IS NOT NULL THEN
        UPDATE finance_entries fe
        SET owner_user_id = COALESCE(t.owner_user_id, owner_id)
        FROM transactions t
        WHERE fe.owner_user_id IS NULL
          AND fe.transaction_id IS NOT NULL
          AND t.id = fe.transaction_id;
        UPDATE finance_entries SET owner_user_id = owner_id WHERE owner_user_id IS NULL;
    END IF;
    IF EXISTS (SELECT 1 FROM finance_entries WHERE owner_user_id IS NULL) THEN
        RAISE EXCEPTION 'finance_entries.owner_user_id backfill failed: create a manager/admin user first';
    END IF;
    ALTER TABLE finance_entries
        ALTER COLUMN owner_user_id SET NOT NULL;
    CREATE INDEX IF NOT EXISTS idx_finance_entries_owner_user_id ON finance_entries (owner_user_id, entry_date DESC);

    -- Attendance
    ALTER TABLE attendance ADD COLUMN IF NOT EXISTS owner_user_id BIGINT;
    -- Prefer employee.manager_user_id when possible.
    UPDATE attendance a
    SET owner_user_id = e.manager_user_id
    FROM employees e
    WHERE a.owner_user_id IS NULL
      AND a.employee_id IS NOT NULL
      AND e.id = a.employee_id;
    IF owner_id IS NOT NULL THEN
        UPDATE attendance SET owner_user_id = owner_id WHERE owner_user_id IS NULL;
    END IF;
    IF EXISTS (SELECT 1 FROM attendance WHERE owner_user_id IS NULL) THEN
        RAISE EXCEPTION 'attendance.owner_user_id backfill failed: create a manager/admin user first';
    END IF;
    ALTER TABLE attendance
        ALTER COLUMN owner_user_id SET NOT NULL;
    ALTER TABLE attendance
        DROP CONSTRAINT IF EXISTS attendance_employee_name_attendance_date_key;
    CREATE INDEX IF NOT EXISTS idx_attendance_owner_user_id ON attendance (owner_user_id, attendance_date DESC);
    CREATE UNIQUE INDEX IF NOT EXISTS attendance_owner_employee_name_date_unique
        ON attendance (owner_user_id, employee_name, attendance_date);

    -- Closing history
    ALTER TABLE closing_history ADD COLUMN IF NOT EXISTS owner_user_id BIGINT;
    IF owner_id IS NOT NULL THEN
        UPDATE closing_history SET owner_user_id = owner_id WHERE owner_user_id IS NULL;
    END IF;
    IF EXISTS (SELECT 1 FROM closing_history WHERE owner_user_id IS NULL) THEN
        RAISE EXCEPTION 'closing_history.owner_user_id backfill failed: create a manager/admin user first';
    END IF;
    ALTER TABLE closing_history
        ALTER COLUMN owner_user_id SET NOT NULL;
    CREATE INDEX IF NOT EXISTS idx_closing_history_owner_user_id ON closing_history (owner_user_id, tanggal DESC);

    -- Activity logs
    ALTER TABLE activity_logs ADD COLUMN IF NOT EXISTS owner_user_id BIGINT;
    IF owner_id IS NOT NULL THEN
        UPDATE activity_logs SET owner_user_id = owner_id WHERE owner_user_id IS NULL;
    END IF;
    IF EXISTS (SELECT 1 FROM activity_logs WHERE owner_user_id IS NULL) THEN
        RAISE EXCEPTION 'activity_logs.owner_user_id backfill failed: create a manager/admin user first';
    END IF;
    ALTER TABLE activity_logs
        ALTER COLUMN owner_user_id SET NOT NULL;
    CREATE INDEX IF NOT EXISTS idx_activity_logs_owner_user_id ON activity_logs (owner_user_id, logged_at DESC);
END $$;
-- +goose StatementEnd

-- Settings: move from singleton row (id=1) to per-owner row keyed by owner_user_id.
-- +goose StatementBegin
DO $$
DECLARE
    owner_id BIGINT;
BEGIN
    SELECT id INTO owner_id FROM users WHERE role IN ('manager','admin') ORDER BY id ASC LIMIT 1;

    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='settings_new') THEN
        CREATE TABLE settings_new (
            owner_user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
            business_name TEXT NOT NULL DEFAULT '',
            business_address TEXT NOT NULL DEFAULT '',
            business_phone TEXT NOT NULL DEFAULT '',
            receipt_footer TEXT NOT NULL DEFAULT '',
            default_payment_method TEXT NOT NULL DEFAULT '',
            printer_name TEXT NOT NULL DEFAULT '',
            printer_type TEXT NOT NULL DEFAULT 'system',
            printer_host TEXT NOT NULL DEFAULT '',
            printer_port INTEGER NOT NULL DEFAULT 9100,
            printer_mac TEXT NOT NULL DEFAULT '',
            paper_size TEXT NOT NULL DEFAULT '',
            auto_print BOOLEAN NOT NULL DEFAULT FALSE,
            notifications BOOLEAN NOT NULL DEFAULT TRUE,
            track_stock BOOLEAN NOT NULL DEFAULT FALSE,
            rounding_price BOOLEAN NOT NULL DEFAULT FALSE,
            auto_backup BOOLEAN NOT NULL DEFAULT FALSE,
            cashier_pin BOOLEAN NOT NULL DEFAULT FALSE,
            currency_code TEXT NOT NULL DEFAULT 'IDR',
            updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
        );
    END IF;

    IF owner_id IS NOT NULL THEN
        INSERT INTO settings_new (
            owner_user_id,
            business_name, business_address, business_phone, receipt_footer, default_payment_method,
            printer_name, printer_type, printer_host, printer_port, printer_mac,
            paper_size, auto_print, notifications, track_stock, rounding_price, auto_backup, cashier_pin,
            currency_code, updated_at
        )
        SELECT
            owner_id,
            business_name, business_address, business_phone, receipt_footer, default_payment_method,
            printer_name,
            COALESCE(printer_type, 'system'),
            COALESCE(printer_host, ''),
            COALESCE(printer_port, 9100),
            COALESCE(printer_mac, ''),
            paper_size, auto_print, notifications, track_stock, rounding_price, auto_backup, cashier_pin,
            currency_code, updated_at
        FROM settings
        WHERE id=1
        ON CONFLICT (owner_user_id) DO NOTHING;
    END IF;

    DROP TABLE IF EXISTS settings;
    ALTER TABLE settings_new RENAME TO settings;
END $$;
-- +goose StatementEnd

-- +goose Down
-- NOTE: Down migrations for tenantization are intentionally not provided (data-loss risk).
