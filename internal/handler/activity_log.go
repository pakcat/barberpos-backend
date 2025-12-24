package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/repository"
	"barberpos-backend/internal/server/authctx"
	"github.com/go-chi/chi/v5"
)

type ActivityLogHandler struct {
	Repo      repository.ActivityLogRepository
	Employees repository.EmployeeRepository
}

func (h ActivityLogHandler) RegisterRoutes(r chi.Router) {
	r.Post("/logs", h.create)
	r.Get("/logs", h.list)
}

func (h ActivityLogHandler) create(w http.ResponseWriter, r *http.Request) {
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
		Title     string `json:"title"`
		Message   string `json:"message"`
		Actor     string `json:"actor"`
		Type      string `json:"type"`
		Timestamp string `json:"timestamp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.Title == "" || req.Message == "" {
		writeError(w, http.StatusBadRequest, "title and message are required")
		return
	}

	ts := time.Now()
	if req.Timestamp != "" {
		if parsed, err := time.Parse(time.RFC3339, req.Timestamp); err == nil {
			ts = parsed
		}
	}
	typ := domain.LogInfo
	switch req.Type {
	case "warning":
		typ = domain.LogWarning
	case "error":
		typ = domain.LogError
	}
	actor := req.Actor
	if actor == "" {
		actor = "System"
	}

	id, err := h.Repo.Create(r.Context(), ownerID, repository.CreateActivityLogInput{
		Title:     req.Title,
		Message:   req.Message,
		Actor:     actor,
		Type:      typ,
		Timestamp: ts,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
}

func (h ActivityLogHandler) list(w http.ResponseWriter, r *http.Request) {
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
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	items, err := h.Repo.List(r.Context(), ownerID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]map[string]any, 0, len(items))
	for _, l := range items {
		resp = append(resp, map[string]any{
			"id":        l.ID,
			"title":     l.Title,
			"message":   l.Message,
			"actor":     l.Actor,
			"type":      string(l.Type),
			"timestamp": l.LoggedAt.Format(time.RFC3339),
			"synced":    l.Synced,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}
