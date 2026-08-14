package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to write JSON response", "error", err)
	}
}

// OK writes a 200 JSON response.
func OK(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, data)
}

// Error writes a JSON error response with the given status code.
func Error(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// ServiceError writes a JSON error response for service dependencies.
func ServiceError(w http.ResponseWriter, status int, service string, err string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "degraded",
		"service": service,
		"error":   err,
	})
}
