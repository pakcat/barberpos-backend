package handler

import (
	"net/http"
	"strings"

	"barberpos-backend/internal/repository"
	"barberpos-backend/internal/server/authctx"
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
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	data, err := h.Repo.Summary(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"transaksiHariIni":  data.TodayTransactions,
		"omzetHariIni":      data.TodayRevenue,
		"customerHariIni":   data.TodayCustomers,
		"layananTerjual":    data.ServicesSold,
		"totalRevenue":      data.TotalRevenue,
		"totalTransactions": data.TotalTransactions,
		"todayRevenue":      data.TodayRevenue,
	})
}

func (h DashboardHandler) topServices(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	items, err := h.Repo.TopServices(r.Context(), user.ID, 5)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDashboardItems(items))
}

func (h DashboardHandler) topStaff(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	items, err := h.Repo.TopStaff(r.Context(), user.ID, 5)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDashboardItems(items))
}

func (h DashboardHandler) sales(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rangeParam := strings.ToLower(r.URL.Query().Get("range"))
	days := 30
	switch rangeParam {
	case "1d", "today", "hari ini", "hari":
		days = 1
	case "7d", "minggu ini", "minggu", "week":
		days = 7
	case "30d", "bulan ini", "bulan", "month":
		days = 30
	}
	points, err := h.Repo.SalesSeries(r.Context(), user.ID, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]map[string]any, 0, len(points))
	for _, p := range points {
		resp = append(resp, map[string]any{
			"label": p.Label,
			"value": p.Amount,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func toDashboardItems(items []repository.DashboardItem) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, map[string]any{
			"name":   it.Name,
			"amount": it.Amount,
			"qty":    it.Count,
			"count":  it.Count,
		})
	}
	return out
}
