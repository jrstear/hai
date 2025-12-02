# Speaker Clustering: Denormalized Approach for Selective Embedding Storage

## Motivation

For large-scale deployment (100k+ users), storing ALL embeddings could become expensive. A denormalized approach allows:
1. **Selective embedding storage** - Only store embeddings that meet certain criteria
2. **Policy flexibility** - Easier to change storage policy without schema changes
3. **Fast queries** - Direct SpeakerID access without joins
4. **Cost optimization** - Store embeddings only when valuable

## Schema Design

### Denormalized Segment Schema

```go
type Segment struct {
    ID                int64     `json:"id"`
    SpeakerEmbeddingID *string  `json:"speaker_embedding_id"` // NULL if embedding not stored
    SpeakerID         *string   `json:"speaker_id"`            // NULL if no match found, set if match >= threshold
    RecordingID       string    `json:"recording_id"`
    LocalSpeakerID    *string   `json:"local_speaker_id"`
    StartTime         float64   `json:"start_time"`
    EndTime           float64   `json:"end_time"`
    Duration          float64   `json:"duration"`
    CreatedAt         time.Time `json:"created_at"`
}
```

**Key points:**
- `SpeakerEmbeddingID`: Optional - only set if embedding was stored
- `SpeakerID`: Optional - set if match found above threshold (e.g., 0.85)
- Both can be NULL (no embedding stored, no match found)
- Both can be set (embedding stored, match found)
- `SpeakerEmbeddingID` set but `SpeakerID` NULL (embedding stored, no match yet)

**Important:** When multiple speakers match above threshold, always use the **highest similarity match** (not just the first one). `FindSimilarSpeakers` returns results sorted by similarity descending, so the first match is the best.

### SpeakerEmbedding Schema (Unchanged)

```go
type SpeakerEmbedding struct {
    ID              string    `json:"id"`
    SpeakerID       *string   `json:"speaker_id"`        // NULL until clustered or matched
    RecordingID     string    `json:"recording_id"`
    LocalSpeakerID  string    `json:"local_speaker_id"`
    Embedding       []float32 `json:"embedding"`
    DurationSeconds float64   `json:"duration_seconds"`
    CreatedAt       time.Time `json:"created_at"`
}
```

## Embedding Storage Policy

### Decision Criteria

**Store embedding if ANY of:**
1. **Duration threshold**: Total speaking time >= X seconds (e.g., 30s)
2. **Novelty threshold**: Distance from nearest centroid > Y (e.g., similarity < 0.80)
3. **Match found**: Match found above threshold (store for centroid updates)
4. **First occurrence**: First time seeing this local speaker in recording

**Don't store if:**
- Short duration (< threshold) AND close match to existing centroid (similarity >= 0.90)
- Low quality embedding (zero magnitude, etc.)

### Policy Configuration

```go
type EmbeddingStoragePolicy struct {
    MinDurationSeconds    float64 // Minimum duration to store (e.g., 30.0)
    NoveltyThreshold     float64 // Store if similarity < this (e.g., 0.80)
    AlwaysStoreOnMatch   bool    // Always store if match found (for centroid updates)
    AlwaysStoreFirst     bool    // Always store first occurrence in recording
}
```

## Export Flow (Selective Storage)

```go
for localSpeakerID, embedding := range result.SpeakerEmbeddings {
    // 1. Calculate duration for this local speaker
    durationSeconds := calculateDuration(localSpeakerID, result.Segments)
    
    // 2. Try to match against centroids
    // Get multiple matches (limit > 1) to find the highest similarity match
    matches := storage.FindSimilarSpeakers(ctx, embedding, threshold=0.85, limit=10, onlyCentroids=true)
    // Find the highest similarity match (matches are already sorted by similarity descending)
    var bestMatch *SpeakerMatch = nil
    if len(matches) > 0 {
        bestMatch = matches[0] // First match has highest similarity (already sorted)
    }
    
    // 3. Decide whether to store embedding
    shouldStore := false
    var speakerEmbeddingID *string = nil
    
    if shouldStoreEmbedding(embedding, durationSeconds, bestMatch, policy) {
        // Store embedding
        speakerEmbedding := &SpeakerEmbedding{
            ID:              generateEmbeddingID(),
            RecordingID:     recordingID,
            LocalSpeakerID:  localSpeakerID,
            Embedding:       embedding,
            DurationSeconds: durationSeconds,
            SpeakerID:       nil, // Will be set if match found or during clustering
            CreatedAt:       now,
        }
        storage.CreateSpeakerEmbedding(ctx, speakerEmbedding)
        speakerEmbeddingID = &speakerEmbedding.ID
        
        // If match found, update embedding's SpeakerID
        if bestMatch != nil && bestMatch.Similarity >= 0.85 {
            speakerEmbedding.SpeakerID = &bestMatch.Speaker.ID
            storage.UpdateSpeakerEmbedding(ctx, speakerEmbedding)
        }
    }
    
    // 4. Determine SpeakerID for segments
    var segmentSpeakerID *string = nil
    if bestMatch != nil && bestMatch.Similarity >= 0.85 {
        segmentSpeakerID = &bestMatch.Speaker.ID
    }
    // If no match: segmentSpeakerID stays NULL (will be set during clustering)
    
    // 5. Create segments with both IDs
    for _, seg := range segmentsForLocalSpeaker(localSpeakerID, result.Segments) {
        segment := &Segment{
            SpeakerEmbeddingID: speakerEmbeddingID, // NULL if not stored
            SpeakerID:          segmentSpeakerID,   // NULL if no match
            RecordingID:        recordingID,
            LocalSpeakerID:     &localSpeakerID,
            // ... other fields
        }
        storage.CreateSegment(ctx, segment)
    }
}
```

### Storage Decision Function

```go
func shouldStoreEmbedding(
    embedding []float32,
    durationSeconds float64,
    bestMatch *SpeakerMatch,
    policy EmbeddingStoragePolicy,
) bool {
    // Always store if match found (for centroid updates)
    if policy.AlwaysStoreOnMatch && bestMatch != nil && bestMatch.Similarity >= 0.85 {
        return true
    }
    
    // Always store first occurrence
    if policy.AlwaysStoreFirst {
        // Check if this is first occurrence (implementation depends on tracking)
        return true
    }
    
    // Store if duration meets threshold
    if durationSeconds >= policy.MinDurationSeconds {
        return true
    }
    
    // Store if novel (far from existing centroids)
    if bestMatch == nil || bestMatch.Similarity < policy.NoveltyThreshold {
        return true
    }
    
    // Don't store: short duration and close match
    return false
}
```

## Clustering Flow

### With Selective Storage

**Challenge:** Not all embeddings are stored, so clustering only works on stored embeddings.

**Options:**
1. **Cluster only stored embeddings** - Simpler, but may miss some speakers
2. **Re-extract embeddings for clustering** - More accurate, but requires re-processing
3. **Hybrid**: Cluster stored embeddings, then match unstored segments to clusters

**Recommended: Option 1 (cluster stored embeddings only)**

```go
func ClusterSpeakers(ctx context.Context) error {
    // 1. Load all stored embeddings (only ones that were stored)
    embeddings := storage.ListAllEmbeddings(ctx)
    
    // 2. Run DBSCAN clustering
    clusters := clusterEmbeddings(embeddings, eps=0.15, minSamples=2)
    
    // 3. Compute centroids for each cluster
    for _, cluster := range clusters {
        centroid := computeWeightedCentroid(cluster.Embeddings)
        speakerID := createOrUpdateCentroidSpeaker(centroid, cluster)
        
        // 4. Update embeddings
        for _, embedding := range cluster.Embeddings {
            embedding.SpeakerID = &speakerID
            storage.UpdateSpeakerEmbedding(ctx, embedding)
        }
        
        // 5. Update segments (denormalized - need to update both)
        // Find all segments with these embedding IDs
        for _, embedding := range cluster.Embeddings {
            segments := storage.GetSegmentsBySpeakerEmbedding(ctx, embedding.ID)
            for _, segment := range segments {
                segment.SpeakerID = &speakerID
                storage.UpdateSegment(ctx, segment)
            }
        }
    }
    
    // 6. Handle singletons (noise/outliers)
    for _, singleton := range singletons {
        speakerID := createSpeakerForSingleton(singleton)
        singleton.SpeakerID = &speakerID
        storage.UpdateSpeakerEmbedding(ctx, singleton)
        
        // Update segments
        segments := storage.GetSegmentsBySpeakerEmbedding(ctx, singleton.ID)
        for _, segment := range segments {
            segment.SpeakerID = &speakerID
            storage.UpdateSegment(ctx, segment)
        }
    }
    
    return nil
}
```

## Query Implications

### Fast Queries (Using SpeakerID)

```go
// Get all segments for a speaker - FAST (no join needed)
segments := storage.GetSegmentsBySpeaker(ctx, speakerID)
```

**Query:** `SELECT * FROM segments WHERE speaker_id = ?`

### Tracking Queries (Using SpeakerEmbeddingID)

```go
// Get all segments that used a specific embedding
segments := storage.GetSegmentsBySpeakerEmbedding(ctx, embeddingID)
```

**Query:** `SELECT * FROM segments WHERE speaker_embedding_id = ?`

### Segments Without Stored Embeddings

```go
// Find segments that don't have stored embeddings (for analysis)
segments := storage.GetSegmentsWithoutEmbeddings(ctx)
```

**Query:** `SELECT * FROM segments WHERE speaker_embedding_id IS NULL`

## Benefits of Denormalized Approach

1. **Cost Control**: Only store embeddings that meet criteria
2. **Policy Flexibility**: Easy to change storage policy without schema changes
3. **Fast Queries**: Direct SpeakerID access (no joins)
4. **Backward Compatibility**: Can still track which embeddings were used
5. **Gradual Migration**: Can start storing all, then tighten policy later

## Trade-offs

### Pros
- ✅ Cost savings (fewer embeddings stored)
- ✅ Fast queries (no joins)
- ✅ Policy flexibility
- ✅ Can optimize storage based on value

### Cons
- ❌ Denormalization (SpeakerID stored in both places)
- ❌ Need to update both fields during clustering
- ❌ Some segments won't have embeddings (can't re-cluster those)
- ❌ More complex export logic (storage decision)

## Migration Path

1. **Phase 1**: Implement denormalized schema (both fields)
2. **Phase 2**: Start with "store all" policy (backward compatible)
3. **Phase 3**: Gradually tighten policy based on metrics
4. **Phase 4**: Monitor cost savings and clustering quality

## Impact on Current Beads

### Beads That Need Updates

1. **hai-hql** (Schema): 
   - Change Segment schema to have both SpeakerEmbeddingID (optional) and SpeakerID (optional)
   - Keep SpeakerEmbedding schema as-is

2. **hai-mi6** (Export):
   - Add embedding storage policy logic
   - Update export to conditionally store embeddings
   - Set both SpeakerEmbeddingID and SpeakerID on segments

3. **hai-2e8** (Clustering):
   - Update clustering to update both SpeakerEmbedding.SpeakerID and Segment.SpeakerID
   - Handle segments without stored embeddings

4. **hai-ckn** (Matching):
   - Can use SpeakerID directly (simpler queries)
   - Still support SpeakerEmbeddingID queries for tracking

5. **hai-qhp** (Speaker name mapping):
   - Can use Segment.SpeakerID directly (simpler)
   - Fall back to SpeakerEmbeddingID if SpeakerID is NULL

## Recommended Policy (Initial)

```go
var DefaultEmbeddingStoragePolicy = EmbeddingStoragePolicy{
    MinDurationSeconds:  30.0,  // Store if speaker spoke >= 30 seconds
    NoveltyThreshold:    0.80,  // Store if similarity < 0.80 (novel speaker)
    AlwaysStoreOnMatch:  true,  // Always store if match found (for centroid updates)
    AlwaysStoreFirst:    true,  // Always store first occurrence in recording
}
```

This ensures:
- Long conversations are stored (high value)
- Novel speakers are stored (important for discovery)
- Matched speakers are stored (for centroid updates)
- First occurrences are stored (important for initial clustering)

## Cost Analysis (100k users)

**Assumptions:**
- Average 2 hours of audio per user per day
- Average 3 speakers per hour
- Average 5 segments per speaker per hour
- Embedding size: 256 floats = 1KB per embedding

**Current approach (store all):**
- Embeddings per day: 100k users × 2 hours × 3 speakers = 600k embeddings
- Storage per day: 600k × 1KB = 600MB
- Storage per year: 600MB × 365 = 219GB

**With selective storage (estimate 50% reduction):**
- Embeddings per day: 300k (50% stored)
- Storage per day: 300MB
- Storage per year: 110GB

**Savings:** ~109GB/year (50% reduction)

Plus:
- Reduced indexing costs
- Faster clustering (fewer embeddings to process)
- Lower query costs

