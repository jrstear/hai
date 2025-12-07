# Lifelog Storage Plan

## Current State

### What We Have
- ✅ **Lifelog fetching**: `onboard/internal/fetch/lifelog.go` - Fetches lifelogs from Limitless API
- ✅ **Lifelog storage**: Saved to `data/YYYY/MM/DD/lifelog.json` (per-day JSON files)
- ✅ **Comparison script**: `cmd/diarize/compare_with_lifelogs.py` - Compares lifelog blockquotes with diarization segments
- ✅ **Beads issue**: `hai-7rz` - "Fetch Lifelogs for Comparison" (open, but only about fetching)

### What's Missing
- ❌ **No lifelog storage in Elasticsearch** - Currently only in JSON files
- ❌ **No lifelog schema** - Not part of the Storage interface
- ❌ **No automated comparison** - Only manual comparison script exists
- ❌ **No speaker name mapping** - Can't map lifelog speaker names to our speaker IDs

## Lifelog Structure

From `onboard/internal/fetch/lifelog.go`:

```go
type Lifelog struct {
    ID        string    `json:"id"`
    Title     string    `json:"title"`
    Markdown  string    `json:"markdown"`
    StartTime time.Time `json:"startTime"`
    EndTime   time.Time `json:"endTime"`
    Contents  []Content `json:"contents"`
}

type Content struct {
    Type              string  `json:"type"`              // "blockquote", "text", etc.
    Content           string  `json:"content"`           // The actual text/transcript
    StartTime         string  `json:"startTime,omitempty"`
    EndTime           string  `json:"endTime,omitempty"`
    StartOffsetMs     int     `json:"startOffsetMs,omitempty"`  // Milliseconds from lifelog start
    EndOffsetMs       int     `json:"endOffsetMs,omitempty"`
    SpeakerName       string  `json:"speakerName,omitempty"`    // "You", "Unknown", "Jon Stearley", etc.
    SpeakerIdentifier *string `json:"speakerIdentifier,omitempty"`
}
```

## Use Cases for Lifelog Storage

### 1. **Speaker Name Mapping**
- **Goal**: Map lifelog speaker names ("You", "Jon Stearley", "Unknown") to our global speaker IDs (`spkr_abc123`)
- **Method**: Time-based matching - find diarization segments that overlap with lifelog blockquotes
- **Benefit**: Users can see actual names instead of "SPEAKER_00", "SPEAKER_01"

### 2. **Transcript Search**
- **Goal**: Full-text search across lifelog transcripts
- **Method**: Store lifelog content in Elasticsearch with full-text indexing
- **Benefit**: "Find all conversations about X" queries

### 3. **Validation & Comparison**
- **Goal**: Compare Limitless API speaker identification with our diarization
- **Method**: Query both lifelog blockquotes and diarization segments by time range
- **Benefit**: Validate accuracy, identify discrepancies

### 4. **Rich Metadata**
- **Goal**: Store lifelog titles, markdown, and other metadata
- **Method**: Store full lifelog documents in Elasticsearch
- **Benefit**: Rich context for segments (what conversation was about)

## Proposed Schema

### Option 1: Store Lifelogs as Separate Documents

**Index**: `lifelogs`

```go
type Lifelog struct {
    ID        string    `json:"id"`         // Limitless API lifelog ID
    Title     string    `json:"title"`      // Lifelog title
    Markdown  string    `json:"markdown"`   // Full markdown content
    StartTime time.Time `json:"start_time"` // When lifelog starts (UTC)
    EndTime   time.Time `json:"end_time"`   // When lifelog ends (UTC)
    CreatedAt time.Time `json:"created_at"` // When we fetched it
}

type LifelogBlockquote struct {
    ID              string    `json:"id"`               // Generated ID
    LifelogID       string    `json:"lifelog_id"`       // References Lifelog.ID
    RecordingID     *string   `json:"recording_id"`     // Optional: Which recording this overlaps with
    Content         string    `json:"content"`          // Transcript text
    SpeakerName     string    `json:"speaker_name"`     // "You", "Unknown", etc.
    SpeakerID       *string   `json:"speaker_id"`       // Optional: Mapped to our global speaker ID
    StartOffsetMs   int       `json:"start_offset_ms"`  // Milliseconds from lifelog start
    EndOffsetMs     int       `json:"end_offset_ms"`    // Milliseconds from lifelog end
    StartTime       time.Time `json:"start_time"`       // Absolute start time (UTC)
    EndTime         time.Time `json:"end_time"`         // Absolute end time (UTC)
    CreatedAt       time.Time `json:"created_at"`
}
```

**Benefits**:
- ✅ Separate lifelogs from diarization data
- ✅ Can query lifelogs independently
- ✅ Full-text search on transcripts
- ✅ Can map blockquotes to segments later

### Option 2: Store Blockquotes as Segments with Source

**Modify existing Segment schema**:

```go
type Segment struct {
    // ... existing fields ...
    Source          string    `json:"source"`           // "diarization" or "lifelog"
    LifelogID       *string   `json:"lifelog_id"`       // If source is "lifelog"
    SpeakerName     *string   `json:"speaker_name"`     // If source is "lifelog"
    Transcript      *string   `json:"transcript"`       // If source is "lifelog"
}
```

**Benefits**:
- ✅ Unified query interface (all segments in one place)
- ✅ Easier comparison (same structure)
- ❌ Mixes two different data sources
- ❌ Different time references (lifelog uses offsets, diarization uses seconds)

### Option 3: Hybrid Approach (Recommended)

**Store both separately, link via time-based queries**:

1. **Lifelogs index**: Store full lifelog documents
2. **LifelogBlockquotes index**: Store individual blockquotes (for easier querying)
3. **Segments index**: Keep existing diarization segments
4. **Mapping logic**: Time-based matching to link blockquotes to segments

**Benefits**:
- ✅ Clean separation of concerns
- ✅ Can query each independently
- ✅ Can link them via time-based matching
- ✅ Full-text search on lifelog content
- ✅ Can map speaker names to speaker IDs

## Recommended Implementation

### Phase 1: Add Lifelog Storage to Interface

Add to `storage/interface.go`:

```go
// Lifelog operations

// CreateLifelog creates a new lifelog document
CreateLifelog(ctx context.Context, lifelog *Lifelog) error

// GetLifelog retrieves a lifelog by ID
GetLifelog(ctx context.Context, id string) (*Lifelog, error)

// ListLifelogs lists lifelogs, optionally filtered by time range
ListLifelogs(ctx context.Context, startTime *time.Time, endTime *time.Time) ([]*Lifelog, error)

// CreateLifelogBlockquote creates a blockquote from a lifelog
CreateLifelogBlockquote(ctx context.Context, blockquote *LifelogBlockquote) error

// GetLifelogBlockquotesByTimeRange retrieves blockquotes within a time range
GetLifelogBlockquotesByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*LifelogBlockquote, error)

// MapBlockquoteToSegment links a lifelog blockquote to a diarization segment
MapBlockquoteToSegment(ctx context.Context, blockquoteID string, segmentID int64, speakerID string) error
```

### Phase 2: Implement in Elasticsearch

- Add `lifelogs` and `lifelog_blockquotes` indices
- Full-text search on `content` field
- Time-based queries for matching

### Phase 3: Speaker Name Mapping

- Time-based matching algorithm:
  1. For each lifelog blockquote, find overlapping diarization segments
  2. Match by time overlap (e.g., > 50% overlap)
  3. Map `SpeakerName` → `SpeakerID` via the matched segments
  4. Store mapping in `LifelogBlockquote.SpeakerID`

### Phase 4: Export Integration

- Update `onboard/internal/export2elastic/` to also export lifelogs
- After diarization completes, export lifelogs for the same time period
- Run speaker name mapping automatically

## Comparison Use Cases

### 1. Time-Based Matching

```go
// Find all lifelog blockquotes that overlap with a diarization segment
blockquotes, err := storage.GetLifelogBlockquotesByTimeRange(
    ctx,
    segmentStartTime,
    segmentEndTime,
)

// For each blockquote, check if it overlaps with the segment
for _, blockquote := range blockquotes {
    if overlaps(blockquote, segment) {
        // Map speaker name to speaker ID
        storage.MapBlockquoteToSegment(ctx, blockquote.ID, segment.ID, segment.SpeakerID)
    }
}
```

### 2. Speaker Name Resolution

```go
// User sees "SPEAKER_00" in UI
// Query: Find lifelog blockquotes that match this segment
blockquotes := findMatchingBlockquotes(segment)

// If found, show speaker name instead
if len(blockquotes) > 0 {
    speakerName := blockquotes[0].SpeakerName  // "Jon Stearley"
    // Display: "Jon Stearley" instead of "SPEAKER_00"
}
```

### 3. Full-Text Search

```go
// User searches: "conversations about AI"
// Query Elasticsearch for lifelog blockquotes containing "AI"
results := searchLifelogContent("AI")

// For each result, find corresponding diarization segments
for _, blockquote := range results {
    segments := findSegmentsByTimeRange(blockquote.StartTime, blockquote.EndTime)
    // Show both transcript and audio segments
}
```

## Benefits of Storing in Elasticsearch

1. **Unified Storage**: All data in one place (speakers, recordings, segments, lifelogs)
2. **Full-Text Search**: Search transcripts across all lifelogs
3. **Time-Based Queries**: Efficient queries by time range
4. **Speaker Mapping**: Link lifelog names to our speaker IDs
5. **Rich Context**: Titles, markdown, and metadata available for segments
6. **Comparison**: Easy to compare Limitless API vs. local diarization

## Next Steps

1. **Create beads issue**: "Store lifelogs in Elasticsearch and implement speaker name mapping"
2. **Add to schema**: Define `Lifelog` and `LifelogBlockquote` types
3. **Extend Storage interface**: Add lifelog operations
4. **Implement in Elasticsearch**: Add indices and operations
5. **Update export2elastic**: Export lifelogs after fetching
6. **Implement mapping**: Time-based matching algorithm

## Questions to Consider

1. **Should we store full lifelog JSON or just blockquotes?**
   - **Recommendation**: Store both - full lifelog for context, blockquotes for querying

2. **How to handle time alignment?**
   - Lifelogs use `StartOffsetMs` (relative to lifelog start)
   - Diarization uses seconds (relative to recording start)
   - **Solution**: Convert both to absolute UTC timestamps for matching

3. **What if a blockquote matches multiple segments?**
   - **Solution**: Use overlap percentage - map to segment with highest overlap

4. **Should speaker name mapping be automatic or manual?**
   - **Recommendation**: Automatic with manual override capability












