package handler

import (
	"context"
	"net/http"
	"time"

	"barberpos-backend/internal/ports"
	"github.com/go-chi/chi/v5"
)

// HealthHandler exposes a readiness probe.
type HealthHandler struct {
	DB ports.HealthChecker
}

func (h HealthHandler) RegisterRoutes(r chi.Router) {
	r.Get("/health", h.handleHealth)
}

func (h HealthHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	status := "ok"
	if err := h.DB.Health(ctx); err != nil {
		status = "degraded"
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}
