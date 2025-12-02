# Speaker Name Mapping Algorithm Design

## Overview

Map lifelog speaker names (e.g., "You", "Jon Stearley", "Unknown") to global speaker IDs (`spkr_xxxxx`) by matching lifelog blockquotes with diarization segments using time-based overlap.

## Current State

### Data Structures

**LifelogBlockquote:**
- `StartTime` / `EndTime`: Absolute UTC timestamps (already converted)
- `SpeakerName`: Human-readable name ("You", "Unknown", "Jon Stearley")
- `SpeakerID`: Optional, to be populated by this algorithm

**Segment:**
- `StartTime` / `EndTime`: Float64 seconds, **relative to recording start**
- `SpeakerID`: Global speaker ID (`spkr_xxxxx`)
- `RecordingID`: Which recording this belongs to

**Recording:**
- `StartTime`: Absolute UTC timestamp (when recording started)
- `ID`: Format `rec_YYYY_MM_DD_HH`

### Key Challenge: Time Alignment

- **Blockquotes**: Absolute UTC timestamps
- **Segments**: Relative to recording start (need to add recording start time to get absolute UTC)

## Algorithm Design

### Phase 1: Find Overlapping Segments

For each lifelog blockquote:

1. **Query recordings that overlap with blockquote time range:**
   ```go
   recordings, err := storage.ListRecordings(ctx, &blockquote.StartTime, &blockquote.EndTime)
   ```

2. **For each overlapping recording:**
   - Calculate segment absolute times:
     ```go
     segmentStartUTC := recording.StartTime.Add(time.Duration(segment.StartTime) * time.Second)
     segmentEndUTC := recording.StartTime.Add(time.Duration(segment.EndTime) * time.Second)
     ```
   - Or query segments directly:
     ```go
     // Convert blockquote times to relative times for this recording
     blockquoteStartRel := blockquote.StartTime.Sub(recording.StartTime).Seconds()
     blockquoteEndRel := blockquote.EndTime.Sub(recording.StartTime).Seconds()
     
     segments, err := storage.GetSegmentsByTimeRange(
         ctx,
         recording.ID,
         blockquoteStartRel,
         blockquoteEndRel,
     )
     ```

3. **Calculate overlap for each segment:**
   ```go
   func calculateOverlap(blockquoteStart, blockquoteEnd, segmentStart, segmentEnd time.Time) (overlapDuration time.Duration, overlapPercentage float64) {
       // Find intersection
       overlapStart := max(blockquoteStart, segmentStart)
       overlapEnd := min(blockquoteEnd, segmentEnd)
       
       if overlapStart.After(overlapEnd) {
           return 0, 0 // No overlap
       }
       
       overlapDuration = overlapEnd.Sub(overlapStart)
       blockquoteDuration := blockquoteEnd.Sub(blockquoteStart)
       segmentDuration := segmentEnd.Sub(segmentStart)
       
       // Overlap percentage: intersection / union (Jaccard similarity)
       // Or: intersection / min(blockquote, segment) for stricter matching
       overlapPercentage = float64(overlapDuration) / float64(min(blockquoteDuration, segmentDuration))
       
       return overlapDuration, overlapPercentage
   }
   ```

### Phase 2: Match Selection Strategy

**Option A: Highest Overlap (Recommended)**
- Find segment with highest overlap percentage
- Require minimum threshold (e.g., > 50%)
- If multiple segments from same speaker, use highest overlap
- If segments from different speakers, use highest overlap (may indicate misalignment)

**Option B: Majority Vote**
- Find all overlapping segments
- Group by speaker ID
- Use speaker ID with most total overlap time
- Require minimum threshold

**Option C: Weighted by Overlap**
- Find all overlapping segments
- Weight each speaker ID by overlap percentage
- Use speaker ID with highest weighted score

**Recommendation: Option A (Highest Overlap)**
- Simpler to implement
- More intuitive (best match wins)
- Handles edge cases naturally

### Phase 3: Edge Cases

1. **No Overlapping Segments:**
   - Leave `SpeakerID` as `NULL`
   - Log warning for debugging

2. **Overlap Below Threshold:**
   - Leave `SpeakerID` as `NULL`
   - Log warning with overlap percentage

3. **Multiple Segments, Same Speaker:**
   - Use highest overlap segment
   - All segments from same speaker, so result is unambiguous

4. **Multiple Segments, Different Speakers:**
   - Use highest overlap segment
   - Log warning if overlap difference is small (may indicate ambiguity)
   - Consider: If second-best is close (e.g., within 10%), mark as ambiguous?

5. **Blockquote Spans Multiple Recordings:**
   - Process each recording separately
   - Use segment with highest overlap across all recordings
   - Update `RecordingID` field to indicate which recording matched

6. **Special Speaker Names:**
   - **"You"**: Treat normally (map to speaker ID)
   - **"Unknown"**: Treat normally (map to speaker ID if match found)
   - **Empty/blank**: Leave unmapped

### Phase 4: Update Strategy

**When to Run:**
- Option 1: During lifelog export (automatic)
- Option 2: Separate batch job (manual/periodic)
- Option 3: On-demand via API/CLI

**Skip Logic:**
- Check if `SpeakerID` is already set
- If set, skip (unless `--reprocess` flag)
- Allows incremental updates

**Batch Processing:**
- Process all blockquotes for a lifelog
- Process all lifelogs for a date range
- Use bulk update operations

## Algorithm Pseudocode

```go
func MapSpeakerNames(ctx context.Context, storage storage.Storage, blockquotes []*LifelogBlockquote) error {
    const minOverlapThreshold = 0.5 // 50% overlap required
    
    for _, blockquote := range blockquotes {
        // Skip if already mapped
        if blockquote.SpeakerID != nil {
            continue
        }
        
        // Find overlapping recordings
        recordings, err := storage.ListRecordings(ctx, &blockquote.StartTime, &blockquote.EndTime)
        if err != nil {
            return err
        }
        
        var bestMatch *Segment
        var bestOverlap float64
        
        // Check each recording
        for _, recording := range recordings {
            // Convert blockquote times to relative times for this recording
            blockquoteStartRel := blockquote.StartTime.Sub(recording.StartTime).Seconds()
            blockquoteEndRel := blockquote.EndTime.Sub(recording.StartTime).Seconds()
            
            // Query segments in time range
            segments, err := storage.GetSegmentsByTimeRange(
                ctx,
                recording.ID,
                blockquoteStartRel,
                blockquoteEndRel,
            )
            if err != nil {
                continue // Skip this recording
            }
            
            // Calculate overlap for each segment
            for _, segment := range segments {
                // Convert segment times to absolute UTC
                segmentStartUTC := recording.StartTime.Add(time.Duration(segment.StartTime) * time.Second)
                segmentEndUTC := recording.StartTime.Add(time.Duration(segment.EndTime) * time.Second)
                
                // Calculate overlap
                overlapDuration, overlapPercentage := calculateOverlap(
                    blockquote.StartTime,
                    blockquote.EndTime,
                    segmentStartUTC,
                    segmentEndUTC,
                )
                
                // Track best match
                if overlapPercentage > bestOverlap {
                    bestOverlap = overlapPercentage
                    bestMatch = segment
                }
            }
        }
        
        // Apply match if above threshold
        if bestMatch != nil && bestOverlap >= minOverlapThreshold {
            blockquote.SpeakerID = &bestMatch.SpeakerID
            blockquote.RecordingID = &bestMatch.RecordingID
            
            // Update in storage
            if err := storage.UpdateLifelogBlockquote(ctx, blockquote); err != nil {
                log.Printf("Failed to update blockquote %s: %v", blockquote.ID, err)
            }
        } else {
            log.Printf("No match found for blockquote %s (best overlap: %.2f%%)", blockquote.ID, bestOverlap*100)
        }
    }
    
    return nil
}
```

## Overlap Calculation Formula

**Jaccard Similarity (Intersection over Union):**
```
overlap = intersection(blockquote, segment)
union = union(blockquote, segment)
overlapPercentage = overlap / union
```

**Stricter Matching (Intersection over Minimum):**
```
overlap = intersection(blockquote, segment)
minDuration = min(blockquote.duration, segment.duration)
overlapPercentage = overlap / minDuration
```

**Recommendation: Use "Intersection over Minimum"**
- More strict (requires good alignment)
- Prevents small segments from matching large blockquotes
- Better for speaker identification accuracy

## Threshold Selection

**Minimum Overlap Threshold: 50%**

Rationale:
- Too low (< 30%): Risk of false matches
- Too high (> 80%): May miss valid matches due to timing differences
- 50%: Balanced, allows for some timing variance

**Considerations:**
- May need tuning based on real data
- Could be configurable per use case
- Could use different thresholds for different speaker names (e.g., stricter for "Unknown")

## Performance Considerations

1. **Query Optimization:**
   - Use `ListRecordings` with time range filter (indexed)
   - Use `GetSegmentsByTimeRange` per recording (indexed)
   - Avoid loading all segments and filtering in memory

2. **Batch Processing:**
   - Process blockquotes in batches (e.g., per lifelog, per day)
   - Use bulk update operations
   - Cache recording lookups

3. **Incremental Updates:**
   - Skip already-mapped blockquotes
   - Process only new/unmapped blockquotes
   - Support reprocessing with flag

## Implementation Location

**Option 1: Go Backend (Recommended for Initial Implementation)**
- Add to `onboard/internal/export2elastic/` or new `onboard/internal/mapping/` package
- Can run during export or as separate CLI tool
- Easy to test and debug
- Can be called from onboarding server

**Option 2: Kibana App (Future Enhancement)**
- Visual interface for reviewing matches
- Manual correction of ambiguous matches
- Interactive exploration of overlaps
- Requires Kibana setup (separate bead)

**Recommendation:**
- Start with Go implementation (Option 1)
- Add Kibana app later for visualization and manual correction (Option 2)

## Testing Strategy

1. **Unit Tests:**
   - Overlap calculation function
   - Edge cases (no overlap, exact match, partial overlap)
   - Threshold logic

2. **Integration Tests:**
   - Test with real data samples
   - Verify correct speaker ID assignment
   - Test edge cases (multiple recordings, ambiguous matches)

3. **Validation:**
   - Compare mapped results with manual inspection
   - Track mapping accuracy metrics
   - Log ambiguous cases for review

## Future Enhancements

1. **Confidence Scores:**
   - Store overlap percentage as confidence metric
   - Use for UI display (high confidence vs. low confidence)

2. **Manual Correction:**
   - Allow users to manually correct mappings
   - Store correction history
   - Learn from corrections

3. **Multi-Speaker Blockquotes:**
   - Handle blockquotes that span multiple speakers
   - Split blockquote or mark as multi-speaker

4. **Transcript Similarity:**
   - Use transcript text similarity as additional signal
   - Combine with time overlap for better accuracy

5. **Learning from Patterns:**
   - Track which speaker names map to which speaker IDs
   - Use historical mappings to improve future matches









