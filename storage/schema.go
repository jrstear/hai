package storage

import (
	"time"
)

// Speaker represents a unique voice identity that can appear across multiple recordings
type Speaker struct {
	ID        string    `json:"id"`         // Global speaker ID (UUID format: spkr_xxxxx)
	Embedding []float32 `json:"embedding"`  // 256-dimensional embedding vector
	FirstSeen time.Time `json:"first_seen"` // When this speaker was first detected
	LastSeen  time.Time `json:"last_seen"`  // When this speaker was last detected
	ContactID *string   `json:"contact_id"` // Optional: Associated contact ID (NULL if not associated)
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Recording represents a single audio file (typically 1-hour chunk)
type Recording struct {
	ID             string     `json:"id"`              // Recording ID (format: rec_YYYY_MM_DD_HH)
	FilePath       string     `json:"file_path"`       // Path to audio file (e.g., "data/2025/11/22/15.ogg")
	StartTime      time.Time  `json:"start_time"`      // When recording started (UTC)
	Duration       float64    `json:"duration"`        // Duration in seconds
	SampleRate     *int       `json:"sample_rate"`     // Audio sample rate (optional)
	Format         *string    `json:"format"`          // Audio format: "ogg", "mp3", etc. (optional)
	DiarizedAt     *time.Time `json:"diarized_at"`     // When diarization completed (optional)
	ProcessingTime *float64   `json:"processing_time"` // Diarization processing time in seconds (optional)
	RTF            *float64   `json:"rtf"`             // Real-time factor (processing_time / duration) (optional)
	Device         *string    `json:"device"`          // Device used for diarization: "mps", "cpu", "cuda" (optional)
	CreatedAt      time.Time  `json:"created_at"`
}

// Segment represents a time period during which a single speaker speaks
// Equivalent to Limitless API's "blockquote" type
type Segment struct {
	ID             int64      `json:"id"`              // Auto-increment ID (SQLite) or generated ID (Elasticsearch)
	SpeakerID      string     `json:"speaker_id"`      // Global speaker ID (references Speaker.ID)
	RecordingID    string     `json:"recording_id"`    // Recording ID (references Recording.ID)
	LocalSpeakerID *string    `json:"local_speaker_id"` // Original SPEAKER_XX from diarization (optional, for debugging)
	StartTime      float64    `json:"start_time"`      // Start time in seconds (relative to recording start)
	EndTime        float64    `json:"end_time"`        // End time in seconds (relative to recording start)
	Duration       float64    `json:"duration"`        // Duration in seconds (end_time - start_time)
	StartByteOffset *int64    `json:"start_byte_offset"` // Byte offset for HTTP Range requests (optional, NULL if not indexed)
	EndByteOffset   *int64    `json:"end_byte_offset"`   // Byte offset for HTTP Range requests (optional, NULL if not indexed)
	CreatedAt      time.Time  `json:"created_at"`
}

// SpeakerMatch represents a speaker match result from similarity search
type SpeakerMatch struct {
	Speaker    *Speaker `json:"speaker"`
	Similarity float64  `json:"similarity"` // Cosine similarity score (0.0 to 1.0)
}

// EmbeddingDimension is the fixed dimension for speaker embeddings
const EmbeddingDimension = 256

// SimilarityThreshold is the default threshold for speaker matching
// Speakers with similarity >= this threshold are considered the same person
const SimilarityThreshold = 0.85

