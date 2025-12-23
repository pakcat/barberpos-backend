package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type HomeHandler struct{}

func (h HomeHandler) RegisterRoutes(r chi.Router) {
	r.Get("/posts/1", h.welcome)
}

func (h HomeHandler) welcome(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"title": "Hello",
		"body":  "Welcome to Barber POS",
	})
}
