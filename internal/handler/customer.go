package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

type CustomerHandler struct {
	Repo repository.CustomerRepository
}

func (h CustomerHandler) RegisterRoutes(r chi.Router) {
	r.Get("/customers", h.list)
	r.Post("/customers", h.upsert)
	r.Delete("/customers/{id}", h.delete)
}

func (h CustomerHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.Repo.List(r.Context(), 500)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	var req struct {
		ID      *int64 `json:"id"`
		Name    string `json:"name"`
		Phone   string `json:"phone"`
		Email   string `json:"email"`
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Phone == "" {
		http.Error(w, "name and phone are required", http.StatusBadRequest)
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
	saved, err := h.Repo.Upsert(r.Context(), c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.Repo.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
