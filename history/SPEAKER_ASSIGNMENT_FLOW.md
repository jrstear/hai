# Speaker Assignment Flow: Embeddings → Segments

## Current State

**Segment Schema:**
```go
type Segment struct {
    ID              int64     // Auto-increment or generated ID
    SpeakerID       string    // Global speaker ID (references Speaker.ID)
    RecordingID     string    // Recording ID
    LocalSpeakerID  *string   // Original SPEAKER_00, SPEAKER_01, etc.
    StartTime       float64   // Relative to recording start
    EndTime         float64   // Relative to recording end
    Duration        float64   // Duration in seconds
    CreatedAt       time.Time
}
```

**Current Export Flow:**
1. Extract embeddings from diarization result
2. Match embeddings to existing speakers (cosine similarity)
3. If match found: use existing Speaker.ID
4. If no match: create new Speaker with that embedding
5. Assign Speaker.ID to segments

## New Clustering Approach

### The Question

**When a diary is loaded, what speaker should be assigned to segments?**

### Answer: Use Embedding as Temporary Speaker

**During Export (before clustering):**
1. Store all embeddings in `SpeakerEmbedding` table (SpeakerID = NULL)
2. For immediate segment assignment, create a temporary `Speaker` record:
   - Use the embedding directly as the speaker's embedding (temporary centroid)
   - This allows segments to reference a Speaker.ID immediately
   - Speaker is marked as "temporary" or "pre-clustered"
3. Assign this temporary Speaker.ID to segments

**During Clustering:**
1. Cluster all embeddings (including temporary ones)
2. Compute centroid for each cluster
3. Create/update canonical `Speaker` records (centroids)
4. Update `SpeakerEmbedding` records: set SpeakerID to centroid
5. Update `Segment` records: replace temporary SpeakerID with centroid SpeakerID

### Alternative: Defer Assignment

**Option: Leave segments unassigned until clustering**
- Set Segment.SpeakerID = NULL initially
- After clustering, assign centroid SpeakerID to segments
- **Problem**: Segments need SpeakerID for queries, so this breaks existing functionality

### Recommended Approach: Temporary Speakers

**Phase 1: Export (hai-mi6)**
```go
// For each embedding in diarization result
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
    
    // 2. Try to match against existing centroids (for immediate use)
    matches := storage.FindSimilarSpeakers(ctx, embedding, threshold=0.85, limit=1, onlyCentroids=true)
    
    var speakerID string
    if len(matches) > 0 {
        // Use existing centroid
        speakerID = matches[0].Speaker.ID
    } else {
        // Create temporary speaker (using embedding as temporary centroid)
        speakerID = generateSpeakerID()
        tempSpeaker := &Speaker{
            ID:        speakerID,
            Embedding: embedding, // Use embedding directly as temporary centroid
            FirstSeen: now,
            LastSeen:  now,
        }
        storage.CreateSpeaker(ctx, tempSpeaker)
    }
    
    // 3. Assign to segments
    speakerMap[localSpeakerID] = speakerID
}
```

**Phase 2: Clustering (hai-t0b)**
```go
// After clustering, update segments
for _, cluster := range clusters {
    centroidSpeakerID := cluster.CentroidSpeaker.ID
    
    // Update all embeddings in cluster
    for _, embedding := range cluster.Embeddings {
        embedding.SpeakerID = &centroidSpeakerID
        storage.UpdateSpeakerEmbedding(ctx, embedding)
    }
    
    // Update all segments that reference temporary speakers in this cluster
    for _, embedding := range cluster.Embeddings {
        // Find all segments for this recording with this local speaker
        segments := storage.GetSegmentsByRecordingAndLocalSpeaker(ctx, embedding.RecordingID, embedding.LocalSpeakerID)
        for _, segment := range segments {
            // Replace temporary speaker ID with centroid speaker ID
            segment.SpeakerID = centroidSpeakerID
            storage.UpdateSegment(ctx, segment)
        }
    }
    
    // Delete temporary speakers (or mark as merged)
    for _, embedding := range cluster.Embeddings {
        if embedding.SpeakerID != nil && embedding.SpeakerID != &centroidSpeakerID {
            // This was a temporary speaker, delete it
            storage.DeleteSpeaker(ctx, *embedding.SpeakerID)
        }
    }
}
```

## Segment Schema Changes

**No changes needed!** The Segment schema already has:
- `SpeakerID` (string) - references Speaker.ID
- This works for both temporary and centroid speakers

## Summary

1. **During export**: Create temporary Speaker records (one per embedding) and assign to segments
2. **During clustering**: Create centroid Speakers, update segments to reference centroids
3. **Segment schema**: No changes needed - SpeakerID field works for both temporary and centroid speakers
4. **Assignment timing**: 
   - Immediate: Temporary speaker assigned during export
   - Final: Centroid speaker assigned during clustering

## Open Questions

1. **How to identify temporary speakers?**
   - Option A: Add `IsTemporary` flag to Speaker (but we removed IsCentroid for similar reasons)
   - Option B: Check if Speaker.ID is referenced by any SpeakerEmbedding with SpeakerID = NULL
   - Option C: Don't mark them - just replace during clustering

2. **What if clustering fails or is delayed?**
   - Segments still work with temporary speakers
   - Queries still work
   - Just less optimal (more speakers than necessary)

3. **Should we update segments immediately or batch?**
   - Immediate: Update as we cluster
   - Batch: Update all segments after clustering completes
   - Recommendation: Batch (simpler, atomic)

