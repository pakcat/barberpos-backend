package server

import (
	"net/http"
	"strconv"
	"strings"

	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/server/authctx"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware validates JWT and sets current user in context.
func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
				writeAuthError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}
			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				writeAuthError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok || claims["token_type"] != "access" {
				writeAuthError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			sub, _ := claims["sub"].(string)
			email, _ := claims["email"].(string)
			roleStr, _ := claims["role"].(string)
			id, err := strconv.ParseInt(sub, 10, 64)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "invalid subject")
				return
			}
			ctx := authctx.WithCurrentUser(r.Context(), authctx.CurrentUser{
				ID:    id,
				Email: email,
				Role:  domain.UserRole(roleStr),
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole ensures the user has one of the allowed roles.
func RequireRole(roles ...domain.UserRole) func(http.Handler) http.Handler {
	allowed := make(map[domain.UserRole]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := authctx.FromContext(r.Context())
			if u == nil {
				writeAuthError(w, http.StatusForbidden, "forbidden")
				return
			}
			if len(allowed) == 0 {
				next.ServeHTTP(w, r)
				return
			}
			if _, ok := allowed[u.Role]; !ok {
				writeAuthError(w, http.StatusForbidden, "forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + http.StatusText(status) + `","message":"` + message + `"}`))
}
