package handler

import (
	"encoding/json"
	"net/http"
)

type apiError struct {
	Code   int    `json:"code"`
	Status string `json:"status"`
}

type apiResponse struct {
	Status  string    `json:"status"`
	Message string    `json:"message"`
	Data    any       `json:"data"`
	Error   *apiError `json:"error,omitempty"`
}

func writeRawJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	if status >= 400 {
		writeRawJSON(w, status, apiResponse{
			Status:  "error",
			Message: "",
			Data:    payload,
			Error: &apiError{
				Code:   status,
				Status: http.StatusText(status),
			},
		})
		return
	}
	writeRawJSON(w, status, apiResponse{
		Status:  "ok",
		Message: "",
		Data:    payload,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	if status < 400 {
		status = http.StatusInternalServerError
	}
	writeRawJSON(w, status, apiResponse{
		Status:  "error",
		Message: message,
		Data:    nil,
		Error: &apiError{
			Code:   status,
			Status: http.StatusText(status),
		},
	})
}

func writeErrorWithErr(w http.ResponseWriter, status int, message string, err error) {
	if err == nil {
		writeError(w, status, message)
		return
	}
	if message == "" {
		writeError(w, status, err.Error())
		return
	}
	writeError(w, status, message+": "+err.Error())
}
