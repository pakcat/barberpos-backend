package handler

import (
	"encoding/json"
	"net/http"

	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/repository"
	"barberpos-backend/internal/server/authctx"
	"github.com/go-chi/chi/v5"
)

type SettingsHandler struct {
	Repo repository.SettingsRepository
}

func (h SettingsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/settings", h.get)
	r.Put("/settings", h.save)
}

func (h SettingsHandler) get(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	s, err := h.Repo.Get(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toSettingsResponse(s))
}

func (h SettingsHandler) save(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req domain.Settings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	current, err := h.Repo.Get(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if req.CurrencyCode == "" {
		req.CurrencyCode = current.CurrencyCode
	}
	s, err := h.Repo.Save(r.Context(), user.ID, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toSettingsResponse(s))
}

func toSettingsResponse(s *domain.Settings) map[string]any {
	return map[string]any{
		"businessName":         s.BusinessName,
		"businessAddress":      s.BusinessAddress,
		"businessPhone":        s.BusinessPhone,
		"receiptFooter":        s.ReceiptFooter,
		"defaultPaymentMethod": s.DefaultPaymentMethod,
		"printerName":          s.PrinterName,
		"printerType":          s.PrinterType,
		"printerHost":          s.PrinterHost,
		"printerPort":          s.PrinterPort,
		"printerMac":           s.PrinterMac,
		"paperSize":            s.PaperSize,
		"autoPrint":            s.AutoPrint,
		"notifications":        s.Notifications,
		"trackStock":           s.TrackStock,
		"roundingPrice":        s.RoundingPrice,
		"autoBackup":           s.AutoBackup,
		"cashierPin":           s.CashierPin,
		"currencyCode":         s.CurrencyCode,
	}
}
