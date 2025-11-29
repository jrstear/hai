# Schema Design Discussion Summary

## Decisions Made

### 1. Storage Approach: Normalized DB (Option A)
- ✅ **Segments in normalized database table** (NOT JSON)
- ✅ **Remove recordings array from JSON** (query DB instead)
- ✅ **Minimal JSON files** - only embeddings, no segments

### 2. Terminology
- **Limitless API uses**: `blockquote` for speaking segments, `startOffsetMs`/`endOffsetMs` (milliseconds)
- **Our internal term**: `segment` (time period when single speaker speaks)
- **Compatible**: Our schema can map to/from Limitless format

### 3. JSON File Role
**Question**: Are we using JSON at all anymore?

**Answer**: Yes, but minimal:
- **Keep**: Speaker embeddings (for quick per-file access)
- **Keep**: Basic metadata (diarization timestamp, counts)
- **Remove**: Segments array (moved to DB)
- **Remove**: Recordings list (query DB instead)

**Why keep JSON?**
- Backup/cache if DB unavailable
- Human-readable for debugging
- Quick per-file access to embeddings
- But DB is source of truth

### 4. Byte Offsets: Yes, Store Them
**Decision**: Store byte offsets for fast seeking

**Rationale**:
- Scalability: Multi-user service needs fast playback
- HTTP Range requests: Standard `Range: bytes=start-end` support
- DASH-style streaming: Industry standard for segmented media
- CDN-friendly: Works with any HTTP server

**Implementation**:
- Parse OGG packets to build timestamp → byte_offset index
- Store in segments table: `start_byte_offset`, `end_byte_offset`
- Use HTTP Range requests for playback
- Standard approach (not proprietary)

### 5. Schema Structure

**Core Tables:**
1. `speakers` - Global speaker registry with embeddings
2. `recordings` - Audio file metadata
3. `segments` - Normalized segments table (NOT JSON)

**Key Fields:**
- Time-based: `start_time`, `end_time` (seconds, float)
- Byte-based: `start_byte_offset`, `end_byte_offset` (integers, nullable)
- References: `speaker_id`, `recording_id`, `local_speaker_id`

## Decisions Finalized ✅

### 1. Speaker ID Generation Strategy
- **Decision**: **UUID + cosine similarity matching**
- Generate UUID for each new speaker
- Use cosine similarity to match across recordings
- Threshold: Start with 0.85, tune as needed

### 2. Segment Merging
- **Decision**: **Keep all segments** (even < 0.5s)
- Databases are for precision and aggregation
- Can aggregate in queries as needed

### 3. JSON File Lifecycle
- **Decision**: **Keep JSON files for now**
- Useful for debugging
- Backup if DB unavailable
- Can prune later as needed
- DB is source of truth

### 4. Byte Offset Calculation
- **Decision**: **Use ffprobe** (similar to HLS approach)
- Command: `ffprobe -show_frames -select_streams a -of compact -show_entries frame=best_effort_timestamp_time,pkt_pos <file>`
- Extract `pkt_pos` for each frame, match timestamps to nearest frame
- ✅ Confirmed working for OGG/Opus files

## Implementation Sequence

1. **Design schema** (hai-tw5) ← Current
2. **Create DB tables** (hai-tw5)
3. **Build import tool** (hai-d8z) - depends on #1
4. **Update diarization pipeline** (hai-tw5) - save minimal JSON + import to DB
5. **Build byte offset indexer** (hai-645) - depends on #1
6. **Test with existing data**

## Next Steps Discussion

Ready to discuss:
- Speaker ID generation strategy
- Byte offset calculation approach (which library/tool)
- Migration path for existing JSON files
- API design for querying segments
- Contact association workflow

