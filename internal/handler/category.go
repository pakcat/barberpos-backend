package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"barberpos-backend/internal/repository"
	"barberpos-backend/internal/server/authctx"
	"github.com/go-chi/chi/v5"
)

type CategoryHandler struct {
	Repo      repository.CategoryRepository
	Employees repository.EmployeeRepository
}

func (h CategoryHandler) RegisterRoutes(r chi.Router) {
	r.Get("/categories", h.list)
	r.Post("/categories", h.upsert)
	r.Delete("/categories/{id}", h.delete)
}

func (h CategoryHandler) list(w http.ResponseWriter, r *http.Request) {
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

	items, err := h.Repo.List(r.Context(), ownerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(items) == 0 {
		_ = h.Repo.SeedDefaults(r.Context(), ownerID)
		if seeded, seedErr := h.Repo.List(r.Context(), ownerID); seedErr == nil {
			items = seeded
		}
	}
	resp := make([]map[string]any, 0, len(items))
	for _, c := range items {
		resp = append(resp, map[string]any{
			"id":   c.ID,
			"name": c.Name,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h CategoryHandler) upsert(w http.ResponseWriter, r *http.Request) {
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
		ID   *int64 `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	c, err := h.Repo.Upsert(r.Context(), ownerID, req.Name, req.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":   c.ID,
		"name": c.Name,
	})
}

func (h CategoryHandler) delete(w http.ResponseWriter, r *http.Request) {
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
