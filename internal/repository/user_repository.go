package repository

import (
	"context"
	"errors"
	"time"

	"barberpos-backend/internal/db"
	"barberpos-backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

type UserRepository struct {
	DB *db.Postgres
}

type CreateUserParams struct {
	Name         string
	Email        string
	Phone        string
	Address      string
	Region       string
	Role         domain.UserRole
	PasswordHash *string
	IsGoogle     bool
}

func (r UserRepository) Create(ctx context.Context, p CreateUserParams) (*domain.User, error) {
	query := `
		INSERT INTO users (name, email, phone, address, region, role, password_hash, is_google, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8, now(), now())
		RETURNING id, name, email, phone, address, region, role, is_google, password_hash, created_at, updated_at
	`
	row := r.DB.Pool.QueryRow(ctx, query, p.Name, p.Email, p.Phone, p.Address, p.Region, p.Role, p.PasswordHash, p.IsGoogle)
	return scanUser(row)
}

func (r UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, name, email, phone, address, region, role, is_google, password_hash, created_at, updated_at
		FROM users
		WHERE email=$1 AND deleted_at IS NULL
	`
	row := r.DB.Pool.QueryRow(ctx, query, email)
	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

func (r UserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	query := `
		SELECT id, name, email, phone, address, region, role, is_google, password_hash, created_at, updated_at
		FROM users
		WHERE id=$1 AND deleted_at IS NULL
	`
	row := r.DB.Pool.QueryRow(ctx, query, id)
	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

type EmailAccountInfo struct {
	IsGoogle bool
	Deleted  bool
}

func (r UserRepository) GetEmailAccountInfo(ctx context.Context, email string) (*EmailAccountInfo, error) {
	var isGoogle bool
	var deletedAt *time.Time
	err := r.DB.Pool.QueryRow(ctx, `
		SELECT is_google, deleted_at
		FROM users
		WHERE lower(email) = lower($1)
		LIMIT 1
	`, email).Scan(&isGoogle, &deletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &EmailAccountInfo{
		IsGoogle: isGoogle,
		Deleted:  deletedAt != nil,
	}, nil
}

func (r UserRepository) UpdatePassword(ctx context.Context, id int64, hash string) error {
	_, err := r.DB.Pool.Exec(ctx, `
		UPDATE users
		SET password_hash=$1, updated_at=now()
		WHERE id=$2 AND deleted_at IS NULL
	`, hash, id)
	return err
}

func scanUser(row interface {
	Scan(dest ...any) error
}) (*domain.User, error) {
	var (
		u    domain.User
		role string
	)
	if err := row.Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.Phone,
		&u.Address,
		&u.Region,
		&role,
		&u.IsGoogle,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	); err != nil {
		return nil, err
	}
	u.Role = domain.UserRole(role)
	return &u, nil
}

// ErrNotFound is returned when a record does not exist.
var ErrNotFound = errors.New("not found")

// IsDuplicate detects unique constraint violation.
func IsDuplicate(err error) bool {
	return db.IsUniqueViolation(err)
}
