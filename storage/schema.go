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
	ID                 int64     `json:"id"`                   // Auto-increment ID (SQLite) or generated ID (Elasticsearch)
	SpeakerEmbeddingID *string   `json:"speaker_embedding_id"` // Optional: References SpeakerEmbedding.ID (NULL if embedding not stored)
	SpeakerID          *string   `json:"speaker_id"`           // Optional: Global speaker ID (NULL if no match, set if match >= threshold)
	RecordingID        string    `json:"recording_id"`         // Recording ID (references Recording.ID)
	LocalSpeakerID     *string   `json:"local_speaker_id"`     // Original SPEAKER_XX from diarization (optional, for debugging)
	BlockquoteID       *string   `json:"blockquote_id"`        // Optional: Best matching blockquote ID (for transcript lookup)
	StartTime          float64   `json:"start_time"`           // Start time in seconds (relative to recording start)
	EndTime            float64   `json:"end_time"`             // End time in seconds (relative to recording start)
	Duration           float64   `json:"duration"`             // Duration in seconds (end_time - start_time)
	StartByteOffset    *int64    `json:"start_byte_offset"`    // Byte offset for HTTP Range requests (optional, NULL if not indexed)
	EndByteOffset      *int64    `json:"end_byte_offset"`      // Byte offset for HTTP Range requests (optional, NULL if not indexed)
	CreatedAt          time.Time `json:"created_at"`
}

// SpeakerMatch represents a speaker match result from similarity search
type SpeakerMatch struct {
	Speaker    *Speaker `json:"speaker"`
	Similarity float64  `json:"similarity"` // Cosine similarity score (0.0 to 1.0)
}

// Lifelog represents a lifelog document from the Limitless API
type Lifelog struct {
	ID        string    `json:"id"`         // Limitless API lifelog ID
	Title     string    `json:"title"`      // Lifelog title
	Markdown  string    `json:"markdown"`   // Full markdown content
	StartTime time.Time `json:"start_time"` // When lifelog starts (UTC)
	EndTime   time.Time `json:"end_time"`   // When lifelog ends (UTC)
	CreatedAt time.Time `json:"created_at"` // When we fetched it
}

// LifelogBlockquote represents a blockquote (speaker segment) within a lifelog
// Equivalent to Limitless API's "blockquote" content type
type LifelogBlockquote struct {
	ID            string    `json:"id"`              // Generated ID
	LifelogID     string    `json:"lifelog_id"`      // References Lifelog.ID
	RecordingID   *string   `json:"recording_id"`    // Optional: Which recording this overlaps with
	Content       string    `json:"content"`         // Transcript text
	SpeakerName   string    `json:"speaker_name"`    // "You", "Unknown", "Jon Stearley", etc. (from Limitless)
	SpeakerID     *string   `json:"speaker_id"`      // Optional: Mapped to our global speaker ID (from diarization)
	ContactID     *string   `json:"contact_id"`      // Optional: Associated contact ID (user-assigned, from contacts index)
	StartOffsetMs int       `json:"start_offset_ms"` // Milliseconds from lifelog start
	EndOffsetMs   int       `json:"end_offset_ms"`   // Milliseconds from lifelog end
	StartTime     time.Time `json:"start_time"`      // Absolute start time (UTC)
	EndTime       time.Time `json:"end_time"`        // Absolute end time (UTC)
	CreatedAt     time.Time `json:"created_at"`
}

// EmbeddingDimension is the fixed dimension for speaker embeddings
const EmbeddingDimension = 256

// SimilarityThreshold is the default threshold for speaker matching
// Speakers with similarity >= this threshold are considered the same person
// Based on analysis: 0.85 matches 73.8% of embeddings, 0.82 matches 85.0%
// Using 0.82 as a better balance between precision and recall
const SimilarityThreshold = 0.82

// SpeakerEmbedding stores individual speaker embeddings from recordings
// Used for clustering and centroid computation
type SpeakerEmbedding struct {
	ID              string    `json:"id"`               // Unique ID for this embedding
	SpeakerID       *string   `json:"speaker_id"`       // Optional: Points to Speaker.ID (centroid), NULL until clustered or matched
	RecordingID     string    `json:"recording_id"`     // Which recording this came from
	LocalSpeakerID  string    `json:"local_speaker_id"` // SPEAKER_00, SPEAKER_01, etc. from that recording
	Embedding       []float32 `json:"embedding"`        // The actual embedding vector (256 dimensions)
	DurationSeconds float64   `json:"duration_seconds"` // Total seconds this speaker spoke in this recording (sum of segment durations)
	CreatedAt       time.Time `json:"created_at"`
}
