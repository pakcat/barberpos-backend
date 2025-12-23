package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"barberpos-backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

type CategoryHandler struct {
	Repo repository.CategoryRepository
}

func (h CategoryHandler) RegisterRoutes(r chi.Router) {
	r.Get("/categories", h.list)
	r.Post("/categories", h.upsert)
	r.Delete("/categories/{id}", h.delete)
}

func (h CategoryHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.Repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
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
	c, err := h.Repo.Upsert(r.Context(), req.Name, req.ID)
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
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.Repo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
