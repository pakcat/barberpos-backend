package handler

import (
	"net/http"

	"barberpos-backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

type RegionHandler struct {
	Repo repository.RegionRepository
}

func (h RegionHandler) RegisterRoutes(r chi.Router) {
	r.Get("/regions", h.list)
}

func (h RegionHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.Repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(items) == 0 {
		_ = h.Repo.SeedDefaults(r.Context())
		if seeded, seedErr := h.Repo.List(r.Context()); seedErr == nil {
			items = seeded
		}
	}
	resp := make([]map[string]any, 0, len(items))
	for _, region := range items {
		resp = append(resp, map[string]any{
			"id":   region.ID,
			"name": region.Name,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}
