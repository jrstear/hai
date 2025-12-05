# Migration Plan for Existing Diarization Results (hai-d8z)

## Current State

### Elasticsearch Indices (Already Designed & Implemented)

All three indices are already created in `storage/elasticsearch.go`:

1. **`speakers` index**:
   - `id` (keyword)
   - `embedding` (dense_vector, 256 dims, cosine similarity)
   - `first_seen`, `last_seen` (date)
   - `contact_id` (keyword, optional)
   - `created_at`, `updated_at` (date)

2. **`recordings` index**:
   - `id` (keyword) - format: `rec_YYYY_MM_DD_HH`
   - `file_path` (keyword)
   - `start_time` (date)
   - `duration_seconds` (float)
   - `sample_rate`, `format`, `device` (optional)
   - `diarization_metadata`: `diarized_at`, `processing_time`, `rtf`
   - `created_at` (date)

3. **`segments` index**:
   - `id` (keyword) - generated ID
   - `speaker_id` (keyword) - references Speaker.ID
   - `recording_id` (keyword) - references Recording.ID
   - `local_speaker_id` (keyword, optional) - original SPEAKER_XX
   - `start_time`, `end_time`, `duration` (float)
   - `start_byte_offset`, `end_byte_offset` (long, optional)
   - `created_at` (date)

### Export Logic (Already Implemented)

The `export2elastic` module already has all the logic we need:

- **`ExportResult()`** function:
  - Matches speakers using kNN search (cosine similarity)
  - Creates/updates speaker records
  - Creates recording records
  - Creates segment records
  - Has skip logic (checks if segments already exist)

- **Speaker matching**:
  - Uses `FindSimilarSpeakers()` with threshold 0.85
  - Handles zero-magnitude embeddings (logs warning, creates speaker without embedding)
  - Maps local speaker IDs (SPEAKER_XX) to global speaker IDs (spkr_xxxxx)

- **Skip logic**:
  - Checks if segments already exist for a recording
  - Returns `wasSkipped = true` if found
  - Prevents duplicate indexing

### Existing Tools

- **`cmd/load-es`**: CLI tool that loads a single diarization JSON file
  - Reads JSON file
  - Calls `ExportResult()`
  - Handles skip logic
  - Prints status messages

## Migration Plan

### Approach: Reuse Existing Export Logic

Since `ExportResult()` already does everything we need, the migration tool should:

1. **Scan for all diarization JSON files**:
   - Pattern: `data/YYYY/MM/DD/HH.json`
   - Recursively find all matching files
   - Exclude `lifelog.json` files

2. **For each JSON file**:
   - Read and parse the JSON
   - Derive audio file path (replace `.json` with `.ogg`)
   - Call `ExportResult()` (or use the existing `load-es` logic)
   - Handle errors gracefully (log and continue)
   - Report progress

3. **Progress reporting**:
   - Show total files found
   - Show files processed
   - Show files skipped (already in ES)
   - Show files failed
   - Show summary statistics (speakers created, segments indexed)

### Implementation Options

**Option A: New migration CLI tool**
- Create `cmd/migrate-diarization/main.go`
- Scans for all JSON files
- Processes them in batches or sequentially
- Provides progress output

**Option B: Extend existing `load-es` tool**
- Add `--all` or `--directory` flag
- Recursively process all JSON files in a directory
- Reuse existing logic

**Option C: Use existing `load-es` in a script**
- Write a shell script that finds all JSON files
- Calls `load-es` for each file
- Simpler but less efficient (multiple ES connections)

### Recommended: Option A (New Migration Tool)

**Benefits:**
- Single Elasticsearch connection (more efficient)
- Better error handling and progress reporting
- Can process in batches
- Can be run as a one-time migration

**Structure:**
```
cmd/migrate-diarization/
└── main.go
    - Scans data/ directory for *.json files (excluding lifelog.json)
    - For each file:
      - Read JSON
      - Derive audio path
      - Call ExportResult()
      - Track statistics
    - Print summary
```

### Data Flow

```
Existing JSON files (data/YYYY/MM/DD/HH.json)
    ↓
Migration tool reads JSON
    ↓
Parse diarization.Result
    ↓
ExportResult() (existing function)
    ├── Match speakers (kNN search)
    ├── Create/update speakers
    ├── Create recording
    └── Create segments (bulk)
    ↓
Elasticsearch indices
    ├── speakers
    ├── recordings
    └── segments
```

### Skip Logic

The migration should respect existing data:
- If segments already exist for a recording → skip (already migrated)
- If speakers already exist → update `last_seen` timestamp
- If recording already exists → update metadata

This allows:
- Re-running migration safely (idempotent)
- Migrating incrementally
- Resuming after errors

### Error Handling

- **Invalid JSON**: Log error, skip file, continue
- **Missing audio file**: Log warning, continue (audio path is optional for segments)
- **Elasticsearch errors**: Log error, skip file, continue
- **Speaker matching errors**: Log warning, create new speaker, continue

### Statistics to Report

- Total files found
- Files processed successfully
- Files skipped (already in ES)
- Files failed
- Speakers created
- Speakers matched (existing)
- Recordings created
- Segments indexed

## Questions to Consider

1. **Processing order**: Should we process chronologically (oldest first) or by date?
   - **Recommendation**: Process in chronological order (oldest first) so speaker matching works better

2. **Batch processing**: Process all at once or in batches?
   - **Recommendation**: Process sequentially (one file at a time) for simplicity and better error handling

3. **Dry run mode**: Should we support a dry-run to see what would be migrated?
   - **Recommendation**: Yes, add `--dry-run` flag that shows what would be done without actually doing it

4. **Resume capability**: Should we track which files have been processed?
   - **Recommendation**: No need - skip logic already handles this. Re-running is safe.

## Summary

✅ **Indices are designed and implemented** - All three indices exist with proper mappings
✅ **Export logic exists** - `ExportResult()` does everything we need
✅ **Skip logic exists** - Prevents duplicate indexing
✅ **Schema is defined** - All types in `storage/schema.go`

**Next step**: Create migration tool that scans for JSON files and calls `ExportResult()` for each.











