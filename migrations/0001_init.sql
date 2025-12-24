-- +goose Up
-- Core reference tables
CREATE TABLE IF NOT EXISTS regions (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    phone TEXT NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    region TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL DEFAULT 'manager' CHECK (role IN ('admin','manager','staff')),
    is_google BOOLEAN NOT NULL DEFAULT FALSE,
    password_hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS categories (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS customers (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    phone TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS products (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    category TEXT NOT NULL DEFAULT '',
    price BIGINT NOT NULL CHECK (price >= 0),
    image TEXT NOT NULL DEFAULT '',
    track_stock BOOLEAN NOT NULL DEFAULT FALSE,
    stock INTEGER NOT NULL DEFAULT 0,
    min_stock INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS stocks (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT REFERENCES products(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT '',
    image TEXT NOT NULL DEFAULT '',
    stock INTEGER NOT NULL DEFAULT 0,
    transactions INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS transactions (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    transacted_date DATE NOT NULL,
    transacted_time TEXT NOT NULL,
    amount BIGINT NOT NULL CHECK (amount >= 0),
    payment_method TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'paid' CHECK (status IN ('paid','refund')),
    stylist TEXT NOT NULL DEFAULT '',
    customer_id BIGINT REFERENCES customers(id) ON DELETE SET NULL,
    customer_name TEXT,
    customer_phone TEXT,
    customer_email TEXT,
    customer_address TEXT,
    customer_visits INTEGER,
    customer_last_visit TEXT,
    shift_id TEXT,
    operator_name TEXT,
    payment_intent_id TEXT,
    payment_reference TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions (transacted_date);

CREATE TABLE IF NOT EXISTS transaction_items (
    id BIGSERIAL PRIMARY KEY,
    transaction_id BIGINT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    product_id BIGINT REFERENCES products(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT '',
    price BIGINT NOT NULL CHECK (price >= 0),
    qty INTEGER NOT NULL CHECK (qty > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS employees (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    role TEXT NOT NULL,
    phone TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL,
    join_date DATE NOT NULL,
    commission NUMERIC(6,2),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS attendance (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT REFERENCES employees(id) ON DELETE SET NULL,
    employee_name TEXT NOT NULL,
    attendance_date DATE NOT NULL,
    check_in TIMESTAMPTZ,
    check_out TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'present',
    source TEXT NOT NULL DEFAULT 'cashier',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (employee_name, attendance_date)
);

CREATE TABLE IF NOT EXISTS finance_entries (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    amount BIGINT NOT NULL CHECK (amount >= 0),
    category TEXT NOT NULL,
    entry_date DATE NOT NULL,
    type TEXT NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    staff TEXT,
    service TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS membership_state (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    used_quota INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS membership_topups (
    id BIGSERIAL PRIMARY KEY,
    amount BIGINT NOT NULL CHECK (amount >= 0),
    manager TEXT NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    topup_date TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS settings (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    business_name TEXT NOT NULL DEFAULT '',
    business_address TEXT NOT NULL DEFAULT '',
    business_phone TEXT NOT NULL DEFAULT '',
    receipt_footer TEXT NOT NULL DEFAULT '',
    default_payment_method TEXT NOT NULL DEFAULT '',
    printer_name TEXT NOT NULL DEFAULT '',
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

CREATE TABLE IF NOT EXISTS activity_logs (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    actor TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'info',
    logged_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    synced BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS closing_history (
    id BIGSERIAL PRIMARY KEY,
    tanggal DATE NOT NULL,
    shift TEXT NOT NULL,
    karyawan TEXT NOT NULL,
    shift_id TEXT,
    operator_name TEXT NOT NULL DEFAULT '',
    total BIGINT NOT NULL,
    status TEXT NOT NULL,
    catatan TEXT NOT NULL DEFAULT '',
    fisik TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE IF EXISTS closing_history;
DROP TABLE IF EXISTS activity_logs;
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS membership_topups;
DROP TABLE IF EXISTS membership_state;
DROP TABLE IF EXISTS finance_entries;
DROP TABLE IF EXISTS attendance;
DROP TABLE IF EXISTS employees;
DROP TABLE IF EXISTS transaction_items;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS stocks;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS customers;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS regions;
