package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/repository"
	"barberpos-backend/internal/server/authctx"
	"github.com/go-chi/chi/v5"
)

type ProductAdminHandler struct {
	Repo      repository.ProductRepository
	UploadDir string
}

func (h ProductAdminHandler) RegisterRoutes(r chi.Router) {
	r.Post("/products", h.upsert)
	r.Delete("/products/{id}", h.delete)
	r.Post("/products/{id}/image", h.uploadImage)
}

func (h ProductAdminHandler) upsert(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		ID         *int64 `json:"id"`
		Name       string `json:"name"`
		Category   string `json:"category"`
		Price      int64  `json:"price"`
		Image      string `json:"image"`
		TrackStock bool   `json:"trackStock"`
		Stock      int    `json:"stock"`
		MinStock   int    `json:"minStock"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	p := domain.Product{
		Name:       req.Name,
		Category:   req.Category,
		Price:      domain.Money{Amount: req.Price},
		Image:      req.Image,
		TrackStock: req.TrackStock,
		Stock:      req.Stock,
		MinStock:   req.MinStock,
	}
	if req.ID != nil {
		p.ID = *req.ID
	}
	saved, err := h.Repo.Save(r.Context(), user.ID, p)
	if err != nil {
		if err == repository.ErrNotFound {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":         saved.ID,
		"name":       saved.Name,
		"category":   saved.Category,
		"price":      saved.Price.Amount,
		"image":      saved.Image,
		"trackStock": saved.TrackStock,
		"stock":      saved.Stock,
		"minStock":   saved.MinStock,
	})
}

func (h ProductAdminHandler) delete(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.Repo.Delete(r.Context(), user.ID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h ProductAdminHandler) uploadImage(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
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
	ext := ".jpg"
	if mime == "image/png" {
		ext = ".png"
	}

	// Ensure product exists and is owned by this manager.
	if _, err := h.Repo.GetByID(r.Context(), user.ID, id); err != nil {
		if err == repository.ErrNotFound {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	baseDir := h.UploadDir
	if baseDir == "" {
		baseDir = "uploads"
	}
	dir := filepath.Join(baseDir, "products", fmt.Sprintf("%d", user.ID), fmt.Sprintf("%d", id))
	if err := os.MkdirAll(dir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	fullPath := filepath.Join(dir, filename)
	if err := os.WriteFile(fullPath, bytes, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	imagePath := fmt.Sprintf("/uploads/products/%d/%d/%s", user.ID, id, filename)
	if err := h.Repo.UpdateImage(r.Context(), user.ID, id, imagePath); err != nil {
		if err == repository.ErrNotFound {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"image": imagePath,
	})
}
