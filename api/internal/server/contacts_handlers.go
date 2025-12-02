package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"hai/api/internal/contacts"

	"github.com/go-chi/chi/v5"
)

// HandleListContacts handles GET /api/contacts
func (s *APIServer) HandleListContacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, &ErrMethodNotAllowed{Method: r.Method})
		return
	}

	start := time.Now()

	// Parse query parameters
	filters := &contacts.ContactFilters{}
	if knownStr := r.URL.Query().Get("known"); knownStr != "" {
		known, err := strconv.ParseBool(knownStr)
		if err == nil {
			filters.Known = &known
		}
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filters.Search = search
	}

	// List contacts
	listStart := time.Now()
	contactList, total, err := s.contacts.ListContacts(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	listElapsed := time.Since(listStart)
	log.Printf("[TIMING] ListContacts: %v (%d contacts)", listElapsed, total)

	// Compute known status for each contact (TODO: optimize with batch query)
	knownStart := time.Now()
	for i := range contactList {
		known, err := s.computeKnownStatus(r.Context(), contactList[i].ID)
		if err == nil {
			contactList[i].Known = known
		}
	}
	knownElapsed := time.Since(knownStart)
	totalElapsed := time.Since(start)
	log.Printf("[TIMING] computeKnownStatus (all %d contacts): %v", len(contactList), knownElapsed)
	log.Printf("[TIMING] HandleListContacts total: %v", totalElapsed)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contacts.ContactListResponse{
		Contacts: contactList,
		Total:    total,
	})
}

// HandleGetContact handles GET /api/contacts/:id
func (s *APIServer) HandleGetContact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, &ErrMethodNotAllowed{Method: r.Method})
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "contact ID is required"})
		return
	}

	contact, err := s.contacts.GetContact(r.Context(), id)
	if err != nil {
		if err.Error() == "contact not found: "+id {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Compute known status
	known, err := s.computeKnownStatus(r.Context(), contact.ID)
	if err == nil {
		contact.Known = known
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contact)
}

// HandleCreateContact handles POST /api/contacts
func (s *APIServer) HandleCreateContact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, &ErrMethodNotAllowed{Method: r.Method})
		return
	}

	var contact contacts.Contact
	if err := json.NewDecoder(r.Body).Decode(&contact); err != nil {
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "invalid request body: " + err.Error()})
		return
	}

	if err := s.contacts.CreateContact(r.Context(), &contact); err != nil {
		if err.Error() == "contact already exists: "+contact.ID {
			writeError(w, http.StatusConflict, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(contact)
}

// HandleUpdateContact handles PUT /api/contacts/:id
func (s *APIServer) HandleUpdateContact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, &ErrMethodNotAllowed{Method: r.Method})
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "contact ID is required"})
		return
	}

	var updates contacts.Contact
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "invalid request body: " + err.Error()})
		return
	}

	if err := s.contacts.UpdateContact(r.Context(), id, &updates); err != nil {
		if err.Error() == "contact not found: "+id {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Return updated contact
	contact, err := s.contacts.GetContact(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contact)
}

// computeKnownStatus checks if any speaker has this contact_id
func (s *APIServer) computeKnownStatus(ctx context.Context, contactID string) (bool, error) {
	speakers, err := s.storage.ListSpeakers(ctx, &contactID)
	if err != nil {
		return false, err
	}
	return len(speakers) > 0, nil
}

// ErrBadRequest is returned when request is invalid
type ErrBadRequest struct {
	Message string
}

func (e *ErrBadRequest) Error() string {
	return e.Message
}

