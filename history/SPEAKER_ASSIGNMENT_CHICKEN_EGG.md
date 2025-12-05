# Speaker Assignment: Chicken-and-Egg Problem

## The Problem

**Chicken-and-egg issue:**
1. First diary: No speakers exist yet, so we can't match against centroids
2. Segments need SpeakerID for queries (e.g., "find all segments for speaker X")
3. But we don't know the final speaker until clustering
4. How do we assign speakers to segments immediately?

## Current Approach (Proposed)

**During Export:**
- Create a Speaker record for each embedding (using embedding as speaker's embedding)
- Assign this Speaker.ID to segments
- These are "single-embedding speakers" (not centroids yet)

**During Clustering:**
- Cluster embeddings
- Create centroid Speakers
- Update segments: single-embedding SpeakerID → centroid SpeakerID

**Issues:**
- When do we create these speakers? Always? Only when no match?
- First diary: We'd create speakers for every embedding
- This works, but creates many speakers that will later be merged

## Alternative: Point Segments at SpeakerEmbedding

**Schema Change:**
```go
type Segment struct {
    // ... other fields ...
    SpeakerEmbeddingID string // References SpeakerEmbedding.ID instead of Speaker.ID
    // OR
    SpeakerID *string // Make nullable, point to SpeakerEmbedding.ID initially
}
```

**Flow:**
1. During export: Point segments at SpeakerEmbedding.ID
2. During clustering: Set SpeakerEmbedding.SpeakerID to centroid
3. Update segments: SpeakerEmbeddingID → use SpeakerEmbedding.SpeakerID

**Issues:**
- Segments don't have SpeakerID until clustering
- Queries like "find all segments for speaker X" won't work until clustering
- Need to join through SpeakerEmbedding to get SpeakerID

## Better Approach: Always Create Speakers

**Key insight:** There's no such thing as "temporary" speakers. Every embedding gets a Speaker record immediately.

**Flow:**

### During Export (First Diary or Any Diary)

```go
for localSpeakerID, embedding := range result.SpeakerEmbeddings {
    // 1. Store embedding in SpeakerEmbedding table
    speakerEmbedding := &SpeakerEmbedding{
        ID:            generateEmbeddingID(),
        RecordingID:   recordingID,
        LocalSpeakerID: localSpeakerID,
        Embedding:     embedding,
        DurationSeconds: calculateDuration(localSpeakerID, result.Segments),
        SpeakerID:     nil, // Not clustered yet
    }
    storage.CreateSpeakerEmbedding(ctx, speakerEmbedding)
    
    // 2. Try to match against existing centroids (if any exist)
    var speakerID string
    matches := storage.FindSimilarSpeakers(ctx, embedding, threshold=0.85, limit=1, onlyCentroids=true)
    
    if len(matches) > 0 {
        // Match found: use existing centroid
        speakerID = matches[0].Speaker.ID
        
        // Update SpeakerEmbedding to point to this centroid
        speakerEmbedding.SpeakerID = &speakerID
        storage.UpdateSpeakerEmbedding(ctx, speakerEmbedding)
    } else {
        // No match: create new Speaker (single-embedding speaker)
        // This is NOT temporary - it's a real speaker, just not a centroid yet
        speakerID = generateSpeakerID()
        speaker := &Speaker{
            ID:        speakerID,
            Embedding: embedding, // Use embedding directly (single-embedding speaker)
            FirstSeen: now,
            LastSeen:  now,
        }
        storage.CreateSpeaker(ctx, speaker)
        
        // SpeakerEmbedding.SpeakerID stays NULL (will be set during clustering)
    }
    
    // 3. Assign Speaker.ID to segments
    speakerMap[localSpeakerID] = speakerID
}
```

### During Clustering

```go
// After clustering, we have:
// - Clusters of embeddings (same person)
// - Centroids (computed from clusters)

for _, cluster := range clusters {
    // Create or update centroid Speaker
    centroidSpeakerID := createOrUpdateCentroidSpeaker(cluster)
    
    // Update all embeddings in cluster
    for _, embedding := range cluster.Embeddings {
        embedding.SpeakerID = &centroidSpeakerID
        storage.UpdateSpeakerEmbedding(ctx, embedding)
    }
    
    // Update all segments that reference single-embedding speakers in this cluster
    for _, embedding := range cluster.Embeddings {
        // Find segments for this recording/local speaker
        segments := storage.GetSegmentsByRecordingAndLocalSpeaker(ctx, embedding.RecordingID, embedding.LocalSpeakerID)
        for _, segment := range segments {
            // If segment's SpeakerID is a single-embedding speaker in this cluster, update it
            if isSingleEmbeddingSpeaker(segment.SpeakerID, cluster) {
                segment.SpeakerID = centroidSpeakerID
                storage.UpdateSegment(ctx, segment)
            }
        }
    }
    
    // Optionally: Delete single-embedding speakers that were merged into centroid
    // (or keep them for history/audit trail)
}
```

## Why This Works

1. **First diary:** Creates speakers for every embedding (single-embedding speakers)
2. **Subsequent diaries:** Tries to match against centroids first, creates new speakers if no match
3. **Clustering:** Merges single-embedding speakers into centroids, updates segments
4. **Queries always work:** Segments always have a valid SpeakerID

## The "Temporary" Misconception

**There are no "temporary" speakers.** Every speaker is real:
- **Single-embedding speakers:** Real speakers, just not centroids yet
- **Centroid speakers:** Real speakers, computed from clusters

The distinction is:
- Single-embedding: One embedding = one speaker
- Centroid: Multiple embeddings = one speaker (computed centroid)

## When Speakers Are Created

**Always during export:**
- If match found: Use existing centroid
- If no match: Create new single-embedding speaker

**No conditional logic needed** - we always create speakers if no match exists.

## Segment Schema: No Changes Needed

The current schema works:
- `SpeakerID` (string) - references Speaker.ID
- Works for both single-embedding and centroid speakers
- No need to make it nullable or point at SpeakerEmbedding

## Summary

**The solution:**
1. Always create Speaker records during export (if no match found)
2. These are "single-embedding speakers" (not temporary, just not centroids)
3. Clustering merges them into centroids and updates segments
4. No schema changes needed
5. Queries always work (segments always have SpeakerID)

**Key insight:** The "chicken-and-egg" problem is solved by always creating speakers. There's no need for "temporary" speakers or pointing segments at embeddings - we just create real speakers immediately, and clustering refines them later.











