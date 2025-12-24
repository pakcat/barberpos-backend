package handler

import (
	"net/http"
	"strconv"

	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/repository"
	"barberpos-backend/internal/server/authctx"
	"github.com/go-chi/chi/v5"
)

type ProductHandler struct {
	Repo     repository.ProductRepository
	Employees repository.EmployeeRepository
	Currency string
}

func (h ProductHandler) RegisterRoutes(r chi.Router) {
	r.Get("/products", h.listProducts)
	r.Get("/services", h.listServices)
}

func (h ProductHandler) listProducts(w http.ResponseWriter, r *http.Request) {
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
		// Best-effort: seed a minimal default catalog so the app isn't empty on fresh installs.
		// If it fails (e.g., read-only DB), just return the empty list.
		_ = h.Repo.SeedDefaults(r.Context(), ownerID)
		if seeded, seedErr := h.Repo.List(r.Context(), ownerID); seedErr == nil {
			items = seeded
		}
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
