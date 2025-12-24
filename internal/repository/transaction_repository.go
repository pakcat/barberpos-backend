package repository

import (
	"context"
	"fmt"
	"time"

	"barberpos-backend/internal/db"
	"barberpos-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type TransactionRepository struct {
	DB *db.Postgres
}

func (r TransactionRepository) GetByCode(ctx context.Context, code string) (*domain.Transaction, error) {
	row := r.DB.Pool.QueryRow(ctx, `
		SELECT id, code, transacted_date, transacted_time, amount, payment_method, status, stylist, stylist_id,
		       customer_name, customer_phone, customer_email, customer_address, customer_visits, customer_last_visit,
		       shift_id, operator_name, created_at, updated_at, deleted_at
		FROM transactions
		WHERE deleted_at IS NULL AND code = $1
		LIMIT 1
	`, code)
	var t domain.Transaction
	var status string
	var customerName, customerPhone, customerEmail, customerAddress pgtype.Text
	var visits pgtype.Int4
	var lastVisit pgtype.Text
	var shiftID pgtype.Text
	var opName pgtype.Text
	var stylistID pgtype.Int8
	var deletedAt pgtype.Timestamptz
	if err := row.Scan(
		&t.ID, &t.Code, &t.Date, &t.Time, &t.Amount.Amount, &t.PaymentMethod, &status, &t.Stylist, &stylistID,
		&customerName, &customerPhone, &customerEmail, &customerAddress, &visits, &lastVisit,
		&shiftID, &opName, &t.CreatedAt, &t.UpdatedAt, &deletedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if stylistID.Valid {
		t.StylistID = &stylistID.Int64
	}
	t.Status = domain.TransactionStatus(status)
	t.Customer = &domain.TransactionCustomerSnapshot{
		Name:    customerName.String,
		Phone:   customerPhone.String,
		Email:   customerEmail.String,
		Address: customerAddress.String,
	}
	if visits.Valid {
		v := int(visits.Int32)
		t.Customer.Visits = &v
	}
	if lastVisit.Valid {
		lv := lastVisit.String
		t.Customer.LastVisit = &lv
	}
	if shiftID.Valid {
		s := shiftID.String
		t.ShiftID = &s
	}
	if opName.Valid {
		t.OperatorName = opName.String
	}
	if deletedAt.Valid {
		d := deletedAt.Time
		t.DeletedAt = &d
	}

	itemRows, err := r.DB.Pool.Query(ctx, `
		SELECT transaction_id, id, product_id, name, category, price, qty, created_at
		FROM transaction_items
		WHERE transaction_id=$1 AND deleted_at IS NULL
	`, t.ID)
	if err != nil {
		return nil, err
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var it domain.TransactionItem
		var txID int64
		if err := itemRows.Scan(&txID, &it.ID, &it.ProductID, &it.Name, &it.Category, &it.Price.Amount, &it.Qty, &it.CreatedAt); err != nil {
			return nil, err
		}
		it.TransactionID = txID
		t.Items = append(t.Items, it)
	}
	if err := itemRows.Err(); err != nil {
		return nil, err
	}
	return &t, nil
}

type CreateTransactionInput struct {
	PaymentMethod     string
	Stylist           string
	StylistID         *int64
	CustomerName      string
	CustomerPhone     string
	CustomerEmail     string
	CustomerAddr      string
	CustomerVisits    *int
	CustomerLastVisit *string
	ShiftID           *string
	OperatorName      string
	Amount            int64
	Items             []CreateTransactionItem
}

type CreateTransactionItem struct {
	ProductID *int64
	Name      string
	Category  string
	Price     int64
	Qty       int
}

func (r TransactionRepository) Create(ctx context.Context, in CreateTransactionInput, after func(context.Context, pgx.Tx) error) (*domain.Transaction, error) {
	tx, err := r.DB.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	code := fmt.Sprintf("ORD-%d", time.Now().UnixNano()/1e6)
	now := time.Now()
	var id int64
	_, err = tx.Exec(ctx, "SET LOCAL synchronous_commit TO OFF")
	if err != nil {
		// non-fatal; continue
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions
		(code, transacted_date, transacted_time, amount, payment_method, status, stylist, stylist_id,
		 customer_name, customer_phone, customer_email, customer_address, customer_visits, customer_last_visit,
		 shift_id, operator_name, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16, now(), now())
		RETURNING id
	`, code, now.Format("2006-01-02"), now.Format("15:04"), in.Amount, in.PaymentMethod, domain.TransactionPaid, in.Stylist, in.StylistID,
		in.CustomerName, in.CustomerPhone, in.CustomerEmail, in.CustomerAddr, in.CustomerVisits, in.CustomerLastVisit,
		in.ShiftID, in.OperatorName).Scan(&id)
	if err != nil {
		return nil, err
	}

	for _, item := range in.Items {
		_, err := tx.Exec(ctx, `
			INSERT INTO transaction_items (transaction_id, product_id, name, category, price, qty, created_at)
			VALUES ($1,$2,$3,$4,$5,$6, now())
		`, id, item.ProductID, item.Name, item.Category, item.Price, item.Qty)
		if err != nil {
			return nil, err
		}
	}

	if after != nil {
		if err := after(ctx, tx); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &domain.Transaction{
		ID:            id,
		Code:          code,
		Date:          now,
		Time:          now.Format("15:04"),
		Amount:        domain.Money{Amount: in.Amount},
		PaymentMethod: in.PaymentMethod,
		Status:        domain.TransactionPaid,
		Stylist:       in.Stylist,
		StylistID:     in.StylistID,
		Customer: &domain.TransactionCustomerSnapshot{
			Name:      in.CustomerName,
			Phone:     in.CustomerPhone,
			Email:     in.CustomerEmail,
			Address:   in.CustomerAddr,
			Visits:    in.CustomerVisits,
			LastVisit: in.CustomerLastVisit,
		},
		Items:     mapItems(in.Items),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func mapItems(items []CreateTransactionItem) []domain.TransactionItem {
	var out []domain.TransactionItem
	for _, it := range items {
		out = append(out, domain.TransactionItem{
			ProductID: it.ProductID,
			Name:      it.Name,
			Category:  it.Category,
			Price:     domain.Money{Amount: it.Price},
			Qty:       it.Qty,
		})
	}
	return out
}

func (r TransactionRepository) List(ctx context.Context, limit int) ([]domain.Transaction, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id, code, transacted_date, transacted_time, amount, payment_method, status, stylist, stylist_id,
		       customer_name, customer_phone, customer_email, customer_address, customer_visits, customer_last_visit,
		       shift_id, operator_name, created_at, updated_at
		FROM transactions
		WHERE deleted_at IS NULL
		ORDER BY transacted_date DESC, id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []domain.Transaction
	var ids []int64
	for rows.Next() {
		var t domain.Transaction
		var status string
		var customerName, customerPhone, customerEmail, customerAddress pgtype.Text
		var visits pgtype.Int4
		var lastVisit pgtype.Text
		var shiftID pgtype.Text
		var opName pgtype.Text
		var stylistID pgtype.Int8
		if err := rows.Scan(
			&t.ID, &t.Code, &t.Date, &t.Time, &t.Amount.Amount, &t.PaymentMethod, &status, &t.Stylist, &stylistID,
			&customerName, &customerPhone, &customerEmail, &customerAddress, &visits, &lastVisit,
			&shiftID, &opName, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if stylistID.Valid {
			t.StylistID = &stylistID.Int64
		}
		t.Status = domain.TransactionStatus(status)
		t.Customer = &domain.TransactionCustomerSnapshot{
			Name:    customerName.String,
			Phone:   customerPhone.String,
			Email:   customerEmail.String,
			Address: customerAddress.String,
		}
		if visits.Valid {
			v := int(visits.Int32)
			t.Customer.Visits = &v
		}
		if lastVisit.Valid {
			lv := lastVisit.String
			t.Customer.LastVisit = &lv
		}
		ids = append(ids, t.ID)
		txs = append(txs, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return txs, nil
	}

	itemRows, err := r.DB.Pool.Query(ctx, `
		SELECT transaction_id, id, product_id, name, category, price, qty, created_at
		FROM transaction_items
		WHERE transaction_id = ANY($1)
	`, ids)
	if err != nil {
		return nil, err
	}
	defer itemRows.Close()

	itemsByTx := make(map[int64][]domain.TransactionItem)
	for itemRows.Next() {
		var it domain.TransactionItem
		var txID int64
		if err := itemRows.Scan(&txID, &it.ID, &it.ProductID, &it.Name, &it.Category, &it.Price.Amount, &it.Qty, &it.CreatedAt); err != nil {
			return nil, err
		}
		itemsByTx[txID] = append(itemsByTx[txID], it)
	}
	if err := itemRows.Err(); err != nil {
		return nil, err
	}

	for i := range txs {
		txs[i].Items = itemsByTx[txs[i].ID]
	}

	return txs, nil
}

func (r TransactionRepository) ListFiltered(ctx context.Context, startDate, endDate *time.Time) ([]domain.Transaction, error) {
	query := `
		SELECT id, code, transacted_date, transacted_time, amount, payment_method, status, stylist, stylist_id,
		       customer_name, customer_phone, customer_email, customer_address, customer_visits, customer_last_visit,
		       shift_id, operator_name, created_at, updated_at
		FROM transactions
		WHERE deleted_at IS NULL
	`
	args := make([]any, 0, 2)
	if startDate != nil {
		query += fmt.Sprintf(" AND transacted_date >= $%d", len(args)+1)
		args = append(args, startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		query += fmt.Sprintf(" AND transacted_date <= $%d", len(args)+1)
		args = append(args, endDate.Format("2006-01-02"))
	}
	query += " ORDER BY transacted_date DESC, id DESC"

	rows, err := r.DB.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []domain.Transaction
	var ids []int64
	for rows.Next() {
		var t domain.Transaction
		var status string
		var customerName, customerPhone, customerEmail, customerAddress pgtype.Text
		var visits pgtype.Int4
		var lastVisit pgtype.Text
		var shiftID pgtype.Text
		var opName pgtype.Text
		var stylistID pgtype.Int8
		if err := rows.Scan(
			&t.ID, &t.Code, &t.Date, &t.Time, &t.Amount.Amount, &t.PaymentMethod, &status, &t.Stylist, &stylistID,
			&customerName, &customerPhone, &customerEmail, &customerAddress, &visits, &lastVisit,
			&shiftID, &opName, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if stylistID.Valid {
			t.StylistID = &stylistID.Int64
		}
		t.Status = domain.TransactionStatus(status)
		t.Customer = &domain.TransactionCustomerSnapshot{
			Name:    customerName.String,
			Phone:   customerPhone.String,
			Email:   customerEmail.String,
			Address: customerAddress.String,
		}
		if visits.Valid {
			v := int(visits.Int32)
			t.Customer.Visits = &v
		}
		if lastVisit.Valid {
			lv := lastVisit.String
			t.Customer.LastVisit = &lv
		}
		ids = append(ids, t.ID)
		txs = append(txs, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return txs, nil
	}

	itemRows, err := r.DB.Pool.Query(ctx, `
		SELECT transaction_id, id, product_id, name, category, price, qty, created_at
		FROM transaction_items
		WHERE transaction_id = ANY($1)
	`, ids)
	if err != nil {
		return nil, err
	}
	defer itemRows.Close()

	itemsByTx := make(map[int64][]domain.TransactionItem)
	for itemRows.Next() {
		var it domain.TransactionItem
		var txID int64
		if err := itemRows.Scan(&txID, &it.ID, &it.ProductID, &it.Name, &it.Category, &it.Price.Amount, &it.Qty, &it.CreatedAt); err != nil {
			return nil, err
		}
		itemsByTx[txID] = append(itemsByTx[txID], it)
	}
	if err := itemRows.Err(); err != nil {
		return nil, err
	}

	for i := range txs {
		txs[i].Items = itemsByTx[txs[i].ID]
	}

	return txs, nil
}

func (r TransactionRepository) MarkPaidByCode(ctx context.Context, code string) error {
	ct, err := r.DB.Pool.Exec(ctx, `
		UPDATE transactions
		SET status='paid',
		    refunded_at=NULL,
		    refunded_by=NULL,
		    refund_note='',
		    deleted_at=NULL,
		    updated_at=now()
		WHERE code=$1
	`, code)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type RefundTransactionParams struct {
	Code       string
	Note       string
	Delete     bool
	RefundedBy *int64
}

type RefundItem struct {
	ProductID *int64
	Qty       int
}

func (r TransactionRepository) RefundByCode(ctx context.Context, in RefundTransactionParams, after func(context.Context, pgx.Tx, domain.Transaction, []RefundItem, int) error) error {
	tx, err := r.DB.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		SELECT id, code, transacted_date, transacted_time, amount, payment_method, status
		FROM transactions
		WHERE code=$1
		FOR UPDATE
	`, in.Code)
	var t domain.Transaction
	var status string
	if err := row.Scan(&t.ID, &t.Code, &t.Date, &t.Time, &t.Amount.Amount, &t.PaymentMethod, &status); err != nil {
		if err == pgx.ErrNoRows {
			return ErrNotFound
		}
		return err
	}
	t.Status = domain.TransactionStatus(status)

	itemRows, err := tx.Query(ctx, `
		SELECT product_id, qty
		FROM transaction_items
		WHERE transaction_id=$1 AND deleted_at IS NULL
	`, t.ID)
	if err != nil {
		return err
	}
	units := 0
	var items []RefundItem
	for itemRows.Next() {
		var productID pgtype.Int8
		var qty int
		if err := itemRows.Scan(&productID, &qty); err != nil {
			itemRows.Close()
			return err
		}
		units += qty
		var pid *int64
		if productID.Valid {
			pid = &productID.Int64
		}
		items = append(items, RefundItem{ProductID: pid, Qty: qty})
	}
	if err := itemRows.Err(); err != nil {
		itemRows.Close()
		return err
	}
	itemRows.Close()
	if units == 0 {
		units = 1
	}

	// Keep transaction for audit; default behavior from UI is "refund & delete".
	_, err = tx.Exec(ctx, `
		UPDATE transactions
		SET status='refund',
		    refunded_at=now(),
		    refunded_by=$1,
		    refund_note=$2,
		    deleted_at=CASE WHEN $3 THEN now() ELSE NULL END,
		    updated_at=now()
		WHERE id=$4
	`, in.RefundedBy, in.Note, in.Delete, t.ID)
	if err != nil {
		return err
	}

	if after != nil {
		if err := after(ctx, tx, t, items, units); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
