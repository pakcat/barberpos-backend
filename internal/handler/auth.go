package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/server/authctx"
	"barberpos-backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type AuthHandler struct {
	Service *service.AuthService
}

func (h AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/auth/register", h.register)
	r.Post("/auth/login", h.login)
	r.Post("/auth/google", h.loginGoogle)
	r.Post("/auth/staff", h.loginStaff)
	r.Post("/auth/refresh", h.refresh)
	r.Post("/auth/forgot-password", h.forgotPassword)
	r.Post("/auth/reset-password", h.resetPassword)
}

func (h AuthHandler) RegisterProtectedRoutes(r chi.Router) {
	r.Post("/auth/change-password", h.changePassword)
}

func (h AuthHandler) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Phone    string `json:"phone"`
		Address  string `json:"address"`
		Region   string `json:"region"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	res, err := h.Service.Register(r.Context(), service.RegisterInput{
		Name:     req.Name,
		Email:    strings.ToLower(req.Email),
		Password: req.Password,
		Phone:    req.Phone,
		Address:  req.Address,
		Region:   req.Region,
		Role:     domain.UserRole(req.Role),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeAuthResponse(w, res)
}

func (h AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	res, err := h.Service.Login(r.Context(), service.LoginInput{
		Email:    strings.ToLower(req.Email),
		Password: req.Password,
	})
	if err != nil {
		status := http.StatusUnauthorized
		if err == service.ErrInvalidCredentials {
			status = http.StatusUnauthorized
		}
		writeError(w, status, err.Error())
		return
	}
	writeAuthResponse(w, res)
}

func (h AuthHandler) loginGoogle(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDToken string `json:"idToken"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Phone   string `json:"phone"`
		Address string `json:"address"`
		Region  string `json:"region"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	res, err := h.Service.LoginWithGoogle(r.Context(), service.GoogleLoginInput{
		IDToken: req.IDToken,
		Email:   strings.ToLower(req.Email),
		Name:    req.Name,
		Phone:   req.Phone,
		Address: req.Address,
		Region:  req.Region,
	})
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	writeAuthResponse(w, res)
}

// loginStaff authenticates active employees (stylists) by phone or email (no password yet).
func (h AuthHandler) loginStaff(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phone string `json:"phone"`
		Email string `json:"email"`
		Name  string `json:"name"`
		Pin   string `json:"pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	res, err := h.Service.LoginEmployee(r.Context(), service.EmployeeLoginInput{
		Phone: req.Phone,
		Email: strings.ToLower(req.Email),
		Name:  req.Name,
		Pin:   req.Pin,
	})
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	writeAuthResponse(w, res)
}

func (h AuthHandler) refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	res, err := h.Service.Refresh(r.Context(), service.RefreshInput{RefreshToken: req.RefreshToken})
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	writeAuthResponse(w, res)
}

func (h AuthHandler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	code, _ := h.Service.ForgotPassword(r.Context(), strings.ToLower(req.Email))
	writeJSON(w, http.StatusOK, map[string]string{"code": code})
}

func (h AuthHandler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := h.Service.ResetPassword(r.Context(), req.Token, req.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h AuthHandler) changePassword(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.NewPassword == "" || req.CurrentPassword == "" {
		writeError(w, http.StatusBadRequest, "currentPassword and newPassword are required")
		return
	}
	if err := h.Service.ChangePassword(r.Context(), user.ID, req.CurrentPassword, req.NewPassword); err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func writeAuthResponse(w http.ResponseWriter, res *service.AuthResult) {
	userPayload := map[string]any{
		"id":       strconv.FormatInt(res.User.ID, 10),
		"name":     res.User.Name,
		"email":    res.User.Email,
		"role":     string(res.User.Role),
		"phone":    res.User.Phone,
		"address":  res.User.Address,
		"region":   res.User.Region,
		"isGoogle": res.User.IsGoogle,
	}
	if len(res.Permissions) > 0 {
		userPayload["permissions"] = res.Permissions
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token":        res.AccessToken,
		"refreshToken": res.RefreshToken,
		"expiresAt":    res.ExpiresAt.UTC().Format(time.RFC3339),
		"user":         userPayload,
	})
}
