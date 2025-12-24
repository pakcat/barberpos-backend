package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/repository"
	"barberpos-backend/internal/server/authctx"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

type EmployeeHandler struct {
	Repo repository.EmployeeRepository
}

func (h EmployeeHandler) RegisterRoutes(r chi.Router) {
	r.Get("/employees", h.list)
	r.Post("/employees", h.upsert)
	r.Delete("/employees/{id}", h.delete)
}

func (h EmployeeHandler) list(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	items, err := h.Repo.List(r.Context(), user.ID, 500)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]map[string]any, 0, len(items))
	for _, e := range items {
		resp = append(resp, map[string]any{
			"id":            e.ID,
			"managerUserId": e.ManagerID,
			"name":          e.Name,
			"role":          e.Role,
			"phone":         e.Phone,
			"email":         e.Email,
			"joinDate":      e.JoinDate.Format("2006-01-02"),
			"commission":    e.Commission,
			"active":        e.Active,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h EmployeeHandler) upsert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID         *int64   `json:"id"`
		Name       string   `json:"name"`
		Role       string   `json:"role"`
		Phone      string   `json:"phone"`
		Email      string   `json:"email"`
		Pin        string   `json:"pin"`
		JoinDate   string   `json:"joinDate"`
		Commission *float64 `json:"commission"`
		Active     *bool    `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.ID == nil && req.Pin == "" {
		writeError(w, http.StatusBadRequest, "pin is required")
		return
	}
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	joinDate := time.Now()
	if req.JoinDate != "" {
		if t, err := time.Parse("2006-01-02", req.JoinDate); err == nil {
			joinDate = t
		}
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	e := domain.Employee{
		ManagerID:  &user.ID,
		Name:       req.Name,
		Role:       req.Role,
		Phone:      req.Phone,
		Email:      req.Email,
		JoinDate:   joinDate,
		Commission: req.Commission,
		Active:     active,
	}
	if req.ID != nil {
		e.ID = *req.ID
	}
	if req.Pin != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Pin), bcrypt.DefaultCost)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to hash pin")
			return
		}
		e.PinHash = ptr(string(hash))
	}
	saved, err := h.Repo.Save(r.Context(), user.ID, e)
	if err != nil {
		if err == repository.ErrNotFound {
			writeError(w, http.StatusNotFound, "employee not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":            saved.ID,
		"managerUserId": saved.ManagerID,
		"name":          saved.Name,
		"role":          saved.Role,
		"phone":         saved.Phone,
		"email":         saved.Email,
		"joinDate":      saved.JoinDate.Format("2006-01-02"),
		"commission":    saved.Commission,
		"active":        saved.Active,
	})
}

func (h EmployeeHandler) delete(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.Repo.Delete(r.Context(), user.ID, id); err != nil {
		if err == repository.ErrNotFound {
			writeError(w, http.StatusNotFound, "employee not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func ptr[T any](v T) *T { return &v }
