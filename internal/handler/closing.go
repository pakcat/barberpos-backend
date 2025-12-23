package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"barberpos-backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

type ClosingHandler struct {
	Repo repository.ClosingRepository
}

func (h ClosingHandler) RegisterRoutes(r chi.Router) {
	r.Get("/closing/summary", h.summary)
	r.Post("/closing", h.create)
}

func (h ClosingHandler) summary(w http.ResponseWriter, r *http.Request) {
	data, err := h.Repo.Summary(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "invalid payload", http.StatusBadRequest)
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
	if err := h.Repo.Create(r.Context(), repository.CreateClosingInput{
		Tanggal:      date,
		Shift:        req.Shift,
		Karyawan:     req.Karyawan,
		ShiftID:      shiftID,
		OperatorName: req.OperatorName,
		Total:        req.Total,
		Status:       req.Status,
		Catatan:      req.Catatan,
		Fisik:        req.Fisik,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
