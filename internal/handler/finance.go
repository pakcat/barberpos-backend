package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

type FinanceHandler struct {
	Repo repository.FinanceRepository
}

func (h FinanceHandler) RegisterRoutes(r chi.Router) {
	r.Get("/finance", h.list)
	r.Post("/finance", h.create)
}

func (h FinanceHandler) list(w http.ResponseWriter, r *http.Request) {
	startDate, err := parseDateQuery(r, "startDate")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid startDate")
		return
	}
	endDate, err := parseDateQuery(r, "endDate")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid endDate")
		return
	}
	if startDate != nil && endDate != nil && startDate.After(*endDate) {
		writeError(w, http.StatusBadRequest, "startDate must be before endDate")
		return
	}

	var items []domain.FinanceEntry
	if startDate != nil || endDate != nil {
		items, err = h.Repo.ListFiltered(r.Context(), startDate, endDate)
	} else {
		items, err = h.Repo.List(r.Context(), 200)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]map[string]any, 0, len(items))
	for _, fe := range items {
		resp = append(resp, map[string]any{
			"id":       fe.ID,
			"title":    fe.Title,
			"amount":   fe.Amount.Amount,
			"category": fe.Category,
			"date":     fe.Date.Format("2006-01-02"),
			"type":     string(fe.Type),
			"note":     fe.Note,
			"staff":    fe.Staff,
			"service":  fe.Service,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h FinanceHandler) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title    string  `json:"title"`
		Amount   int64   `json:"amount"`
		Category string  `json:"category"`
		Date     string  `json:"date"`
		Type     string  `json:"type"`
		Note     string  `json:"note"`
		Staff    *string `json:"staff"`
		Service  *string `json:"service"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	dt := time.Now()
	if req.Date != "" {
		if t, err := time.Parse("2006-01-02", req.Date); err == nil {
			dt = t
		}
	}
	fe, err := h.Repo.Create(r.Context(), repository.CreateFinanceInput{
		Title:    req.Title,
		Amount:   req.Amount,
		Category: req.Category,
		Date:     dt,
		Type:     domain.FinanceEntryType(req.Type),
		Note:     req.Note,
		Staff:    req.Staff,
		Service:  req.Service,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       fe.ID,
		"title":    fe.Title,
		"amount":   fe.Amount.Amount,
		"category": fe.Category,
		"date":     fe.Date.Format("2006-01-02"),
		"type":     string(fe.Type),
		"note":     fe.Note,
		"staff":    fe.Staff,
		"service":  fe.Service,
	})
}
