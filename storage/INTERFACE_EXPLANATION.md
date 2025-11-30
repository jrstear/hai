# Storage Interface: Purpose and Usage

## Why Does the Interface Exist?

The `Storage` interface exists to **decouple your application code from the specific storage backend**. This allows you to:

1. **Switch backends without changing application code** - Use SQLite for development, Elasticsearch for production
2. **Test with different backends** - Test with SQLite locally, validate with Elasticsearch
3. **Support multiple backends simultaneously** - Some users might prefer SQLite, others Elasticsearch
4. **Simplify development** - Start with SQLite (zero setup), scale to Elasticsearch when needed

## How/When Would It Be Used?

The interface is used in **three main places** in your architecture:

### 1. **Onboarding/Data Import** (`onboard/internal/export2elastic/`)

**When**: After diarization completes, when importing results to storage

**What happens**:
```go
// In onboard/internal/export2elastic/exporter.go

func ExportDiarizationResults(
    diarizationResult *diarization.Result,
    storage storage.Storage,  // ← Interface here!
) error {
    // 1. Match speakers (find existing or create new)
    for localSpeakerID, embedding := range diarizationResult.SpeakerEmbeddings {
        matches, err := storage.FindSimilarSpeakers(
            ctx,
            embedding,
            storage.SimilarityThreshold, // 0.85
            1, // limit to best match
        )
        
        var speakerID string
        if len(matches) > 0 && matches[0].Similarity >= storage.SimilarityThreshold {
            // Use existing speaker
            speakerID = matches[0].Speaker.ID
            // Update last_seen
            storage.UpdateSpeaker(ctx, &storage.Speaker{
                ID: speakerID,
                LastSeen: time.Now(),
            })
        } else {
            // Create new speaker
            speakerID = generateSpeakerID()
            storage.CreateSpeaker(ctx, &storage.Speaker{
                ID: speakerID,
                Embedding: embedding,
                FirstSeen: time.Now(),
                LastSeen: time.Now(),
            })
        }
        
        // Map local ID to global ID
        speakerMap[localSpeakerID] = speakerID
    }
    
    // 2. Create recording
    recording := &storage.Recording{
        ID: generateRecordingID(diarizationResult.AudioFile),
        FilePath: diarizationResult.AudioFile,
        StartTime: parseStartTime(diarizationResult.AudioFile),
        Duration: diarizationResult.AudioDuration,
        DiarizedAt: &time.Now(),
    }
    storage.CreateRecording(ctx, recording)
    
    // 3. Create segments (bulk insert)
    segments := make([]*storage.Segment, 0, len(diarizationResult.Segments))
    for _, seg := range diarizationResult.Segments {
        segments = append(segments, &storage.Segment{
            SpeakerID: speakerMap[seg.Speaker], // Map local → global
            RecordingID: recording.ID,
            LocalSpeakerID: &seg.Speaker, // Keep original for debugging
            StartTime: seg.Start,
            EndTime: seg.End,
            Duration: seg.Duration,
        })
    }
    storage.CreateSegments(ctx, segments)
    
    return nil
}
```

**Key point**: The export code doesn't know or care if it's writing to SQLite or Elasticsearch - it just uses the interface.

### 2. **Backend API** (`cmd/api/` - to be implemented)

**When**: When the frontend queries for speakers, segments, recordings

**What happens**:
```go
// In cmd/api/handlers.go

type APIHandler struct {
    storage storage.Storage  // ← Interface here!
}

// GET /api/speakers
func (h *APIHandler) ListSpeakers(w http.ResponseWriter, r *http.Request) {
    speakers, err := h.storage.ListSpeakers(ctx, nil)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    json.NewEncoder(w).Encode(speakers)
}

// GET /api/speakers/:id/segments
func (h *APIHandler) GetSpeakerSegments(w http.ResponseWriter, r *http.Request) {
    speakerID := mux.Vars(r)["id"]
    segments, err := h.storage.GetSegmentsBySpeaker(ctx, speakerID)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    json.NewEncoder(w).Encode(segments)
}

// GET /api/recordings/:id/segments?start=10.5&end=20.3
func (h *APIHandler) GetRecordingSegments(w http.ResponseWriter, r *http.Request) {
    recordingID := mux.Vars(r)["id"]
    startTime, _ := strconv.ParseFloat(r.URL.Query().Get("start"), 64)
    endTime, _ := strconv.ParseFloat(r.URL.Query().Get("end"), 64)
    
    segments, err := h.storage.GetSegmentsByTimeRange(
        ctx,
        recordingID,
        startTime,
        endTime,
    )
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    json.NewEncoder(w).Encode(segments)
}

// GET /api/audio/:recordingID/segment/:segmentID
// Returns byte range for HTTP Range request
func (h *APIHandler) GetSegmentByteRange(w http.ResponseWriter, r *http.Request) {
    segmentID, _ := strconv.ParseInt(mux.Vars(r)["segmentID"], 10, 64)
    segment, err := h.storage.GetSegment(ctx, segmentID)
    if err != nil {
        http.Error(w, err.Error(), 404)
        return
    }
    
    if segment.StartByteOffset != nil && segment.EndByteOffset != nil {
        // Serve audio with HTTP Range request
        w.Header().Set("Content-Range", 
            fmt.Sprintf("bytes %d-%d/*", *segment.StartByteOffset, *segment.EndByteOffset))
        // ... serve audio file with Range header
    }
}
```

**Key point**: The API handlers don't know if they're querying SQLite or Elasticsearch - they just use the interface.

### 3. **Initialization/Configuration** (`cmd/api/main.go`)

**When**: At application startup, choosing which backend to use

**What happens**:
```go
// In cmd/api/main.go

func main() {
    var s storage.Storage
    
    // Choose backend based on environment/config
    backend := os.Getenv("STORAGE_BACKEND") // "sqlite" or "elasticsearch"
    
    switch backend {
    case "sqlite":
        dbPath := os.Getenv("SQLITE_PATH") // e.g., "data/speakers.db"
        sqliteStorage, err := sqlite.NewStorage(dbPath)
        if err != nil {
            log.Fatal(err)
        }
        s = sqliteStorage
        
    case "elasticsearch", "":
        // Default to Elasticsearch
        esURL := os.Getenv("ELASTICSEARCH_URL") // e.g., "http://localhost:9200"
        esStorage, err := elasticsearch.NewStorage(esURL)
        if err != nil {
            log.Fatal(err)
        }
        s = esStorage
        
    default:
        log.Fatalf("Unknown storage backend: %s", backend)
    }
    
    defer s.Close()
    
    // Now use the interface - doesn't matter which backend!
    handler := &APIHandler{storage: s}
    
    // ... start HTTP server
}
```

**Key point**: You choose the backend once at startup, then everything else uses the interface.

## Where Are Results Stored and Used?

### Storage Location

**SQLite**:
- File: `data/speakers.db` (or configurable path)
- Tables: `speakers`, `recordings`, `segments`
- Local file on disk

**Elasticsearch**:
- Indices: `speakers`, `recordings`, `segments`
- Running in Docker container (port 9200)
- Network-accessible, can be distributed

### Data Flow

```
┌─────────────────────────────────────────────────────────────┐
│  Phase 1: Onboarding (Native macOS)                         │
│                                                              │
│  1. User submits API key + date range                       │
│  2. Fetch audio from Limitless API                          │
│  3. Run diarization (Python, MPS acceleration)              │
│  4. Save JSON results to: data/YYYY/MM/DD/HH.json          │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  export2elastic/ExportDiarizationResults()           │  │
│  │  Uses: storage.Storage interface                     │  │
│  │  Writes to: SQLite or Elasticsearch                  │  │
│  └──────────────────────────────────────────────────────┘  │
│           ↓                                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Storage Backend (SQLite or Elasticsearch)           │  │
│  │  - Speakers with embeddings                          │  │
│  │  - Recordings metadata                               │  │
│  │  - Segments (normalized, not JSON)                   │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│  Phase 2: Full App (Docker)                                 │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Frontend (React/Vue)                                │  │
│  │  - Query interface                                   │  │
│  │  - Audio playback                                    │  │
│  │  - Analytics dashboard                               │  │
│  └──────────────────────────────────────────────────────┘  │
│           ↕ HTTP requests                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Backend API (Go)                                    │  │
│  │  Uses: storage.Storage interface                     │  │
│  │  Reads from: SQLite or Elasticsearch                 │  │
│  │                                                       │  │
│  │  Endpoints:                                          │  │
│  │  - GET /api/speakers                                 │  │
│  │  - GET /api/speakers/:id/segments                    │  │
│  │  - GET /api/recordings/:id/segments                  │  │
│  │  - GET /api/audio/:id/segment/:id (with Range)      │  │
│  └──────────────────────────────────────────────────────┘  │
│           ↕ Queries                                          │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Storage Backend (SQLite or Elasticsearch)           │  │
│  │  - Same data as Phase 1                              │  │
│  │  - Queryable via interface                           │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Example: Complete Flow

**1. Onboarding (Write)**:
```go
// User runs: ./hai-onboard
// Enters API key, selects date range
// System:
//   - Fetches audio
//   - Diarizes (saves to data/2025/11/22/15.json)
//   - Calls: export2elastic.ExportDiarizationResults()

// export2elastic uses storage interface:
storage.CreateSpeaker(...)      // Creates speaker with embedding
storage.CreateRecording(...)    // Creates recording metadata
storage.CreateSegments(...)     // Bulk inserts all segments

// Data is now in storage (SQLite or Elasticsearch)
```

**2. Frontend Query (Read)**:
```go
// User opens frontend: http://localhost:3001
// Clicks "Show all speakers"

// Frontend calls: GET /api/speakers
// Backend uses storage interface:
speakers := storage.ListSpeakers(ctx, nil)

// Returns JSON to frontend:
[
  {"id": "spkr_abc123", "first_seen": "2025-11-22T15:00:00Z", ...},
  {"id": "spkr_def456", "first_seen": "2025-11-22T16:00:00Z", ...},
]

// User clicks on a speaker
// Frontend calls: GET /api/speakers/spkr_abc123/segments
// Backend uses storage interface:
segments := storage.GetSegmentsBySpeaker(ctx, "spkr_abc123")

// Returns all segments where this speaker spoke
```

**3. Audio Playback (Read)**:
```go
// User clicks play on a segment
// Frontend calls: GET /api/audio/rec_2025_11_22_15/segment/12345

// Backend uses storage interface:
segment := storage.GetSegment(ctx, 12345)

// Returns segment with byte offsets:
{
  "id": 12345,
  "start_byte_offset": 123456,
  "end_byte_offset": 234567,
  ...
}

// Backend serves audio file with HTTP Range header:
// Range: bytes=123456-234567
// Frontend can stream just that segment!
```

## Benefits of the Interface

### 1. **Development Flexibility**
```go
// Local development: Use SQLite (fast, no setup)
STORAGE_BACKEND=sqlite SQLITE_PATH=./data/speakers.db ./cmd/api

// Testing: Use Elasticsearch (validate production behavior)
STORAGE_BACKEND=elasticsearch ELASTICSEARCH_URL=http://localhost:9200 ./cmd/api

// Production: Use Elasticsearch (scalable, distributed)
STORAGE_BACKEND=elasticsearch ELASTICSEARCH_URL=https://es.production.com ./cmd/api
```

### 2. **Easy Testing**
```go
// In tests, you can use a mock or in-memory SQLite
func TestGetSpeakerSegments(t *testing.T) {
    // Use in-memory SQLite for fast tests
    storage := sqlite.NewInMemoryStorage()
    defer storage.Close()
    
    // Test your code using the interface
    handler := &APIHandler{storage: storage}
    // ... test handler methods
}
```

### 3. **Gradual Migration**
```go
// Start with SQLite
storage := sqlite.NewStorage("data/speakers.db")

// Later, migrate to Elasticsearch without changing application code
storage := elasticsearch.NewStorage("http://localhost:9200")

// Same interface, different backend!
```

## Summary

The `Storage` interface is the **abstraction layer** that:

1. **Decouples** your application from storage implementation
2. **Enables** switching between SQLite and Elasticsearch
3. **Simplifies** development and testing
4. **Standardizes** how data is stored and queried

**Used in**:
- `onboard/internal/export2elastic/` - Writing diarization results
- `cmd/api/` - Reading data for frontend queries
- Tests - Using in-memory or mock storage

**Results stored in**:
- SQLite: Local file (`data/speakers.db`)
- Elasticsearch: Docker container (port 9200) or remote cluster

**Results used by**:
- Frontend web app (via Backend API)
- Analytics queries
- Audio playback (with byte offsets for streaming)

The interface makes your code **storage-agnostic** - you write code once, and it works with any backend that implements the interface!

