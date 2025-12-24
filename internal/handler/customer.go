package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/repository"
	"barberpos-backend/internal/server/authctx"
	"github.com/go-chi/chi/v5"
)

type CustomerHandler struct {
	Repo      repository.CustomerRepository
	Employees repository.EmployeeRepository
}

func (h CustomerHandler) RegisterRoutes(r chi.Router) {
	r.Get("/customers", h.list)
	r.Post("/customers", h.upsert)
	r.Delete("/customers/{id}", h.delete)
}

func (h CustomerHandler) list(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ownerID, err := resolveOwnerID(r.Context(), *user, h.Employees)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	items, err := h.Repo.List(r.Context(), ownerID, 500)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]map[string]any, 0, len(items))
	for _, c := range items {
		resp = append(resp, map[string]any{
			"id":      c.ID,
			"name":    c.Name,
			"phone":   c.Phone,
			"email":   c.Email,
			"address": c.Address,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h CustomerHandler) upsert(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ownerID, err := resolveOwnerID(r.Context(), *user, h.Employees)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		ID      *int64 `json:"id"`
		Name    string `json:"name"`
		Phone   string `json:"phone"`
		Email   string `json:"email"`
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.Name == "" || req.Phone == "" {
		writeError(w, http.StatusBadRequest, "name and phone are required")
		return
	}
	c := domain.Customer{
		ID:      0,
		Name:    req.Name,
		Phone:   req.Phone,
		Email:   req.Email,
		Address: req.Address,
	}
	if req.ID != nil {
		c.ID = *req.ID
	}
	saved, err := h.Repo.Upsert(r.Context(), ownerID, c)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":      saved.ID,
		"name":    saved.Name,
		"phone":   saved.Phone,
		"email":   saved.Email,
		"address": saved.Address,
	})
}

func (h CustomerHandler) delete(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ownerID, err := resolveOwnerID(r.Context(), *user, h.Employees)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.Repo.Delete(r.Context(), ownerID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
