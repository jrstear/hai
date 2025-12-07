# Storage Package

This package provides a unified storage abstraction layer for speakers, recordings, and segments. It supports both SQLite and Elasticsearch backends through a common interface.

## Overview

The storage package consists of:

- **`schema.go`**: Core data types (Speaker, Recording, Segment)
- **`interface.go`**: Storage interface and helper functions
- **`sqlite/`**: SQLite implementation (to be implemented)
- **`elasticsearch/`**: Elasticsearch implementation (to be implemented)
- **`migrations/`**: SQL migration files for SQLite

## Usage

```go
import "hai/storage"

// Create a storage instance (SQLite or Elasticsearch)
var s storage.Storage

// Create a speaker
speaker := &storage.Speaker{
    ID:        "spkr_abc123",
    Embedding: []float32{...}, // 256 floats
    FirstSeen: time.Now(),
    LastSeen:  time.Now(),
}
err := s.CreateSpeaker(ctx, speaker)

// Find similar speakers
matches, err := s.FindSimilarSpeakers(
    ctx,
    embedding,
    storage.SimilarityThreshold, // 0.85
    10, // limit
)

// Create a recording
recording := &storage.Recording{
    ID:        "rec_2025_11_22_15",
    FilePath:  "data/2025/11/22/15.ogg",
    StartTime: time.Now(),
    Duration:  3600.0,
}
err := s.CreateRecording(ctx, recording)

// Create segments (bulk)
segments := []*storage.Segment{...}
count, err := s.CreateSegments(ctx, segments)

// Query segments
segments, err := s.GetSegmentsBySpeaker(ctx, "spkr_abc123")
```

## Interface Methods

### Speaker Operations

- `CreateSpeaker(ctx, speaker)` - Create a new speaker
- `GetSpeaker(ctx, id)` - Get speaker by ID
- `FindSimilarSpeakers(ctx, embedding, threshold, limit)` - Find similar speakers using cosine similarity
- `UpdateSpeaker(ctx, speaker)` - Update speaker (contact_id, last_seen, etc.)
- `ListSpeakers(ctx, contactID)` - List all speakers, optionally filtered by contact

### Recording Operations

- `CreateRecording(ctx, recording)` - Create a new recording
- `GetRecording(ctx, id)` - Get recording by ID
- `GetRecordingByPath(ctx, filePath)` - Get recording by file path
- `ListRecordings(ctx, startTime, endTime)` - List recordings, optionally filtered by time range
- `UpdateRecording(ctx, recording)` - Update recording (diarized_at, etc.)

### Segment Operations

- `CreateSegment(ctx, segment)` - Create a single segment
- `CreateSegments(ctx, segments)` - Bulk create segments (more efficient)
- `GetSegment(ctx, id)` - Get segment by ID
- `GetSegmentsBySpeaker(ctx, speakerID)` - Get all segments for a speaker
- `GetSegmentsByRecording(ctx, recordingID)` - Get all segments for a recording
- `GetSegmentsByTimeRange(ctx, recordingID, startTime, endTime)` - Get segments in time range
- `UpdateSegmentByteOffsets(ctx, segmentID, startByteOffset, endByteOffset)` - Update byte offsets

## Helper Functions

### CosineSimilarity

Computes cosine similarity between two embedding vectors:

```go
similarity := storage.CosineSimilarity(embedding1, embedding2)
// Returns value between -1.0 and 1.0, where 1.0 means identical
```

### FindBestMatch

Finds the speaker with the highest similarity to a given embedding:

```go
match := storage.FindBestMatch(embedding, candidates, threshold)
if match != nil {
    // Found a match with similarity >= threshold
    speakerID := match.Speaker.ID
    similarity := match.Similarity
}
```

### ValidateEmbedding

Validates that an embedding has the correct dimension (256):

```go
err := storage.ValidateEmbedding(embedding)
if err != nil {
    // Invalid embedding dimension
}
```

## Error Handling

The storage interface defines common errors:

- `storage.ErrNotFound` - Resource not found
- `storage.ErrDuplicateKey` - Duplicate key (ID already exists)
- `storage.ErrInvalidEmbedding` - Invalid embedding dimension

## Constants

- `storage.EmbeddingDimension` - Fixed dimension for embeddings (256)
- `storage.SimilarityThreshold` - Default similarity threshold (0.85)

## Implementation Status

- ✅ **Schema**: Complete (Go types, SQL migrations, ES mappings)
- ✅ **Interface**: Complete (all methods defined)
- 🔲 **SQLite**: To be implemented (hai-hh1 dependency)
- 🔲 **Elasticsearch**: To be implemented (hai-hh1)

## See Also

- `storage/SCHEMA.md` - Detailed schema documentation
- `storage/migrations/001_initial_schema.sql` - SQLite schema
- `storage/elasticsearch_mappings.json` - Elasticsearch index mappings













