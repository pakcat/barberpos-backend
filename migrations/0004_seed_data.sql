-- +goose Up
-- Seed regions (38 provinces)
INSERT INTO regions (name) VALUES
('Aceh'), ('Sumatera Utara'), ('Sumatera Barat'), ('Riau'), ('Kepulauan Riau'),
('Jambi'), ('Sumatera Selatan'), ('Bangka Belitung'), ('Bengkulu'), ('Lampung'),
('DKI Jakarta'), ('Jawa Barat'), ('Banten'), ('Jawa Tengah'), ('DI Yogyakarta'),
('Jawa Timur'), ('Bali'), ('Nusa Tenggara Barat'), ('Nusa Tenggara Timur'),
('Kalimantan Barat'), ('Kalimantan Tengah'), ('Kalimantan Selatan'), ('Kalimantan Timur'),
('Kalimantan Utara'), ('Sulawesi Utara'), ('Gorontalo'), ('Sulawesi Tengah'),
('Sulawesi Barat'), ('Sulawesi Selatan'), ('Sulawesi Tenggara'), ('Maluku'),
('Maluku Utara'), ('Papua'), ('Papua Barat'), ('Papua Barat Daya'),
('Papua Selatan'), ('Papua Tengah'), ('Papua Pegunungan')
ON CONFLICT (name) DO NOTHING;

-- Seed categories
INSERT INTO categories (name) VALUES ('Layanan'), ('Produk') ON CONFLICT (name) DO NOTHING;

-- Seed admin/manager users (password: Admin123!)
INSERT INTO users (id, name, email, phone, address, region, role, is_google, password_hash, created_at, updated_at)
VALUES
  (1, 'Admin', 'admin@barberpos.test', '', '', '', 'admin', false, '$2a$10$vsU42RJjgch.CxwJGvMDnu.iSa7QS3Zt25xGxnML/Dz7rH2aewLkq', now(), now()),
  (2, 'Manager', 'manager@barberpos.test', '', '', '', 'manager', false, '$2a$10$vsU42RJjgch.CxwJGvMDnu.iSa7QS3Zt25xGxnML/Dz7rH2aewLkq', now(), now())
ON CONFLICT (id) DO NOTHING;

-- Seed default settings row
INSERT INTO settings (id, business_name, business_address, business_phone, receipt_footer, default_payment_method, printer_name, paper_size, auto_print, notifications, track_stock, rounding_price, auto_backup, cashier_pin, currency_code, updated_at)
VALUES (1, 'BarberPOS', '', '', 'Terima kasih', 'cash', '', '80mm', false, true, true, false, false, false, 'IDR', now())
ON CONFLICT (id) DO NOTHING;

-- +goose Down
-- No down migration for seed
