package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// BlockquoteResponse represents a blockquote with formatted fields for the UI
type BlockquoteResponse struct {
	ID          string  `json:"id"`
	LifelogID   string  `json:"lifelog_id"`
	LifelogTitle string `json:"lifelog_title,omitempty"`
	SpeakerName string  `json:"speaker_name"`
	SpeakerID   *string `json:"speaker_id,omitempty"` // Optional: Global speaker ID (populated after matching)
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

