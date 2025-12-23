package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/repository"
	"barberpos-backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type TransactionHandler struct {
	Repo       repository.TransactionRepository
	Currency   string
	Membership *service.MembershipService
}

func (h TransactionHandler) RegisterRoutes(r chi.Router) {
	r.Post("/orders", h.createOrder)
	r.Get("/transactions", h.listTransactions)
}

type orderPayload struct {
	Items         []orderLine `json:"items"`
	Total         int64       `json:"total"`
	Paid          int64       `json:"paid"`
	Change        int64       `json:"change"`
	PaymentMethod string      `json:"paymentMethod"`
	Stylist       string      `json:"stylist"`
	StylistID     *int64      `json:"stylistId"`
	Customer      string      `json:"customer"`
	ShiftID       string      `json:"shiftId"`
}

type orderLine struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Price    int64  `json:"price"`
	Qty      int    `json:"qty"`
}

func (h TransactionHandler) createOrder(w http.ResponseWriter, r *http.Request) {
	var req orderPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	items := make([]repository.CreateTransactionItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, repository.CreateTransactionItem{
			Name:     it.Name,
			Category: it.Category,
			Price:    it.Price,
			Qty:      it.Qty,
		})
	}

	unitsToConsume := countUnits(req.Items)

	tx, err := h.Repo.Create(r.Context(), repository.CreateTransactionInput{
		PaymentMethod: req.PaymentMethod,
		Stylist:       req.Stylist,
		StylistID:     req.StylistID,
		CustomerName:  req.Customer,
		Amount:        req.Total,
		Items:         items,
		ShiftID:       strPtr(req.ShiftID),
	}, func(ctx context.Context, tx pgx.Tx) error {
		if h.Membership == nil {
			return nil
		}
		_, err := h.Membership.ConsumeWithTx(ctx, tx, unitsToConsume)
		return err
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":            strconv.FormatInt(tx.ID, 10),
		"code":          tx.Code,
		"total":         req.Total,
		"paid":          req.Paid,
		"change":        req.Change,
		"paymentMethod": req.PaymentMethod,
		"items":         toOrderLines(tx.Items),
	})
}

func (h TransactionHandler) listTransactions(w http.ResponseWriter, r *http.Request) {
	txs, err := h.Repo.List(r.Context(), 200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]map[string]any, 0, len(txs))
	for _, t := range txs {
		customer := map[string]any{
			"name":      "",
			"phone":     "",
			"email":     "",
			"address":   "",
			"visits":    nil,
			"lastVisit": nil,
		}
		if t.Customer != nil {
			customer["name"] = t.Customer.Name
			customer["phone"] = t.Customer.Phone
			customer["email"] = t.Customer.Email
			customer["address"] = t.Customer.Address
			customer["visits"] = t.Customer.Visits
			customer["lastVisit"] = t.Customer.LastVisit
		}

		resp = append(resp, map[string]any{
			"id":            strconv.FormatInt(t.ID, 10),
			"code":          t.Code,
			"date":          t.Date.Format("2006-01-02"),
			"time":          t.Time,
			"amount":        t.Amount.Amount,
			"paymentMethod": t.PaymentMethod,
			"status":        string(t.Status),
			"stylistId":     t.StylistID,
			"items":         toOrderLines(t.Items),
			"customer":      customer,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func toOrderLines(items []domain.TransactionItem) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, map[string]any{
			"name":     it.Name,
			"category": it.Category,
			"price":    it.Price.Amount,
			"qty":      it.Qty,
		})
	}
	return out
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func countUnits(items []orderLine) int {
	sum := 0
	for _, it := range items {
		sum += it.Qty
	}
	if sum == 0 {
		return 1
	}
	return sum
}
