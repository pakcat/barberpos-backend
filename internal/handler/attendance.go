package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/repository"
	"barberpos-backend/internal/server/authctx"
	"github.com/go-chi/chi/v5"
)

type AttendanceHandler struct {
	Repo      repository.AttendanceRepository
	Employees repository.EmployeeRepository
}

func (h AttendanceHandler) RegisterRoutes(r chi.Router) {
	r.Post("/attendance/checkin", h.checkIn)
	r.Post("/attendance/checkout", h.checkOut)
	r.Get("/attendance", h.listMonth)
	r.Get("/attendance/daily", h.listDaily)
}

func (h AttendanceHandler) checkIn(w http.ResponseWriter, r *http.Request) {
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
	var req struct {
		EmployeeID   *int64 `json:"employeeId"`
		EmployeeName string `json:"employeeName"`
		Source       string `json:"source"`
		Date         string `json:"date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.EmployeeName == "" {
		writeError(w, http.StatusBadRequest, "employeeName is required")
		return
	}
	date := time.Now()
	if req.Date != "" {
		if t, err := time.Parse("2006-01-02", req.Date); err == nil {
			date = t
		}
	}
	if err := h.Repo.CheckIn(r.Context(), ownerID, req.EmployeeName, req.EmployeeID, date, req.Source); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h AttendanceHandler) checkOut(w http.ResponseWriter, r *http.Request) {
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
	var req struct {
		EmployeeID   *int64 `json:"employeeId"`
		EmployeeName string `json:"employeeName"`
		Source       string `json:"source"`
		Date         string `json:"date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.EmployeeName == "" {
		writeError(w, http.StatusBadRequest, "employeeName is required")
		return
	}
	date := time.Now()
	if req.Date != "" {
		if t, err := time.Parse("2006-01-02", req.Date); err == nil {
			date = t
		}
	}
	if err := h.Repo.CheckOut(r.Context(), ownerID, req.EmployeeName, req.EmployeeID, date, req.Source); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h AttendanceHandler) listMonth(w http.ResponseWriter, r *http.Request) {
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
	name := r.URL.Query().Get("employeeName")
	monthStr := r.URL.Query().Get("month")
	if name == "" || monthStr == "" {
		writeError(w, http.StatusBadRequest, "employeeName and month are required")
		return
	}
	month, err := time.Parse("2006-01", monthStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid month format")
		return
	}
	items, err := h.Repo.GetMonth(r.Context(), ownerID, name, month)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]map[string]any, 0, len(items))
	for _, a := range items {
		resp = append(resp, map[string]any{
			"id":           a.ID,
			"employeeId":   a.EmployeeID,
			"employeeName": a.EmployeeName,
			"date":         a.Date.Format("2006-01-02"),
			"checkIn":      timeOrNil(a.CheckIn),
			"checkOut":     timeOrNil(a.CheckOut),
			"status":       string(a.Status),
			"source":       a.Source,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h AttendanceHandler) listDaily(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if user.Role == domain.RoleStaff {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	ownerID, err := resolveOwnerID(r.Context(), *user, h.Employees)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		writeError(w, http.StatusBadRequest, "date is required")
		return
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid date format")
		return
	}
	items, err := h.Repo.GetDaily(r.Context(), ownerID, date)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]map[string]any, 0, len(items))
	for _, a := range items {
		resp = append(resp, map[string]any{
			"id":           a.ID,
			"employeeId":   a.EmployeeID,
			"employeeName": a.EmployeeName,
			"date":         a.Date.Format("2006-01-02"),
			"checkIn":      timeOrNil(a.CheckIn),
			"checkOut":     timeOrNil(a.CheckOut),
			"status":       string(a.Status),
			"source":       a.Source,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func timeOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}
