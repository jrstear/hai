# Voice Terminology & Design Considerations

## Terminology: "Speakers" vs "Voices"

### User Consideration
The term "speakers" implies actual people, but what we're storing are **voice signatures/embeddings**. The term "voices" would be more accurate.

### Impact
If we rename, it affects:
- Database table names: `speakers` → `voices`
- Column names: `speaker_id` → `voice_id`
- Variable names throughout codebase
- Documentation

### Decision Needed
- **Option A**: Rename before implementation (cleaner)
- **Option B**: Use "voices" terminology from the start in new code
- **Option C**: Plan for migration later (keep current "speakers" for now)

**Recommendation**: Option B - Use "voices" terminology in schema design and implementation going forward. Update existing references as we implement.

## Voice-to-Contact Mapping Strategy

### Problem
A single contact (person) may have multiple distinct voice signatures:
- Different recording conditions
- Voice changes over time
- Different emotional states
- Background noise variations

### Options

#### Option A: 1:Many Mapping (Contact → Multiple Voices)
**Structure**:
```sql
CREATE TABLE contact_voices (
  contact_id TEXT,
  voice_id TEXT,
  priority INTEGER,  -- Order: 1 = best/first to try
  match_frequency INTEGER,  -- How often this voice matches
  created_at TIMESTAMP
);
```

**Pros**:
- Handles voice variations naturally
- Can try multiple voices for matching

**Cons**:
- Computation latency (check multiple embeddings)
- Wasted time/energy on failed matches
- More complex matching logic

#### Option B: 1:1 Mapping (Contact → Single Voice)
**Structure**:
```sql
ALTER TABLE voices ADD COLUMN contact_id TEXT;
-- One voice per contact
```

**Pros**:
- Simple, fast lookup
- Low latency matching

**Cons**:
- May not handle voice variations well
- Might miss matches if voice has changed

### User's Thoughts
- Order voices by "best" (most frequent matches)
- But concerned about computation latency and wasted time
- Might need to default to 1:1 and optimize later

### Decision Needed
Before contact association implementation, decide:
1. Initial approach (1:1 vs 1:many)
2. How to handle voice variations
3. Matching strategy (try all voices? or pick best?)

**Recommendation**: Start with 1:1 for simplicity, add 1:many support later if needed.

## Voice Clustering and Pruning

### Problem
The voices table may grow large with mostly unused entries:
- Every diarization run may create new voices
- Many voices may be rare/unused
- Storage and matching overhead

### Solution: Clustering and Pruning

**Approach**:
1. **Cluster similar voices** using cosine similarity
2. **Find tight clusters** (high similarity within cluster)
3. **Take centroid** (representative voice for cluster)
4. **Prune redundant/unused voices**

### Implementation Considerations

#### Clustering Algorithm
- Hierarchical clustering with cosine distance
- DBSCAN with cosine distance
- K-means (if we know number of clusters)
- Approximate nearest neighbor (for large datasets)

#### When to Run
- **Periodic**: Scheduled job (e.g., weekly)
- **On-demand**: When voices table exceeds threshold
- **Continuous**: Background process, incremental updates

#### Thresholds
- **Tight cluster**: Cosine similarity > 0.9 within cluster?
- **Prune unused**: Voices with < N segments? < M hours?
- **Merge threshold**: When to merge two voices?

#### Pruning Strategy
```sql
-- Example: Find voices to prune
SELECT voice_id, COUNT(*) as segment_count
FROM segments
GROUP BY voice_id
HAVING COUNT(*) < 10;  -- Less than 10 segments
```

#### Merging Voices
When merging two voices:
1. Update all segment references: `voice_id = old → new`
2. Recompute embedding (average or keep best)
3. Delete old voice record
4. Update contact associations

### User's Concerns
- Prevent voices table from getting "large and mostly unused"
- Cosine similarity should "point the way" for clustering
- Need to find "tight clusters" and take "centroid"

### Decision Needed
1. Clustering approach (algorithm, thresholds)
2. Pruning criteria (unused time, segment count)
3. When to run (periodic vs on-demand)
4. How to merge while preserving data integrity

**Recommendation**: Design clustering/pruning as future optimization (Priority 2), implement basic voice storage first, then add clustering when voices table grows.

## Related Beads Issues

- [Rename consideration] - Terminology change from "speakers" to "voices"
- [Mapping strategy] - 1:1 vs 1:many contact-to-voice mapping
- [Clustering/pruning] - Voice clustering and table pruning strategy

## Next Steps

1. Decide on terminology ("voices" vs "speakers")
2. Decide on mapping strategy (1:1 vs 1:many)
3. Plan clustering/pruning as future optimization
4. Update schema design based on decisions
5. Implement basic voice storage and matching
6. Add clustering/pruning later when needed

