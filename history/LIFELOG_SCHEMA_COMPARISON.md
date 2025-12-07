# Lifelog vs. Existing Schema: Overlap Analysis

## Overview

This document compares lifelog content (from Limitless API) with our existing Elasticsearch schema (speakers, recordings, segments) to identify overlaps and differences.

## Side-by-Side Comparison

### 1. **Segments vs. Lifelog Blockquotes**

| Aspect | **Segment** (Our Schema) | **Lifelog Blockquote** (Limitless API) | Overlap? |
|--------|-------------------------|----------------------------------------|----------|
| **Purpose** | Time period when a speaker speaks (from diarization) | Time period when a speaker speaks (from Limitless API) | ✅ **YES - Same concept** |
| **Time Range** | `start_time`, `end_time` (float64, seconds relative to recording) | `startOffsetMs`, `endOffsetMs` (int, milliseconds relative to lifelog start) | ⚠️ **PARTIAL - Different units/reference** |
| **Speaker ID** | `speaker_id` (global ID: `spkr_abc123`) | `speakerName` (string: "You", "Unknown", "Jon Stearley") | ⚠️ **PARTIAL - Different format** |
| **Speaker Reference** | References `Speaker` table via `speaker_id` | Has `speakerIdentifier` (e.g., "user") | ❌ **NO - Different systems** |
| **Transcript/Content** | ❌ **NO** - No transcript text | ✅ **YES** - `content` field with transcript | ❌ **NO - Missing in segments** |
| **Source** | Our local diarization (pyannote.audio) | Limitless API (their diarization) | ❌ **NO - Different sources** |
| **Recording Reference** | `recording_id` (which audio file) | No direct recording reference | ❌ **NO - Missing in lifelog** |
| **Byte Offsets** | `start_byte_offset`, `end_byte_offset` (for HTTP Range) | ❌ **NO** | ❌ **NO - Missing in lifelog** |
| **Metadata** | Just time + speaker | Has transcript, speaker name, timestamps | ❌ **NO - Segments are minimal** |

**Key Insight**: They represent the **same concept** (who spoke when) but from **different sources** with **different data**:
- Segments: Audio-based, no transcripts, global speaker IDs
- Blockquotes: API-based, has transcripts, speaker names

### 2. **Speakers vs. Lifelog Speaker Names**

| Aspect | **Speaker** (Our Schema) | **Lifelog Speaker** | Overlap? |
|--------|-------------------------|---------------------|----------|
| **Identity** | Global speaker ID (`spkr_abc123`) | Speaker name ("You", "Unknown", "Jon Stearley") | ⚠️ **PARTIAL - Need mapping** |
| **Embedding** | ✅ **YES** - 256-dim voice embedding | ❌ **NO** | ❌ **NO - Missing in lifelog** |
| **Matching** | Cosine similarity on embeddings | Name-based (exact string match) | ❌ **NO - Different methods** |
| **Persistence** | Cross-recording (same speaker across files) | Per-lifelog (names may vary) | ⚠️ **PARTIAL - Need normalization** |
| **Contact Link** | `contact_id` (can link to contacts) | `speakerIdentifier` (e.g., "user") | ⚠️ **PARTIAL - Different systems** |

**Key Insight**: Lifelog speaker names are **human-readable labels** that need to be **mapped** to our global speaker IDs via time-based matching.

### 3. **Recordings vs. Lifelogs**

| Aspect | **Recording** (Our Schema) | **Lifelog** (Limitless API) | Overlap? |
|--------|---------------------------|----------------------------|----------|
| **Purpose** | Audio file metadata (1-hour chunk) | Conversation/document metadata | ⚠️ **PARTIAL - Different granularity** |
| **Time Range** | `start_time`, `duration` (single hour) | `startTime`, `endTime` (variable duration) | ⚠️ **PARTIAL - Different scope** |
| **File Reference** | `file_path` (audio file: `15.ogg`) | ❌ **NO** - No direct file reference | ❌ **NO - Missing in lifelog** |
| **Content** | Just metadata (no content) | `markdown`, `title`, `contents[]` | ❌ **NO - Rich content in lifelog** |
| **Segments/Blockquotes** | Has segments (via `recording_id`) | Has blockquotes (in `contents[]`) | ⚠️ **PARTIAL - Similar but separate** |

**Key Insight**: Recordings are **audio file containers**, lifelogs are **conversation containers**. They can overlap in time but serve different purposes.

## Detailed Overlap Analysis

### ✅ **What Overlaps**

1. **Time-based speaker segments**
   - Both represent "who spoke when"
   - Both have start/end times
   - Both associate a speaker with a time range

2. **Speaker identification**
   - Both identify speakers
   - Both can be used to track who said what
   - Both can be linked (via time-based matching)

3. **Time ranges**
   - Both use start/end times
   - Both can be queried by time range
   - Both can be compared for overlap

### ⚠️ **What Partially Overlaps (Needs Mapping)**

1. **Speaker IDs vs. Names**
   - Segments: `speaker_id` = `"spkr_abc123"` (global, embedding-based)
   - Blockquotes: `speakerName` = `"You"` or `"Jon Stearley"` (human-readable)
   - **Solution**: Time-based matching to map names → IDs

2. **Time Units**
   - Segments: Seconds (relative to recording start)
   - Blockquotes: Milliseconds (relative to lifelog start)
   - **Solution**: Convert both to absolute UTC timestamps for matching

3. **Recording Reference**
   - Segments: Have `recording_id` (which audio file)
   - Blockquotes: No direct recording reference
   - **Solution**: Match by time overlap (lifelog time range overlaps with recording time range)

### ❌ **What Doesn't Overlap (Unique to Each)**

#### **Unique to Segments:**
- ✅ **Voice embeddings** (256-dim vectors for speaker matching)
- ✅ **Byte offsets** (for HTTP Range requests, audio streaming)
- ✅ **Recording reference** (which audio file this came from)
- ✅ **Local speaker IDs** (SPEAKER_00, SPEAKER_01 for debugging)

#### **Unique to Lifelogs:**
- ✅ **Transcripts** (`content` field with actual spoken text)
- ✅ **Speaker names** (human-readable: "You", "Jon Stearley")
- ✅ **Rich metadata** (title, markdown, headings)
- ✅ **Conversation context** (what the conversation was about)
- ✅ **Multiple content types** (blockquotes, headings, text)

## Visual Comparison

```
┌─────────────────────────────────────────────────────────────┐
│  Recording (Our Schema)                                      │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  rec_2025_11_22_15                                    │  │
│  │  file_path: data/2025/11/22/15.ogg                   │  │
│  │  start_time: 2025-11-22T15:00:00Z                    │  │
│  │  duration: 3600s                                      │  │
│  └───────────────────────────────────────────────────────┘  │
│           ↓ contains                                          │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  Segments (Our Schema)                                │  │
│  │  - speaker_id: spkr_abc123                            │  │
│  │  - start_time: 120.5s (relative to recording)        │  │
│  │  - end_time: 125.3s                                   │  │
│  │  - start_byte_offset: 123456                          │  │
│  │  - end_byte_offset: 234567                            │  │
│  │  - ❌ NO TRANSCRIPT                                   │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  Lifelog (Limitless API)                                     │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  id: "77oWJ14VgPcsDdQzFbL6"                          │  │
│  │  title: "A wide-ranging conversation"                │  │
│  │  startTime: 2025-11-22T21:26:23-07:00                │  │
│  │  endTime: 2025-11-22T21:49:00-07:00                  │  │
│  │  markdown: "## Initial remarks..."                   │  │
│  └───────────────────────────────────────────────────────┘  │
│           ↓ contains                                          │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  Blockquotes (Limitless API)                          │  │
│  │  - speakerName: "You"                                 │  │
│  │  - startOffsetMs: 2000 (relative to lifelog)         │  │
│  │  - endOffsetMs: 4000                                  │  │
│  │  - content: "No, I think they're halfway there..."   │  │
│  │  - ❌ NO BYTE OFFSETS                                 │  │
│  │  - ❌ NO RECORDING REFERENCE                          │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  Overlap: Time-Based Matching                                │
│                                                              │
│  Segment: 2025-11-22T15:02:00Z - 2025-11-22T15:02:05Z      │
│  Blockquote: 2025-11-22T21:26:25-07:00 (converts to UTC)   │
│                                                              │
│  If times overlap → Map "You" → spkr_abc123                 │
└─────────────────────────────────────────────────────────────┘
```

## What This Means for Storage

### Option 1: Store Separately (Recommended)

**Keep lifelogs and segments separate** because:
- ✅ Different sources (API vs. local diarization)
- ✅ Different purposes (transcripts vs. audio segments)
- ✅ Different data (names vs. embeddings)
- ✅ Can link via time-based matching when needed

**Storage:**
- `segments` index: Our diarization segments (existing)
- `lifelogs` index: Full lifelog documents (new)
- `lifelog_blockquotes` index: Individual blockquotes (new)

**Linking:**
- Time-based matching algorithm
- Store mapping: `blockquote.speaker_id` ← mapped from segment

### Option 2: Merge into Segments

**Add lifelog data to segments** by:
- Adding optional fields: `transcript`, `speaker_name`, `lifelog_id`
- Marking source: `source = "diarization"` or `"lifelog"`

**Problems:**
- ❌ Mixes two different data sources
- ❌ Different time references (seconds vs. milliseconds)
- ❌ Different speaker identification (IDs vs. names)
- ❌ Confusing queries (which source do you want?)

### Option 3: Store Blockquotes as Separate "Transcript Segments"

**Create a new entity** that's similar to segments but for transcripts:
- `transcript_segments` index
- Similar structure to segments but with transcript content
- Link to segments via time-based matching

**Problems:**
- ❌ Duplicates segment structure
- ❌ Still need to link them
- ❌ More complexity

## Recommended Approach

**Store separately, link via time-based matching:**

1. **Keep existing schema** (speakers, recordings, segments)
2. **Add lifelog indices**:
   - `lifelogs` - Full lifelog documents
   - `lifelog_blockquotes` - Individual blockquotes
3. **Add mapping fields**:
   - `lifelog_blockquote.speaker_id` - Mapped from segment (optional)
   - `lifelog_blockquote.recording_id` - Which recording overlaps (optional)
   - `lifelog_blockquote.matched_segment_ids` - Array of segment IDs that overlap
4. **Time-based matching**:
   - Convert both to absolute UTC timestamps
   - Find overlaps (e.g., > 50% overlap)
   - Map speaker names to speaker IDs

## Key Differences Summary

| Feature | Segments | Lifelog Blockquotes |
|---------|----------|---------------------|
| **Source** | Local diarization | Limitless API |
| **Speaker ID** | Global ID (`spkr_abc123`) | Name (`"You"`, `"Jon Stearley"`) |
| **Transcript** | ❌ No | ✅ Yes |
| **Time Units** | Seconds (relative) | Milliseconds (relative) |
| **Byte Offsets** | ✅ Yes | ❌ No |
| **Recording Ref** | ✅ Yes | ❌ No |
| **Metadata** | Minimal | Rich (title, markdown) |
| **Matching Method** | Embedding similarity | Time-based overlap |

## Conclusion

**Overlap**: Both represent "who spoke when" - same concept, different implementations.

**Key Differences**:
- Segments: Audio-based, no transcripts, embedding-based speaker matching
- Blockquotes: API-based, has transcripts, name-based speaker identification

**Best Approach**: Store separately, link via time-based matching. This preserves the strengths of each (embeddings for segments, transcripts for blockquotes) while allowing them to complement each other.













