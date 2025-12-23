package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"barberpos-backend/internal/repository"
	"barberpos-backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type MembershipHandler struct {
	Service *service.MembershipService
}

func (h MembershipHandler) RegisterRoutes(r chi.Router) {
	r.Get("/membership", h.state)
	r.Put("/membership", h.updateState)
	r.Get("/membership/topups", h.listTopups)
	r.Post("/membership/topups", h.createTopup)
}

func (h MembershipHandler) state(w http.ResponseWriter, r *http.Request) {
	s, err := h.Service.GetState(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"usedQuota":    s.UsedQuota,
		"freeUsed":     s.FreeUsed,
		"freeQuota":    1000,
		"topupBalance": s.TopupBal,
	})
}

func (h MembershipHandler) updateState(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UsedQuota int `json:"usedQuota"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	s, err := h.Service.SetUsedQuota(r.Context(), req.UsedQuota)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"usedQuota":    s.UsedQuota,
		"freeUsed":     s.FreeUsed,
		"freeQuota":    1000,
		"topupBalance": s.TopupBal,
	})
}

func (h MembershipHandler) listTopups(w http.ResponseWriter, r *http.Request) {
	items, err := h.Service.Repo.ListTopups(r.Context(), 200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]map[string]any, 0, len(items))
	for _, t := range items {
		resp = append(resp, map[string]any{
			"id":      t.ID,
			"amount":  t.Amount.Amount,
			"manager": t.Manager,
			"note":    t.Note,
			"date":    t.Date.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h MembershipHandler) createTopup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Amount  int64  `json:"amount"`
		Manager string `json:"manager"`
		Note    string `json:"note"`
		Date    string `json:"date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.Manager == "" {
		writeError(w, http.StatusBadRequest, "manager is required")
		return
	}
	dt := time.Now()
	if req.Date != "" {
		if t, err := time.Parse(time.RFC3339, req.Date); err == nil {
			dt = t
		}
	}
	topup, err := h.Service.Repo.CreateTopup(r.Context(), repository.CreateTopupInput{
		Amount:  req.Amount,
		Manager: req.Manager,
		Note:    req.Note,
		Date:    dt,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":      topup.ID,
		"amount":  topup.Amount.Amount,
		"manager": topup.Manager,
		"note":    topup.Note,
		"date":    topup.Date.Format(time.RFC3339),
	})
}
