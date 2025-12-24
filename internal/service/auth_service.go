package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"barberpos-backend/internal/config"
	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/repository"
	fbauth "firebase.google.com/go/v4/auth"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

type AuthService struct {
	Config       config.Config
	Users        repository.UserRepository
	Employees    repository.EmployeeRepository
	Logger       *slog.Logger
	FirebaseAuth *fbauth.Client
}

type AuthResult struct {
	AccessToken  string
	RefreshToken string
	User         domain.User
	ExpiresAt    time.Time
}

type RegisterInput struct {
	Name     string
	Email    string
	Password string
	Phone    string
	Address  string
	Region   string
	Role     domain.UserRole
}

type LoginInput struct {
	Email    string
	Password string
}

type GoogleLoginInput struct {
	IDToken string
	Email   string
	Name    string
	Phone   string
	Address string
	Region  string
}

type EmployeeLoginInput struct {
	Phone string
	Email string
	Name  string
	Pin   string
}

type RefreshInput struct {
	RefreshToken string
}

func (s AuthService) Register(ctx context.Context, in RegisterInput) (*AuthResult, error) {
	if in.Role == "" {
		in.Role = domain.RoleManager
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	user, err := s.Users.Create(ctx, repository.CreateUserParams{
		Name:         in.Name,
		Email:        in.Email,
		Phone:        in.Phone,
		Address:      in.Address,
		Region:       in.Region,
		Role:         in.Role,
		PasswordHash: ptr(string(hash)),
		IsGoogle:     false,
	})
	if err != nil {
		if repository.IsDuplicate(err) {
			if info, infoErr := s.Users.GetEmailAccountInfo(ctx, in.Email); infoErr == nil && info.IsGoogle {
				return nil, fmt.Errorf("email already used, please login with Google")
			}
			return nil, fmt.Errorf("email already used")
		}
		return nil, err
	}
	return s.issueTokens(user)
}

func (s AuthService) Login(ctx context.Context, in LoginInput) (*AuthResult, error) {
	user, err := s.Users.GetByEmail(ctx, in.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if user.PasswordHash == nil {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(in.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return s.issueTokens(user)
}

func (s AuthService) LoginWithGoogle(ctx context.Context, in GoogleLoginInput) (*AuthResult, error) {
	// Prefer Firebase Auth verification if available; otherwise fallback to Google ID token validation when client ID provided.
	switch {
	case s.FirebaseAuth != nil:
		if _, err := s.FirebaseAuth.VerifyIDToken(ctx, in.IDToken); err != nil {
			return nil, fmt.Errorf("firebase token invalid: %w", err)
		}
	case s.Config.GoogleClientID != "":
		if _, err := idtoken.Validate(ctx, in.IDToken, s.Config.GoogleClientID); err != nil {
			return nil, fmt.Errorf("google token invalid: %w", err)
		}
	}

	user, err := s.Users.GetByEmail(ctx, in.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// create new user
			user, err = s.Users.Create(ctx, repository.CreateUserParams{
				Name:         in.Name,
				Email:        in.Email,
				Phone:        in.Phone,
				Address:      in.Address,
				Region:       in.Region,
				Role:         domain.RoleManager,
				PasswordHash: nil,
				IsGoogle:     true,
			})
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return s.issueTokens(user)
}

// LoginEmployee authenticates active employees (stylist/staff) by phone or email and issues staff tokens.
// This avoids password/PIN for now; can be extended with a PIN column when needed.
func (s AuthService) LoginEmployee(ctx context.Context, in EmployeeLoginInput) (*AuthResult, error) {
	if (in.Phone == "" && in.Email == "") || in.Pin == "" {
		return nil, ErrInvalidCredentials
	}
	emp, err := s.Employees.GetByPhoneOrEmail(ctx, in.Phone, in.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if !emp.Active {
		return nil, ErrInvalidCredentials
	}
	if emp.PinHash == nil || bcrypt.CompareHashAndPassword([]byte(*emp.PinHash), []byte(in.Pin)) != nil {
		return nil, ErrInvalidCredentials
	}

	// Ensure a corresponding user exists so refresh tokens keep working.
	userEmail := strings.ToLower(strings.TrimSpace(emp.Email))
	if userEmail == "" {
		phone := strings.TrimSpace(emp.Phone)
		if phone == "" {
			userEmail = fmt.Sprintf("staff-%d@barberpos.local", emp.ID)
		} else {
			// Keep only digits so the derived email is stable and unique-ish per phone number.
			digits := make([]rune, 0, len(phone))
			for _, r := range phone {
				if r >= '0' && r <= '9' {
					digits = append(digits, r)
				}
			}
			if len(digits) == 0 {
				userEmail = fmt.Sprintf("staff-%d@barberpos.local", emp.ID)
			} else {
				userEmail = fmt.Sprintf("staff+%s@barberpos.local", string(digits))
			}
		}
	}
	user, err := s.Users.GetByEmail(ctx, userEmail)
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return nil, err
		}
		user, err = s.Users.Create(ctx, repository.CreateUserParams{
			Name:         emp.Name,
			Email:        userEmail,
			Phone:        emp.Phone,
			Address:      "", // employees table has no address; leave blank
			Region:       "",
			Role:         domain.RoleStaff,
			PasswordHash: emp.PinHash,
			IsGoogle:     false,
		})
		if err != nil {
			if repository.IsDuplicate(err) {
				user, err = s.Users.GetByEmail(ctx, userEmail)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
	}

	// Backfill password_hash if an existing staff user was created without it (older bug).
	if user.PasswordHash == nil && emp.PinHash != nil {
		_ = s.Users.UpdatePassword(ctx, user.ID, *emp.PinHash)
		user.PasswordHash = emp.PinHash
	}
	user.Role = domain.RoleStaff // enforce staff role for employee login
	return s.issueTokens(user)
}

func (s AuthService) Refresh(ctx context.Context, in RefreshInput) (*AuthResult, error) {
	token, err := jwt.Parse(in.RefreshToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.Config.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}
	if claims["token_type"] != "refresh" {
		return nil, ErrInvalidToken
	}
	sub, ok := claims["sub"].(string)
	if !ok {
		return nil, ErrInvalidToken
	}
	userID, err := strconv.ParseInt(sub, 10, 64)
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.Users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}
	return s.issueTokens(user)
}

func (s AuthService) ForgotPassword(ctx context.Context, email string) (string, error) {
	// Always succeed; return static code as requested.
	_, _ = s.Users.GetByEmail(ctx, email) // attempt to check existence; ignore error for privacy.
	return "1234", nil
}

func (s AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	if token != "1234" {
		return ErrInvalidToken
	}
	// Without a user identifier, we cannot update a specific account; act as dummy success.
	// If you want to update a user, extend payload to include email and set the password there.
	_ = newPassword
	return nil
}

func (s AuthService) ChangePassword(ctx context.Context, userID int64, current, next string) error {
	if next == "" {
		return errors.New("new password is required")
	}
	user, err := s.Users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.PasswordHash == nil {
		return ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(current)); err != nil {
		return ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return s.Users.UpdatePassword(ctx, userID, string(hash))
}

func (s AuthService) issueTokens(user *domain.User) (*AuthResult, error) {
	now := time.Now()
	accessExp := now.Add(s.Config.AccessTokenTTL)
	refreshExp := now.Add(s.Config.RefreshTokenTTL)

	access, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":        fmt.Sprintf("%d", user.ID),
		"email":      user.Email,
		"role":       user.Role,
		"token_type": "access",
		"exp":        accessExp.Unix(),
		"iat":        now.Unix(),
	}).SignedString([]byte(s.Config.JWTSecret))
	if err != nil {
		return nil, err
	}

	refresh, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":        fmt.Sprintf("%d", user.ID),
		"token_type": "refresh",
		"exp":        refreshExp.Unix(),
		"iat":        now.Unix(),
	}).SignedString([]byte(s.Config.JWTSecret))
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  access,
		RefreshToken: refresh,
		User:         *user,
		ExpiresAt:    accessExp,
	}, nil
}

func ptr[T any](v T) *T { return &v }
