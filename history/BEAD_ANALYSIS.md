# Bead Analysis: hai-d8z, hai-7rz, and hai-s20

## hai-645: Build OGG byte offset indexer
**Priority:** Decremented to 3 (optimization, may never be needed)

**Rationale:** Limitless API already serves audio by millisecond, so byte offset indexing may be unnecessary optimization.

## hai-d8z: Migrate existing diarization results to database

**Current State:**
- Diarization results are stored as JSON files: `data/YYYY/MM/DD/HH.json`
- Each JSON contains: segments, speaker embeddings, metadata
- Files are stored alongside audio files (e.g., `15.ogg` and `15.json`)

**What needs to happen:**
1. Read existing JSON files from `data/` directory structure
2. Extract segments and speaker embeddings
3. Map local speaker IDs (SPEAKER_XX) to global speaker IDs (spkr_xxxxx)
   - Use cosine similarity to match embeddings with existing speakers
   - Create new speakers if no match found
4. Create recording records
5. Insert segments with proper speaker_id and recording_id references
6. Store in Elasticsearch (or SQLite if still using that)

**Scope:**
- One-time migration script/tool
- Processes all existing JSON files
- Handles speaker matching across recordings
- Creates proper relationships (speakers, recordings, segments)

**Dependencies:**
- Storage interface must be implemented (Elasticsearch or SQLite)
- Speaker matching logic (cosine similarity)

**Files to process:**
- Pattern: `data/YYYY/MM/DD/HH.json`
- Example: `onboard/data/2025/11/20/21.json`, `22.json`, `23.json`, etc.

## hai-7rz: Fetch Lifelogs for Comparison

**Current State:**
- ✅ Lifelog fetching already implemented in `onboard/internal/fetch/lifelog.go`
- ✅ Lifelogs are saved to `data/YYYY/MM/DD/lifelog.json`
- ✅ Comparison script exists: `cmd/diarize/compare_with_lifelogs.py`

**What the bead says:**
- "Fetch lifelogs from Limitless API for Nov 22, 2025 3-7pm to compare speaker identification with local diarization results"

**Status:**
- **Already implemented!** The onboarding server fetches lifelogs automatically
- Comparison script exists but is manual
- This bead may be outdated or referring to a specific test case

**Recommendation:**
- Check if this is a specific test case or general feature
- If general feature: Close as already implemented
- If test case: Update to be more specific

## hai-s20: Store lifelogs in Elasticsearch and implement speaker name mapping

**Current Description:**
"Add lifelog storage to Elasticsearch backend. Store lifelog documents and blockquotes as separate indices. Implement time-based matching to map lifelog speaker names (e.g., 'You', 'Jon Stearley') to our global speaker IDs. Enable full-text search on lifelog transcripts."

### Analysis: Should This Be Split?

**Two distinct concerns:**

1. **Store lifelogs in Elasticsearch**
   - Add lifelog schema to Storage interface
   - Create `lifelogs` and `lifelog_blockquotes` indices
   - Implement CRUD operations
   - Export lifelogs from onboarding server
   - **Independent feature** - can be done standalone

2. **Implement speaker name mapping**
   - Time-based matching algorithm (lifelog blockquotes ↔ diarization segments)
   - Map `SpeakerName` → `SpeakerID`
   - Store mapping in `LifelogBlockquote.SpeakerID`
   - **Depends on**: Both lifelogs AND segments being stored
   - **More complex** - requires matching logic

**Recommendation: Split into two beads**

**Reasoning:**
- **Separation of concerns**: Storage vs. mapping logic
- **Dependencies**: Mapping requires both lifelogs and segments stored
- **Incremental development**: Can store lifelogs first, add mapping later
- **Testing**: Easier to test storage independently from mapping

**Proposed split:**

1. **hai-s20a**: Store lifelogs in Elasticsearch
   - Add lifelog schema
   - Implement storage operations
   - Export from onboarding server
   - Full-text search on transcripts

2. **hai-s20b**: Implement speaker name mapping
   - Time-based matching algorithm
   - Map lifelog speaker names to global speaker IDs
   - Store mappings
   - **Depends on**: hai-s20a (lifelogs stored) + segments stored

**Alternative: Keep together but clarify phases**
- Phase 1: Storage (independent)
- Phase 2: Mapping (depends on Phase 1 + segments)













