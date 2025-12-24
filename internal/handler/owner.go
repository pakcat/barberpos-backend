package handler

import (
	"context"
	"errors"

	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/repository"
	"barberpos-backend/internal/server/authctx"
)

func resolveOwnerID(ctx context.Context, user authctx.CurrentUser, employees repository.EmployeeRepository) (int64, error) {
	switch user.Role {
	case domain.RoleManager, domain.RoleAdmin:
		return user.ID, nil
	case domain.RoleStaff:
		if user.Email == "" {
			return 0, errors.New("staff email is required")
		}
		emp, err := employees.GetByEmail(ctx, user.Email)
		if err != nil {
			return 0, errors.New("employee not found")
		}
		if emp.ManagerID == nil {
			return 0, errors.New("employee has no manager")
		}
		return *emp.ManagerID, nil
	default:
		return 0, errors.New("invalid role")
	}
}

