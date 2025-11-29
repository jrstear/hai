# Implementation Ready Summary

## All Decisions Finalized ✅

### 1. Storage Architecture
- ✅ **Normalized database** for segments (NOT JSON)
- ✅ **Minimal JSON files** (embeddings only, backup/debugging)
- ✅ **Database location**: `data/speakers.db`

### 2. Schema Design
- ✅ **speakers** table: Global speaker registry with embeddings
- ✅ **recordings** table: Audio file metadata
- ✅ **segments** table: Normalized segments (time + optional byte offsets)
- ✅ **contact_id** in speakers table (external ID reference)

### 3. Speaker Identification
- ✅ **UUID + cosine similarity** matching
- ✅ **Threshold**: 0.85 (tune as needed)
- ✅ **Matching timing**: During import (synchronous)
- ✅ **Per-user isolation**: Only check speakers within user's recordings

### 4. Byte Offsets
- ✅ **Optional**: NULL if unavailable (don't halt/error)
- ✅ **Method**: ffprobe (separate call, pyannote doesn't provide)
- ✅ **Format**: HTTP Range request compatible (DASH-style)
- ✅ **Calculation**: Background/async, can retry later

### 5. Performance
- ✅ **In-memory cosine similarity** (fine for < 1k speakers)
- ✅ **Future scaling**: Documented in beads issue `hai-a2z`

### 6. Contacts Integration
- ✅ **Initial approach**: vCard export (`data/contacts/contacts.vcf`)
- ✅ **Future**: Full sync with Google People API + macOS Contacts
- ✅ **Beads issue**: `hai-eog` created

### 7. Error Handling
- ✅ **Byte offsets**: Skip if unavailable (NULL in DB)
- ✅ **Matching**: Fallback to new UUID if computation fails
- ✅ **DB locks**: Use SQLite WAL mode

## Key Documents

1. **`history/SPEAKER_DATABASE_SCHEMA.md`** - Full schema design
2. **`history/CONTACTS_INTEGRATION_NOTES.md`** - Contacts export steps
3. **`history/BYTE_OFFSET_EFFICIENCY_CHECK.md`** - Byte offset analysis
4. **`history/REMAINING_CONSIDERATIONS.md`** - All decisions documented

## Beads Issues

1. **`hai-tw5`**: Design and implement normalized speaker database schema
2. **`hai-645`**: Build OGG byte offset indexer (depends on hai-tw5)
3. **`hai-d8z`**: Migrate existing diarization results to database (depends on hai-tw5)
4. **`hai-eog`**: Integrate contacts from macOS and Google Contacts
5. **`hai-a2z`**: Plan for scaling speaker matching beyond 1k speakers

## Next Steps

1. ✅ Schema design (done)
2. ✅ All decisions finalized (done)
3. 🔲 Create SQLite database with schema
4. 🔲 Build import tool (JSON → DB)
5. 🔲 Update diarization pipeline (save minimal JSON + import to DB)
6. 🔲 Build byte offset indexer (using ffprobe)
7. 🔲 Implement speaker matching (cosine similarity)
8. 🔲 Test with existing `15.json`

## User Actions Needed

1. **Export contacts**: 
   - macOS: Contacts app > File > Export > Export vCard
   - Google: contacts.google.com > Export > vCard format
   - Save to `data/contacts/contacts.vcf`

2. **Review documents** (user mentioned they'll read more carefully)

## Ready to Implement! 🚀

All planning complete, all decisions made, ready to start coding the database schema and import pipeline.

