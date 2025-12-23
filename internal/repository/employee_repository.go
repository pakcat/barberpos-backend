package repository

import (
	"context"

	"barberpos-backend/internal/db"
	"barberpos-backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

type EmployeeRepository struct {
	DB *db.Postgres
}

func (r EmployeeRepository) List(ctx context.Context, limit int) ([]domain.Employee, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id, name, role, phone, email, join_date, commission, active, created_at, updated_at
		FROM employees
		WHERE deleted_at IS NULL
		ORDER BY name ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Employee
	for rows.Next() {
		var e domain.Employee
		if err := rows.Scan(&e.ID, &e.Name, &e.Role, &e.Phone, &e.Email, &e.JoinDate, &e.Commission, &e.Active, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, e)
	}
	return items, rows.Err()
}

func (r EmployeeRepository) Upsert(ctx context.Context, e domain.Employee) (*domain.Employee, error) {
	err := r.DB.Pool.QueryRow(ctx, `
		INSERT INTO employees (id, name, role, phone, email, join_date, commission, active, created_at, updated_at)
		VALUES (COALESCE($1, nextval('employees_id_seq')), $2,$3,$4,$5,$6,$7,$8, now(), now())
		ON CONFLICT (id) DO UPDATE SET
			name=EXCLUDED.name,
			role=EXCLUDED.role,
			phone=EXCLUDED.phone,
			email=EXCLUDED.email,
			join_date=EXCLUDED.join_date,
			commission=EXCLUDED.commission,
			active=EXCLUDED.active,
			updated_at=now(),
			deleted_at=NULL
		RETURNING id, name, role, phone, email, join_date, commission, active, created_at, updated_at
	`, nullableID(e.ID), e.Name, e.Role, e.Phone, e.Email, e.JoinDate, e.Commission, e.Active).
		Scan(&e.ID, &e.Name, &e.Role, &e.Phone, &e.Email, &e.JoinDate, &e.Commission, &e.Active, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r EmployeeRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.DB.Pool.Exec(ctx, `UPDATE employees SET deleted_at = now() WHERE id=$1`, id)
	return err
}

func (r EmployeeRepository) Get(ctx context.Context, id int64) (*domain.Employee, error) {
	row := r.DB.Pool.QueryRow(ctx, `
		SELECT id, name, role, phone, email, join_date, commission, active, created_at, updated_at
		FROM employees
		WHERE id=$1 AND deleted_at IS NULL
	`, id)
	var e domain.Employee
	if err := row.Scan(&e.ID, &e.Name, &e.Role, &e.Phone, &e.Email, &e.JoinDate, &e.Commission, &e.Active, &e.CreatedAt, &e.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &e, nil
}
