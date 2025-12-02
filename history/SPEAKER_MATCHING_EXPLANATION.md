# Speaker Matching: Why Cosine Similarity?

## The Problem

When we diarize a recording, pyannote.audio produces:
- **Local speaker IDs**: `SPEAKER_00`, `SPEAKER_01`, etc. (unique to that recording)
- **Speaker embeddings**: 256-dimensional vectors representing each speaker's voice signature
- **Segments**: Time periods where each speaker spoke

## The Key Insight

**Each recording has its own set of local speaker IDs.** They are NOT shared across recordings.

Example:
- Recording A (Nov 22, 2pm): `SPEAKER_00`, `SPEAKER_01`
- Recording B (Nov 22, 3pm): `SPEAKER_00`, `SPEAKER_01` (different people!)
- Recording C (Nov 22, 4pm): `SPEAKER_00`, `SPEAKER_01` (different again!)

The same person (e.g., "Jon") might be:
- `SPEAKER_00` in Recording A
- `SPEAKER_01` in Recording B
- `SPEAKER_00` in Recording C

## Why We Need Global Speaker IDs

We want to know: **"Is this the same person I've seen before?"**

To answer this, we need:
1. **Global speaker IDs** that persist across recordings (e.g., `spkr_abc123`)
2. **A way to match** local speaker embeddings to global speaker IDs

## How Cosine Similarity Works

When we process a new recording:

1. **Extract embeddings** from diarization (one per local speaker)
2. **For each embedding**, search all existing global speakers using cosine similarity
3. **If similarity ≥ 0.85**: This is the same person → use existing global speaker ID
4. **If similarity < 0.85**: This is a new person → create new global speaker ID

### Example Flow

```
Recording A (Nov 22, 2pm):
  SPEAKER_00 → embedding_A → No match → Create spkr_abc123
  SPEAKER_01 → embedding_B → No match → Create spkr_def456

Recording B (Nov 22, 3pm):
  SPEAKER_00 → embedding_C → No match → Create spkr_ghi789
  SPEAKER_01 → embedding_B → Match! (0.92 similarity) → Use spkr_def456

Recording C (Nov 22, 4pm):
  SPEAKER_00 → embedding_A → Match! (0.88 similarity) → Use spkr_abc123
  SPEAKER_01 → embedding_D → No match → Create spkr_jkl012
```

## Why Embeddings Vary

The same person's embedding can vary slightly between recordings due to:
- **Recording conditions**: Background noise, microphone quality, distance
- **Time**: Voice changes over days/weeks
- **Emotional state**: Stress, excitement, fatigue
- **Context**: Phone call vs. in-person, quiet vs. noisy environment

But they should still be **similar enough** (cosine similarity > 0.85) to match.

## Why Not Just Use Local Speaker IDs?

If we only used local speaker IDs:
- ❌ Can't track the same person across recordings
- ❌ Can't answer "Who did I talk to most this week?"
- ❌ Can't build a contact database
- ❌ Can't map lifelog speaker names to diarization speakers

With global speaker IDs + cosine matching:
- ✅ Track people across time
- ✅ Build speaker statistics
- ✅ Map to contacts
- ✅ Answer cross-recording queries

## The Matching Process

See `onboard/internal/export2elastic/types.go`:

```go
// For each local speaker embedding from the new recording
for localSpeakerID, embedding := range result.SpeakerEmbeddings {
    // Search all existing global speakers
    matches := storage.FindSimilarSpeakers(embedding, threshold=0.85, limit=1)
    
    if matches[0].Similarity >= 0.85 {
        // Same person → use existing global ID
        speakerMap[localSpeakerID] = matches[0].Speaker.ID
    } else {
        // New person → create new global ID
        newID := generateSpeakerID()
        createSpeaker(newID, embedding)
        speakerMap[localSpeakerID] = newID
    }
}
```

## Summary

- **Local speaker IDs** (`SPEAKER_00`) are per-recording and not shared
- **Global speaker IDs** (`spkr_abc123`) persist across recordings
- **Cosine similarity** matches local embeddings to global IDs
- **Same person** in different recordings gets the same global ID (if similarity ≥ 0.85)
- **Different people** get different global IDs

This is why we need cosine similarity: to bridge the gap between per-recording local IDs and cross-recording global identities.

