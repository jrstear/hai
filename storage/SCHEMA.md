# Unified Storage Schema Design

This document describes the unified schema design for both SQLite and Elasticsearch backends.

## Overview

The schema consists of five core entities:
1. **Speakers**: Global speaker registry with voice embeddings
2. **Recordings**: Audio file metadata
3. **Segments**: Time-based speaker segments with optional byte offsets
4. **Lifelogs**: Daily lifelog documents from Limitless API (markdown transcripts)
5. **LifelogBlockquotes**: Speaker segments within lifelogs (transcript excerpts)

## Core Entities

### Speaker

A unique voice identity that can appear across multiple recordings.

**Fields:**
- `id` (string): Global speaker ID (UUID format: `spkr_xxxxx`)
- `embedding` ([]float32): 256-dimensional embedding vector
- `first_seen` (time.Time): When this speaker was first detected
- `last_seen` (time.Time): When this speaker was last detected
- `contact_id` (string, optional): Associated contact ID (NULL if not associated)
- `created_at` (time.Time): Creation timestamp
- `updated_at` (time.Time): Last update timestamp

**Storage:**
- **SQLite**: `embedding` stored as BLOB (1024 bytes for 256 floats)
- **Elasticsearch**: `embedding` stored as `dense_vector` with cosine similarity

### Recording

A single audio file (typically 1-hour chunk).

**Fields:**
- `id` (string): Recording ID (format: `rec_YYYY_MM_DD_HH`)
- `file_path` (string): Path to audio file (e.g., `data/2025/11/22/15.ogg`)
- `start_time` (time.Time): When recording started (UTC)
- `duration` (float64): Duration in seconds
- `sample_rate` (int, optional): Audio sample rate
- `format` (string, optional): Audio format: "ogg", "mp3", etc.
- `diarized_at` (time.Time, optional): When diarization completed
- `processing_time` (float64, optional): Diarization processing time in seconds
- `rtf` (float64, optional): Real-time factor (processing_time / duration)
- `device` (string, optional): Device used: "mps", "cpu", "cuda"
- `created_at` (time.Time): Creation timestamp

**Storage:**
- **SQLite**: Standard table with indexes on `start_time`
- **Elasticsearch**: Document with date fields and keyword fields

### Segment

A time period during which a single speaker speaks. Equivalent to Limitless API's "blockquote" type.

**Fields:**
- `id` (int64): Auto-increment ID (SQLite) or generated ID (Elasticsearch)
- `speaker_id` (string): Global speaker ID (references Speaker.ID)
- `recording_id` (string): Recording ID (references Recording.ID)
- `local_speaker_id` (string, optional): Original SPEAKER_XX from diarization (for debugging)
- `blockquote_id` (string, optional): Best matching blockquote ID (for transcript lookup)
- `start_time` (float64): Start time in seconds (relative to recording start)
- `end_time` (float64): End time in seconds (relative to recording start)
- `duration` (float64): Duration in seconds (end_time - start_time)
- `start_byte_offset` (int64, optional): Byte offset for HTTP Range requests (NULL if not indexed)
- `end_byte_offset` (int64, optional): Byte offset for HTTP Range requests (NULL if not indexed)
- `created_at` (time.Time): Creation timestamp

**Storage:**
- **SQLite**: Normalized table with foreign keys and indexes
- **Elasticsearch**: Document with float fields for time and long fields for byte offsets

### Lifelog

A daily lifelog document from the Limitless API containing markdown-formatted transcripts.

**Fields:**
- `id` (string): Limitless API lifelog ID
- `title` (string): Lifelog title
- `markdown` (string): Full markdown content (includes all blockquotes and formatting)
- `start_time` (time.Time): When lifelog starts (UTC)
- `end_time` (time.Time): When lifelog ends (UTC)
- `created_at` (time.Time): When we fetched/imported it

**Storage:**
- **SQLite**: Not currently stored (Elasticsearch only)
- **Elasticsearch**: Document in `lifelogs` index with date fields and text field for markdown

**Notes:**
- Lifelogs are imported from Limitless API
- Each lifelog represents one day's worth of transcripts
- The markdown field contains the full formatted transcript with blockquotes

### LifelogBlockquote

A speaker segment (blockquote) within a lifelog. Contains transcript text and speaker information.
Equivalent to Limitless API's "blockquote" content type.

**Fields:**
- `id` (string): Generated ID (unique identifier)
- `lifelog_id` (string): References Lifelog.ID (which lifelog this belongs to)
- `recording_id` (string, optional): Which recording this overlaps with (populated by matching tool)
- `content` (string): Transcript text (what was said)
- `speaker_name` (string): Speaker name from Limitless ("You", "Unknown", "Jon Stearley", etc.)
- `speaker_id` (string, optional): Mapped to our global speaker ID (populated by matching tool)
- `start_offset_ms` (int): Milliseconds from lifelog start
- `end_offset_ms` (int): Milliseconds from lifelog end
- `start_time` (time.Time): Absolute start time (UTC)
- `end_time` (time.Time): Absolute end time (UTC)
- `created_at` (time.Time): Creation timestamp

**Storage:**
- **SQLite**: Not currently stored (Elasticsearch only)
- **Elasticsearch**: Document in `lifelog_blockquotes` index with:
  - Text field for `content` (searchable)
  - Keyword fields for IDs and speaker_name
  - Date fields for time ranges
  - Integer fields for offset milliseconds

**Notes:**
- Blockquotes are extracted from lifelog markdown during import
- `recording_id` and `speaker_id` are populated by the matching tool (`onboard/cmd/match-blockquotes-to-segments`)
- Matching is done via time-based overlap calculation between blockquote time ranges and segment time ranges
- Once matched, segments store `blockquote_id` for fast transcript lookup

## SQLite Schema

See `migrations/001_initial_schema.sql` for the complete SQL schema.

**Key features:**
- WAL mode enabled for better concurrency
- Foreign key constraints enabled
- Indexes on common query patterns:
  - `idx_speaker_segments`: Find all segments for a speaker
  - `idx_recording_segments`: Find all segments in a recording
  - `idx_time_range`: Time range queries
  - `idx_byte_range`: Byte range queries for HTTP Range requests

## Elasticsearch Schema

See `elasticsearch_mappings.json` for the complete index mapping.

**Key features:**
- Five separate indices: `speakers`, `recordings`, `segments`, `lifelogs`, `lifelog_blockquotes`
- `dense_vector` field for speaker embeddings with cosine similarity
- kNN search enabled for speaker matching
- Date fields for time-based queries
- Keyword fields for exact matches (IDs, file paths)
- Text fields for searchable content (lifelog markdown, blockquote content)

**Index structure:**
```
speakers/
  - id (keyword)
  - embedding (dense_vector, 256 dims, cosine similarity)
  - first_seen, last_seen, created_at, updated_at (date)
  - contact_id (keyword, not indexed)

recordings/
  - id (keyword)
  - file_path (keyword)
  - start_time, diarized_at, created_at (date)
  - duration_seconds, processing_time, rtf (float)
  - sample_rate (integer)
  - format, device (keyword)

segments/
  - id (keyword)
  - speaker_id, recording_id, local_speaker_id, blockquote_id (keyword)
  - start_time, end_time, duration (float)
  - start_byte_offset, end_byte_offset (long)
  - created_at (date)

lifelogs/
  - id (keyword)
  - title (keyword)
  - markdown (text)
  - start_time, end_time, created_at (date)

lifelog_blockquotes/
  - id (keyword)
  - lifelog_id, recording_id, speaker_id (keyword)
  - content (text)
  - speaker_name (keyword)
  - start_offset_ms, end_offset_ms (integer)
  - start_time, end_time, created_at (date)
```

## Speaker Matching

### Strategy: UUID + Cosine Similarity

**Process:**
1. When new recording is diarized:
   - Extract speaker embeddings
   - For each embedding:
     - Compute cosine similarity with all existing speaker embeddings
     - If max similarity >= threshold (0.85): use existing speaker_id
     - Else: generate new UUID, create new speaker record

**Similarity Threshold:**
- Default: 0.85 (configurable)
- Tune based on false positives/negatives
- May need per-speaker thresholds (some voices more distinct)

**Implementation:**
- **SQLite**: Load all embeddings, compute cosine similarity in application
- **Elasticsearch**: Use kNN search with `dense_vector` field

## Byte Offset Indexing

### Purpose

For scalable streaming (multi-user service):
- HTTP Range requests: `Range: bytes=12345-67890`
- DASH-style segments: Direct byte access without decoding
- Fast seeking: No need to decode from start of file
- CDN-friendly: Standard HTTP range requests work everywhere

### Implementation

**Process:**
1. Parse OGG file to build packet index (using `ffprobe`)
2. Map time → packet → byte offset
3. Store byte offsets in segments table
4. Use HTTP Range requests for playback

**Format:**
- OGG uses packets (not frames like MP3)
- Each packet has a timestamp (granulepos)
- Build index: `timestamp → byte_offset`

**Tools:**
- `ffprobe` for extracting frame info:
  ```bash
  ffprobe -show_frames -select_streams a -of compact \
    -show_entries frame=best_effort_timestamp_time,pkt_pos <file.ogg>
  ```

**Storage:**
- `start_byte_offset` and `end_byte_offset` are optional (NULL if not indexed)
- Can be calculated asynchronously/background
- Don't halt processing if unavailable

## Data Flow

### Diarization → Storage

```
1. Run diarization on audio file
   ↓
2. Extract segments + embeddings
   ↓
3. Save minimal JSON (embeddings only, no segments)
   ↓
4. Import to storage:
   - Match/create speakers (with embeddings)
   - Create recording entry
   - Insert all segments (with time, later add byte offsets)
   ↓
5. Build byte offset index (async/background)
   ↓
6. Update segments with byte offsets
```

## Query Patterns

### Find all segments for a speaker

**SQLite:**
```sql
SELECT * FROM segments 
WHERE speaker_id = 'spkr_abc123' 
ORDER BY start_time;
```

**Elasticsearch:**
```json
{
  "query": {
    "term": { "speaker_id": "spkr_abc123" }
  },
  "sort": [{ "start_time": "asc" }]
}
```

### Find similar speakers (embedding similarity)

**SQLite:**
```go
// Load all embeddings, compute cosine similarity in application
speakers := loadAllSpeakers()
similarities := computeCosineSimilarity(newEmbedding, speakers)
matches := filterByThreshold(similarities, 0.85)
```

**Elasticsearch:**
```json
{
  "knn": {
    "field": "embedding",
    "query_vector": [0.1, 0.2, ...],
    "k": 10,
    "num_candidates": 100
  },
  "min_score": 0.85
}
```

### Get segment for playback (with byte range)

**SQLite:**
```sql
SELECT start_byte_offset, end_byte_offset 
FROM segments 
WHERE id = 12345;
```

**Elasticsearch:**
```json
{
  "query": {
    "term": { "id": "12345" }
  },
  "_source": ["start_byte_offset", "end_byte_offset"]
}
```

## Constants

- **EmbeddingDimension**: 256 (fixed dimension for speaker embeddings)
- **SimilarityThreshold**: 0.85 (default threshold for speaker matching)

## Migration Strategy

### Phase 1: Schema Creation
- Create SQLite tables (via migration)
- Create Elasticsearch indices (via mapping)

### Phase 2: Import Existing Data
- Read existing JSON files
- Extract segments and embeddings
- Import to storage
- Map local speaker IDs to global IDs

### Phase 3: Update Diarization Pipeline
- Modify diarization to import directly to storage
- Save minimal JSON (embeddings only)

### Phase 4: Byte Offset Indexing
- Build OGG parser/indexer
- Calculate byte offsets for segments
- Update segments table

## Data Relationships

### Lifelog ↔ LifelogBlockquote
- One-to-many: A lifelog contains many blockquotes
- Blockquotes reference their parent lifelog via `lifelog_id`

### LifelogBlockquote ↔ Segment
- Many-to-one (via matching): Multiple blockquotes can match to one segment (best match wins)
- Matching is done via time-based overlap calculation
- Once matched:
  - Blockquote stores `recording_id` and `speaker_id`
  - Segment stores `blockquote_id` for fast transcript lookup

### Segment ↔ Recording
- Many-to-one: Many segments belong to one recording
- Segments reference recording via `recording_id`

### Segment ↔ Speaker
- Many-to-one: Many segments belong to one speaker
- Segments reference speaker via `speaker_id`

### Speaker ↔ Contact
- One-to-one (optional): A speaker can be associated with a contact
- Speaker stores `contact_id` when associated

## Terminology Alignment

**Limitless API uses:**
- `lifelog` (document type) for daily markdown transcripts
- `blockquote` (content type) for speaking segments within lifelogs
- `startOffsetMs` / `endOffsetMs` (milliseconds from lifelog start)
- `startTime` / `endTime` (ISO 8601 timestamps)

**Our internal terminology:**
- **Lifelog**: A daily lifelog document from Limitless API (markdown transcripts)
- **LifelogBlockquote**: A speaker segment within a lifelog (transcript excerpt with speaker name)
- **Segment**: A time period during which a single speaker speaks (from our diarization, equivalent to Limitless "blockquote")
- **Speaker**: A unique voice identity (can be matched across recordings)
- **Recording**: A single audio file (1-hour chunk)

**Key Distinction:**
- **LifelogBlockquote**: Transcript data from Limitless API (external source)
- **Segment**: Diarization data from our own processing (internal source)
- They are matched together via time-based overlap to link transcripts to our segments

