# Resume Guide - Next Session

## 📚 What to Review Before Starting

### Essential Reading (Start Here)

1. **`history/IMPLEMENTATION_READY_SUMMARY.md`**
   - Complete overview of all decisions made
   - Full checklist of what's ready for implementation
   - Quick reference for key decisions

2. **`history/SPEAKER_DATABASE_SCHEMA.md`**
   - Complete database schema design
   - Table definitions (speakers, recordings, segments)
   - Indexes, foreign keys, byte offsets
   - Data flow and query patterns

3. **`history/REMAINING_CONSIDERATIONS.md`**
   - All finalized decisions
   - Error handling approach
   - Migration strategy
   - Performance considerations

### Quick Reference (Optional)

4. **`history/SCHEMA_DISCUSSION_SUMMARY.md`**
   - Quick summary of discussion points
   - Key architectural decisions

5. **`AGENTS.md`**
   - Beads workflow reminder
   - Issue tracking practices

## 🎯 Next Recommended Steps

### Step 1: Start Database Schema Implementation (Priority 1)

**Issue**: `hai-tw5` - Design and implement normalized speaker database schema

**What to do:**
1. Create SQLite database at `data/speakers.db`
2. Implement schema from `history/SPEAKER_DATABASE_SCHEMA.md`
3. Create migration script to set up tables

**Key files to create:**
- `cmd/db/schema.sql` - SQL schema definitions
- `cmd/db/init.py` or `cmd/db/init.go` - Database initialization script
- Or Python script using `sqlite3` module

**Reference docs:**
- `history/SPEAKER_DATABASE_SCHEMA.md` - Full schema design
- `history/REMAINING_CONSIDERATIONS.md` - Error handling, WAL mode

### Step 2: Migration Tool (After Schema)

**Issue**: `hai-d8z` - Migrate existing diarization results to database

**What to do:**
1. Build import tool that reads `data/audio/2025/11/22/15.json`
2. Extract segments and embeddings
3. Import to database with speaker matching (UUID + cosine similarity)

**Reference docs:**
- `history/SPEAKER_DATABASE_SCHEMA.md` - Migration strategy section
- `history/REMAINING_CONSIDERATIONS.md` - Speaker matching workflow

### Step 3: Byte Offset Indexer (Can be parallel)

**Issue**: `hai-645` - Build OGG byte offset indexer

**What to do:**
1. Use ffprobe to extract frame info
2. Build timestamp → byte_offset mapping
3. Update segments table with byte offsets

**Reference docs:**
- `history/BYTE_OFFSET_EFFICIENCY_CHECK.md` - ffprobe approach
- `history/REMAINING_CONSIDERATIONS.md` - Optional byte offsets

## 🔍 Quick Commands to Get Started

```bash
# See what's ready to work on
bd ready --json

# Check issue details
bd show hai-tw5 --json

# Review implementation status
cat history/IMPLEMENTATION_READY_SUMMARY.md

# Check current schema design
cat history/SPEAKER_DATABASE_SCHEMA.md
```

## 📋 Implementation Checklist

From `history/IMPLEMENTATION_READY_SUMMARY.md`:

1. ✅ Schema design (done)
2. ✅ All decisions finalized (done)
3. 🔲 Create SQLite database with schema
4. 🔲 Build import tool (JSON → DB)
5. 🔲 Update diarization pipeline (save minimal JSON + import to DB)
6. 🔲 Build byte offset indexer (using ffprobe)
7. 🔲 Implement speaker matching (cosine similarity)
8. 🔲 Test with existing `15.json`

## 🎯 Recommended Starting Point

**Start with**: `hai-tw5` (Database Schema)

**Why:**
- Foundation for all other work
- No dependencies (other issues depend on this)
- Clear, well-documented design ready
- Can test immediately with SQLite

**First task:**
Create `cmd/db/init.py` or `cmd/db/schema.sql` and initialize `data/speakers.db` with the three core tables.

## 📁 Key Files to Reference

### Planning Docs (in `history/`)
- `IMPLEMENTATION_READY_SUMMARY.md` - **Start here**
- `SPEAKER_DATABASE_SCHEMA.md` - **Implementation reference**
- `REMAINING_CONSIDERATIONS.md` - **Decisions & edge cases**

### Existing Code (to understand context)
- `cmd/diarize/diarize.py` - Current diarization output format
- `data/audio/2025/11/22/15.json` - Example data to migrate

### Configuration
- `AGENTS.md` - Beads workflow
- `.gitignore` - What's protected (`.env`, etc.)

## 💡 Tips

1. **Test incrementally**: Create schema, test with one recording first
2. **Use existing data**: `15.json` is your test case (1,462 segments, 5 speakers)
3. **Check beads issues**: Each has documentation references in comments
4. **Speaker matching**: Start simple, optimize later (in-memory cosine similarity)

## 🚀 Ready to Start?

1. Read `history/IMPLEMENTATION_READY_SUMMARY.md` (5 min)
2. Review `history/SPEAKER_DATABASE_SCHEMA.md` schema section (10 min)
3. Check `bd ready --json` for current status
4. Start implementing `hai-tw5` (database schema)

## 🤔 Design Considerations to Review

Before implementing the schema, review these considerations documented in beads:

1. **`hai-lkp`**: Rename "speakers" to "voices" terminology
   - More accurate: voice signatures vs actual people
   - Affects table names, columns, variables

2. **`hai-9qz`**: Voice-to-contact mapping strategy (1:1 vs 1:many)
   - One contact may have multiple voice signatures
   - Need to decide before contact association

3. **`hai-avb`**: Voice clustering and pruning
   - Prevent table bloat with unused voices
   - Use cosine similarity for clustering

See `history/VOICE_TERMINOLOGY_CONSIDERATIONS.md` for detailed analysis.

Good luck! 🎉

