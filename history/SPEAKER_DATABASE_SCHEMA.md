# Speaker Database Schema Design

## Terminology Alignment

**Limitless API uses:**
- `blockquote` (type) for speaking segments
- `startOffsetMs` / `endOffsetMs` (milliseconds from lifelog start)
- `startTime` / `endTime` (ISO 8601 timestamps)

**Our internal terminology:**
- **Segment**: A time period during which a single speaker speaks (equivalent to Limitless "blockquote")
- **Speaker**: A unique voice identity (can be matched across recordings)
- **Recording**: A single audio file (1-hour chunk)

## Schema Design: Normalized Database

### Core Tables

```sql
-- Global speaker registry
CREATE TABLE speakers (
  id TEXT PRIMARY KEY,                    -- e.g., 'spkr_abc123'
  embedding BLOB NOT NULL,                -- 256 floats as binary (1024 bytes)
  first_seen TIMESTAMP NOT NULL,
  last_seen TIMESTAMP NOT NULL,
  contact_id TEXT,                        -- NULL until associated with contact
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (contact_id) REFERENCES contacts(id)
);

-- Audio recordings metadata
CREATE TABLE recordings (
  id TEXT PRIMARY KEY,                    -- e.g., 'rec_2025_11_22_15'
  file_path TEXT NOT NULL UNIQUE,         -- 'data/audio/2025/11/22/15.ogg'
  start_time TIMESTAMP NOT NULL,          -- When recording started
  duration_seconds REAL NOT NULL,         -- Total duration
  sample_rate INTEGER,                    -- Audio sample rate
  format TEXT,                            -- 'ogg', 'mp3', etc.
  diarized_at TIMESTAMP,                  -- When diarization completed
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Speaker segments (normalized, NOT JSON)
CREATE TABLE segments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  speaker_id TEXT NOT NULL,               -- Global speaker ID
  recording_id TEXT NOT NULL,             -- Which recording
  local_speaker_id TEXT,                  -- Original SPEAKER_XX from diarization
  
  -- Time-based references (always present)
  start_time REAL NOT NULL,               -- Start in seconds (float)
  end_time REAL NOT NULL,                 -- End in seconds (float)
  
  -- Byte-based references (for fast seeking, DASH-style)
  start_byte_offset INTEGER,              -- NULL if not indexed yet
  end_byte_offset INTEGER,                -- NULL if not indexed yet
  
  -- Metadata
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  
  FOREIGN KEY (speaker_id) REFERENCES speakers(id),
  FOREIGN KEY (recording_id) REFERENCES recordings(id),
  
  -- Indexes for common queries
  INDEX idx_speaker_segments (speaker_id, start_time),
  INDEX idx_recording_segments (recording_id, start_time, end_time),
  INDEX idx_time_range (recording_id, start_time, end_time),
  INDEX idx_byte_range (recording_id, start_byte_offset, end_byte_offset)
);
```

## JSON File Role (Minimal)

**Question: Do we still need JSON files?**

**Answer: Yes, but minimal - just for:**
1. **Initial diarization results** (before DB import)
2. **Backup/cache** (if DB unavailable)
3. **Debugging** (human-readable inspection)

**What stays in JSON:**
```json
{
  "audio_file": "data/audio/2025/11/22/15.ogg",
  "diarized_at": "2025-11-28T17:47:30Z",
  "speaker_count": 5,
  "segment_count": 1462,
  "speaker_embeddings": {
    "SPEAKER_00": [256 floats],
    ...
  }
  // NO segments array - that's in DB
  // NO recordings list - query DB instead
}
```

**What moves to DB:**
- ✅ All segments (normalized table)
- ✅ Speaker embeddings (speakers table)
- ✅ Recording metadata (recordings table)
- ✅ Cross-recording speaker matching

## Byte Offset Indexing

### Why Byte Offsets?

For scalable streaming (multi-user service):
- **HTTP Range requests**: `Range: bytes=12345-67890`
- **DASH-style segments**: Direct byte access without decoding
- **Fast seeking**: No need to decode from start of file
- **CDN-friendly**: Standard HTTP range requests work everywhere

### OGG Format Considerations

OGG uses **packets** (not frames like MP3):
- Each packet has a timestamp (granulepos)
- Packets can be variable size
- Need to build index: `timestamp → byte_offset`

**Implementation approach:**
1. Parse OGG file to build packet index
2. Map time → packet → byte offset
3. Store byte offsets in segments table
4. Use HTTP Range requests for playback

**Standard approach:**
- Use **HTTP Range requests** (RFC 7233)
- Format: `Range: bytes=start-end`
- DASH uses this for segment delivery
- Works with any HTTP server/CDN

### Byte Offset Calculation

**Process:**
1. Run `ffprobe` to extract frame info:
   ```bash
   ffprobe -show_frames -select_streams a -of compact \
     -show_entries frame=best_effort_timestamp_time,pkt_pos <file.ogg>
   ```
2. Parse output to build timestamp → byte_offset map
3. For each segment (start_time, end_time):
   - Find frame with timestamp nearest to start_time
   - Use `pkt_pos` as `start_byte_offset`
   - Find frame with timestamp nearest to end_time
   - Use `pkt_pos` as `end_byte_offset`
4. Store in segments table

**Example ffprobe output:**
```
frame|best_effort_timestamp_time=0.000000|pkt_pos=101
frame|best_effort_timestamp_time=0.013500|pkt_pos=181
frame|best_effort_timestamp_time=0.033500|pkt_pos=258
```

**Tools:**
- ✅ `ffprobe` (confirmed working for OGG/Opus)
- Similar approach to HLS byte indexing

## Data Flow

### Diarization → Storage

```
1. Run diarization on audio file
   ↓
2. Extract segments + embeddings
   ↓
3. Save minimal JSON (embeddings only, no segments)
   ↓
4. Import to DB:
   - Create/update speakers (with embeddings)
   - Create recording entry
   - Insert all segments (with time, later add byte offsets)
   ↓
5. Build byte offset index (async/background)
   ↓
6. Update segments with byte offsets
```

### Query Patterns

**"Show all segments for speaker X":**
```sql
SELECT * FROM segments 
WHERE speaker_id = 'spkr_abc123' 
ORDER BY start_time;
```

**"Find recordings with this speaker":**
```sql
SELECT DISTINCT recording_id FROM segments 
WHERE speaker_id = 'spkr_abc123';
```

**"Get segment for playback (with byte range)":**
```sql
SELECT start_byte_offset, end_byte_offset 
FROM segments 
WHERE id = 12345;
```

**"Find similar speakers (embedding similarity)":**
```sql
-- Use vector similarity search (requires extension or application-level)
-- Or: Load embeddings, compute cosine similarity in app
```

## Migration Strategy

### Phase 1: DB Schema
- Create tables
- Define indexes
- Set up foreign keys

### Phase 2: Import Existing Data
- Read existing JSON files
- Extract segments
- Import to DB
- Map local speaker IDs to global IDs

### Phase 3: Update Diarization Pipeline
- Modify `diarize.py` to:
  - Save minimal JSON (embeddings only)
  - Import segments to DB
  - Create/update speakers

### Phase 4: Byte Offset Indexing
- Build OGG parser/indexer
- Calculate byte offsets for segments
- Update segments table

## Decisions Made

### 1. Speaker ID Generation
- **Decision**: **UUID + cosine similarity matching**
- Generate UUID for each new speaker
- Use cosine similarity to match speakers across recordings
- Threshold TBD (likely 0.8-0.9 for "same speaker")
- Future: Can add hash-based deterministic IDs later if needed

### 2. Segment Granularity
- **Decision**: **Keep all segments** (even < 0.5s)
- Databases are for precision and aggregation
- Can aggregate/merge in queries as needed
- Preserves full detail for analysis

### 3. Byte Offset Calculation
- **Decision**: **Use ffprobe** (similar to HLS approach)
- Command: `ffprobe -show_frames -select_streams a -of compact -show_entries frame=best_effort_timestamp_time,pkt_pos <file>`
- Extract `pkt_pos` (packet position/byte offset) for each frame
- Match timestamps to nearest frame for byte offset
- Store exact packet boundaries

### 4. JSON File Retention
- **Decision**: **Keep JSON files for now**
- Useful for debugging
- Backup if DB unavailable
- Can prune later as needed
- DB is source of truth, JSON is convenience/backup

## Speaker Matching Strategy

### UUID + Cosine Similarity

**Process:**
1. When new recording is diarized:
   - Extract speaker embeddings
   - For each embedding:
     - Compute cosine similarity with all existing speaker embeddings
     - If max similarity > threshold (e.g., 0.85):
       - Use existing speaker_id
     - Else:
       - Generate new UUID, create new speaker record

**Cosine Similarity:**
```python
from sklearn.metrics.pairwise import cosine_similarity
import numpy as np

def match_speaker(new_embedding, existing_embeddings, threshold=0.85):
    similarities = cosine_similarity([new_embedding], existing_embeddings)[0]
    max_sim = np.max(similarities)
    if max_sim > threshold:
        return existing_speaker_ids[np.argmax(similarities)]
    return None  # New speaker
```

**Threshold Selection:**
- Start with 0.85 (conservative)
- Tune based on false positives/negatives
- May need per-speaker thresholds (some voices more distinct)

**Speaker Merging:**
- If two UUIDs later determined to be same person:
  - Update all segments: `UPDATE segments SET speaker_id = 'merged_id' WHERE speaker_id = 'old_id'`
  - Delete old speaker record
  - Or: Keep both, mark one as alias (future enhancement)

## Next Steps

1. ✅ Design schema (this document)
2. ✅ Decisions: UUID + cosine, ffprobe, keep JSON, keep all segments
3. 🔲 Create DB migration scripts
4. 🔲 Build import tool (JSON → DB)
5. 🔲 Update diarization pipeline
6. 🔲 Build byte offset indexer (using ffprobe)
7. 🔲 Implement speaker matching (cosine similarity)
8. 🔲 Test with existing data

