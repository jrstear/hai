package server

import (
	"encoding/json"
	"net/http"
)

// HandleHealth handles GET /api/health
func (s *APIServer) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, &ErrMethodNotAllowed{Method: r.Method})
		return
	}

	// Check storage health
	ctx := r.Context()
	if err := s.storage.Health(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// ErrMethodNotAllowed is returned when HTTP method is not allowed
type ErrMethodNotAllowed struct {
	Method string
}

func (e *ErrMethodNotAllowed) Error() string {
	return "method not allowed: " + e.Method
}








