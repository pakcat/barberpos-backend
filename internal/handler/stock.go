package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"barberpos-backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

type StockHandler struct {
	Repo repository.StockRepository
}

func (h StockHandler) RegisterRoutes(r chi.Router) {
	r.Get("/stock", h.list)
	r.Post("/stock/adjust", h.adjust)
	r.Get("/stock/{id}/history", h.history)
}

func (h StockHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.Repo.List(r.Context(), 500)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]map[string]any, 0, len(items))
	for _, s := range items {
		resp = append(resp, map[string]any{
			"id":           s.ID,
			"productId":    s.ProductID,
			"name":         s.Name,
			"category":     s.Category,
			"image":        s.Image,
			"stock":        s.Stock,
			"transactions": s.Transactions,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h StockHandler) adjust(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StockID   int64  `json:"stockId"`
		Change    int    `json:"change"`
		Type      string `json:"type"`
		Note      string `json:"note"`
		ProductID *int64 `json:"productId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.StockID == 0 {
		writeError(w, http.StatusBadRequest, "stockId is required")
		return
	}
	if req.Type == "" {
		req.Type = "adjust"
	}
	stock, err := h.Repo.Adjust(r.Context(), repository.AdjustStockInput{
		StockID:   req.StockID,
		Change:    req.Change,
		Type:      req.Type,
		Note:      req.Note,
		ProductID: req.ProductID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":           stock.ID,
		"stock":        stock.Stock,
		"transactions": stock.Transactions,
	})
}

func (h StockHandler) history(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	stockID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	items, err := h.Repo.History(r.Context(), stockID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}
