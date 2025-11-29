# Remaining Schema & Implementation Considerations

## Decisions Made ✅
1. UUID + cosine similarity (threshold 0.85)
2. ffprobe for byte offsets (confirmed working)
3. Keep all segments (no merging)
4. Keep JSON files for now (backup/debugging)

## Additional Considerations

### 1. Contacts Table Schema
**Question**: Do we need a `contacts` table, or just reference external IDs?

**From seed.md**: Contacts come from Google/Apple contacts integration.

**Options**:
- **Option A**: Minimal - just store external contact ID (e.g., `google_contact_abc123`)
  - Pros: Simple, no sync needed
  - Cons: Need to query external API for name/picture
- **Option B**: Full contacts table with sync
  - Pros: Fast queries, works offline
  - Cons: Need sync logic, data duplication

**Recommendation**: Start with Option A (external ID reference), add sync table later if needed.

**Schema addition**:
```sql
-- In speakers table, already have:
contact_id TEXT  -- Could be 'google_abc123' or 'apple_xyz789'
-- No contacts table initially, just reference external IDs
```

### 2. Database Location
**Question**: Where should the SQLite database live?

**Options**:
- `data/speakers.db` (alongside audio files)
- `data/db/speakers.db` (separate db directory)
- `.beads/` (with issue tracker)
- Root directory `speakers.db`

**Recommendation**: `data/speakers.db` (co-located with audio data)

### 3. Speaker Matching Workflow ✅
**Decision**: **During import (synchronous)**
- When diarization completes, extract embeddings
- Match immediately against existing speakers using cosine similarity
- If match found (similarity > 0.85), use existing speaker_id
- If no match, create new speaker with UUID
- Load all existing embeddings into memory for fast batch comparison
- Per-user isolation: only check speakers within user's recordings (not cross-user)

**Note**: Pyannote outputs only timestamps (seconds), not byte offsets. Byte offsets require separate ffprobe call.

### 4. Speaker Merging Workflow
**Question**: How to handle when two UUIDs are determined to be the same person?

**Scenario**: User manually identifies two speakers as the same person.

**Process**:
1. User action: "Merge speaker A and speaker B"
2. Update all segments: `UPDATE segments SET speaker_id = 'kept_id' WHERE speaker_id = 'merged_id'`
3. Update speaker record: `UPDATE speakers SET embedding = (average or keep one), last_seen = MAX(...)`
4. Delete old speaker record: `DELETE FROM speakers WHERE id = 'merged_id'`

**Future**: Could add `merged_into` field to track history.

### 5. Error Handling
**Questions**:
- What if ffprobe fails? (skip byte offsets, retry later?)
- What if cosine similarity computation fails? (fallback to new UUID?)
- What if DB is locked? (retry, queue?)

**Recommendation**: 
- Byte offsets: Make optional (NULL if not available), can retry later
- Matching: Fallback to new UUID if computation fails
- DB locks: Use SQLite WAL mode, handle retries

### 6. Performance & Indexing ✅
**Decision**: **In-memory cosine similarity** (fine for < 1000 speakers per user)
- Load all speaker embeddings into memory during matching
- Compute cosine similarity in Python (using sklearn or numpy)
- Per-user isolation ensures we only check speakers within user's recordings
- 1k speakers seems reasonable starting point

**Future scaling** (years, if needed):
- SQLite vector extensions (vector0, sqlite-vss)
- Approximate nearest neighbor search
- External vector DB (Qdrant, Pinecone) if multi-user service
- See beads issue `hai-a2z` for future optimization notes

**Indexes in schema**:
- Indexes on `speaker_id`, `recording_id`, time ranges, byte ranges

### 7. Concurrency
**Question**: What if multiple diarizations run simultaneously?

**Considerations**:
- SQLite supports concurrent reads (WAL mode)
- Writes are serialized (fine for our use case)
- Speaker matching needs to be atomic (check-then-insert)

**Recommendation**: Use SQLite WAL mode, handle retries for conflicts.

### 8. Migration from Existing JSON
**Question**: How to handle existing `15.json` file?

**Process**:
1. Read JSON file
2. Extract segments and embeddings
3. For each speaker embedding:
   - Try to match with existing speakers (cosine similarity)
   - If match: use existing speaker_id
   - If no match: create new speaker with UUID
4. Create recording entry
5. Insert all segments with speaker_id references
6. Calculate byte offsets (async/background)

**Tool**: Create `cmd/diarize/import_to_db.py` script

### 9. API/Query Design (Future)
**Not needed now**, but consider:
- REST API for querying segments?
- GraphQL?
- Direct SQL queries from app?

**Recommendation**: Start with direct SQL, add API layer later.

### 10. Testing Strategy
**Questions**:
- Unit tests for speaker matching?
- Integration tests for import?
- Test data?

**Recommendation**: 
- Start with manual testing on existing `15.json`
- Add tests as we encounter edge cases

## Next Implementation Steps

1. ✅ Schema design (done)
2. ✅ Decisions finalized (done)
3. 🔲 Create SQLite database with schema
4. 🔲 Build import tool (JSON → DB)
5. 🔲 Update diarization pipeline (save minimal JSON + import to DB)
6. 🔲 Build byte offset indexer (using ffprobe)
7. 🔲 Implement speaker matching (cosine similarity)
8. 🔲 Test with existing `15.json`

## Decisions Finalized ✅

1. ✅ **Contacts**: External ID reference only (for now), vCard export as starter data
   - See `history/CONTACTS_INTEGRATION_NOTES.md` for export steps
   - Beads issue `hai-eog` created for future contacts integration
2. ✅ **Database location**: `data/speakers.db` (confirmed)
3. ✅ **Speaker matching**: During import (synchronous), per-user isolation
4. ✅ **Byte offsets**: Optional (NULL if unavailable), don't halt/error
   - Pyannote doesn't provide byte offsets (only timestamps)
   - Requires separate ffprobe call
   - See `history/BYTE_OFFSET_EFFICIENCY_CHECK.md` for analysis
5. ✅ **Performance**: In-memory cosine similarity (fine for < 1k speakers)
   - Beads issue `hai-a2z` created for future scaling notes

**Ready to proceed with implementation!**

