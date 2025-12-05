# Speaker Embedding Update Issue

## Current Behavior

When a speaker matches an existing global speaker:

```go
if len(matches) > 0 && matches[0].Similarity >= storage.SimilarityThreshold {
    // Found a match - use existing speaker
    speakerID = matches[0].Speaker.ID
    
    // Update last_seen timestamp
    update := &storage.Speaker{
        ID:       speakerID,
        LastSeen: now,
    }
    if err := e.storage.UpdateSpeaker(ctx, update); err != nil {
        return nil, err
    }
}
```

**The new embedding is discarded.** Only the `LastSeen` timestamp is updated.

## The Problem

1. **First-seen bias**: The embedding from the first recording where we see a speaker is permanently stored
2. **No refinement**: Later recordings with potentially better quality/more data don't improve the embedding
3. **Suboptimal matching**: Early recordings might have:
   - Less audio data (shorter recording)
   - Poorer quality (noisy, distant microphone)
   - Less representative conditions (unusual emotional state, background noise)

### Example Scenario

```
Recording A (Nov 22, 2pm, 5 minutes, noisy):
  SPEAKER_00 → embedding_A (low quality) → Create spkr_abc123

Recording B (Nov 22, 3pm, 1 hour, clear):
  SPEAKER_00 → embedding_B (high quality, 0.92 similarity) → Match! → Discard embedding_B, keep embedding_A

Recording C (Nov 22, 4pm, 30 minutes, clear):
  SPEAKER_00 → embedding_C (high quality, 0.90 similarity) → Match! → Discard embedding_C, keep embedding_A
```

Result: We're stuck with the low-quality embedding from Recording A, even though we have better embeddings available.

## Potential Solutions

### Option 1: Update to Most Recent Embedding

**Simple approach**: Always update to the newest embedding when a match is found.

```go
if len(matches) > 0 && matches[0].Similarity >= storage.SimilarityThreshold {
    speakerID = matches[0].Speaker.ID
    
    // Update embedding to newest
    update := &storage.Speaker{
        ID:        speakerID,
        Embedding: embedding,  // Use new embedding
        LastSeen:  now,
    }
    e.storage.UpdateSpeaker(ctx, update)
}
```

**Pros:**
- Simple to implement
- Always uses most recent embedding
- Adapts to voice changes over time

**Cons:**
- May lose a better embedding if a newer one is worse
- No consideration of recording quality/duration

### Option 2: Weighted Average of Embeddings

**Average approach**: Maintain a running average of all embeddings seen for this speaker.

```go
// Store: embedding, embedding_count, total_embedding_sum
// On match: new_embedding = (old_embedding * count + new_embedding) / (count + 1)
```

**Pros:**
- More stable representation
- Incorporates all recordings
- Less sensitive to single bad recordings

**Cons:**
- More complex (need to track count/sum)
- May dilute distinctive features
- Requires schema changes

### Option 3: Best Quality Embedding

**Quality-based**: Track recording metadata (duration, quality metrics) and keep the "best" embedding.

```go
// Store: embedding, recording_duration, recording_quality_score
// On match: Compare quality, keep better embedding
```

**Pros:**
- Uses highest quality representation
- Considers recording characteristics

**Cons:**
- Need to define "quality" metrics
- More complex
- May not adapt to voice changes

### Option 4: Hybrid: Update if Better Quality

**Smart update**: Update embedding if new one is from a "better" recording (longer duration, higher quality).

```go
existing := matches[0].Speaker
newDuration := result.AudioDuration
existingDuration := getRecordingDuration(existing.FirstSeen)

if newDuration > existingDuration * 1.5 {  // 50% longer
    // Update to new embedding
    update.Embedding = embedding
}
```

**Pros:**
- Balances quality and recency
- Simple heuristic
- Prevents degradation from poor recordings

**Cons:**
- Heuristic may not always be correct
- Need recording duration metadata

### Option 5: Store Multiple Embeddings (1:Many)

**Multiple embeddings**: Store all embeddings for a speaker, use best match.

This relates to `hai-9qz` (speaker-to-contact mapping strategy).

**Pros:**
- Handles voice variations naturally
- Can use best match for each query
- Most flexible

**Cons:**
- More storage
- More complex matching logic
- Computation overhead

## Recommendation

**Start with Option 1 (Update to Most Recent)** for simplicity, then consider Option 4 (Hybrid) if quality becomes an issue.

**Rationale:**
- Simple to implement (one-line change)
- Adapts to voice changes over time
- Better than current "first-seen" bias
- Can refine later if needed

## Implementation

Update `onboard/internal/export2elastic/types.go`:

```go
if len(matches) > 0 && matches[0].Similarity >= storage.SimilarityThreshold {
    // Found a match - use existing speaker
    speakerID = matches[0].Speaker.ID
    
    // Update embedding to newest (refines representation over time)
    update := &storage.Speaker{
        ID:        speakerID,
        Embedding: embedding,  // Update to new embedding
        LastSeen:  now,
    }
    if err := e.storage.UpdateSpeaker(ctx, update); err != nil {
        return nil, err
    }
}
```

## Related Issues

- `hai-9qz`: Speaker-to-contact mapping strategy (1:1 vs 1:many) - relates to Option 5
- Future: Speaker embedding quality metrics
- Future: Speaker embedding clustering/refinement











