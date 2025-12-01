# Speaker-Embedding-Segment Linking: Solving First-Seen Bias

## The Problem with Current Approach

**Scenario:**
1. First diary: Creates speakers (single-embedding) for each embedding
2. Clustering: Merges into centroids
3. Second diary: Matches against centroids (e.g., 0.92 similarity)
   - Uses existing centroid speaker
   - **BUT: Doesn't save the new embedding!**
4. Result: Centroid still based on first-seen embedding (first-seen bias persists)

**The issue:** Even though we store embeddings in `SpeakerEmbedding` table, if we match against centroids, we might not be saving the new embedding, or if we do save it, the centroid isn't updated to include it.

## Proposed Solution: Segments → Embeddings → Speakers

### Schema Changes

**Current:**
```go
type Segment struct {
    SpeakerID string // Direct reference to Speaker.ID
    // ...
}

type SpeakerEmbedding struct {
    SpeakerID *string // Points to Speaker.ID (centroid)
    // ...
}
```

**Proposed (Normalized):**
```go
type Segment struct {
    SpeakerEmbeddingID string // References SpeakerEmbedding.ID (not Speaker.ID directly)
    // No SpeakerID field - queries join through SpeakerEmbedding
    // ...
}

type SpeakerEmbedding struct {
    SpeakerID *string // Points to Speaker.ID (centroid), NULL until clustered
    // ...
}
```

### Flow

**During Export:**
1. **Always save all embeddings** in `SpeakerEmbedding` table (even when match found)
2. Try to match against centroids
3. If match found: Set `SpeakerEmbedding.SpeakerID` to centroid
4. If no match: Leave `SpeakerEmbedding.SpeakerID = NULL` (will be set during clustering)
5. **Point segments at `SpeakerEmbedding.ID`** (not Speaker.ID directly)

**During Clustering:**
1. Cluster all embeddings (including new ones)
2. Recompute centroids (including new embeddings in calculation)
3. Update `SpeakerEmbedding.SpeakerID` to point to centroid
4. **Segments automatically "see" the updated speaker** through the embedding

## Benefits

### 1. Solves First-Seen Bias Completely

**Before:**
- First diary: Creates speaker with embedding A
- Clustering: Centroid = embedding A
- Second diary: Matches, uses centroid, but embedding B is not included in centroid
- Result: Centroid still based on embedding A

**After:**
- First diary: Creates speaker, saves embedding A
- Clustering: Centroid = embedding A
- Second diary: Matches, saves embedding B, points to centroid
- Clustering (re-run): Centroid = weighted average of [A, B] (includes new embedding!)
- Result: Centroid improves with each new embedding

### 2. Tracks Which Embeddings Were Used

**Metadata:**
- `SpeakerEmbedding.CreatedAt` - When embedding was created
- `Speaker.UpdatedAt` - When centroid was last computed

**Can determine:**
- Which embeddings were included in centroid calculation
- Which new embeddings need to be included in next clustering
- Can optimize clustering to only recompute centroids with new embeddings

### 3. Preserves All Data

- All embeddings saved (no data loss)
- Can recompute centroids with different algorithms/thresholds
- Can analyze embedding evolution over time

## Query Implications

### Current Queries (Direct Speaker Reference)

```go
// Get all segments for a speaker
segments := storage.GetSegmentsBySpeaker(ctx, speakerID)
```

**Query:** `SELECT * FROM segments WHERE speaker_id = ?`

### New Queries (Through Embedding)

```go
// Get all segments for a speaker
// Need to join: segments → speaker_embeddings → speakers
segments := storage.GetSegmentsBySpeaker(ctx, speakerID)
```

**Query:**
```sql
SELECT s.* FROM segments s
JOIN speaker_embeddings se ON s.speaker_embedding_id = se.id
WHERE se.speaker_id = ?
```

**Or in Elasticsearch:**
```json
{
  "query": {
    "bool": {
      "must": [
        { "term": { "speaker_embedding_id": { ... } } }
      ]
    }
  },
  "join": {
    "from": "speaker_embeddings",
    "to": "segments",
    "on": "speaker_embedding_id"
  }
}
```

**Actually simpler in Elasticsearch:**
- Can use nested query or join query
- Or: Add `speaker_id` field to segments (denormalized) for faster queries

## Hybrid Approach: Denormalize for Performance

**Best of both worlds:**

```go
type Segment struct {
    SpeakerEmbeddingID string // Primary reference (for tracking)
    SpeakerID          string // Denormalized (for fast queries)
    // ...
}
```

**Benefits:**
- `SpeakerEmbeddingID`: Tracks which embedding this segment came from
- `SpeakerID`: Fast queries without joins
- When clustering updates `SpeakerEmbedding.SpeakerID`, also update `Segment.SpeakerID`

**Trade-off:**
- Slight denormalization (SpeakerID stored in both places)
- Need to update segments when clustering changes speaker assignments
- But queries stay fast

## Implementation

### Export Flow

```go
for localSpeakerID, embedding := range result.SpeakerEmbeddings {
    // 1. Always save embedding
    speakerEmbedding := &SpeakerEmbedding{
        ID:            generateEmbeddingID(),
        RecordingID:   recordingID,
        LocalSpeakerID: localSpeakerID,
        Embedding:     embedding,
        DurationSeconds: calculateDuration(localSpeakerID, result.Segments),
        SpeakerID:     nil, // Will be set during clustering or if match found
        CreatedAt:     now,
    }
    storage.CreateSpeakerEmbedding(ctx, speakerEmbedding)
    
    // 2. Try to match against centroids
    matches := storage.FindSimilarSpeakers(ctx, embedding, threshold=0.85, limit=1, onlyCentroids=true)
    
    if len(matches) > 0 && matches[0].Similarity >= 0.85 {
        // Match found: point embedding at centroid
        speakerEmbedding.SpeakerID = &matches[0].Speaker.ID
        storage.UpdateSpeakerEmbedding(ctx, speakerEmbedding)
    }
    // If no match: SpeakerID stays NULL, will be set during clustering
    
    // 3. Point segments at embedding
    speakerMap[localSpeakerID] = speakerEmbedding.ID // Use embedding ID, not speaker ID
}
```

### Clustering Flow

```go
// After clustering, update embeddings (segments automatically see updated speaker through embedding)
for _, cluster := range clusters {
    centroidSpeakerID := createOrUpdateCentroidSpeaker(cluster)
    
    // Update all embeddings in cluster
    for _, embedding := range cluster.Embeddings {
        embedding.SpeakerID = &centroidSpeakerID
        storage.UpdateSpeakerEmbedding(ctx, embedding)
        
        // No need to update segments - they reference SpeakerEmbedding, which now points to centroid
        // Queries will automatically get the updated speaker through the join
    }
}
```

## Schema Changes Required

### Option A: Segments Point to Embeddings Only

```go
type Segment struct {
    SpeakerEmbeddingID string // References SpeakerEmbedding.ID
    // Remove: SpeakerID (query through embedding)
}
```

**Pros:**
- Single source of truth
- No denormalization

**Cons:**
- Queries require joins
- More complex query logic

### Option B: Normalized (Chosen)

```go
type Segment struct {
    SpeakerEmbeddingID string // References SpeakerEmbedding.ID
    // No SpeakerID field - queries join through SpeakerEmbedding
    // ...
}
```

**Pros:**
- Maintains normalization principles
- Single source of truth
- Simpler design
- ES is fast enough for joins

**Cons:**
- Queries require joins (but ES handles this efficiently)
- If performance becomes an issue, can add denormalization later

## Recommendation: Normalized Approach (No Denormalization)

**Use only SpeakerEmbeddingID:**
- `SpeakerEmbeddingID`: Tracks embedding (solves first-seen bias)
- No `SpeakerID` in Segment (queries join through SpeakerEmbedding)

**When clustering updates:**
- Update `SpeakerEmbedding.SpeakerID` only
- Segments automatically "see" updated speaker through embedding reference

**Rationale:**
- Maintains normalization principles
- ES is fast enough for joins
- If performance becomes an issue, we can address it then (avoid premature optimization)
- Cleaner, simpler design

This gives us:
- ✅ All embeddings preserved
- ✅ Centroids recomputed with new embeddings
- ✅ Normalized schema (no denormalization)
- ✅ Tracks which embedding each segment came from
- ✅ Solves first-seen bias completely

