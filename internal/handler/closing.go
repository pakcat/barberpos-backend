package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"barberpos-backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

type ClosingHandler struct {
	Repo repository.ClosingRepository
}

func (h ClosingHandler) RegisterRoutes(r chi.Router) {
	r.Get("/closing/summary", h.summary)
	r.Get("/closing", h.list)
	r.Post("/closing", h.create)
}

func (h ClosingHandler) summary(w http.ResponseWriter, r *http.Request) {
	data, err := h.Repo.Summary(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"totalCash":    data.TotalCash,
		"totalNonCash": data.TotalNonCash,
		"totalCard":    data.TotalCard,
	})
}

func (h ClosingHandler) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tanggal      string `json:"tanggal"`
		Shift        string `json:"shift"`
		Karyawan     string `json:"karyawan"`
		ShiftID      string `json:"shiftId"`
		OperatorName string `json:"operatorName"`
		Total        int64  `json:"total"`
		Status       string `json:"status"`
		Catatan      string `json:"catatan"`
		Fisik        string `json:"fisik"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	date := time.Now()
	if req.Tanggal != "" {
		if t, err := time.Parse("2006-01-02", req.Tanggal); err == nil {
			date = t
		}
	}
	var shiftID *string
	if req.ShiftID != "" {
		shiftID = &req.ShiftID
	}
	if req.Status == "" {
		req.Status = "closed"
	}
	id, err := h.Repo.Create(r.Context(), repository.CreateClosingInput{
		Tanggal:      date,
		Shift:        req.Shift,
		Karyawan:     req.Karyawan,
		ShiftID:      shiftID,
		OperatorName: req.OperatorName,
		Total:        req.Total,
		Status:       req.Status,
		Catatan:      req.Catatan,
		Fisik:        req.Fisik,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "id": id})
}

func (h ClosingHandler) list(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	items, err := h.Repo.List(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]map[string]any, 0, len(items))
	for _, c := range items {
		resp = append(resp, map[string]any{
			"id":           c.ID,
			"tanggal":      c.Tanggal.Format("2006-01-02"),
			"shift":        c.Shift,
			"karyawan":     c.Karyawan,
			"shiftId":      c.ShiftID,
			"operatorName": c.OperatorName,
			"total":        c.Total,
			"status":       c.Status,
			"catatan":      c.Catatan,
			"fisik":        c.Fisik,
			"createdAt":    c.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, resp)
}
