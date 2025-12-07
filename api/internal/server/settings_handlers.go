package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// HandleGetSetting handles GET /api/settings/{key}
func (s *APIServer) HandleGetSetting(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, &ErrMethodNotAllowed{Method: r.Method})
		return
	}

	key := chi.URLParam(r, "key")
	if key == "" {
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "setting key is required"})
		return
	}

	ctx := r.Context()
	value, err := s.storage.GetSetting(ctx, key)
	if err != nil {
		if err.Error() == "resource not found" {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"key":   key,
		"value": value,
	})
}

// HandleSetSetting handles PUT /api/settings/{key}
func (s *APIServer) HandleSetSetting(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, &ErrMethodNotAllowed{Method: r.Method})
		return
	}

	key := chi.URLParam(r, "key")
	if key == "" {
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "setting key is required"})
		return
	}

	var req struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "invalid request body: " + err.Error()})
		return
	}

	ctx := r.Context()
	if err := s.storage.SetSetting(ctx, key, req.Value); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"key":   key,
		"value": req.Value,
	})
}
