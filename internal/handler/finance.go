package handler

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"barberpos-backend/internal/domain"
	"barberpos-backend/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/xuri/excelize/v2"
)

type FinanceHandler struct {
	Repo repository.FinanceRepository
}

func (h FinanceHandler) RegisterRoutes(r chi.Router) {
	r.Get("/finance", h.list)
	r.Get("/finance/export", h.export)
	r.Post("/finance", h.create)
}

func (h FinanceHandler) list(w http.ResponseWriter, r *http.Request) {
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

	var items []domain.FinanceEntry
	if startDate != nil || endDate != nil {
		items, err = h.Repo.ListFiltered(r.Context(), startDate, endDate)
	} else {
		items, err = h.Repo.List(r.Context(), 200)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := make([]map[string]any, 0, len(items))
	for _, fe := range items {
		resp = append(resp, map[string]any{
			"id":              fe.ID,
			"title":           fe.Title,
			"amount":          fe.Amount.Amount,
			"category":        fe.Category,
			"date":            fe.Date.Format("2006-01-02"),
			"type":            string(fe.Type),
			"note":            fe.Note,
			"transactionCode": fe.TransactionCode,
			"staff":           fe.Staff,
			"service":         fe.Service,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h FinanceHandler) export(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

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

	var items []domain.FinanceEntry
	if startDate != nil || endDate != nil {
		items, err = h.Repo.ListFiltered(r.Context(), startDate, endDate)
	} else {
		items, err = h.Repo.List(r.Context(), 2000)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	filenameSuffix := time.Now().Format("20060102_150405")
	if startDate != nil && endDate != nil {
		filenameSuffix = fmt.Sprintf("%s_%s", startDate.Format("20060102"), endDate.Format("20060102"))
	}

	switch format {
	case "csv":
		data, err := exportFinanceCSV(items)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"finance_%s.csv\"", filenameSuffix))
		_, _ = w.Write(data)
		return
	case "xlsx", "excel":
		data, err := exportFinanceXLSX(items)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"finance_%s.xlsx\"", filenameSuffix))
		_, _ = w.Write(data)
		return
	default:
		writeError(w, http.StatusBadRequest, "invalid format (use csv or xlsx)")
		return
	}
}

func exportFinanceCSV(items []domain.FinanceEntry) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := csv.NewWriter(buf)
	_ = w.Write([]string{"id", "title", "amount", "category", "date", "type", "note", "transaction_code", "staff", "service"})
	for _, fe := range items {
		_ = w.Write([]string{
			strconv.FormatInt(fe.ID, 10),
			fe.Title,
			strconv.FormatInt(fe.Amount.Amount, 10),
			fe.Category,
			fe.Date.Format("2006-01-02"),
			string(fe.Type),
			fe.Note,
			derefString(fe.TransactionCode),
			derefString(fe.Staff),
			derefString(fe.Service),
		})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func exportFinanceXLSX(items []domain.FinanceEntry) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Finance"
	index, err := f.NewSheet(sheet)
	if err != nil {
		return nil, err
	}
	f.DeleteSheet("Sheet1")
	f.SetActiveSheet(index)

	header := []string{"ID", "Title", "Amount", "Category", "Date", "Type", "Note", "Transaction Code", "Staff", "Service"}
	for c, v := range header {
		cell, _ := excelize.CoordinatesToCellName(c+1, 1)
		_ = f.SetCellValue(sheet, cell, v)
	}
	for r, fe := range items {
		row := r + 2
		values := []any{
			fe.ID,
			fe.Title,
			fe.Amount.Amount,
			fe.Category,
			fe.Date.Format("2006-01-02"),
			string(fe.Type),
			fe.Note,
			derefString(fe.TransactionCode),
			derefString(fe.Staff),
			derefString(fe.Service),
		}
		for c, v := range values {
			cell, _ := excelize.CoordinatesToCellName(c+1, row)
			_ = f.SetCellValue(sheet, cell, v)
		}
	}

	_ = f.SetColWidth(sheet, "A", "A", 10)
	_ = f.SetColWidth(sheet, "B", "B", 28)
	_ = f.SetColWidth(sheet, "C", "C", 14)
	_ = f.SetColWidth(sheet, "D", "D", 18)
	_ = f.SetColWidth(sheet, "E", "E", 12)
	_ = f.SetColWidth(sheet, "F", "F", 10)
	_ = f.SetColWidth(sheet, "G", "G", 28)
	_ = f.SetColWidth(sheet, "H", "H", 18)
	_ = f.SetColWidth(sheet, "I", "I", 18)
	_ = f.SetColWidth(sheet, "J", "J", 18)

	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#1F2937"}, Pattern: 1},
	})
	_ = f.SetCellStyle(sheet, "A1", "J1", style)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func (h FinanceHandler) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title    string  `json:"title"`
		Amount   int64   `json:"amount"`
		Category string  `json:"category"`
		Date     string  `json:"date"`
		Type     string  `json:"type"`
		Note     string  `json:"note"`
		TransactionCode *string `json:"transactionCode"`
		Staff    *string `json:"staff"`
		Service  *string `json:"service"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	dt := time.Now()
	if req.Date != "" {
		if t, err := time.Parse("2006-01-02", req.Date); err == nil {
			dt = t
		}
	}
	fe, err := h.Repo.Create(r.Context(), repository.CreateFinanceInput{
		Title:    req.Title,
		Amount:   req.Amount,
		Category: req.Category,
		Date:     dt,
		Type:     domain.FinanceEntryType(req.Type),
		Note:     req.Note,
		TransactionCode: req.TransactionCode,
		Staff:    req.Staff,
		Service:  req.Service,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":              fe.ID,
		"title":           fe.Title,
		"amount":          fe.Amount.Amount,
		"category":        fe.Category,
		"date":            fe.Date.Format("2006-01-02"),
		"type":            string(fe.Type),
		"note":            fe.Note,
		"transactionCode": fe.TransactionCode,
		"staff":           fe.Staff,
		"service":         fe.Service,
	})
}
