package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

type ProductAdminHandler struct {
	Repo repository.ProductRepository
}

func (h ProductAdminHandler) RegisterRoutes(r chi.Router) {
	r.Post("/products", h.upsert)
	r.Delete("/products/{id}", h.delete)
}

func (h ProductAdminHandler) upsert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID         *int64 `json:"id"`
		Name       string `json:"name"`
		Category   string `json:"category"`
		Price      int64  `json:"price"`
		Image      string `json:"image"`
		TrackStock bool   `json:"trackStock"`
		Stock      int    `json:"stock"`
		MinStock   int    `json:"minStock"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	p := domain.Product{
		Name:       req.Name,
		Category:   req.Category,
		Price:      domain.Money{Amount: req.Price},
		Image:      req.Image,
		TrackStock: req.TrackStock,
		Stock:      req.Stock,
		MinStock:   req.MinStock,
	}
	if req.ID != nil {
		p.ID = *req.ID
	}
	saved, err := h.Repo.Upsert(r.Context(), p)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":         saved.ID,
		"name":       saved.Name,
		"category":   saved.Category,
		"price":      saved.Price.Amount,
		"image":      saved.Image,
		"trackStock": saved.TrackStock,
		"stock":      saved.Stock,
		"minStock":   saved.MinStock,
	})
}

func (h ProductAdminHandler) delete(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
