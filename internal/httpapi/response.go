package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
)

type errorEnvelope struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code        string     `json:"code"`
	Message     string     `json:"message"`
	RequestID   string     `json:"requestId"`
	FieldErrors []fieldErr `json:"fieldErrors,omitempty"`
}

type fieldErr struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		return
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	requestID := requestIDFromContext(r.Context())
	if requestID == "" {
		requestID = strings.TrimSpace(r.Header.Get("X-Request-ID"))
	}
	writeJSON(w, status, errorEnvelope{
		Error: apiError{
			Code:      code,
			Message:   message,
			RequestID: requestID,
		},
	})
}
