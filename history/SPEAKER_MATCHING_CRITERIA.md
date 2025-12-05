# Speaker Matching Criteria: "No Match Found"

## Current Matching Logic

**In `onboard/internal/export2elastic/types.go`:**

```go
// Find similar speakers using kNN search
matches, err := e.storage.FindSimilarSpeakers(
    ctx,
    embedding,
    storage.SimilarityThreshold,  // 0.85
    1, // Only need the best match
)

var speakerID string
if len(matches) > 0 && matches[0].Similarity >= storage.SimilarityThreshold {
    // Found a match - use existing speaker
    speakerID = matches[0].Speaker.ID
} else {
    // No match found - create new speaker
    speakerID = generateSpeakerID()
    // ... create speaker ...
}
```

## "No Match Found" Criteria

**A match is NOT found when:**
1. `len(matches) == 0` - No speakers found in search (e.g., first diary, no speakers exist yet)
2. `matches[0].Similarity < 0.85` - Best match is below threshold (not similar enough)

**A match IS found when:**
- `len(matches) > 0` AND `matches[0].Similarity >= 0.85`

## With Clustering Approach

### Phase 1: Before Clustering (First Diary)

**Scenario:** No speakers exist yet
- `FindSimilarSpeakers` returns empty results
- `len(matches) == 0` → **No match found**
- **Action:** Create new Speaker (single-embedding speaker)

### Phase 2: After Some Clustering (Subsequent Diaries)

**Scenario:** Centroids exist from previous clustering

**Option A: Match against centroids only (recommended)**
```go
matches := storage.FindSimilarSpeakers(
    ctx,
    embedding,
    threshold=0.85,
    limit=1,
    onlyCentroids=true, // NEW: Only search centroids
)
```

**Criteria:**
- If `len(matches) > 0` AND `similarity >= 0.85`: **Match found** → use centroid
- If `len(matches) == 0` OR `similarity < 0.85`: **No match found** → create new single-embedding speaker

**Why only centroids?**
- Faster: fewer speakers to search
- More accurate: centroids are better representations
- Avoids matching against single-embedding speakers that will be merged later

**Option B: Match against all speakers (current behavior)**
```go
matches := storage.FindSimilarSpeakers(
    ctx,
    embedding,
    threshold=0.85,
    limit=1,
    // No onlyCentroids flag - searches all speakers
)
```

**Criteria:**
- If `len(matches) > 0` AND `similarity >= 0.85`: **Match found** → use existing speaker
- If `len(matches) == 0` OR `similarity < 0.85`: **No match found** → create new speaker

**Issues:**
- May match against single-embedding speakers that will later be merged
- Less efficient (searches more speakers)
- May create duplicate speakers if clustering hasn't run yet

## Recommended Approach

**Match against centroids only (after hai-ckn is implemented):**

```go
// Try to match against centroids (if any exist)
matches := storage.FindSimilarSpeakers(
    ctx,
    embedding,
    storage.SimilarityThreshold, // 0.85
    1,
    onlyCentroids=true, // Only search centroids
)

var speakerID string
if len(matches) > 0 && matches[0].Similarity >= storage.SimilarityThreshold {
    // Match found: use existing centroid
    speakerID = matches[0].Speaker.ID
    
    // Update SpeakerEmbedding to point to this centroid
    speakerEmbedding.SpeakerID = &speakerID
    storage.UpdateSpeakerEmbedding(ctx, speakerEmbedding)
} else {
    // No match found: create new single-embedding speaker
    speakerID = generateSpeakerID()
    speaker := &Speaker{
        ID:        speakerID,
        Embedding: embedding, // Use embedding directly
        FirstSeen: now,
        LastSeen:  now,
    }
    storage.CreateSpeaker(ctx, speaker)
    
    // SpeakerEmbedding.SpeakerID stays NULL (will be set during clustering)
}
```

## Edge Cases

### 1. First Diary (No Centroids Exist)

**Scenario:** No speakers in database yet
- `FindSimilarSpeakers` with `onlyCentroids=true` returns empty
- `len(matches) == 0` → **No match found**
- **Action:** Create new single-embedding speaker

### 2. Similarity Just Below Threshold (e.g., 0.84)

**Scenario:** Best match has similarity 0.84 (below 0.85 threshold)
- `len(matches) > 0` BUT `similarity < 0.85` → **No match found**
- **Action:** Create new single-embedding speaker
- **Later:** Clustering may merge this with the 0.84 match if they're in the same cluster

### 3. Multiple Matches Above Threshold

**Scenario:** Multiple centroids match (e.g., 0.92, 0.88, 0.86)
- `FindSimilarSpeakers` with `limit=1` returns best match (0.92)
- `similarity >= 0.85` → **Match found**
- **Action:** Use best match (0.92)

### 4. No Centroids Yet (Before First Clustering)

**Scenario:** Speakers exist but clustering hasn't run yet
- All speakers are single-embedding (not centroids)
- `FindSimilarSpeakers` with `onlyCentroids=true` returns empty
- `len(matches) == 0` → **No match found**
- **Action:** Create new single-embedding speaker
- **Later:** Clustering will merge all single-embedding speakers into centroids

## Summary

**"No match found" when:**
1. No speakers exist (first diary)
2. No centroids exist (before first clustering)
3. Best match similarity < 0.85 (below threshold)

**"Match found" when:**
- At least one centroid exists AND
- Best match similarity >= 0.85

**Action on "no match found":**
- Create new single-embedding Speaker
- Assign to segments immediately
- Will be merged into centroid during clustering

**Action on "match found":**
- Use existing centroid Speaker
- Assign to segments immediately
- Update SpeakerEmbedding.SpeakerID to point to centroid











