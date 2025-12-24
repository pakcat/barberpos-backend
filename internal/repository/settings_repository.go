package repository

import (
	"context"
	"errors"
	"time"

	"barberpos-backend/internal/db"
	"barberpos-backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

type SettingsRepository struct {
	DB *db.Postgres
}

func defaultSettings() domain.Settings {
	return domain.Settings{
		BusinessName:         "BarberPOS",
		BusinessAddress:      "",
		BusinessPhone:        "",
		ReceiptFooter:        "Terima kasih",
		DefaultPaymentMethod: "cash",
		PrinterName:          "",
		PrinterType:          "system",
		PrinterHost:          "",
		PrinterPort:          9100,
		PrinterMac:           "",
		PaperSize:            "80mm",
		AutoPrint:            false,
		Notifications:        true,
		TrackStock:           true,
		RoundingPrice:        false,
		AutoBackup:           false,
		CashierPin:           false,
		CurrencyCode:         "IDR",
	}
}

func (r SettingsRepository) Get(ctx context.Context, ownerUserID int64) (*domain.Settings, error) {
	row := r.DB.Pool.QueryRow(ctx, `
		SELECT business_name, business_address, business_phone, receipt_footer, default_payment_method,
		       printer_name, printer_type, printer_host, printer_port, printer_mac,
		       paper_size, auto_print, notifications, track_stock, rounding_price, auto_backup, cashier_pin, currency_code, updated_at
		FROM settings
		WHERE owner_user_id=$1
	`, ownerUserID)
	var s domain.Settings
	if err := row.Scan(
		&s.BusinessName, &s.BusinessAddress, &s.BusinessPhone, &s.ReceiptFooter, &s.DefaultPaymentMethod,
		&s.PrinterName, &s.PrinterType, &s.PrinterHost, &s.PrinterPort, &s.PrinterMac,
		&s.PaperSize, &s.AutoPrint, &s.Notifications, &s.TrackStock, &s.RoundingPrice, &s.AutoBackup, &s.CashierPin, &s.CurrencyCode, &s.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			def := defaultSettings()
			return r.Save(ctx, ownerUserID, def)
		}
		return nil, err
	}
	return &s, nil
}

func (r SettingsRepository) Save(ctx context.Context, ownerUserID int64, s domain.Settings) (*domain.Settings, error) {
	err := r.DB.Pool.QueryRow(ctx, `
		INSERT INTO settings (owner_user_id, business_name, business_address, business_phone, receipt_footer, default_payment_method,
		                      printer_name, printer_type, printer_host, printer_port, printer_mac,
		                      paper_size, auto_print, notifications, track_stock, rounding_price, auto_backup, cashier_pin, currency_code, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19, now())
		ON CONFLICT (owner_user_id) DO UPDATE SET
			business_name=EXCLUDED.business_name,
			business_address=EXCLUDED.business_address,
			business_phone=EXCLUDED.business_phone,
			receipt_footer=EXCLUDED.receipt_footer,
			default_payment_method=EXCLUDED.default_payment_method,
			printer_name=EXCLUDED.printer_name,
			printer_type=EXCLUDED.printer_type,
			printer_host=EXCLUDED.printer_host,
			printer_port=EXCLUDED.printer_port,
			printer_mac=EXCLUDED.printer_mac,
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
		          printer_name, printer_type, printer_host, printer_port, printer_mac,
		          paper_size, auto_print, notifications, track_stock, rounding_price, auto_backup, cashier_pin, currency_code, updated_at
	`, ownerUserID, s.BusinessName, s.BusinessAddress, s.BusinessPhone, s.ReceiptFooter, s.DefaultPaymentMethod,
		s.PrinterName, s.PrinterType, s.PrinterHost, s.PrinterPort, s.PrinterMac,
		s.PaperSize, s.AutoPrint, s.Notifications, s.TrackStock, s.RoundingPrice, s.AutoBackup, s.CashierPin, s.CurrencyCode).Scan(
		&s.BusinessName, &s.BusinessAddress, &s.BusinessPhone, &s.ReceiptFooter, &s.DefaultPaymentMethod,
		&s.PrinterName, &s.PrinterType, &s.PrinterHost, &s.PrinterPort, &s.PrinterMac,
		&s.PaperSize, &s.AutoPrint, &s.Notifications, &s.TrackStock, &s.RoundingPrice, &s.AutoBackup, &s.CashierPin, &s.CurrencyCode, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r SettingsRepository) HasQrisImage(ctx context.Context, ownerUserID int64) (bool, error) {
	var ok bool
	err := r.DB.Pool.QueryRow(ctx, `
		SELECT qris_image IS NOT NULL
		FROM settings
		WHERE owner_user_id=$1
	`, ownerUserID).Scan(&ok)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return ok, nil
}

func (r SettingsRepository) SetQrisImage(ctx context.Context, ownerUserID int64, bytes []byte, mime string) error {
	_, err := r.DB.Pool.Exec(ctx, `
		UPDATE settings
		SET qris_image=$2, qris_image_mime=$3, qris_image_updated_at=now(), updated_at=now()
		WHERE owner_user_id=$1
	`, ownerUserID, bytes, mime)
	return err
}

func (r SettingsRepository) ClearQrisImage(ctx context.Context, ownerUserID int64) error {
	_, err := r.DB.Pool.Exec(ctx, `
		UPDATE settings
		SET qris_image=NULL, qris_image_mime='', qris_image_updated_at=NULL, updated_at=now()
		WHERE owner_user_id=$1
	`, ownerUserID)
	return err
}

func (r SettingsRepository) GetQrisImage(ctx context.Context, ownerUserID int64) (bytes []byte, mime string, updatedAt *time.Time, err error) {
	err = r.DB.Pool.QueryRow(ctx, `
		SELECT qris_image, qris_image_mime, qris_image_updated_at
		FROM settings
		WHERE owner_user_id=$1
	`, ownerUserID).Scan(&bytes, &mime, &updatedAt)
	if err != nil {
		return nil, "", nil, err
	}
	if len(bytes) == 0 {
		return nil, "", updatedAt, pgx.ErrNoRows
	}
	return bytes, mime, updatedAt, nil
}
