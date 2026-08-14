package httpserver

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is the standard error response body.
type ErrorResponse struct {
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Service string `json:"service,omitempty"`
}

// Error writes a JSON error response.
func Error(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	resp := ErrorResponse{
		Status: statusText(status),
		Error:  message,
	}
	json.NewEncoder(w).Encode(resp)
}

// ServiceError writes a JSON error response for a specific service health failure.
func ServiceError(w http.ResponseWriter, status int, service, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	resp := ErrorResponse{
		Status:  "degraded",
		Service: service,
		Error:   message,
	}
	json.NewEncoder(w).Encode(resp)
}

func statusText(code int) string {
	switch {
	case code >= 500:
		return "error"
	case code >= 400:
		return "fail"
	default:
		return "ok"
	}
}
