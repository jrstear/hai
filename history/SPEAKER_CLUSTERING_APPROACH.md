# Speaker Clustering Approach: Store All, Cluster, Use Centroids

## Overview

Store all speaker embeddings from all recordings, periodically cluster them to find groups (same person), compute **duration-weighted centroids** (weighted mean) for each cluster, and match only against centroids for performance.

**Key Design Decisions:**
- **No `IsCentroid` flag**: Speakers table only contains centroids by design - no flag needed
- **Computation in Go**: Elasticsearch is storage only - clustering happens in Go/Python
- **DBSCAN singletons**: Noise/outliers (label = -1) become single-point clusters (their own centroid)
- **Centroid = Duration-Weighted Mean**: Average of all embeddings weighted by seconds spoken (longer recordings = more reliable = higher weight)

## Key Benefits

1. **No data loss**: All embeddings preserved, can re-cluster later
2. **Better representation**: Centroid is more stable than single embedding
3. **Performance**: Match against centroids only (much smaller set)
4. **Flexibility**: Can re-cluster with different algorithms/thresholds
5. **Quality**: Avoids first-seen bias, uses best representation

## Architecture

### Current Schema (Single Embedding per Speaker)

```go
type Speaker struct {
    ID        string    // Global speaker ID
    Embedding []float32 // Single embedding (first-seen or most recent)
    FirstSeen time.Time
    LastSeen  time.Time
    ContactID *string
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### Proposed Schema (All Embeddings + Centroids)

**Option A: Separate Embeddings Table (Recommended)**

```go
// Speaker represents a canonical speaker (cluster centroid)
// Note: All speakers in this table are centroids by design - no flag needed
type Speaker struct {
    ID        string    // Global speaker ID (e.g., spkr_abc123)
    Embedding []float32 // Centroid embedding (canonical representation = mean of cluster)
    FirstSeen time.Time
    LastSeen  time.Time
    ContactID *string
    CreatedAt time.Time
    UpdatedAt time.Time
}

// SpeakerEmbedding stores all embeddings from all recordings
type SpeakerEmbedding struct {
    ID            string    // Unique ID for this embedding
    SpeakerID     *string   // NULL until clustered, then points to centroid speaker
    RecordingID   string    // Which recording this came from
    LocalSpeakerID string   // SPEAKER_00, SPEAKER_01, etc. from that recording
    Embedding     []float32 // The actual embedding vector
    DurationSeconds float64 // Total seconds this speaker spoke in this recording (sum of segment durations)
    CreatedAt     time.Time
}
```

**Option B: Embeddings Array in Speaker**

```go
type Speaker struct {
    ID         string      // Global speaker ID
    Embedding  []float32   // Centroid embedding
    Embeddings [][]float32 // All embeddings (for re-clustering)
    IsCentroid bool
    // ... other fields
}
```

**Recommendation: Option A** - Cleaner separation, easier to query, scales better.

## Workflow

### 1. Initial Export (Store All Embeddings)

When exporting a diarization result:

```go
// For each local speaker embedding
for localSpeakerID, embedding := range result.SpeakerEmbeddings {
    // Calculate total duration this speaker spoke in this recording
    // (sum of all segment durations for this local speaker)
    durationSeconds := 0.0
    for _, seg := range result.Segments {
        if seg.Speaker == localSpeakerID {
            durationSeconds += seg.Duration
        }
    }
    
    // Store embedding (always, even if match found)
    speakerEmbedding := &SpeakerEmbedding{
        ID:            generateEmbeddingID(),
        RecordingID:   recordingID,
        LocalSpeakerID: localSpeakerID,
        Embedding:     embedding,
        DurationSeconds: durationSeconds, // Store duration for weighting
        SpeakerID:     nil, // Not clustered yet
        CreatedAt:     now,
    }
    storage.CreateSpeakerEmbedding(ctx, speakerEmbedding)
    
    // Try to match against centroids (for immediate use)
    matches := storage.FindSimilarSpeakers(ctx, embedding, threshold=0.85, limit=1)
    if len(matches) > 0 {
        // Use existing centroid for segments
        speakerMap[localSpeakerID] = matches[0].Speaker.ID
    } else {
        // Create temporary speaker ID for segments
        // Will be replaced during clustering
        tempID := generateSpeakerID()
        speakerMap[localSpeakerID] = tempID
    }
}
```

### 2. Periodic Clustering

Run clustering job (e.g., daily, weekly, or when N new embeddings added):

```go
func ClusterSpeakers(ctx context.Context) error {
    // 1. Load all unclustered embeddings
    embeddings := storage.ListUnclusteredEmbeddings(ctx)
    
    // 2. Run clustering algorithm (DBSCAN, hierarchical, etc.)
    clusters := clusterEmbeddings(embeddings, threshold=0.85)
    
    // 3. For each cluster:
    for _, cluster := range clusters {
        // Compute weighted centroid (duration-weighted average)
        // Weight each embedding by total seconds that speaker spoke in that recording
        weightedEmbeddings := make([]WeightedEmbedding, len(cluster.Embeddings))
        for i, emb := range cluster.Embeddings {
            weightedEmbeddings[i] = WeightedEmbedding{
                Embedding: emb.Embedding,
                Weight:    emb.DurationSeconds, // Weight by duration
            }
        }
        centroid := computeWeightedCentroid(weightedEmbeddings)
        
        // Create or update speaker (centroid)
        speaker := &Speaker{
            ID:        generateSpeakerID(),
            Embedding: centroid, // Mean of all embeddings in cluster
            FirstSeen: min(cluster.Embeddings, func(e) time.Time { return e.CreatedAt }),
            LastSeen:  max(cluster.Embeddings, func(e) time.Time { return e.CreatedAt }),
        }
        storage.CreateSpeaker(ctx, speaker)
        
        // Update all embeddings in cluster to point to this speaker
        for _, embedding := range cluster.Embeddings {
            embedding.SpeakerID = &speaker.ID
            storage.UpdateSpeakerEmbedding(ctx, embedding)
        }
    }
    
    // 4. Handle singletons (DBSCAN noise/outliers - label = -1)
    // Create speaker (centroid) for each singleton - they become single-point clusters
    for _, singleton := range singletons {
        speaker := &Speaker{
            ID:        generateSpeakerID(),
            Embedding: singleton.Embedding, // Single point = its own centroid
            FirstSeen: singleton.CreatedAt,
            LastSeen:  singleton.CreatedAt,
        }
        storage.CreateSpeaker(ctx, speaker)
        singleton.SpeakerID = &speaker.ID
        storage.UpdateSpeakerEmbedding(ctx, singleton)
    }
}
```

### 3. Matching (Use Centroids Only)

When matching new embeddings:

```go
// Only search against centroid speakers
matches := storage.FindSimilarSpeakers(
    ctx,
    embedding,
    threshold=0.85,
    limit=1,
    onlyCentroids=true, // NEW: Only match against centroids
)
```

### 4. Re-clustering

Periodically re-cluster all embeddings (e.g., monthly):

```go
func ReclusterAllSpeakers(ctx context.Context) error {
    // 1. Load ALL embeddings (including already clustered)
    allEmbeddings := storage.ListAllEmbeddings(ctx)
    
    // 2. Run clustering
    clusters := clusterEmbeddings(allEmbeddings, threshold=0.85)
    
    // 3. Update centroids and speaker assignments
    // (Similar to initial clustering, but update existing speakers)
}
```

## Computation Location

**Elasticsearch is storage only** - clustering computation happens in Go (or Python subprocess).

**Process:**
1. Load all embeddings from Elasticsearch into memory
2. Run clustering algorithm (DBSCAN, hierarchical, etc.) in Go
3. Compute centroids for each cluster
4. Save results back to Elasticsearch:
   - Create/update Speaker records (centroids)
   - Update SpeakerEmbedding records (set SpeakerID)

**Why not in Elasticsearch?**
- Elasticsearch doesn't have built-in clustering algorithms
- kNN search is for similarity search, not clustering
- Clustering requires iterative algorithms (DBSCAN, k-means) that ES doesn't support
- Better to use Go/Python libraries (sklearn, scipy) for clustering

**Performance:**
- Load embeddings: ~1-10 seconds for 10,000 embeddings
- Clustering: ~1-10 seconds (DBSCAN O(n log n) with spatial indexing)
- Save results: ~1-5 seconds
- Total: ~5-25 seconds for 10,000 embeddings (acceptable for background job)

## Clustering Algorithm

### Option 1: DBSCAN (Density-Based)

**Pros:**
- Handles noise/outliers (singletons)
- No need to specify number of clusters
- Good for variable-density clusters

**Cons:**
- Need to tune `eps` (distance threshold) and `minPts` (minimum points)

**How DBSCAN handles outliers:**
- DBSCAN has a `minPts` (min_samples) parameter - minimum points required to form a cluster
- Points that don't meet the density requirement (fewer than `minPts` neighbors within `eps` distance) are labeled as **noise** (label = -1)
- These noise points are **not** automatically clustered - they remain as singletons
- **Our handling**: Create a speaker (centroid) for each singleton - they become single-point clusters

**Example:**
```python
from sklearn.cluster import DBSCAN
from sklearn.metrics.pairwise import cosine_distances

# Convert cosine similarity to distance
distances = 1 - cosine_similarity_matrix
clustering = DBSCAN(eps=0.15, min_samples=2, metric='precomputed')
labels = clustering.fit_predict(distances)

# Labels: [0, 0, 1, 1, -1, 2, 2, 2]
# -1 = noise/outlier (singleton)
# 0, 1, 2 = cluster IDs
```

**Tuning:**
- `eps=0.15` means: points within 0.15 cosine distance are neighbors
- `min_samples=2` means: need at least 2 points to form a cluster
- If `min_samples=1`, every point becomes its own cluster (not useful)
- If `min_samples=3`, need at least 3 points to form a cluster (stricter)

### Option 2: Hierarchical Clustering

**Pros:**
- Can choose number of clusters after the fact
- Produces dendrogram (useful for visualization)
- Deterministic

**Cons:**
- O(n²) or O(n² log n) complexity
- Need to choose linkage method

```python
from scipy.cluster.hierarchy import linkage, fcluster
from scipy.spatial.distance import pdist

# Cosine distance
distances = pdist(embeddings, metric='cosine')
linkage_matrix = linkage(distances, method='average')
labels = fcluster(linkage_matrix, t=0.15, criterion='distance')
```

### Option 3: K-Means (if we know approximate number of speakers)

**Pros:**
- Fast
- Simple

**Cons:**
- Need to know/estimate number of clusters
- Assumes spherical clusters

### Recommendation: DBSCAN

- Handles unknown number of speakers
- Good for variable cluster sizes
- Identifies outliers (singletons)

## Centroid Computation

### Option 1: Weighted Average (Duration-Weighted Mean) ⭐ **RECOMMENDED**

**Key insight**: Embeddings from recordings where the speaker spoke more (longer duration) should be weighted more heavily, as they're more reliable/representative.

**Weighting**: Use total seconds the speaker spoke in that recording (sum of segment durations).

```go
type WeightedEmbedding struct {
    Embedding []float32
    Weight    float64 // Duration in seconds this speaker spoke
}

func computeWeightedCentroid(embeddings []WeightedEmbedding) []float32 {
    if len(embeddings) == 0 {
        return nil
    }
    
    dim := len(embeddings[0].Embedding)
    centroid := make([]float32, dim)
    totalWeight := 0.0
    
    // Weighted sum
    for _, we := range embeddings {
        weight := we.Weight
        totalWeight += weight
        for i, v := range we.Embedding {
            centroid[i] += float32(weight) * v
        }
    }
    
    // Normalize by total weight
    if totalWeight > 0 {
        for i := range centroid {
            centroid[i] /= float32(totalWeight)
        }
    }
    
    return centroid
}
```

**Why this is better:**
- ✅ More reliable embeddings (from longer recordings) contribute more
- ✅ Accounts for quality differences (5-minute vs 1-hour recording)
- ✅ Still smooths out variations (like simple mean)
- ✅ Standard weighted average formula

**Example:**
```
Cluster has 3 embeddings:
- Embedding A: 300 seconds (5 minutes) of speech
- Embedding B: 1800 seconds (30 minutes) of speech  
- Embedding C: 600 seconds (10 minutes) of speech

Total weight: 2700 seconds
Centroid = (A * 300 + B * 1800 + C * 600) / 2700
         = (A * 0.11 + B * 0.67 + C * 0.22)
```

### Option 2: Simple Average (Unweighted Mean)

```go
func computeCentroid(embeddings [][]float32) []float32 {
    centroid := make([]float32, len(embeddings[0]))
    for _, emb := range embeddings {
        for i, v := range emb {
            centroid[i] += v
        }
    }
    for i := range centroid {
        centroid[i] /= float32(len(embeddings))
    }
    return centroid
}
```

**Pros:**
- Simple
- Standard approach

**Cons:**
- Treats all embeddings equally (5-minute recording = 1-hour recording)
- Less accurate representation

### Option 3: Best Representative (Highest Similarity to Others)

```go
func findBestRepresentative(embeddings [][]float32) []float32 {
    bestIdx := 0
    bestScore := 0.0
    
    for i, emb1 := range embeddings {
        score := 0.0
        for j, emb2 := range embeddings {
            if i != j {
                score += cosineSimilarity(emb1, emb2)
            }
        }
        if score > bestScore {
            bestScore = score
            bestIdx = i
        }
    }
    
    return embeddings[bestIdx]
}
```

**Pros:**
- Uses actual embedding (not computed)
- Most similar to others in cluster

**Cons:**
- Doesn't account for duration/quality
- May pick a poor-quality embedding if it happens to be similar

### Recommendation: Weighted Average (Duration-Weighted)

- **Weighted mean** - average weighted by duration (seconds spoken)
- More reliable embeddings contribute more
- Accounts for recording quality differences
- Still smooths out variations
- Formula: `centroid[i] = sum(embedding[i] * duration) / sum(durations)`

## Implementation Steps

### Phase 1: Schema Changes

1. Add `SpeakerEmbedding` type to `storage/schema.go`
2. Add methods to `storage/interface.go`:
   - `CreateSpeakerEmbedding(ctx, embedding)`
   - `ListUnclusteredEmbeddings(ctx)`
   - `ListAllEmbeddings(ctx)`
   - `UpdateSpeakerEmbedding(ctx, embedding)`
   - `FindSimilarSpeakers(ctx, embedding, onlyCentroids bool)`
3. Implement in `storage/elasticsearch.go`

### Phase 2: Store All Embeddings

1. Update `export2elastic/types.go` to always store embeddings
2. Store embedding even when match found
3. Match against centroids for immediate segment assignment

### Phase 3: Clustering Implementation

1. Create `onboard/internal/clustering/` package
2. Implement DBSCAN clustering
3. Implement centroid computation
4. Create CLI tool: `cmd/cluster-speakers/main.go`

### Phase 4: Periodic Clustering

1. Add clustering job to onboarding server (or separate service)
2. Schedule periodic clustering (daily/weekly)
3. Handle re-clustering of all embeddings

### Phase 5: Update Matching

1. Update `FindSimilarSpeakers` to only search centroids
2. Update export logic to use centroid matching

## Elasticsearch Schema

### New Index: `speaker_embeddings`

```json
{
  "mappings": {
    "properties": {
      "id": { "type": "keyword" },
      "speaker_id": { "type": "keyword" },  // NULL until clustered
      "recording_id": { "type": "keyword" },
      "local_speaker_id": { "type": "keyword" },
      "embedding": {
        "type": "dense_vector",
        "dims": 256,
        "index": true,
        "similarity": "cosine"
      },
      "duration_seconds": { "type": "float" },  // For weighted centroid computation
      "created_at": { "type": "date" }
    }
  }
}
```

### Updated Index: `speakers`

**No changes needed** - all speakers in this index are centroids by design.

## Performance Considerations

### Matching Performance

**Before (current):**
- Match against all speakers (grows linearly)
- O(n) where n = number of speakers

**After (centroids only):**
- Match against centroids only (much smaller set)
- O(k) where k = number of unique people (clusters)
- Typically k << n (e.g., 10 people vs 1000 embeddings)

### Clustering Performance

- DBSCAN: O(n log n) with spatial indexing, O(n²) worst case
- For 10,000 embeddings: ~1-10 seconds
- Can run in background, doesn't block exports

### Storage

- Each embedding: 256 floats × 4 bytes = 1KB
- 10,000 embeddings = ~10MB
- Negligible compared to audio/segment data

## Migration Strategy

1. **Backward compatible**: Keep current schema working
2. **Gradual migration**: 
   - Start storing all embeddings alongside current logic
   - Run clustering in background
   - Switch matching to centroids once clusters stable
3. **Re-cluster existing data**: Run clustering on all historical embeddings

## Open Questions

1. **When to cluster?**
   - After N new embeddings? (e.g., every 100)
   - Time-based? (daily, weekly)
   - On-demand? (manual trigger)

2. **Clustering threshold?**
   - Same as matching threshold? (0.85)
   - Stricter? (0.90 for clusters, 0.85 for matching)

3. **Handle singletons?**
   - Create speaker for each? (yes, recommended)
   - Mark as "unclustered" and handle later?

4. **Re-clustering frequency?**
   - Monthly? Weekly?
   - When contact mappings change?

5. **Update existing segments?**
   - When re-clustering changes speaker assignments, update segments?
   - Or keep historical assignments?

## Related Beads

- `hai-9qz`: Speaker-to-contact mapping (1:1 vs 1:many) - relates to clustering strategy
- `hai-126`: Update speaker embeddings when match found - this approach solves this
- Future: Speaker embedding quality metrics
- Future: Clustering visualization/debugging tools

## Recommendation

**This is an excellent approach!** It solves multiple problems:
- ✅ No data loss (all embeddings preserved)
- ✅ Better representation (centroids)
- ✅ Performance (match against centroids only)
- ✅ Flexibility (can re-cluster)
- ✅ Quality (avoids first-seen bias)

**Implementation priority:**
1. Phase 1: Schema changes (store all embeddings)
2. Phase 2: Update export to store all embeddings
3. Phase 3: Basic clustering (DBSCAN)
4. Phase 4: Switch matching to centroids
5. Phase 5: Periodic clustering job

