package export2elastic

import (
	"context"
	"fmt"
	"log"
	"time"

	"hai/onboard/internal/diarization"
	"hai/storage"
)

// Exporter handles exporting diarization results to Elasticsearch
type Exporter struct {
	storage storage.Storage
}

// NewExporter creates a new exporter instance
func NewExporter(s storage.Storage) *Exporter {
	return &Exporter{
		storage: s,
	}
}

// ExportResult exports a diarization result to Elasticsearch
// Returns the number of speakers matched, segments indexed, a boolean indicating if it was skipped, and any error
// If segments already exist for the recording, they are skipped (not re-indexed)
// Note: Speakers are never created during export - only matched against existing centroids
func (e *Exporter) ExportResult(ctx context.Context, result *diarization.Result, audioFilePath string) (int, int, bool, error) {
	// 1. Match against existing centroid speakers (never creates speakers)
	speakerMap, err := e.matchSpeakers(ctx, result)
	if err != nil {
		return 0, 0, false, err
	}
	
	// Count how many speakers were matched (non-nil values)
	matchedCount := 0
	for _, speakerID := range speakerMap {
		if speakerID != nil {
			matchedCount++
		}
	}

	// 2. Create recording
	recording, err := e.createRecording(ctx, result, audioFilePath)
	if err != nil {
		return 0, 0, false, err
	}

	// 3. Check if segments already exist for this recording
	existingSegments, err := e.storage.GetSegmentsByRecording(ctx, recording.ID)
	if err != nil && err != storage.ErrNotFound {
		return matchedCount, 0, false, err
	}
	
	// If segments already exist, skip indexing (similar to how diarization skips if JSON exists)
	if len(existingSegments) > 0 {
		// Segments already exist - skip indexing
		return matchedCount, len(existingSegments), true, nil
	}

	// 4. Create speaker embeddings and calculate durations
	// TODO (hai-vwg): Implement selective embedding storage policy instead of storing all embeddings
	// For now, store all embeddings to enable clustering. Future: only store if meets criteria
	// (duration threshold, novelty threshold, match status, etc.)
	embeddingMap, err := e.createSpeakerEmbeddings(ctx, result, recording.ID, speakerMap)
	if err != nil {
		return 0, 0, false, err
	}

	// 5. Transform and index segments (with SpeakerEmbeddingID)
	segments, err := e.transformSegments(result, speakerMap, embeddingMap, recording.ID)
	if err != nil {
		return 0, 0, false, err
	}

	// 6. Bulk index segments
	numIndexed, err := e.storage.CreateSegments(ctx, segments)
	if err != nil {
		return matchedCount, 0, false, err
	}

	return matchedCount, numIndexed, false, nil
}

// matchSpeakers matches diarization speaker embeddings to existing centroid speakers
// Returns a map from local speaker ID (e.g., "SPEAKER_00") to global speaker ID (e.g., "spkr_abc123")
// Returns nil SpeakerID if no match found (speakers are only created during clustering)
// Skips speakers with zero-magnitude embeddings (logs warning, returns nil SpeakerID)
func (e *Exporter) matchSpeakers(ctx context.Context, result *diarization.Result) (map[string]*string, error) {
	speakerMap := make(map[string]*string)
	now := time.Now().UTC()

	// For each speaker embedding in the diarization result
	for localSpeakerID, embeddingFloat64 := range result.SpeakerEmbeddings {
		// Convert []float64 to []float32
		embedding := make([]float32, len(embeddingFloat64))
		for i, v := range embeddingFloat64 {
			embedding[i] = float32(v)
		}

		// Validate embedding (dimension and magnitude)
		if err := storage.ValidateEmbedding(embedding); err != nil {
			if err == storage.ErrZeroMagnitudeEmbedding {
				// Zero-magnitude embedding: skip matching
				// This can happen with silent segments or diarization edge cases
				log.Printf("Warning: Speaker %s has zero-magnitude embedding, skipping speaker matching. This may indicate silent segments or a diarization edge case.", localSpeakerID)
				// Return nil SpeakerID - segments will have no speaker assignment
				speakerMap[localSpeakerID] = nil
				continue
			}
			// Other validation errors (wrong dimension) should fail
			return nil, fmt.Errorf("invalid embedding for speaker %s: %w", localSpeakerID, err)
		}

		// Find similar speakers using kNN search against centroids only
		// Request multiple matches to ensure we get the highest similarity match
		// Results are sorted by similarity descending, so matches[0] is the best match
		matches, err := e.storage.FindSimilarSpeakers(
			ctx,
			embedding,
			storage.SimilarityThreshold,
			10, // Get multiple matches to find the highest similarity
			true, // onlyCentroids: only match against centroid speakers
		)
		if err != nil {
			return nil, err
		}

		var speakerID *string = nil
		if len(matches) > 0 && matches[0].Similarity >= storage.SimilarityThreshold {
			// Found a match - use the highest similarity match (first in sorted results)
			speakerID = &matches[0].Speaker.ID
			
			// Update last_seen timestamp
			update := &storage.Speaker{
				ID:       *speakerID,
				LastSeen: now,
			}
			if err := e.storage.UpdateSpeaker(ctx, update); err != nil {
				return nil, err
			}
		}
		// If no match: speakerID stays nil (will be set during clustering)

		speakerMap[localSpeakerID] = speakerID
	}

	return speakerMap, nil
}

// createRecording creates a recording record from diarization result
func (e *Exporter) createRecording(ctx context.Context, result *diarization.Result, audioFilePath string) (*storage.Recording, error) {
	// Extract recording start time from file path
	// Path format: data/YYYY/MM/DD/HH.ogg (in UTC)
	startTime, err := extractRecordingStartTime(audioFilePath)
	if err != nil {
		return nil, err
	}

	// Generate recording ID: rec_YYYY_MM_DD_HH
	recordingID := generateRecordingID(startTime)

	// Check if recording already exists
	existing, err := e.storage.GetRecording(ctx, recordingID)
	if err == nil {
		// Recording exists, update diarization metadata
		update := &storage.Recording{
			ID: recordingID,
		}
		if result.ProcessingTime > 0 {
			update.ProcessingTime = &result.ProcessingTime
		}
		if result.RTF > 0 {
			update.RTF = &result.RTF
		}
		if result.Device != "" {
			update.Device = &result.Device
		}
		now := time.Now().UTC()
		update.DiarizedAt = &now

		if err := e.storage.UpdateRecording(ctx, update); err != nil {
			return nil, err
		}
		return existing, nil
	}
	if err != storage.ErrNotFound {
		return nil, err
	}

	// Create new recording
	recording := &storage.Recording{
		ID:        recordingID,
		FilePath:  audioFilePath,
		StartTime: startTime,
		Duration:  result.AudioDuration,
		CreatedAt: time.Now().UTC(),
	}

	if result.ProcessingTime > 0 {
		recording.ProcessingTime = &result.ProcessingTime
	}
	if result.RTF > 0 {
		recording.RTF = &result.RTF
	}
	if result.Device != "" {
		recording.Device = &result.Device
	}
	now := time.Now().UTC()
	recording.DiarizedAt = &now

	// Extract format from file extension
	if ext := getFileExtension(audioFilePath); ext != "" {
		recording.Format = &ext
	}

	if err := e.storage.CreateRecording(ctx, recording); err != nil {
		return nil, err
	}

	return recording, nil
}

// createSpeakerEmbeddings creates SpeakerEmbedding records for all speakers in the diarization result
// Returns a map from local speaker ID to SpeakerEmbedding ID
// TODO (hai-vwg): Implement selective storage policy - only store embeddings that meet criteria
// (duration threshold, novelty threshold, match status, etc.)
func (e *Exporter) createSpeakerEmbeddings(ctx context.Context, result *diarization.Result, recordingID string, speakerMap map[string]*string) (map[string]string, error) {
	embeddingMap := make(map[string]string)
	now := time.Now().UTC()

	// Calculate total duration for each speaker (sum of segment durations)
	speakerDurations := make(map[string]float64)
	for _, seg := range result.Segments {
		speakerDurations[seg.Speaker] += seg.Duration
	}

	// Create SpeakerEmbedding record for each speaker
	for localSpeakerID, embeddingFloat64 := range result.SpeakerEmbeddings {
		// Convert []float64 to []float32
		embedding := make([]float32, len(embeddingFloat64))
		for i, v := range embeddingFloat64 {
			embedding[i] = float32(v)
		}

		// Validate embedding (skip zero-magnitude embeddings)
		if err := storage.ValidateEmbedding(embedding); err != nil {
			if err == storage.ErrZeroMagnitudeEmbedding {
				// Skip zero-magnitude embeddings (already logged in matchSpeakers)
				continue
			}
			// Other validation errors should fail
			return nil, fmt.Errorf("invalid embedding for speaker %s: %w", localSpeakerID, err)
		}

		// Get duration for this speaker
		duration := speakerDurations[localSpeakerID]

		// Get matched speaker ID (may be nil)
		matchedSpeakerID := speakerMap[localSpeakerID]

		// Create SpeakerEmbedding record
		embeddingID := generateSpeakerEmbeddingID()
		speakerEmbedding := &storage.SpeakerEmbedding{
			ID:              embeddingID,
			SpeakerID:       matchedSpeakerID, // May be nil if no match (will be set during clustering)
			RecordingID:     recordingID,
			LocalSpeakerID:  localSpeakerID,
			Embedding:       embedding,
			DurationSeconds: duration,
			CreatedAt:       now,
		}

		if err := e.storage.CreateSpeakerEmbedding(ctx, speakerEmbedding); err != nil {
			return nil, fmt.Errorf("failed to create speaker embedding for %s: %w", localSpeakerID, err)
		}

		embeddingMap[localSpeakerID] = embeddingID
	}

	return embeddingMap, nil
}

// transformSegments transforms diarization segments to storage segments
// speakerMap maps local speaker ID to global speaker ID (can be nil if no match found)
// embeddingMap maps local speaker ID to SpeakerEmbedding ID
func (e *Exporter) transformSegments(result *diarization.Result, speakerMap map[string]*string, embeddingMap map[string]string, recordingID string) ([]*storage.Segment, error) {
	segments := make([]*storage.Segment, 0, len(result.Segments))
	now := time.Now().UTC()

	for _, diarizationSegment := range result.Segments {
		// Map local speaker ID to global speaker ID (may be nil if no match)
		globalSpeakerID, ok := speakerMap[diarizationSegment.Speaker]
		if !ok {
			return nil, &ErrSpeakerNotFound{LocalSpeakerID: diarizationSegment.Speaker}
		}

		// Get SpeakerEmbeddingID (may not exist if embedding was skipped)
		var speakerEmbeddingID *string
		if embID, exists := embeddingMap[diarizationSegment.Speaker]; exists {
			speakerEmbeddingID = &embID
		}

		segment := &storage.Segment{
			SpeakerEmbeddingID: speakerEmbeddingID, // May be nil if embedding was skipped
			SpeakerID:          globalSpeakerID,    // Can be nil if no match found
			RecordingID:        recordingID,
			LocalSpeakerID:     &diarizationSegment.Speaker,
			StartTime:          diarizationSegment.Start,
			EndTime:            diarizationSegment.End,
			Duration:           diarizationSegment.Duration,
			CreatedAt:          now,
		}

		segments = append(segments, segment)
	}

	return segments, nil
}

