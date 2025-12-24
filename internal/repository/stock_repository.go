package repository

import (
	"context"
	"errors"

	"barberpos-backend/internal/db"
	"barberpos-backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

type StockRepository struct {
	DB *db.Postgres
}

func (r StockRepository) List(ctx context.Context, limit int) ([]domain.Stock, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id, product_id, name, category, image, stock, transactions, created_at, updated_at
		FROM stocks
		WHERE deleted_at IS NULL
		ORDER BY name ASC
		LIMIT $1
	`, limit)
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

func (r StockRepository) Adjust(ctx context.Context, in AdjustStockInput) (*domain.Stock, error) {
	tx, err := r.DB.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var current domain.Stock
	err = tx.QueryRow(ctx, `
		SELECT id, product_id, name, category, image, stock, transactions, created_at, updated_at
		FROM stocks
		WHERE id=$1 FOR UPDATE
	`, in.StockID).Scan(&current.ID, &current.ProductID, &current.Name, &current.Category, &current.Image, &current.Stock, &current.Transactions, &current.CreatedAt, &current.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	newStock := current.Stock + in.Change
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
	`, in.StockID, in.ProductID, in.Change, newStock, in.Note, in.Type)
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

func (r StockRepository) History(ctx context.Context, stockID int64, limit int) ([]map[string]any, error) {
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
