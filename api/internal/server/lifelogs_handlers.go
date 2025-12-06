package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"hai/storage"

	"github.com/go-chi/chi/v5"
)

// BlockquoteResponse represents a blockquote with formatted fields for the UI
type BlockquoteResponse struct {
	ID          string  `json:"id"`
	LifelogID   string  `json:"lifelog_id"`
	LifelogTitle string `json:"lifelog_title,omitempty"`
	SpeakerName string  `json:"speaker_name"`
	SpeakerID   *string `json:"speaker_id,omitempty"` // Optional: Global speaker ID (populated after matching)
	ContactID   *string `json:"contact_id,omitempty"` // Optional: Associated contact ID (user-assigned)
	Content     string  `json:"content"`
	StartTime   string  `json:"start_time"`   // Formatted time (HH:MM:SS)
	EndTime     string  `json:"end_time"`     // Formatted time (HH:MM:SS)
	Duration    float64 `json:"duration"`     // Duration in seconds
	StartOffsetMs int   `json:"start_offset_ms"` // Absolute Unix milliseconds for Limitless API
	EndOffsetMs   int   `json:"end_offset_ms"`   // Absolute Unix milliseconds for Limitless API
}

// HandleGetLifelogs handles GET /api/lifelogs?date=YYYY-MM-DD
// Returns all blockquotes for the specified date, grouped by lifelog
func (s *APIServer) HandleGetLifelogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, &ErrMethodNotAllowed{Method: r.Method})
		return
	}

	// Parse date parameter
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "date parameter is required (format: YYYY-MM-DD)"})
		return
	}

	// Parse date (YYYY-MM-DD format)
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "invalid date format, expected YYYY-MM-DD"})
		return
	}

	// Convert to UTC time range (start of day to end of day)
	startTime := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endTime := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, time.UTC)

	ctx := r.Context()

	// Fetch all lifelogs for this date
	lifelogs, err := s.storage.ListLifelogs(ctx, &startTime, &endTime)
	if err != nil {
		log.Printf("[ERROR] Failed to list lifelogs: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Create a map of lifelog ID to title for quick lookup
	lifelogTitles := make(map[string]string)
	for _, ll := range lifelogs {
		lifelogTitles[ll.ID] = ll.Title
	}

	// Fetch all blockquotes for this date range
	blockquotes, err := s.storage.GetLifelogBlockquotesByTimeRange(ctx, startTime, endTime)
	if err != nil {
		log.Printf("[ERROR] Failed to get blockquotes: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Convert to response format
	responses := make([]BlockquoteResponse, 0, len(blockquotes))
	for _, bq := range blockquotes {
		duration := bq.EndTime.Sub(bq.StartTime).Seconds()
		
		// Format times as HH:MM:SS
		startTimeStr := bq.StartTime.Format("15:04:05")
		endTimeStr := bq.EndTime.Format("15:04:05")

		// Convert absolute timestamps to Unix milliseconds for Limitless API
		startMs := bq.StartTime.UnixMilli()
		endMs := bq.EndTime.UnixMilli()

		responses = append(responses, BlockquoteResponse{
			ID:            bq.ID,
			LifelogID:     bq.LifelogID,
			LifelogTitle:  lifelogTitles[bq.LifelogID],
			SpeakerName:   bq.SpeakerName,
			SpeakerID:     bq.SpeakerID, // Optional: Global speaker ID (if matched to segment)
			ContactID:     bq.ContactID, // Optional: Associated contact ID (user-assigned)
			Content:       bq.Content,
			StartTime:     startTimeStr,
			EndTime:       endTimeStr,
			Duration:      duration,
			StartOffsetMs: int(startMs), // Now using absolute Unix milliseconds
			EndOffsetMs:   int(endMs),   // Now using absolute Unix milliseconds
		})
	}

	// Group by lifelog_id for easier UI rendering
	grouped := make(map[string][]BlockquoteResponse)
	conversationTimings := make(map[string]struct {
		StartMs int `json:"start_ms"`
		EndMs   int `json:"end_ms"`
	})
	
	for _, resp := range responses {
		grouped[resp.LifelogID] = append(grouped[resp.LifelogID], resp)
		
		// Track conversation-level timing (first start, last end) using absolute timestamps
		if timing, exists := conversationTimings[resp.LifelogID]; exists {
			if resp.StartOffsetMs < timing.StartMs {
				timing.StartMs = resp.StartOffsetMs
			}
			if resp.EndOffsetMs > timing.EndMs {
				timing.EndMs = resp.EndOffsetMs
			}
			conversationTimings[resp.LifelogID] = timing
		} else {
			conversationTimings[resp.LifelogID] = struct {
				StartMs int `json:"start_ms"`
				EndMs   int `json:"end_ms"`
			}{
				StartMs: resp.StartOffsetMs, // Already absolute Unix milliseconds
				EndMs:   resp.EndOffsetMs,   // Already absolute Unix milliseconds
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"date":                dateStr,
		"blockquotes":         responses,
		"grouped":             grouped,
		"conversationTimings": conversationTimings,
		"total":               len(responses),
	})
}

// UpdateBlockquoteContactRequest represents a request to update blockquote's contact_id
type UpdateBlockquoteContactRequest struct {
	ContactID *string `json:"contact_id"` // Optional: contact ID to associate, or null to clear
}

// HandleUpdateBlockquoteContact handles PUT /api/blockquotes/{blockquoteId}/contact
// Associates a blockquote with a contact by setting the blockquote's contact_id field
func (s *APIServer) HandleUpdateBlockquoteContact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, &ErrMethodNotAllowed{Method: r.Method})
		return
	}

	blockquoteID := chi.URLParam(r, "blockquoteId")
	if blockquoteID == "" {
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "blockquote ID is required"})
		return
	}

	var req UpdateBlockquoteContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "invalid request body: " + err.Error()})
		return
	}

	ctx := r.Context()

	// Get existing blockquote
	blockquote, err := s.storage.GetLifelogBlockquote(ctx, blockquoteID)
	if err != nil {
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, &ErrBadRequest{Message: "blockquote not found: " + blockquoteID})
			return
		}
		log.Printf("[ERROR] Failed to get blockquote: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Verify contact exists if contact_id is provided
	if req.ContactID != nil && *req.ContactID != "" {
		_, err := s.contacts.GetContact(ctx, *req.ContactID)
		if err != nil {
			if err.Error() == "contact not found: "+*req.ContactID {
				writeError(w, http.StatusNotFound, &ErrBadRequest{Message: "contact not found: " + *req.ContactID})
				return
			}
			log.Printf("[ERROR] Failed to get contact: %v", err)
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}

	// Update blockquote's contact_id
	blockquote.ContactID = req.ContactID

	// Update in storage
	if err := s.storage.UpdateLifelogBlockquote(ctx, blockquote); err != nil {
		log.Printf("[ERROR] Failed to update blockquote: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Return updated blockquote
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         blockquote.ID,
		"contact_id": blockquote.ContactID,
	})
}

// HandleGetLifelogParticipants handles GET /api/lifelogs/{lifelogId}/participants
// Returns a list of unique contact IDs for participants in the conversation
func (s *APIServer) HandleGetLifelogParticipants(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, &ErrMethodNotAllowed{Method: r.Method})
		return
	}

	lifelogID := chi.URLParam(r, "lifelogId")
	if lifelogID == "" {
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "lifelog ID is required"})
		return
	}

	ctx := r.Context()

	// Fetch all blockquotes for this lifelog
	blockquotes, err := s.storage.GetLifelogBlockquotesByLifelog(ctx, lifelogID)
	if err != nil {
		log.Printf("[ERROR] Failed to get blockquotes for lifelog %s: %v", lifelogID, err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Extract unique contact IDs (where contact_id is not nil and not empty)
	contactIDSet := make(map[string]bool)
	for _, bq := range blockquotes {
		if bq.ContactID != nil && *bq.ContactID != "" {
			contactIDSet[*bq.ContactID] = true
		}
	}

	// Convert set to slice
	contactIDs := make([]string, 0, len(contactIDSet))
	for contactID := range contactIDSet {
		contactIDs = append(contactIDs, contactID)
	}

	// Return participants list
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"lifelog_id":  lifelogID,
		"participants": contactIDs,
		"count":       len(contactIDs),
	})
}

