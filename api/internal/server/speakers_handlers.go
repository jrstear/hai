package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"time"

	"hai/api/internal/contacts"
	"hai/storage"

	"github.com/go-chi/chi/v5"
)

// SpeakerWithStats represents a speaker with computed statistics
type SpeakerWithStats struct {
	ID            string    `json:"id"`
	ContactID     *string   `json:"contact_id,omitempty"`
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
	Duration      int64     `json:"duration"`        // Total speaking time in seconds
	DurationDisplay string  `json:"duration_display"` // Formatted duration string (e.g., "0:07 a" or "9:09")
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// HandleListUnassociatedSpeakers handles GET /api/speakers/unassociated
// Returns only speaker data without computing stats from segments.
// Stats (duration, detailed last_seen) should be computed only when a speaker is selected.
func (s *APIServer) HandleListUnassociatedSpeakers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, &ErrMethodNotAllowed{Method: r.Method})
		return
	}

	start := time.Now()
	ctx := r.Context()

	// List all speakers (we'll filter for unassociated in code)
	listStart := time.Now()
	allSpeakers, err := s.storage.ListSpeakers(ctx, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	listElapsed := time.Since(listStart)
	log.Printf("[TIMING] ListSpeakers: %v (%d total speakers)", listElapsed, len(allSpeakers))

	// Get speaker durations via ES aggregation (single query, much faster than N+1)
	durationStart := time.Now()
	speakerDurations, err := s.getSpeakerDurations(ctx)
	if err != nil {
		log.Printf("[WARN] Failed to get speaker durations: %v", err)
		speakerDurations = make(map[string]int64) // Continue with empty map
	}
	durationElapsed := time.Since(durationStart)
	log.Printf("[TIMING] getSpeakerDurations (aggregation): %v", durationElapsed)

	// Filter for unassociated speakers and merge duration data
	unassociated := make([]SpeakerWithStats, 0)
	for _, speaker := range allSpeakers {
		// Skip speakers that already have a contact_id
		if speaker.ContactID != nil && *speaker.ContactID != "" {
			continue
		}

		// Get duration from aggregation result (0 if not found)
		duration := speakerDurations[speaker.ID]

		unassociated = append(unassociated, SpeakerWithStats{
			ID:        speaker.ID,
			ContactID: speaker.ContactID,
			FirstSeen: speaker.FirstSeen,
			LastSeen:  speaker.LastSeen, // Use existing field from speaker document
			Duration:  duration,          // From ES aggregation
			CreatedAt: speaker.CreatedAt,
			UpdatedAt: speaker.UpdatedAt,
		})
	}

	// Sort by decreasing duration (biggest speaker first)
	// For speakers with the same duration, sort by last_seen descending (most recent first)
	sort.Slice(unassociated, func(i, j int) bool {
		if unassociated[i].Duration != unassociated[j].Duration {
			return unassociated[i].Duration > unassociated[j].Duration
		}
		// Same duration: sort by last_seen descending (most recent first)
		return unassociated[i].LastSeen.After(unassociated[j].LastSeen)
	})

	// Format duration display strings with letter suffixes for same-duration speakers
	// Most recently seen gets "a", next gets "b", etc.
	// Only add suffixes when multiple speakers have the same duration
	if len(unassociated) > 0 {
		currentDuration := unassociated[0].Duration
		groupStart := 0
		
		for i := 1; i <= len(unassociated); i++ {
			// Check if we've reached the end or a new duration group
			if i == len(unassociated) || unassociated[i].Duration != currentDuration {
				groupSize := i - groupStart
				
				// Format duration display strings for this group
				for j := 0; j < groupSize; j++ {
					durationStr := formatDuration(unassociated[groupStart+j].Duration)
					
					// Only add suffix if multiple speakers have the same duration
					if groupSize > 1 {
						suffix := string(rune('a' + j))
						unassociated[groupStart+j].DurationDisplay = durationStr + " " + suffix
					} else {
						unassociated[groupStart+j].DurationDisplay = durationStr
					}
				}
				
				// Move to next group
				if i < len(unassociated) {
					currentDuration = unassociated[i].Duration
					groupStart = i
				}
			}
		}
	}

	totalElapsed := time.Since(start)
	log.Printf("[TIMING] HandleListUnassociatedSpeakers total: %v (%d unassociated speakers)", totalElapsed, len(unassociated))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"speakers": unassociated,
		"total":    len(unassociated),
	})
}

// SpeakerStats holds computed statistics for a speaker
type SpeakerStats struct {
	Duration int64     // Total speaking time in seconds
	LastSeen time.Time // Most recent recording time
}

// getSpeakerDurations uses ES aggregation to get sum(duration) grouped by speaker_id
// Returns a map of speaker_id -> total duration in seconds
// This is much more efficient than N+1 queries
func (s *APIServer) getSpeakerDurations(ctx context.Context) (map[string]int64, error) {
	// Use ES aggregation: terms aggregation on speaker_id with sum sub-aggregation on duration
	// Equivalent to: SELECT speaker_id, SUM(duration) FROM segments GROUP BY speaker_id
	query := map[string]interface{}{
		"size": 0, // Don't return documents, only aggregations
		"aggs": map[string]interface{}{
			"by_speaker": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "speaker_id",
					"size":  10000, // Support up to 10k speakers
				},
				"aggs": map[string]interface{}{
					"total_duration": map[string]interface{}{
						"sum": map[string]interface{}{
							"field": "duration",
						},
					},
				},
			},
		},
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal aggregation query: %w", err)
	}

	// Query the segments index using ES aggregation
	res, err := s.esClient.Search(
		s.esClient.Search.WithIndex("segments"),
		s.esClient.Search.WithBody(bytes.NewReader(queryJSON)),
		s.esClient.Search.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute aggregation query: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("aggregation query failed: %s", string(body))
	}

	var result struct {
		Aggregations struct {
			BySpeaker struct {
				Buckets []struct {
					Key          string  `json:"key"`
					TotalDuration struct {
						Value float64 `json:"value"`
					} `json:"total_duration"`
				} `json:"buckets"`
			} `json:"by_speaker"`
		} `json:"aggregations"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode aggregation result: %w", err)
	}

	// Build map of speaker_id -> duration
	durations := make(map[string]int64)
	for _, bucket := range result.Aggregations.BySpeaker.Buckets {
		durations[bucket.Key] = int64(bucket.TotalDuration.Value)
	}

	return durations, nil
}

// formatDuration converts seconds to MM:SS format
func formatDuration(seconds int64) string {
	minutes := seconds / 60
	secs := seconds % 60
	return fmt.Sprintf("%d:%02d", minutes, secs)
}

// computeSpeakerStats calculates duration and last_seen for a speaker
func (s *APIServer) computeSpeakerStats(ctx context.Context, speakerID string) (*SpeakerStats, error) {
	// Get all segments for this speaker
	segments, err := s.storage.GetSegmentsBySpeaker(ctx, speakerID)
	if err != nil {
		return nil, err
	}

	stats := &SpeakerStats{
		Duration: 0,
		LastSeen: time.Time{},
	}

	// Calculate total duration and find most recent recording
	// NOTE: This is doing N+1 queries - one GetRecording call per segment
	// This could be optimized by batching recording fetches
	for _, segment := range segments {
		// Sum duration (duration is already in seconds as float64)
		stats.Duration += int64(segment.Duration)

		// Get recording to find start_time
		// TODO: Optimize this - batch recording fetches or cache recordings
		recording, err := s.storage.GetRecording(ctx, segment.RecordingID)
		if err != nil {
			continue // Skip if recording not found
		}

		// Update last_seen if this recording is more recent
		// Use recording start_time + segment end_time for most accurate last_seen
		segmentEndTime := recording.StartTime.Add(time.Duration(segment.EndTime * float64(time.Second)))
		if segmentEndTime.After(stats.LastSeen) {
			stats.LastSeen = segmentEndTime
		}
	}

	return stats, nil
}

// HandleAssociateSpeaker handles POST /api/contacts/{contactId}/associate-speaker
func (s *APIServer) HandleAssociateSpeaker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, &ErrMethodNotAllowed{Method: r.Method})
		return
	}

	contactID := chi.URLParam(r, "contactId")
	if contactID == "" {
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "contact ID is required"})
		return
	}

	var req contacts.AssociateSpeakerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "invalid request body: " + err.Error()})
		return
	}

	if req.SpeakerID == "" {
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "speaker_id is required"})
		return
	}

	ctx := r.Context()

	// Verify contact exists
	contact, err := s.contacts.GetContact(ctx, contactID)
	if err != nil {
		if err.Error() == "contact not found: "+contactID {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Verify speaker exists
	speaker, err := s.storage.GetSpeaker(ctx, req.SpeakerID)
	if err != nil {
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, &ErrBadRequest{Message: "speaker not found: " + req.SpeakerID})
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Update speaker's contact_id
	speaker.ContactID = &contactID
	if err := s.storage.UpdateSpeaker(ctx, speaker); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Update contact's known status (recompute)
	known, err := s.computeKnownStatus(ctx, contactID)
	if err == nil {
		contact.Known = known
		// Update contact to reflect known status
		update := &contacts.Contact{
			Known: known,
		}
		s.contacts.UpdateContact(ctx, contactID, update)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"contact": contact,
		"speaker": speaker,
	})
}

