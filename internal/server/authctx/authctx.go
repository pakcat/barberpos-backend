package authctx

import (
	"context"

	"barberpos-backend/internal/domain"
)

type contextKey string

const userContextKey contextKey = "currentUser"

type CurrentUser struct {
	ID    int64
	Email string
	Role  domain.UserRole
}

func WithCurrentUser(ctx context.Context, user CurrentUser) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func FromContext(ctx context.Context) *CurrentUser {
	val, ok := ctx.Value(userContextKey).(CurrentUser)
	if !ok {
		return nil
	}
	return &val
}
