package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/repository"
	"barberpos-backend/internal/server/authctx"
	"github.com/go-chi/chi/v5"
)

type NotificationHandler struct {
	Repo repository.NotificationRepository
}

func (h NotificationHandler) RegisterRoutes(r chi.Router) {
	r.Get("/notifications", h.list)
	r.Post("/notifications", h.create)
}

func (h NotificationHandler) list(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	items, err := h.Repo.List(r.Context(), user.ID, 200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]map[string]any, 0, len(items))
	for _, n := range items {
		resp = append(resp, map[string]any{
			"id":        n.ID,
			"title":     n.Title,
			"message":   n.Message,
			"type":      string(n.Type),
			"timestamp": n.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h NotificationHandler) create(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		Title     string `json:"title"`
		Message   string `json:"message"`
		Type      string `json:"type"`
		Timestamp string `json:"timestamp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		req.Title = "Notifikasi"
	}
	ntype := domain.NotificationType(strings.ToLower(strings.TrimSpace(req.Type)))
	switch ntype {
	case domain.NotificationInfo, domain.NotificationWarning, domain.NotificationError:
	default:
		ntype = domain.NotificationInfo
	}
	created := time.Now()
	if req.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, req.Timestamp); err == nil {
			created = t
		}
	}
	notification, err := h.Repo.Create(r.Context(), repository.CreateNotificationInput{
		UserID:  user.ID,
		Title:   req.Title,
		Message: req.Message,
		Type:    ntype,
		Created: created,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":        notification.ID,
		"title":     notification.Title,
		"message":   notification.Message,
		"type":      string(notification.Type),
		"timestamp": notification.CreatedAt.Format(time.RFC3339),
	})
}
