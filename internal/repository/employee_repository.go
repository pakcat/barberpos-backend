package repository

import (
	"context"

	"barberpos-backend/internal/db"
	"barberpos-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type EmployeeRepository struct {
	DB *db.Postgres
}

func (r EmployeeRepository) List(ctx context.Context, managerUserID int64, limit int) ([]domain.Employee, error) {
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id, manager_user_id, name, role, allowed_modules, phone, email, pin_hash, join_date, commission, active, created_at, updated_at
		FROM employees
		WHERE deleted_at IS NULL AND manager_user_id=$1
		ORDER BY name ASC
		LIMIT $2
	`, managerUserID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Employee
	for rows.Next() {
		var e domain.Employee
		var managerID pgtype.Int8
		var modules []string
		if err := rows.Scan(&e.ID, &managerID, &e.Name, &e.Role, &modules, &e.Phone, &e.Email, &e.PinHash, &e.JoinDate, &e.Commission, &e.Active, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		if managerID.Valid {
			e.ManagerID = &managerID.Int64
		}
		e.AllowedModules = modules
		items = append(items, e)
	}
	return items, rows.Err()
}

func (r EmployeeRepository) GetByPhoneOrEmail(ctx context.Context, phone, email string) (*domain.Employee, error) {
	row := r.DB.Pool.QueryRow(ctx, `
		SELECT id, manager_user_id, name, role, allowed_modules, phone, email, pin_hash, join_date, commission, active, created_at, updated_at
		FROM employees
		WHERE deleted_at IS NULL AND (
			(phone <> '' AND phone = $1) OR
			(email <> '' AND lower(email) = lower($2))
		)
		ORDER BY active DESC, id ASC
		LIMIT 1
	`, phone, email)
	var e domain.Employee
	var managerID pgtype.Int8
	var modules []string
	if err := row.Scan(&e.ID, &managerID, &e.Name, &e.Role, &modules, &e.Phone, &e.Email, &e.PinHash, &e.JoinDate, &e.Commission, &e.Active, &e.CreatedAt, &e.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if managerID.Valid {
		e.ManagerID = &managerID.Int64
	}
	e.AllowedModules = modules
	return &e, nil
}

func (r EmployeeRepository) GetByEmail(ctx context.Context, email string) (*domain.Employee, error) {
	row := r.DB.Pool.QueryRow(ctx, `
		SELECT id, manager_user_id, name, role, allowed_modules, phone, email, pin_hash, join_date, commission, active, created_at, updated_at
		FROM employees
		WHERE deleted_at IS NULL AND email <> '' AND lower(email) = lower($1)
		ORDER BY active DESC, id ASC
		LIMIT 1
	`, email)
	var e domain.Employee
	var managerID pgtype.Int8
	var modules []string
	if err := row.Scan(&e.ID, &managerID, &e.Name, &e.Role, &modules, &e.Phone, &e.Email, &e.PinHash, &e.JoinDate, &e.Commission, &e.Active, &e.CreatedAt, &e.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if managerID.Valid {
		e.ManagerID = &managerID.Int64
	}
	e.AllowedModules = modules
	return &e, nil
}

func (r EmployeeRepository) Save(ctx context.Context, managerUserID int64, e domain.Employee) (*domain.Employee, error) {
	modulesArg := e.AllowedModules
	if modulesArg == nil {
		modulesArg = []string{}
	}
	var row pgx.Row
	if e.ID == 0 {
		row = r.DB.Pool.QueryRow(ctx, `
			INSERT INTO employees (manager_user_id, name, role, allowed_modules, phone, email, pin_hash, join_date, commission, active, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, now(), now())
			RETURNING id, manager_user_id, name, role, allowed_modules, phone, email, pin_hash, join_date, commission, active, created_at, updated_at
		`, managerUserID, e.Name, e.Role, modulesArg, e.Phone, e.Email, e.PinHash, e.JoinDate, e.Commission, e.Active)
	} else {
		row = r.DB.Pool.QueryRow(ctx, `
			UPDATE employees
			SET name=$1,
				role=$2,
				allowed_modules=$3,
				phone=$4,
				email=$5,
				pin_hash=COALESCE($6, employees.pin_hash),
				join_date=$7,
				commission=$8,
				active=$9,
				updated_at=now(),
				deleted_at=NULL
			WHERE id=$10 AND manager_user_id=$11
			RETURNING id, manager_user_id, name, role, allowed_modules, phone, email, pin_hash, join_date, commission, active, created_at, updated_at
		`, e.Name, e.Role, modulesArg, e.Phone, e.Email, e.PinHash, e.JoinDate, e.Commission, e.Active, e.ID, managerUserID)
	}
	var managerID pgtype.Int8
	var modules []string
	if err := row.Scan(&e.ID, &managerID, &e.Name, &e.Role, &modules, &e.Phone, &e.Email, &e.PinHash, &e.JoinDate, &e.Commission, &e.Active, &e.CreatedAt, &e.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if managerID.Valid {
		e.ManagerID = &managerID.Int64
	}
	e.AllowedModules = modules
	return &e, nil
}

func (r EmployeeRepository) Delete(ctx context.Context, managerUserID int64, id int64) error {
	ct, err := r.DB.Pool.Exec(ctx, `UPDATE employees SET deleted_at = now() WHERE id=$1 AND manager_user_id=$2`, id, managerUserID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r EmployeeRepository) Get(ctx context.Context, managerUserID int64, id int64) (*domain.Employee, error) {
	row := r.DB.Pool.QueryRow(ctx, `
		SELECT id, manager_user_id, name, role, allowed_modules, phone, email, pin_hash, join_date, commission, active, created_at, updated_at
		FROM employees
		WHERE id=$1 AND manager_user_id=$2 AND deleted_at IS NULL
	`, id, managerUserID)
	var e domain.Employee
	var managerID pgtype.Int8
	var modules []string
	if err := row.Scan(&e.ID, &managerID, &e.Name, &e.Role, &modules, &e.Phone, &e.Email, &e.PinHash, &e.JoinDate, &e.Commission, &e.Active, &e.CreatedAt, &e.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if managerID.Valid {
		e.ManagerID = &managerID.Int64
	}
	e.AllowedModules = modules
	return &e, nil
}
