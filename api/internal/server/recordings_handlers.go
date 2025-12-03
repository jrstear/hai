package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"hai/api/internal/contacts"
	"hai/storage"

	"github.com/go-chi/chi/v5"
)

// SegmentWithTranscript represents a segment with transcript information
type SegmentWithTranscript struct {
	ID          int64   `json:"id"`
	SpeakerID   *string `json:"speaker_id,omitempty"`
	RecordingID string  `json:"recording_id"`
	StartTime   float64 `json:"start_time"`
	EndTime     float64 `json:"end_time"`
	Duration    float64 `json:"duration"`
	Transcript  string  `json:"transcript,omitempty"`
	Time        string  `json:"time"` // Formatted time (recording start + segment start)
}

// HandleGetContactRecordings handles GET /api/contacts/{contactId}/recordings
func (s *APIServer) HandleGetContactRecordings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, &ErrMethodNotAllowed{Method: r.Method})
		return
	}

	contactID := chi.URLParam(r, "contactId")
	if contactID == "" {
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "contact ID is required"})
		return
	}

	ctx := r.Context()

	// Verify contact exists
	var contact *contacts.Contact
	contact, err := s.contacts.GetContact(ctx, contactID)
	if err != nil {
		if err.Error() == "contact not found: "+contactID {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Get all speakers associated with this contact
	speakers, err := s.storage.ListSpeakers(ctx, &contactID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if len(speakers) == 0 {
		// No speakers associated, return empty list
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"contact":  contact,
			"segments": []SegmentWithTranscript{},
			"total":    0,
		})
		return
	}

	// Parse date range filter (optional)
	var startDate, endDate *time.Time
	if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			// Set to start of day
			startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
			startDate = &startOfDay
		}
	}
	if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			// Set to end of day
			endOfDay := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
			endDate = &endOfDay
		}
	}

	// Collect segments from all speakers
	allSegments := make([]*storage.Segment, 0)
	for _, speaker := range speakers {
		segments, err := s.storage.GetSegmentsBySpeaker(ctx, speaker.ID)
		if err != nil {
			continue // Skip on error
		}
		allSegments = append(allSegments, segments...)
	}

	// Convert to response format with transcripts (and filter by date range if specified)
	segmentsWithTranscripts, err := s.enrichSegmentsWithTranscripts(ctx, allSegments, startDate, endDate)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Parse sort parameter
	sortBy := r.URL.Query().Get("sort")
	if sortBy == "time" {
		// Sort by time (ascending)
		sort.Slice(segmentsWithTranscripts, func(i, j int) bool {
			return segmentsWithTranscripts[i].StartTime < segmentsWithTranscripts[j].StartTime
		})
	} else {
		// Default: sort by duration (descending)
		sort.Slice(segmentsWithTranscripts, func(i, j int) bool {
			return segmentsWithTranscripts[i].Duration > segmentsWithTranscripts[j].Duration
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"contact":  contact,
		"segments": segmentsWithTranscripts,
		"total":    len(segmentsWithTranscripts),
	})
}

// HandleGetSpeakerRecordings handles GET /api/speakers/{speakerId}/recordings
func (s *APIServer) HandleGetSpeakerRecordings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, &ErrMethodNotAllowed{Method: r.Method})
		return
	}

	speakerID := chi.URLParam(r, "speakerId")
	if speakerID == "" {
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "speaker ID is required"})
		return
	}

	ctx := r.Context()

	// Verify speaker exists
	speaker, err := s.storage.GetSpeaker(ctx, speakerID)
	if err != nil {
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, &ErrBadRequest{Message: "speaker not found: " + speakerID})
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Parse date range filter (optional)
	var startDate, endDate *time.Time
	if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			// Set to start of day
			startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
			startDate = &startOfDay
		}
	}
	if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			// Set to end of day
			endOfDay := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
			endDate = &endOfDay
		}
	}

	// Get all segments for this speaker
	segments, err := s.storage.GetSegmentsBySpeaker(ctx, speakerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Convert to response format with transcripts (and filter by date range if specified)
	segmentsWithTranscripts, err := s.enrichSegmentsWithTranscripts(ctx, segments, startDate, endDate)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Parse sort parameter
	sortBy := r.URL.Query().Get("sort")
	if sortBy == "time" {
		// Sort by time (ascending)
		sort.Slice(segmentsWithTranscripts, func(i, j int) bool {
			return segmentsWithTranscripts[i].StartTime < segmentsWithTranscripts[j].StartTime
		})
	} else {
		// Default: sort by duration (descending)
		sort.Slice(segmentsWithTranscripts, func(i, j int) bool {
			return segmentsWithTranscripts[i].Duration > segmentsWithTranscripts[j].Duration
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"speaker":  speaker,
		"segments": segmentsWithTranscripts,
		"total":    len(segmentsWithTranscripts),
	})
}

// enrichSegmentsWithTranscripts adds transcript and formatted time to segments
// If startDate or endDate are provided, filters segments to only those whose recording
// start_time falls within the date range.
func (s *APIServer) enrichSegmentsWithTranscripts(ctx context.Context, segments []*storage.Segment, startDate, endDate *time.Time) ([]SegmentWithTranscript, error) {
	result := make([]SegmentWithTranscript, 0, len(segments))

	// Group segments by recording to batch fetch recordings
	recordingMap := make(map[string]*storage.Recording)
	for _, segment := range segments {
		if _, exists := recordingMap[segment.RecordingID]; !exists {
			recording, err := s.storage.GetRecording(ctx, segment.RecordingID)
			if err != nil {
				continue // Skip if recording not found
			}
			recordingMap[segment.RecordingID] = recording
		}
	}

	// Filter segments by date range if specified
	if startDate != nil || endDate != nil {
		filteredSegments := make([]*storage.Segment, 0)
		for _, segment := range segments {
			recording, exists := recordingMap[segment.RecordingID]
			if !exists {
				continue
			}
			// Check if recording start_time is within date range
			if startDate != nil && recording.StartTime.Before(*startDate) {
				continue
			}
			if endDate != nil && recording.StartTime.After(*endDate) {
				continue
			}
			filteredSegments = append(filteredSegments, segment)
		}
		segments = filteredSegments
	}

	// Get blockquotes for transcripts using stored blockquote_id (if available)
	// Batch fetch blockquotes to minimize queries
	blockquoteMap := make(map[string]*storage.LifelogBlockquote)
	blockquoteIDSet := make(map[string]bool)
	segmentsWithBlockquoteID := 0
	for _, segment := range segments {
		if segment.BlockquoteID != nil && *segment.BlockquoteID != "" {
			blockquoteIDSet[*segment.BlockquoteID] = true
			segmentsWithBlockquoteID++
		}
	}

	// Log debug info
	if len(segments) > 0 {
		log.Printf("[DEBUG] enrichSegmentsWithTranscripts: %d segments, %d with blockquote_id, %d unique blockquote IDs", len(segments), segmentsWithBlockquoteID, len(blockquoteIDSet))
	}

	// Fetch all unique blockquotes
	blockquotesFound := 0
	for blockquoteID := range blockquoteIDSet {
		blockquote, err := s.storage.GetLifelogBlockquote(ctx, blockquoteID)
		if err != nil {
			// Log but continue - missing blockquote is not fatal
			log.Printf("[DEBUG] Failed to fetch blockquote %s: %v", blockquoteID, err)
			continue
		}
		blockquoteMap[blockquoteID] = blockquote
		blockquotesFound++
	}
	
	if len(blockquoteIDSet) > 0 {
		log.Printf("[DEBUG] Fetched %d/%d blockquotes successfully", blockquotesFound, len(blockquoteIDSet))
	}

	// Build result with transcripts
	for _, segment := range segments {
		recording, exists := recordingMap[segment.RecordingID]
		if !exists {
			continue // Skip if recording not found
		}

		// Calculate absolute time (recording start + segment start)
		absoluteTime := recording.StartTime.Add(time.Duration(segment.StartTime * float64(time.Second)))

		// Get transcript from blockquote if available
		transcript := ""
		if segment.BlockquoteID != nil && *segment.BlockquoteID != "" {
			if blockquote, exists := blockquoteMap[*segment.BlockquoteID]; exists {
				transcript = blockquote.Content
				if transcript == "" {
					log.Printf("[DEBUG] Blockquote %s has empty content", *segment.BlockquoteID)
				}
			} else {
				log.Printf("[DEBUG] Blockquote %s not found in map (segment ID: %d)", *segment.BlockquoteID, segment.ID)
			}
		}

		result = append(result, SegmentWithTranscript{
			ID:          segment.ID,
			SpeakerID:   segment.SpeakerID,
			RecordingID: segment.RecordingID,
			StartTime:   segment.StartTime,
			EndTime:     segment.EndTime,
			Duration:    segment.Duration,
			Transcript:  transcript,
			Time:        absoluteTime.Format("2006-01-02 15:04:05"),
		})
	}

	return result, nil
}

// HandleGetRecordingAudio handles GET /api/recordings/{recordingId}/audio
func (s *APIServer) HandleGetRecordingAudio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, &ErrMethodNotAllowed{Method: r.Method})
		return
	}

	recordingID := chi.URLParam(r, "recordingId")
	if recordingID == "" {
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "recording ID is required"})
		return
	}

	ctx := r.Context()

	// Get recording
	recording, err := s.storage.GetRecording(ctx, recordingID)
	if err != nil {
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, &ErrBadRequest{Message: "recording not found: " + recordingID})
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Parse query parameters for time range
	var startTime, endTime float64
	if startStr := r.URL.Query().Get("start"); startStr != "" {
		startTime, err = strconv.ParseFloat(startStr, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "invalid start parameter: " + err.Error()})
			return
		}
	}
	if endStr := r.URL.Query().Get("end"); endStr != "" {
		endTime, err = strconv.ParseFloat(endStr, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "invalid end parameter: " + err.Error()})
			return
		}
	}

	// Convert relative times to absolute times
	// If no start/end specified, use full recording duration
	var absoluteStart, absoluteEnd time.Time
	if startTime == 0 && endTime == 0 {
		// No time range specified, use full recording
		absoluteStart = recording.StartTime
		absoluteEnd = recording.StartTime.Add(time.Duration(recording.Duration * float64(time.Second)))
	} else {
		// Time range specified (relative to recording start)
		absoluteStart = recording.StartTime.Add(time.Duration(startTime * float64(time.Second)))
		if endTime == 0 {
			// Only start specified, use recording end
			absoluteEnd = recording.StartTime.Add(time.Duration(recording.Duration * float64(time.Second)))
		} else {
			absoluteEnd = recording.StartTime.Add(time.Duration(endTime * float64(time.Second)))
		}
	}

	// Convert to milliseconds for Limitless API
	startMs := absoluteStart.UnixMilli()
	endMs := absoluteEnd.UnixMilli()

	// Return Limitless API information for client to call directly
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"recording_id": recordingID,
		"api_url":      "https://api.limitless.ai/v1/download-audio",
		"method":       "GET",
		"headers": map[string]string{
			"X-API-Key": "REQUIRED", // Client must provide their own API key
		},
		"query_params": map[string]interface{}{
			"startMs": startMs,
			"endMs":   endMs,
		},
		"absolute_start_time": absoluteStart.Format(time.RFC3339),
		"absolute_end_time":   absoluteEnd.Format(time.RFC3339),
		"start_ms":            startMs,
		"end_ms":              endMs,
		"content_type":        "audio/ogg", // Expected response content type
	})
}

