package storage

import (
	"context"
	"errors"
	"math"
	"time"
)

var (
	// ErrNotFound is returned when a requested resource is not found
	ErrNotFound = errors.New("resource not found")
	// ErrDuplicateKey is returned when attempting to create a resource with an existing key
	ErrDuplicateKey = errors.New("duplicate key")
	// ErrInvalidEmbedding is returned when an embedding has the wrong dimension
	ErrInvalidEmbedding = errors.New("invalid embedding dimension")
	// ErrZeroMagnitudeEmbedding is returned when an embedding has zero magnitude (all zeros)
	ErrZeroMagnitudeEmbedding = errors.New("embedding has zero magnitude")
)

// Storage is the interface that all storage backends must implement
// It provides a unified API for storing and querying speakers, recordings, and segments
type Storage interface {
	// Close closes the storage connection and releases resources
	Close() error

	// Health checks if the storage backend is healthy and accessible
	Health(ctx context.Context) error

	// Speaker operations

	// CreateSpeaker creates a new speaker record
	// Returns ErrDuplicateKey if speaker with same ID already exists
	// Returns ErrInvalidEmbedding if embedding dimension is incorrect
	CreateSpeaker(ctx context.Context, speaker *Speaker) error

	// GetSpeaker retrieves a speaker by ID
	// Returns ErrNotFound if speaker doesn't exist
	GetSpeaker(ctx context.Context, id string) (*Speaker, error)

	// FindSimilarSpeakers finds speakers with similar embeddings using cosine similarity
	// Returns speakers sorted by similarity (highest first)
	// threshold: minimum similarity score (0.0 to 1.0), use SimilarityThreshold constant
	// limit: maximum number of results to return (0 = no limit)
	FindSimilarSpeakers(ctx context.Context, embedding []float32, threshold float64, limit int) ([]SpeakerMatch, error)

	// UpdateSpeaker updates an existing speaker record
	// Only non-zero fields are updated
	// Returns ErrNotFound if speaker doesn't exist
	UpdateSpeaker(ctx context.Context, speaker *Speaker) error

	// ListSpeakers lists all speakers, optionally filtered by contact_id
	// If contactID is not nil, only returns speakers with that contact_id
	// If contactID is nil, returns all speakers
	ListSpeakers(ctx context.Context, contactID *string) ([]*Speaker, error)

	// Recording operations

	// CreateRecording creates a new recording record
	// Returns ErrDuplicateKey if recording with same ID already exists
	CreateRecording(ctx context.Context, recording *Recording) error

	// GetRecording retrieves a recording by ID
	// Returns ErrNotFound if recording doesn't exist
	GetRecording(ctx context.Context, id string) (*Recording, error)

	// GetRecordingByPath retrieves a recording by file path
	// Returns ErrNotFound if recording doesn't exist
	GetRecordingByPath(ctx context.Context, filePath string) (*Recording, error)

	// ListRecordings lists all recordings, optionally filtered by time range
	// If startTime is not nil, only returns recordings starting at or after startTime
	// If endTime is not nil, only returns recordings starting before endTime
	ListRecordings(ctx context.Context, startTime *time.Time, endTime *time.Time) ([]*Recording, error)

	// UpdateRecording updates an existing recording record
	// Only non-zero fields are updated
	// Returns ErrNotFound if recording doesn't exist
	UpdateRecording(ctx context.Context, recording *Recording) error

	// Segment operations

	// CreateSegment creates a new segment record
	CreateSegment(ctx context.Context, segment *Segment) error

	// CreateSegments creates multiple segment records in a single operation (bulk insert)
	// This is more efficient than calling CreateSegment multiple times
	// Returns the number of segments created and any error
	CreateSegments(ctx context.Context, segments []*Segment) (int, error)

	// GetSegment retrieves a segment by ID
	// Returns ErrNotFound if segment doesn't exist
	GetSegment(ctx context.Context, id int64) (*Segment, error)

	// GetSegmentsBySpeaker retrieves all segments for a given speaker
	// Results are sorted by start_time ascending
	GetSegmentsBySpeaker(ctx context.Context, speakerID string) ([]*Segment, error)

	// GetSegmentsByRecording retrieves all segments for a given recording
	// Results are sorted by start_time ascending
	GetSegmentsByRecording(ctx context.Context, recordingID string) ([]*Segment, error)

	// GetSegmentsByTimeRange retrieves segments within a time range for a recording
	// startTime and endTime are relative to the recording start (in seconds)
	// Results are sorted by start_time ascending
	GetSegmentsByTimeRange(ctx context.Context, recordingID string, startTime, endTime float64) ([]*Segment, error)

	// UpdateSegmentByteOffsets updates the byte offsets for a segment
	// This is used after byte offset indexing is complete
	// Returns ErrNotFound if segment doesn't exist
	UpdateSegmentByteOffsets(ctx context.Context, segmentID int64, startByteOffset, endByteOffset int64) error

	// Lifelog operations

	// CreateLifelog creates a new lifelog document
	// Returns ErrDuplicateKey if lifelog with same ID already exists
	CreateLifelog(ctx context.Context, lifelog *Lifelog) error

	// GetLifelog retrieves a lifelog by ID
	// Returns ErrNotFound if lifelog doesn't exist
	GetLifelog(ctx context.Context, id string) (*Lifelog, error)

	// ListLifelogs lists all lifelogs, optionally filtered by time range
	// If startTime is not nil, only returns lifelogs starting at or after startTime
	// If endTime is not nil, only returns lifelogs starting before endTime
	// Results are sorted by start_time ascending
	ListLifelogs(ctx context.Context, startTime *time.Time, endTime *time.Time) ([]*Lifelog, error)

	// UpdateLifelog updates an existing lifelog record
	// Only non-zero fields are updated
	// Returns ErrNotFound if lifelog doesn't exist
	UpdateLifelog(ctx context.Context, lifelog *Lifelog) error

	// CreateLifelogBlockquote creates a blockquote from a lifelog
	// Returns ErrDuplicateKey if blockquote with same ID already exists
	CreateLifelogBlockquote(ctx context.Context, blockquote *LifelogBlockquote) error

	// CreateLifelogBlockquotes creates multiple blockquote records in a single operation (bulk insert)
	// This is more efficient than calling CreateLifelogBlockquote multiple times
	// Returns the number of blockquotes created and any error
	CreateLifelogBlockquotes(ctx context.Context, blockquotes []*LifelogBlockquote) (int, error)

	// GetLifelogBlockquote retrieves a blockquote by ID
	// Returns ErrNotFound if blockquote doesn't exist
	GetLifelogBlockquote(ctx context.Context, id string) (*LifelogBlockquote, error)

	// GetLifelogBlockquotesByLifelog retrieves all blockquotes for a given lifelog
	// Results are sorted by start_time ascending
	GetLifelogBlockquotesByLifelog(ctx context.Context, lifelogID string) ([]*LifelogBlockquote, error)

	// GetLifelogBlockquotesByTimeRange retrieves blockquotes within a time range
	// startTime and endTime are absolute UTC timestamps
	// Results are sorted by start_time ascending
	GetLifelogBlockquotesByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*LifelogBlockquote, error)

	// UpdateLifelogBlockquote updates an existing blockquote record
	// Only non-zero fields are updated
	// Returns ErrNotFound if blockquote doesn't exist
	UpdateLifelogBlockquote(ctx context.Context, blockquote *LifelogBlockquote) error
}

// ValidateEmbedding checks if an embedding has the correct dimension and non-zero magnitude
func ValidateEmbedding(embedding []float32) error {
	if len(embedding) != EmbeddingDimension {
		return ErrInvalidEmbedding
	}

	// Check for zero magnitude (all zeros)
	var sumSquares float64
	for _, v := range embedding {
		sumSquares += float64(v * v)
	}
	if sumSquares == 0 {
		return ErrZeroMagnitudeEmbedding
	}

	return nil
}

// CosineSimilarity computes the cosine similarity between two embedding vectors
// Returns a value between -1.0 and 1.0, where 1.0 means identical
// Both embeddings must have the same dimension (typically 256)
// Formula: cos(θ) = (A · B) / (||A|| * ||B||)
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	// Take square root to get L2 norm
	normA = math.Sqrt(normA)
	normB = math.Sqrt(normB)
	return dotProduct / (normA * normB)
}

// FindBestMatch finds the speaker with the highest similarity to the given embedding
// Returns the best match if similarity >= threshold, otherwise returns nil
// This is a helper function for speaker matching logic
func FindBestMatch(embedding []float32, candidates []*Speaker, threshold float64) *SpeakerMatch {
	if len(candidates) == 0 {
		return nil
	}

	var bestMatch *SpeakerMatch
	bestSimilarity := threshold

	for _, candidate := range candidates {
		similarity := CosineSimilarity(embedding, candidate.Embedding)
		if similarity >= bestSimilarity {
			bestSimilarity = similarity
			bestMatch = &SpeakerMatch{
				Speaker:    candidate,
				Similarity: similarity,
			}
		}
	}

	return bestMatch
}
