package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/repository"
	"barberpos-backend/internal/server/authctx"
	"barberpos-backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type TransactionHandler struct {
	Repo       repository.TransactionRepository
	Currency   string
	Membership *service.MembershipService
	Employees  repository.EmployeeRepository
	Stocks     repository.StockRepository
	Finance    repository.FinanceRepository
}

func (h TransactionHandler) RegisterRoutes(r chi.Router) {
	r.Post("/orders", h.createOrder)
	r.Get("/transactions", h.listTransactions)
	r.Get("/transactions/{code}", h.getByCode)
	r.Post("/transactions/{code}/refund", h.refund)
	r.Post("/transactions/{code}/mark-paid", h.markPaid)
}

type orderPayload struct {
	Items         []orderLine `json:"items"`
	ClientRef     string      `json:"clientRef"`
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
	ProductID *int64 `json:"productId"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	Price     int64  `json:"price"`
	Qty       int    `json:"qty"`
}

func (h TransactionHandler) createOrder(w http.ResponseWriter, r *http.Request) {
	var req orderPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	var ownerID int64
	if h.Membership != nil {
		user := authctx.FromContext(r.Context())
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		resolved, err := resolveOwnerID(r.Context(), *user, h.Employees)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ownerID = resolved
	}

	items := make([]repository.CreateTransactionItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, repository.CreateTransactionItem{
			ProductID: it.ProductID,
			Name:      it.Name,
			Category:  it.Category,
			Price:     it.Price,
			Qty:       it.Qty,
		})
	}

	unitsToConsume := countUnits(req.Items)

	var clientRef *string
	if req.ClientRef != "" {
		clientRef = &req.ClientRef
	}
	tx, err := h.Repo.Create(r.Context(), repository.CreateTransactionInput{
		PaymentMethod: req.PaymentMethod,
		Stylist:       req.Stylist,
		StylistID:     req.StylistID,
		CustomerName:  req.Customer,
		Amount:        req.Total,
		Items:         items,
		ShiftID:       strPtr(req.ShiftID),
		ClientRef:     clientRef,
	}, func(ctx context.Context, tx pgx.Tx) error {
		for _, it := range items {
			if it.ProductID == nil || it.Qty <= 0 {
				continue
			}
			// Best-effort: only affects products that track stock (stocks row exists).
			_ = h.Stocks.AdjustByProductIDWithTx(ctx, tx, *it.ProductID, -it.Qty, "sale", "sale")
		}
		if h.Membership == nil {
			return nil
		}
		_, err := h.Membership.ConsumeWithTx(ctx, tx, ownerID, unitsToConsume)
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
	startDate, err := parseDateQuery(r, "startDate")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid startDate")
		return
	}
	endDate, err := parseDateQuery(r, "endDate")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid endDate")
		return
	}
	if startDate != nil && endDate != nil && startDate.After(*endDate) {
		writeError(w, http.StatusBadRequest, "startDate must be before endDate")
		return
	}

	var txs []domain.Transaction
	if startDate != nil || endDate != nil {
		txs, err = h.Repo.ListFiltered(r.Context(), startDate, endDate)
	} else {
		txs, err = h.Repo.List(r.Context(), 200)
	}
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
			"refundedAt":    t.RefundedAt,
			"refundNote":    t.RefundNote,
			"stylist":       t.Stylist,
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
		m := map[string]any{
			"name":     it.Name,
			"category": it.Category,
			"price":    it.Price.Amount,
			"qty":      it.Qty,
		}
		if it.ProductID != nil {
			m["productId"] = *it.ProductID
		}
		out = append(out, m)
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

func resolveOwnerID(ctx context.Context, user authctx.CurrentUser, employees repository.EmployeeRepository) (int64, error) {
	switch user.Role {
	case domain.RoleManager, domain.RoleAdmin:
		return user.ID, nil
	case domain.RoleStaff:
		if user.Email == "" {
			return 0, errors.New("staff email is required")
		}
		emp, err := employees.GetByEmail(ctx, user.Email)
		if err != nil {
			return 0, errors.New("employee not found")
		}
		if emp.ManagerID == nil {
			return 0, errors.New("employee has no manager")
		}
		return *emp.ManagerID, nil
	default:
		return 0, errors.New("invalid role")
	}
}

func (h TransactionHandler) getByCode(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}
	t, err := h.Repo.GetByCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "transaction not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
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
	writeJSON(w, http.StatusOK, map[string]any{
		"id":            strconv.FormatInt(t.ID, 10),
		"code":          t.Code,
		"date":          t.Date.Format("2006-01-02"),
		"time":          t.Time,
		"amount":        t.Amount.Amount,
		"paymentMethod": t.PaymentMethod,
		"status":        string(t.Status),
		"refundedAt":    t.RefundedAt,
		"refundNote":    t.RefundNote,
		"stylist":       t.Stylist,
		"stylistId":     t.StylistID,
		"items":         toOrderLines(t.Items),
		"customer":      customer,
	})
}

func (h TransactionHandler) refund(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil || (user.Role != domain.RoleManager && user.Role != domain.RoleAdmin) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	code := chi.URLParam(r, "code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}
	var req struct {
		Note   string `json:"note"`
		Delete *bool  `json:"delete"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	deleteFlag := true
	if req.Delete != nil {
		deleteFlag = *req.Delete
	}

	var ownerID int64
	if h.Membership != nil {
		resolved, err := resolveOwnerID(r.Context(), *user, h.Employees)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		ownerID = resolved
	}

	err := h.Repo.RefundByCode(
		r.Context(),
		repository.RefundTransactionParams{
			Code:       code,
			Note:       req.Note,
			Delete:     deleteFlag,
			RefundedBy: &user.ID,
		},
		func(ctx context.Context, tx pgx.Tx, t domain.Transaction, items []repository.RefundItem, units int) error {
			for _, it := range items {
				if it.ProductID == nil || it.Qty <= 0 {
					continue
				}
				_ = h.Stocks.AdjustByProductIDWithTx(ctx, tx, *it.ProductID, it.Qty, "refund", "refund "+code)
			}
			_, _ = h.Finance.CreateWithTx(ctx, tx, repository.CreateFinanceInput{
				Title:    "Refund " + code,
				Amount:   t.Amount.Amount,
				Category: "Refund",
				Date:     time.Now(),
				Type:     domain.FinanceExpense,
				Note:     req.Note,
				TransactionID:   &t.ID,
				TransactionCode: &code,
			})
			if h.Membership == nil {
				return nil
			}
			_, err := h.Membership.RefundWithTx(ctx, tx, ownerID, units)
			return err
		},
	)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "transaction not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h TransactionHandler) markPaid(w http.ResponseWriter, r *http.Request) {
	user := authctx.FromContext(r.Context())
	if user == nil || (user.Role != domain.RoleManager && user.Role != domain.RoleAdmin) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	code := chi.URLParam(r, "code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}
	tx, err := h.Repo.DB.Pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback(r.Context())

	transactionID, err := h.Repo.MarkPaidByCodeWithTx(r.Context(), tx, code)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "transaction not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Best-effort: remove refund finance entry when undoing refund.
	_ = h.Finance.DeleteRefundByTransactionIDWithTx(r.Context(), tx, transactionID)

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
