package handler

import (
	"net/http"
	"strconv"

	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

type ProductHandler struct {
	Repo     repository.ProductRepository
	Currency string
}

func (h ProductHandler) RegisterRoutes(r chi.Router) {
	r.Get("/products", h.listProducts)
	r.Get("/services", h.listServices)
}

func (h ProductHandler) listProducts(w http.ResponseWriter, r *http.Request) {
	items, err := h.Repo.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, toProductResponses(items))
}

// For now services are the same as products (haircut/services catalog).
func (h ProductHandler) listServices(w http.ResponseWriter, r *http.Request) {
	h.listProducts(w, r)
}

func toProductResponses(items []domain.Product) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, p := range items {
		out = append(out, map[string]any{
			"id":         strconv.FormatInt(p.ID, 10),
			"name":       p.Name,
			"category":   p.Category,
			"price":      p.Price.Amount,
			"image":      p.Image,
			"trackStock": p.TrackStock,
			"stock":      p.Stock,
			"minStock":   p.MinStock,
		})
	}
	return out
}
