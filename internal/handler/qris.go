package handler

import (
	"io"
	"net/http"
	"strings"

	"barberpos-backend/internal/repository"
	"barberpos-backend/internal/server/authctx"
	"github.com/go-chi/chi/v5"
)

type QRISHandler struct {
	Settings  repository.SettingsRepository
	Employees repository.EmployeeRepository
}

func (h QRISHandler) RegisterStaffRoutes(r chi.Router) {
	r.Get("/settings/qris", h.get)
}

func (h QRISHandler) RegisterManagerRoutes(r chi.Router) {
	r.Post("/settings/qris", h.upload)
	r.Delete("/settings/qris", h.clear)
}

func (h QRISHandler) get(w http.ResponseWriter, r *http.Request) {
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

	bytes, mime, _, err := h.Settings.GetQrisImage(r.Context(), ownerID)
	if err != nil {
		writeError(w, http.StatusNotFound, "QRIS belum diupload")
		return
	}
	if mime == "" {
		mime = http.DetectContentType(bytes)
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(bytes)
}

func (h QRISHandler) upload(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := r.ParseMultipartForm(6 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	limited := io.LimitReader(file, 5<<20)
	bytes, err := io.ReadAll(limited)
	if err != nil || len(bytes) == 0 {
		writeError(w, http.StatusBadRequest, "file kosong")
		return
	}

	mime := header.Header.Get("Content-Type")
	if mime == "" {
		mime = http.DetectContentType(bytes)
	}
	mime = strings.ToLower(strings.TrimSpace(mime))
	if mime != "image/png" && mime != "image/jpeg" && mime != "image/jpg" {
		writeError(w, http.StatusBadRequest, "format harus PNG/JPG")
		return
	}
	if mime == "image/jpg" {
		mime = "image/jpeg"
	}

	// Ensure settings row exists.
	if _, err := h.Settings.Get(r.Context(), user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.Settings.SetQrisImage(r.Context(), user.ID, bytes, mime); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h QRISHandler) clear(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.Settings.ClearQrisImage(r.Context(), user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
