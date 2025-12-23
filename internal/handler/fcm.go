package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"barberpos-backend/internal/repository"
	"barberpos-backend/internal/server/authctx"
	"github.com/go-chi/chi/v5"
)

type FCMHandler struct {
	Repo repository.FCMRepository
}

func (h FCMHandler) RegisterRoutes(r chi.Router) {
	r.Post("/notifications/token", h.register)
}

func (h FCMHandler) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Platform string `json:"platform"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}
	user := authctx.FromContext(r.Context())
	var userID *int64
	if user != nil {
		userID = &user.ID
	}
	if err := h.Repo.Register(r.Context(), repository.RegisterTokenInput{
		UserID:   userID,
		Token:    req.Token,
		Platform: req.Platform,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
