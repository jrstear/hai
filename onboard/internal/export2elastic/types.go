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
// Returns the number of speakers created/updated, segments indexed, a boolean indicating if it was skipped, and any error
// If segments already exist for the recording, they are skipped (not re-indexed)
func (e *Exporter) ExportResult(ctx context.Context, result *diarization.Result, audioFilePath string) (int, int, bool, error) {
	// 1. Match or create speakers
	speakerMap, err := e.matchSpeakers(ctx, result)
	if err != nil {
		return 0, 0, false, err
	}

	// 2. Create recording
	recording, err := e.createRecording(ctx, result, audioFilePath)
	if err != nil {
		return 0, 0, false, err
	}

	// 3. Check if segments already exist for this recording
	existingSegments, err := e.storage.GetSegmentsByRecording(ctx, recording.ID)
	if err != nil && err != storage.ErrNotFound {
		return len(speakerMap), 0, false, err
	}
	
	// If segments already exist, skip indexing (similar to how diarization skips if JSON exists)
	if len(existingSegments) > 0 {
		// Segments already exist - skip indexing
		return len(speakerMap), len(existingSegments), true, nil
	}

	// 4. Transform and index segments
	segments, err := e.transformSegments(result, speakerMap, recording.ID)
	if err != nil {
		return 0, 0, false, err
	}

	// 5. Bulk index segments
	numIndexed, err := e.storage.CreateSegments(ctx, segments)
	if err != nil {
		return len(speakerMap), 0, false, err
	}

	return len(speakerMap), numIndexed, false, nil
}

// matchSpeakers matches diarization speaker embeddings to existing speakers
// Returns a map from local speaker ID (e.g., "SPEAKER_00") to global speaker ID (e.g., "spkr_abc123")
// Skips speakers with zero-magnitude embeddings (logs warning, creates speaker without embedding matching)
func (e *Exporter) matchSpeakers(ctx context.Context, result *diarization.Result) (map[string]string, error) {
	speakerMap := make(map[string]string)
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
				// Zero-magnitude embedding: skip matching but still create a speaker ID for segments
				// This can happen with silent segments or diarization edge cases
				log.Printf("Warning: Speaker %s has zero-magnitude embedding, skipping speaker matching. This may indicate silent segments or a diarization edge case.", localSpeakerID)
				
				// Create a new speaker without trying to match (we can't store zero embeddings)
				// For now, we'll create a speaker ID but won't store it in Elasticsearch
				// Segments will still reference this speaker ID, but it won't be matchable
				speakerID := generateSpeakerID()
				speakerMap[localSpeakerID] = speakerID
				
				// Log that we're skipping this speaker's embedding storage
				log.Printf("Skipping Elasticsearch storage for speaker %s (zero-magnitude embedding). Segments will still be indexed with speaker ID %s.", localSpeakerID, speakerID)
				continue
			}
			// Other validation errors (wrong dimension) should fail
			return nil, fmt.Errorf("invalid embedding for speaker %s: %w", localSpeakerID, err)
		}

		// Find similar speakers using kNN search
		matches, err := e.storage.FindSimilarSpeakers(
			ctx,
			embedding,
			storage.SimilarityThreshold,
			1, // Only need the best match
		)
		if err != nil {
			return nil, err
		}

		var speakerID string
		if len(matches) > 0 && matches[0].Similarity >= storage.SimilarityThreshold {
			// Found a match - use existing speaker
			speakerID = matches[0].Speaker.ID
			
			// Update last_seen timestamp
			update := &storage.Speaker{
				ID:       speakerID,
				LastSeen: now,
			}
			if err := e.storage.UpdateSpeaker(ctx, update); err != nil {
				return nil, err
			}
		} else {
			// No match found - create new speaker
			speakerID = generateSpeakerID()
			speaker := &storage.Speaker{
				ID:        speakerID,
				Embedding: embedding,
				FirstSeen: now,
				LastSeen:  now,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := e.storage.CreateSpeaker(ctx, speaker); err != nil {
				return nil, err
			}
		}

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

// transformSegments transforms diarization segments to storage segments
func (e *Exporter) transformSegments(result *diarization.Result, speakerMap map[string]string, recordingID string) ([]*storage.Segment, error) {
	segments := make([]*storage.Segment, 0, len(result.Segments))
	now := time.Now().UTC()

	for _, diarizationSegment := range result.Segments {
		// Map local speaker ID to global speaker ID
		globalSpeakerID, ok := speakerMap[diarizationSegment.Speaker]
		if !ok {
			return nil, &ErrSpeakerNotFound{LocalSpeakerID: diarizationSegment.Speaker}
		}

		segment := &storage.Segment{
			SpeakerID:      globalSpeakerID,
			RecordingID:    recordingID,
			LocalSpeakerID: &diarizationSegment.Speaker,
			StartTime:      diarizationSegment.Start,
			EndTime:        diarizationSegment.End,
			Duration:       diarizationSegment.Duration,
			CreatedAt:      now,
		}

		segments = append(segments, segment)
	}

	return segments, nil
}

