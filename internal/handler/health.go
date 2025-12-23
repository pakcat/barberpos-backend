package handler

import (
    "context"
    "encoding/json"
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

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]string{
        "status": status,
    })
}
