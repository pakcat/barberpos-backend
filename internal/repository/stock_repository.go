package repository

import (
	"context"
	"errors"
	"strings"

	"barberpos-backend/internal/db"
	"barberpos-backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

type StockRepository struct {
	DB *db.Postgres
}

func (r StockRepository) SyncFromProducts(ctx context.Context) error {
	tx, err := r.DB.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Remove duplicates (keep smallest id) before enforcing uniqueness.
	if _, err := tx.Exec(ctx, `
		WITH ranked AS (
			SELECT id,
				   ROW_NUMBER() OVER (PARTITION BY product_id ORDER BY id ASC) AS rn
			FROM stocks
			WHERE product_id IS NOT NULL
		)
		DELETE FROM stocks
		WHERE id IN (SELECT id FROM ranked WHERE rn > 1)
	`); err != nil {
		return err
	}

	// Ensure 1 stock row per product_id.
	if _, err := tx.Exec(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS stocks_product_id_unique
		ON stocks (product_id)
		WHERE product_id IS NOT NULL
	`); err != nil {
		return err
	}

	// Create missing stock rows for tracked products.
	if _, err := tx.Exec(ctx, `
		INSERT INTO stocks (product_id, name, category, image, stock, transactions, created_at, updated_at)
		SELECT p.id, p.name, p.category, p.image, p.stock, 0, now(), now()
		FROM products p
		WHERE p.deleted_at IS NULL AND p.track_stock = TRUE
		  AND NOT EXISTS (
			  SELECT 1 FROM stocks s WHERE s.product_id = p.id AND s.deleted_at IS NULL
		  )
	`); err != nil {
		return err
	}

	// Sync tracked product fields.
	if _, err := tx.Exec(ctx, `
		UPDATE stocks s
		SET name = p.name,
			category = p.category,
			image = p.image,
			stock = p.stock,
			updated_at = now(),
			deleted_at = NULL
		FROM products p
		WHERE s.product_id = p.id
		  AND p.deleted_at IS NULL
		  AND p.track_stock = TRUE
	`); err != nil {
		return err
	}

	// Soft-delete stocks for non-tracked products.
	if _, err := tx.Exec(ctx, `
		UPDATE stocks s
		SET deleted_at = now(),
			updated_at = now()
		FROM products p
		WHERE s.product_id = p.id
		  AND (p.deleted_at IS NOT NULL OR p.track_stock = FALSE)
		  AND s.deleted_at IS NULL
	`); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r StockRepository) AdjustByProductIDWithTx(ctx context.Context, tx pgx.Tx, ownerUserID int64, productID int64, delta int, typ string, note string) error {
	row := tx.QueryRow(ctx, `
		SELECT id, stock
		FROM stocks
		WHERE deleted_at IS NULL
		  AND product_id=$1
		  AND EXISTS (
			  SELECT 1 FROM products p
			  WHERE p.id = stocks.product_id AND p.owner_user_id=$2 AND p.deleted_at IS NULL
		  )
		FOR UPDATE
	`, productID, ownerUserID)
	var stockID int64
	var current int
	if err := row.Scan(&stockID, &current); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	newStock := current + delta
	if newStock < 0 {
		newStock = 0
	}
	_, err := tx.Exec(ctx, `
		UPDATE stocks
		SET stock=$1, transactions=transactions+1, updated_at=now()
		WHERE id=$2
	`, newStock, stockID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO stock_history (stock_id, product_id, change, remaining, note, type, created_at)
		VALUES ($1,$2,$3,$4,$5,$6, now())
	`, stockID, productID, delta, newStock, note, typ)
	if err != nil {
		return err
	}
	return nil
}

func (r StockRepository) List(ctx context.Context, ownerUserID int64, limit int) ([]domain.Stock, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT stocks.id, stocks.product_id, stocks.name, stocks.category, stocks.image, stocks.stock, stocks.transactions, stocks.created_at, stocks.updated_at
		FROM stocks
		JOIN products p ON p.id = stocks.product_id
		WHERE stocks.deleted_at IS NULL
		  AND p.deleted_at IS NULL
		  AND p.owner_user_id=$1
		ORDER BY stocks.name ASC
		LIMIT $2
	`, ownerUserID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Stock
	for rows.Next() {
		var s domain.Stock
		if err := rows.Scan(&s.ID, &s.ProductID, &s.Name, &s.Category, &s.Image, &s.Stock, &s.Transactions, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

type AdjustStockInput struct {
	StockID   int64
	Change    int
	Type      string
	Note      string
	ProductID *int64
}

func (r StockRepository) Adjust(ctx context.Context, ownerUserID int64, in AdjustStockInput) (*domain.Stock, error) {
	tx, err := r.DB.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var current domain.Stock
	err = tx.QueryRow(ctx, `
		SELECT id, product_id, name, category, image, stock, transactions, created_at, updated_at
		FROM stocks
		WHERE id=$1
		  AND EXISTS (
			  SELECT 1 FROM products p
			  WHERE p.id = stocks.product_id AND p.owner_user_id=$2 AND p.deleted_at IS NULL
		  )
		FOR UPDATE
	`, in.StockID, ownerUserID).Scan(&current.ID, &current.ProductID, &current.Name, &current.Category, &current.Image, &current.Stock, &current.Transactions, &current.CreatedAt, &current.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	change := in.Change
	switch strings.ToLower(strings.TrimSpace(in.Type)) {
	case "reduce":
		if change > 0 {
			change = -change
		}
	case "recount":
		// Interpret `Change` as absolute stock value.
		if change < 0 {
			change = 0
		}
		change = change - current.Stock
	}

	newStock := current.Stock + change
	if newStock < 0 {
		newStock = 0
	}

	_, err = tx.Exec(ctx, `
		UPDATE stocks
		SET stock=$1, transactions=transactions+1, updated_at=now()
		WHERE id=$2
	`, newStock, in.StockID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO stock_history (stock_id, product_id, change, remaining, note, type, created_at)
		VALUES ($1,$2,$3,$4,$5,$6, now())
	`, in.StockID, in.ProductID, change, newStock, in.Note, in.Type)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	current.Stock = newStock
	current.Transactions = current.Transactions + 1
	return &current, nil
}

func (r StockRepository) History(ctx context.Context, ownerUserID int64, stockID int64, limit int) ([]map[string]any, error) {
	// Ensure stock belongs to caller.
	var exists bool
	if err := r.DB.Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM stocks s
			JOIN products p ON p.id = s.product_id
			WHERE s.id=$1 AND s.deleted_at IS NULL AND p.deleted_at IS NULL AND p.owner_user_id=$2
		)
	`, stockID, ownerUserID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id, change, remaining, note, type, created_at
		FROM stock_history
		WHERE stock_id=$1
		ORDER BY created_at DESC
		LIMIT $2
	`, stockID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []map[string]any
	for rows.Next() {
		var id int64
		var change, remaining int
		var note, typ string
		var createdAt any
		if err := rows.Scan(&id, &change, &remaining, &note, &typ, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"id":        id,
			"change":    change,
			"remaining": remaining,
			"note":      note,
			"type":      typ,
			"createdAt": createdAt,
		})
	}
	return items, rows.Err()
}
