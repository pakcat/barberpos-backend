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

func (r AttendanceRepository) CheckIn(ctx context.Context, name string, employeeID *int64) error {
	today := time.Now().Format("2006-01-02")
	_, err := r.DB.Pool.Exec(ctx, `
		INSERT INTO attendance (employee_id, employee_name, attendance_date, check_in, status, source, created_at)
		VALUES ($1,$2,$3, now(), 'present', 'cashier', now())
		ON CONFLICT (employee_name, attendance_date)
		DO UPDATE SET check_in = EXCLUDED.check_in, status = EXCLUDED.status
	`, employeeID, name, today)
	return err
}

func (r AttendanceRepository) CheckOut(ctx context.Context, name string, employeeID *int64) error {
	today := time.Now().Format("2006-01-02")
	_, err := r.DB.Pool.Exec(ctx, `
		INSERT INTO attendance (employee_id, employee_name, attendance_date, check_out, status, source, created_at)
		VALUES ($1,$2,$3, now(), 'present', 'cashier', now())
		ON CONFLICT (employee_name, attendance_date)
		DO UPDATE SET check_out = EXCLUDED.check_out
	`, employeeID, name, today)
	return err
}

func (r AttendanceRepository) GetMonth(ctx context.Context, name string, month time.Time) ([]domain.Attendance, error) {
	start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	rows, err := r.DB.Pool.Query(ctx, `
		SELECT id, employee_id, employee_name, attendance_date, check_in, check_out, status, source, created_at
		FROM attendance
		WHERE employee_name = $1
		  AND attendance_date >= $2
		  AND attendance_date < $2 + interval '1 month'
		ORDER BY attendance_date ASC
	`, name, start)
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
