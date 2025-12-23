package handler

import (
	"net/http"
	"strings"

	"barberpos-backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

type DashboardHandler struct {
	Repo repository.DashboardRepository
}

func (h DashboardHandler) RegisterRoutes(r chi.Router) {
	r.Get("/dashboard/summary", h.summary)
	r.Get("/dashboard/top-services", h.topServices)
	r.Get("/dashboard/top-staff", h.topStaff)
	r.Get("/dashboard/sales", h.sales)
}

func (h DashboardHandler) summary(w http.ResponseWriter, r *http.Request) {
	data, err := h.Repo.Summary(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"totalRevenue":      data.TotalRevenue,
		"totalTransactions": data.TotalTransactions,
		"todayRevenue":      data.TodayRevenue,
	})
}

func (h DashboardHandler) topServices(w http.ResponseWriter, r *http.Request) {
	items, err := h.Repo.TopServices(r.Context(), 5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, toDashboardItems(items))
}

func (h DashboardHandler) topStaff(w http.ResponseWriter, r *http.Request) {
	items, err := h.Repo.TopStaff(r.Context(), 5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, toDashboardItems(items))
}

func (h DashboardHandler) sales(w http.ResponseWriter, r *http.Request) {
	rangeParam := strings.ToLower(r.URL.Query().Get("range"))
	days := 30
	if rangeParam == "7d" {
		days = 7
	}
	points, err := h.Repo.SalesSeries(r.Context(), days)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, points)
}

func toDashboardItems(items []repository.DashboardItem) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, map[string]any{
			"name":   it.Name,
			"amount": it.Amount,
			"count":  it.Count,
		})
	}
	return out
}
