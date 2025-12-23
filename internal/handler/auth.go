package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"barberpos-backend/internal/domain"
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
	r.Post("/auth/refresh", h.refresh)
	r.Post("/auth/forgot-password", h.forgotPassword)
	r.Post("/auth/reset-password", h.resetPassword)
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
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeAuthResponse(w http.ResponseWriter, res *service.AuthResult) {
	writeJSON(w, http.StatusOK, map[string]any{
		"token":        res.AccessToken,
		"refreshToken": res.RefreshToken,
		"expiresAt":    res.ExpiresAt.UTC().Format(time.RFC3339),
		"user": map[string]any{
			"id":       strconv.FormatInt(res.User.ID, 10),
			"name":     res.User.Name,
			"email":    res.User.Email,
			"role":     string(res.User.Role),
			"phone":    res.User.Phone,
			"address":  res.User.Address,
			"region":   res.User.Region,
			"isGoogle": res.User.IsGoogle,
		},
	})
}
