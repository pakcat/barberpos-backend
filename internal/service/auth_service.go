package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
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
