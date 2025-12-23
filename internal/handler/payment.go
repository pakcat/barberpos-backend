package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type PaymentHandler struct{}

func (h PaymentHandler) RegisterRoutes(r chi.Router) {
	r.Post("/payments/qris", h.qris)
	r.Post("/payments/card", h.card)
}

func (h PaymentHandler) qris(w http.ResponseWriter, r *http.Request) {
	amount := parseAmount(r)
	now := time.Now().UnixMilli()
	writeJSON(w, http.StatusOK, map[string]any{
		"id":        "qris-" + strconv.FormatInt(now, 10),
		"method":    "qris",
		"qrString":  "00020101021226380011ID.CO.QRIS.WWW01189360091100200010000031303UKE51440014ID.CO.QRIS.WWW0215DUMMYQRISEXAMPLE53033605406" + strconv.FormatInt(amount, 10) + "5802ID5908BARBER6010JAKARTA6304ABCD",
		"status":    "pending",
		"expiresAt": time.Now().Add(10 * time.Minute).Format(time.RFC3339),
		"intentId":  "qris-intent-" + strconv.FormatInt(now, 10),
	})
}

func (h PaymentHandler) card(w http.ResponseWriter, r *http.Request) {
	amount := parseAmount(r)
	now := time.Now().UnixMilli()
	writeJSON(w, http.StatusOK, map[string]any{
		"id":        "card-" + strconv.FormatInt(now, 10),
		"method":    "card",
		"status":    "pending",
		"reference": "REF" + strconv.FormatInt(now, 10),
		"intentId":  "card-intent-" + strconv.FormatInt(now, 10),
		"amount":    amount,
	})
}

func parseAmount(r *http.Request) int64 {
	// naive parse; rely on client to send correct amount.
	// In production, validate JSON. Simplified here.
	// Attempt query param or default 0.
	q := r.URL.Query().Get("amount")
	if q == "" {
		return 0
	}
	if v, err := strconv.ParseInt(q, 10, 64); err == nil {
		return v
	}
	return 0
}
