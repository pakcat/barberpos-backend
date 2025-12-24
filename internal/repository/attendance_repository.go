package repository

import (
	"context"
	"time"

	"barberpos-backend/internal/db"
	"barberpos-backend/internal/domain"
)

type AttendanceRepository struct {
	DB *db.Postgres
}

func (r AttendanceRepository) CheckIn(ctx context.Context, ownerUserID int64, name string, employeeID *int64, date time.Time, source string) error {
	if source == "" {
		source = "cashier"
	}
	day := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	_, err := r.DB.Pool.Exec(ctx, `
		INSERT INTO attendance (owner_user_id, employee_id, employee_name, attendance_date, check_in, status, source, created_at)
		VALUES ($1,$2,$3,$4, now(), 'present', $5, now())
		ON CONFLICT (owner_user_id, employee_name, attendance_date)
		DO UPDATE SET check_in = EXCLUDED.check_in, status = EXCLUDED.status, source = EXCLUDED.source
	`, ownerUserID, employeeID, name, day, source)
	return err
}

func (r AttendanceRepository) CheckOut(ctx context.Context, ownerUserID int64, name string, employeeID *int64, date time.Time, source string) error {
	if source == "" {
		source = "cashier"
	}
	day := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	_, err := r.DB.Pool.Exec(ctx, `
		INSERT INTO attendance (owner_user_id, employee_id, employee_name, attendance_date, check_out, status, source, created_at)
		VALUES ($1,$2,$3,$4, now(), 'present', $5, now())
		ON CONFLICT (owner_user_id, employee_name, attendance_date)
		DO UPDATE SET check_out = EXCLUDED.check_out, source = EXCLUDED.source
	`, ownerUserID, employeeID, name, day, source)
	return err
}

func (r AttendanceRepository) GetMonth(ctx context.Context, ownerUserID int64, name string, month time.Time) ([]domain.Attendance, error) {
	start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id, employee_id, employee_name, attendance_date, check_in, check_out, status, source, created_at
		FROM attendance
		WHERE owner_user_id=$1 AND employee_name = $2
		  AND attendance_date >= $3
		  AND attendance_date < $3 + interval '1 month'
		ORDER BY attendance_date ASC
	`, ownerUserID, name, start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Attendance
	for rows.Next() {
		var a domain.Attendance
		var status string
		if err := rows.Scan(&a.ID, &a.EmployeeID, &a.EmployeeName, &a.Date, &a.CheckIn, &a.CheckOut, &status, &a.Source, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.Status = domain.AttendanceStatus(status)
		items = append(items, a)
	}
	return items, rows.Err()
}

func (r AttendanceRepository) GetDaily(ctx context.Context, ownerUserID int64, date time.Time) ([]domain.Attendance, error) {
	day := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id, employee_id, employee_name, attendance_date, check_in, check_out, status, source, created_at
		FROM attendance
		WHERE owner_user_id=$1 AND attendance_date = $2
		ORDER BY employee_name ASC
	`, ownerUserID, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Attendance
	for rows.Next() {
		var a domain.Attendance
		var status string
		if err := rows.Scan(&a.ID, &a.EmployeeID, &a.EmployeeName, &a.Date, &a.CheckIn, &a.CheckOut, &status, &a.Source, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.Status = domain.AttendanceStatus(status)
		items = append(items, a)
	}
	return items, rows.Err()
}
