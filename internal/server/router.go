package server

import (
    "net/http"
    "time"

    "barberpos-backend/internal/handler"
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
)

// NewRouter wires HTTP routes and middleware.
func NewRouter(h handler.HealthHandler) http.Handler {
    r := chi.NewRouter()
    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(middleware.Timeout(60 * time.Second))

    h.RegisterRoutes(r)

    return r
}
