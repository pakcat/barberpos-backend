package repository

import (
	"context"

	"barberpos-backend/internal/db"
	"barberpos-backend/internal/domain"
)

type SettingsRepository struct {
	DB *db.Postgres
}

func (r SettingsRepository) Get(ctx context.Context) (*domain.Settings, error) {
	row := r.DB.Pool.QueryRow(ctx, `
		SELECT business_name, business_address, business_phone, receipt_footer, default_payment_method,
		       printer_name, paper_size, auto_print, notifications, track_stock, rounding_price, auto_backup, cashier_pin, currency_code, updated_at
		FROM settings
		WHERE id=1
	`)
	var s domain.Settings
	if err := row.Scan(
		&s.BusinessName, &s.BusinessAddress, &s.BusinessPhone, &s.ReceiptFooter, &s.DefaultPaymentMethod,
		&s.PrinterName, &s.PaperSize, &s.AutoPrint, &s.Notifications, &s.TrackStock, &s.RoundingPrice, &s.AutoBackup, &s.CashierPin, &s.CurrencyCode, &s.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r SettingsRepository) Save(ctx context.Context, s domain.Settings) (*domain.Settings, error) {
	err := r.DB.Pool.QueryRow(ctx, `
		INSERT INTO settings (id, business_name, business_address, business_phone, receipt_footer, default_payment_method,
		                      printer_name, paper_size, auto_print, notifications, track_stock, rounding_price, auto_backup, cashier_pin, currency_code, updated_at)
		VALUES (1,$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14, now())
		ON CONFLICT (id) DO UPDATE SET
			business_name=EXCLUDED.business_name,
			business_address=EXCLUDED.business_address,
			business_phone=EXCLUDED.business_phone,
			receipt_footer=EXCLUDED.receipt_footer,
			default_payment_method=EXCLUDED.default_payment_method,
			printer_name=EXCLUDED.printer_name,
			paper_size=EXCLUDED.paper_size,
			auto_print=EXCLUDED.auto_print,
			notifications=EXCLUDED.notifications,
			track_stock=EXCLUDED.track_stock,
			rounding_price=EXCLUDED.rounding_price,
			auto_backup=EXCLUDED.auto_backup,
			cashier_pin=EXCLUDED.cashier_pin,
			currency_code=EXCLUDED.currency_code,
			updated_at=now()
		RETURNING business_name, business_address, business_phone, receipt_footer, default_payment_method,
		          printer_name, paper_size, auto_print, notifications, track_stock, rounding_price, auto_backup, cashier_pin, currency_code, updated_at
	`, s.BusinessName, s.BusinessAddress, s.BusinessPhone, s.ReceiptFooter, s.DefaultPaymentMethod, s.PrinterName, s.PaperSize, s.AutoPrint, s.Notifications, s.TrackStock, s.RoundingPrice, s.AutoBackup, s.CashierPin, s.CurrencyCode).Scan(
		&s.BusinessName, &s.BusinessAddress, &s.BusinessPhone, &s.ReceiptFooter, &s.DefaultPaymentMethod,
		&s.PrinterName, &s.PaperSize, &s.AutoPrint, &s.Notifications, &s.TrackStock, &s.RoundingPrice, &s.AutoBackup, &s.CashierPin, &s.CurrencyCode, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
