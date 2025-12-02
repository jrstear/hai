package server

import (
	"encoding/json"
	"net/http"
)

// ErrElasticsearchNotConfigured is returned when ELASTICSEARCH_URL is not set
type ErrElasticsearchNotConfigured struct{}

func (e *ErrElasticsearchNotConfigured) Error() string {
	return "ELASTICSEARCH_URL environment variable is required"
}

// ErrorResponse represents a JSON error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// writeError writes a JSON error response
func writeError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: err.Error(),
	})
}

