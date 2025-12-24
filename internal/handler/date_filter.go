package handler

import (
	"net/http"
	"time"
)

const dateLayout = "2006-01-02"

func parseDateQuery(r *http.Request, key string) (*time.Time, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(dateLayout, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
