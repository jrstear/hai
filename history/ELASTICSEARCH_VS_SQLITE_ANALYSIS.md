# Elasticsearch vs SQLite: Storage Platform Analysis

## Current Requirements Summary

Based on the planned SQLite schema, your system needs:

1. **Speaker Registry**: Store 256-dim embeddings, track first/last seen, link to contacts
2. **Recordings Metadata**: File paths, timestamps, duration, format, diarization status
3. **Segments**: Time ranges (start/end in seconds), byte offsets (for HTTP Range requests), speaker associations
4. **Speaker Matching**: Cosine similarity search across embeddings (threshold ~0.85)
5. **Query Patterns**:
   - Find all segments for a speaker
   - Find recordings containing a speaker
   - Get segments by time range
   - Get byte offsets for playback
   - Find similar speakers (vector similarity)

## Elasticsearch Overview

Elasticsearch is a distributed search and analytics engine built on Apache Lucene. It's designed for:
- Full-text search
- Vector similarity search (kNN)
- Time-series data
- Aggregations and analytics
- Horizontal scaling

## ⚠️ Important: Diarization is Separate

**Elasticsearch does NOT have built-in diarization capabilities.** It is purely a search and analytics engine for already-processed data.

### Your Current Diarization Pipeline

You're using **pyannote.audio** (`pyannote/speaker-diarization-3.1`) to:
1. Process raw audio files (OGG format)
2. Identify speaker segments (who spoke when)
3. Extract speaker embeddings (256-dim vectors)
4. Output structured JSON with segments and embeddings

### Data Flow with Elasticsearch

If you use Elasticsearch, your pipeline would be:

```
Raw Audio (OGG)
    ↓
[pyannote.audio diarization] ← Still needed!
    ↓
JSON with segments + embeddings
    ↓
[Index to Elasticsearch] ← ES only stores/queries this
    ↓
Query via ES (vector search, aggregations, etc.)
```

**Key Point**: Elasticsearch replaces the **storage and query layer**, not the **audio processing layer**. You would still need:
- ✅ Your existing `diarize.py` script (pyannote pipeline)
- ✅ Python environment with PyTorch, pyannote.audio
- ✅ Hugging Face token for model access
- ✅ GPU/CPU for running diarization

**What Elasticsearch adds**:
- Better vector similarity search (vs in-memory Python)
- Scalable storage and querying
- Full-text search (if you add transcripts later)
- Rich analytics and aggregations

**What stays the same**:
- Audio preprocessing (diarization)
- Embedding extraction
- Byte offset calculation (ffprobe)

## Capability Comparison

### ✅ Where Elasticsearch Excels

#### 1. **Native Vector Similarity Search**
- **ES**: Built-in `dense_vector` field type with kNN search
- **SQLite**: Requires application-level cosine similarity (load all embeddings, compute in Python)
- **Impact**: ES can handle millions of speakers efficiently; SQLite approach limited to ~1k speakers in memory

**ES Example**:
```json
PUT /speakers/_mapping
{
  "properties": {
    "embedding": {
      "type": "dense_vector",
      "dims": 256,
      "index": true,
      "similarity": "cosine"
    }
  }
}

GET /speakers/_search
{
  "knn": {
    "field": "embedding",
    "query_vector": [0.1, 0.2, ...],
    "k": 10,
    "num_candidates": 100
  }
}
```

#### 2. **Full-Text Search (Future-Proofing)**
- **ES**: Native full-text search with analyzers, stemming, fuzzy matching
- **SQLite**: Basic `LIKE` queries or FTS5 extension (more limited)
- **Impact**: If you add transcripts later, ES will shine

#### 3. **Complex Aggregations**
- **ES**: Rich aggregation framework (date histograms, nested aggregations, percentiles)
- **SQLite**: Basic GROUP BY, limited aggregation functions
- **Impact**: Better analytics ("speaker activity over time", "most frequent speakers", etc.)

#### 4. **Time-Series Queries**
- **ES**: Optimized for time-based queries, date range queries
- **SQLite**: Works but less optimized
- **Impact**: Better performance for "segments in time range" queries at scale

#### 5. **Horizontal Scaling**
- **ES**: Built for distributed clusters, sharding, replication
- **SQLite**: Single-file, single-machine
- **Impact**: ES can scale to petabytes; SQLite limited by single machine

### ⚠️ Where SQLite is Better

#### 1. **Byte Offset Storage**
- **SQLite**: Simple integer fields, straightforward queries
- **ES**: Works but less natural (numeric fields are fine, but no special optimization)
- **Impact**: Minimal - both can handle this, SQLite is just simpler

#### 2. **Relational Integrity**
- **SQLite**: Foreign keys, referential integrity, ACID transactions
- **ES**: No foreign keys, eventual consistency, no transactions
- **Impact**: ES requires application-level validation; SQLite enforces at DB level

#### 3. **Simple Queries**
- **SQLite**: Standard SQL, easy to understand
- **ES**: Query DSL (JSON), steeper learning curve
- **Impact**: SQLite queries are more intuitive for simple lookups

## Complexity Comparison

### SQLite Approach

**Setup Complexity**: ⭐ Very Low
- Single file database (`data/speakers.db`)
- No server process
- No configuration needed
- Works out of the box

**Operational Complexity**: ⭐ Very Low
- No monitoring needed
- No cluster management
- Backup = copy file
- No resource tuning

**Code Complexity**: ⭐⭐ Low-Medium
- Standard SQL queries
- Application-level cosine similarity (simple Python code)
- Straightforward schema

**Learning Curve**: ⭐ Very Low
- Standard SQL knowledge
- Python libraries (sqlite3, sklearn)

### Elasticsearch Approach

**Setup Complexity**: ⭐⭐⭐⭐ High
- Requires Java runtime (JVM)
- Server process must run
- Configuration files (elasticsearch.yml)
- Memory settings, heap size tuning
- Network ports (9200, 9300)
- Security setup (if needed)

**Operational Complexity**: ⭐⭐⭐⭐ High
- Monitor cluster health
- Manage indices (create, delete, optimize)
- Disk space management
- Memory usage monitoring
- Log management
- Backup/restore procedures
- Version upgrades

**Code Complexity**: ⭐⭐⭐ Medium
- Query DSL (JSON-based, not SQL)
- Mapping definitions (schema)
- Client libraries (elasticsearch-py)
- Error handling for cluster issues
- Retry logic for transient failures

**Learning Curve**: ⭐⭐⭐ Medium-High
- Elasticsearch concepts (indices, documents, mappings)
- Query DSL syntax
- Aggregation framework
- Cluster management basics

## Implementation Time Comparison

**Note**: These estimates assume diarization is already working (your existing `diarize.py` with pyannote). Both approaches use the same diarization preprocessing step - the difference is only in storage and querying.

### SQLite Implementation

**Phase 1: Schema & Basic Operations** (2-4 hours)
- Create database file
- Define tables (speakers, recordings, segments)
- Create indexes
- Basic CRUD operations

**Phase 2: Import Pipeline** (4-6 hours)
- JSON → SQLite import script
- Speaker matching logic (cosine similarity in Python)
- Error handling

**Phase 3: Query Layer** (2-3 hours)
- Query functions for common patterns
- Testing

**Phase 4: Byte Offset Indexing** (3-4 hours)
- ffprobe integration
- Byte offset calculation
- Update segments

**Total: ~11-17 hours** (1.5-2 days of focused work)

### Elasticsearch Implementation

**Phase 1: Setup & Configuration** (4-6 hours)
- Install Elasticsearch
- Configure JVM settings
- Create indices with mappings
- Set up client library
- Test basic operations

**Phase 2: Schema Design (Mappings)** (3-4 hours)
- Design document structure
- Define dense_vector fields for embeddings
- Configure analyzers (if needed)
- Index settings (shards, replicas)

**Phase 3: Import Pipeline** (6-8 hours)
- JSON → Elasticsearch import script
- Bulk indexing operations
- Speaker matching using kNN queries
- Error handling and retries

**Phase 4: Query Layer** (4-6 hours)
- Query DSL for common patterns
- kNN queries for speaker matching
- Aggregations for analytics
- Testing

**Phase 5: Byte Offset Indexing** (3-4 hours)
- Same as SQLite (ffprobe integration)
- Update documents

**Total: ~20-28 hours** (2.5-3.5 days of focused work)

**Additional Ongoing Time**:
- Learning curve: +5-10 hours
- Troubleshooting: +2-5 hours
- Operational setup: +2-4 hours

## Specific Use Case Analysis

### Use Case 1: Speaker Matching (Core Feature)

**SQLite Approach**:
```python
# Load all embeddings into memory
embeddings = load_all_speaker_embeddings()  # ~1k speakers = fine
similarities = cosine_similarity([new_embedding], embeddings)[0]
best_match_idx = np.argmax(similarities)
if similarities[best_match_idx] > 0.85:
    return existing_speaker_ids[best_match_idx]
```
- **Pros**: Simple, works for < 1k speakers
- **Cons**: Doesn't scale beyond ~1k speakers (memory + O(n) search)

**Elasticsearch Approach**:
```python
# kNN query - efficient even with millions
response = es.search(
    index="speakers",
    body={
        "knn": {
            "field": "embedding",
            "query_vector": new_embedding,
            "k": 1,
            "num_candidates": 100
        }
    }
)
```
- **Pros**: Scales to millions, efficient approximate search
- **Cons**: Requires ES server running, more complex setup

**Verdict**: ES wins for scalability, SQLite wins for simplicity at current scale

### Use Case 2: "Find all segments for speaker X"

**SQLite**:
```sql
SELECT * FROM segments 
WHERE speaker_id = 'spkr_abc123' 
ORDER BY start_time;
```
- Simple, fast with index

**Elasticsearch**:
```json
GET /segments/_search
{
  "query": {
    "term": { "speaker_id": "spkr_abc123" }
  },
  "sort": [{ "start_time": "asc" }]
}
```
- Works fine, but SQL is more intuitive

**Verdict**: Tie - both work well, SQLite is simpler

### Use Case 3: "Find recordings with this speaker"

**SQLite**:
```sql
SELECT DISTINCT recording_id FROM segments 
WHERE speaker_id = 'spkr_abc123';
```
- Simple aggregation

**Elasticsearch**:
```json
GET /segments/_search
{
  "query": { "term": { "speaker_id": "spkr_abc123" } },
  "aggs": {
    "recordings": { "terms": { "field": "recording_id" } }
  }
}
```
- More verbose but powerful

**Verdict**: SQLite simpler, ES more powerful for complex aggregations

### Use Case 4: Time Range Queries

**SQLite**:
```sql
SELECT * FROM segments 
WHERE recording_id = 'rec_123' 
  AND start_time >= 100.0 
  AND end_time <= 200.0;
```
- Works well with index

**Elasticsearch**:
```json
GET /segments/_search
{
  "query": {
    "bool": {
      "must": [
        { "term": { "recording_id": "rec_123" } },
        { "range": { "start_time": { "gte": 100.0 } } },
        { "range": { "end_time": { "lte": 200.0 } } }
      ]
    }
  }
}
```
- More verbose, but optimized for time-series

**Verdict**: SQLite simpler, ES potentially faster at very large scale

## Resource Requirements

### SQLite
- **Disk**: ~1-10 MB for database file (grows with data)
- **Memory**: Minimal (OS file cache)
- **CPU**: Low (only during queries)
- **Processes**: None (embedded library)

### Elasticsearch
- **Disk**: ~500 MB - 2 GB (Elasticsearch installation + data)
- **Memory**: 1-4 GB minimum (JVM heap)
- **CPU**: Moderate (JVM overhead, indexing)
- **Processes**: Elasticsearch server (always running)

## Complete Pipeline Comparison

### Current Pipeline (What You Have Now)

```
1. Raw Audio File (15.ogg)
   ↓
2. diarize.py (pyannote.audio)
   - Loads pyannote/speaker-diarization-3.1 model
   - Processes audio with GPU/CPU
   - Extracts segments + embeddings
   ↓
3. JSON Output (15.json)
   - Segments array
   - Speaker embeddings
   - Metadata
```

### Pipeline with SQLite

```
1. Raw Audio File (15.ogg)
   ↓
2. diarize.py (pyannote.audio) ← UNCHANGED
   - Same diarization process
   ↓
3. JSON Output (15.json) ← Still saved (minimal, for backup)
   ↓
4. Import Script (NEW)
   - Reads JSON
   - Matches speakers (cosine similarity in Python)
   - Inserts into SQLite
   ↓
5. SQLite Database (data/speakers.db) ← Storage layer
   - Query via SQL
   - Speaker matching via Python (in-memory)
```

### Pipeline with Elasticsearch

```
1. Raw Audio File (15.ogg)
   ↓
2. diarize.py (pyannote.audio) ← UNCHANGED
   - Same diarization process
   ↓
3. JSON Output (15.json) ← Still saved (optional, for backup)
   ↓
4. Import Script (NEW, different)
   - Reads JSON
   - Matches speakers (kNN query to ES)
   - Bulk indexes to Elasticsearch
   ↓
5. Elasticsearch Cluster ← Storage layer
   - Query via Query DSL
   - Speaker matching via kNN (native)
```

### What Stays the Same (Both Approaches)

- ✅ Audio file format (OGG)
- ✅ Diarization tool (pyannote.audio)
- ✅ Model (pyannote/speaker-diarization-3.1)
- ✅ Embedding extraction (256-dim vectors)
- ✅ JSON output format (at least initially)
- ✅ Byte offset calculation (ffprobe)
- ✅ Python environment requirements

### What Changes

| Component | SQLite | Elasticsearch |
|-----------|--------|---------------|
| **Storage** | SQLite file | ES indices |
| **Speaker Matching** | Python (in-memory) | ES kNN query |
| **Queries** | SQL | Query DSL (JSON) |
| **Setup** | Create DB file | Install & configure ES |
| **Operations** | File backup | Cluster management |

## When to Choose Each

### Choose SQLite If:
- ✅ Single-user or small-scale deployment
- ✅ < 1,000 speakers (current scale)
- ✅ Want minimal operational overhead
- ✅ Prefer simple, standard SQL
- ✅ Local/embedded use case
- ✅ Quick implementation needed
- ✅ No need for full-text search (yet)

### Choose Elasticsearch If:
- ✅ Multi-user service or large scale
- ✅ > 10,000 speakers expected
- ✅ Need full-text search (transcripts)
- ✅ Want rich analytics/aggregations
- ✅ Need horizontal scaling
- ✅ Have DevOps resources
- ✅ Willing to invest in learning curve
- ✅ Future-proofing for growth

## Hybrid Approach (Best of Both Worlds?)

**Option**: Start with SQLite, migrate to ES later

**Pros**:
- Quick start with SQLite
- Learn requirements in practice
- Migrate when you hit SQLite limits
- ES can import from SQLite

**Cons**:
- Migration effort later
- Some code rewrite needed
- But migration is straightforward (export SQLite → import ES)

## Recommendation

### For Your Current Situation:

**Start with SQLite** because:

1. **Scale**: You're at < 100 speakers currently, SQLite handles < 1k easily
2. **Speed**: Can implement in 1-2 days vs 3-4 days for ES
3. **Simplicity**: No server to manage, easier debugging
4. **Focus**: Get the core features working first, optimize later
5. **Migration Path**: Easy to migrate to ES later if needed

### When to Reconsider Elasticsearch:

- You exceed ~1,000 speakers per user
- You add transcript search requirements
- You need multi-user/multi-tenant architecture
- You want advanced analytics (speaker activity patterns, etc.)
- You have dedicated DevOps resources

### Migration Strategy (If Needed Later):

1. Export SQLite data to JSON
2. Create ES indices with mappings
3. Bulk import to ES
4. Update application code to use ES client
5. Test thoroughly
6. Switch over

**Migration effort**: ~1-2 days (much less than initial ES implementation)

## Conclusion

**Elasticsearch is powerful but overkill for your current needs.** The SQLite approach will:
- Get you to production faster
- Be easier to maintain
- Handle your scale for the foreseeable future
- Allow easy migration to ES later if needed

**Elasticsearch becomes compelling when:**
- You need to scale beyond 1k speakers
- You add full-text search (transcripts)
- You need multi-user architecture
- You want advanced analytics

**My recommendation**: **Start with SQLite, plan for ES migration if/when you hit scale limits.**

