package export2elastic

import (
	"context"
	"fmt"
	"log"
	"time"

	"hai/storage"
)

const (
	// MinOverlapThreshold is the minimum overlap percentage required to match a blockquote to a segment
	// 0.5 = 50% overlap required
	MinOverlapThreshold = 0.5
)

// MapSpeakerNames maps lifelog speaker names to global speaker IDs by matching
// lifelog blockquotes with diarization segments using time-based overlap.
//
// For each blockquote, it:
// 1. Finds overlapping recordings
// 2. Queries segments in those recordings that overlap with the blockquote time range
// 3. Calculates overlap percentage for each segment
// 4. Selects the segment with highest overlap (if >= MinOverlapThreshold)
// 5. Updates the blockquote's SpeakerID and RecordingID fields
// 6. Tracks the best blockquote for each segment (for storing blockquote_id on segments)
//
// Blockquotes that already have a SpeakerID set are skipped unless reprocess is true.
//
// After processing all blockquotes, updates segments with their best matching blockquote_id.
//
// Returns the number of blockquotes matched and any error.
func (e *Exporter) MapSpeakerNames(ctx context.Context, blockquotes []*storage.LifelogBlockquote, reprocess bool) (int, error) {
	if len(blockquotes) == 0 {
		return 0, nil
	}

	stats := &mappingStats{
		total:        len(blockquotes),
		alreadyMapped: 0,
		matched:      0,
		unmatched:    0,
	}

	// Track best blockquote for each segment (segmentID -> {blockquoteID, overlap})
	segmentMatches := make(map[int64]*struct {
		blockquoteID string
		overlap      float64
	})

	for _, blockquote := range blockquotes {
		// Skip if already mapped (unless reprocessing)
		if !reprocess && blockquote.SpeakerID != nil {
			stats.alreadyMapped++
			continue
		}

		// Find best matching segment
		bestMatch, bestOverlap, err := e.findBestMatchingSegment(ctx, blockquote)
		if err != nil {
			log.Printf("Error finding match for blockquote %s: %v", blockquote.ID, err)
			stats.unmatched++
			continue
		}

		// Apply match if above threshold
		if bestMatch != nil && bestOverlap >= MinOverlapThreshold {
			// Use SpeakerID directly (it's already a pointer, can be nil if no match)
			blockquote.SpeakerID = bestMatch.SpeakerID
			blockquote.RecordingID = &bestMatch.RecordingID

			// Update in storage
			if err := e.storage.UpdateLifelogBlockquote(ctx, blockquote); err != nil {
				log.Printf("Failed to update blockquote %s: %v", blockquote.ID, err)
				stats.unmatched++
				continue
			}

			// Track best blockquote for this segment
			segmentID := bestMatch.ID
			if existing, exists := segmentMatches[segmentID]; !exists || bestOverlap > existing.overlap {
				segmentMatches[segmentID] = &struct {
					blockquoteID string
					overlap      float64
				}{blockquote.ID, bestOverlap}
			}

			stats.matched++
			speakerIDStr := "nil"
			if bestMatch.SpeakerID != nil {
				speakerIDStr = *bestMatch.SpeakerID
			}
			log.Printf("Mapped blockquote %s (speaker: %s) to speaker ID %s (overlap: %.2f%%)",
				blockquote.ID, blockquote.SpeakerName, speakerIDStr, bestOverlap*100)
		} else {
			stats.unmatched++
			if bestMatch != nil {
				log.Printf("No match found for blockquote %s (best overlap: %.2f%%, below threshold)",
					blockquote.ID, bestOverlap*100)
			} else {
				log.Printf("No match found for blockquote %s (no overlapping segments)", blockquote.ID)
			}
		}
	}

	// Update segments with their best matching blockquote_id
	segmentsUpdated := 0
	for segmentID, match := range segmentMatches {
		// Get the segment
		segment, err := e.storage.GetSegment(ctx, segmentID)
		if err != nil {
			log.Printf("Warning: failed to get segment %d for blockquote update: %v", segmentID, err)
			continue
		}

		// Update blockquote_id
		segment.BlockquoteID = &match.blockquoteID
		if err := e.storage.UpdateSegment(ctx, segment); err != nil {
			log.Printf("Warning: failed to update segment %d with blockquote_id %s: %v", segmentID, match.blockquoteID, err)
			continue
		}

		segmentsUpdated++
	}

	// Log summary
	log.Printf("Speaker name mapping complete: %d total, %d already mapped, %d matched, %d unmatched, %d segments updated with blockquote_id",
		stats.total, stats.alreadyMapped, stats.matched, stats.unmatched, segmentsUpdated)

	return stats.matched, nil
}

// findBestMatchingSegment finds the segment with the highest overlap percentage for a given blockquote.
// Returns the best matching segment, its overlap percentage, and any error.
func (e *Exporter) findBestMatchingSegment(ctx context.Context, blockquote *storage.LifelogBlockquote) (*storage.Segment, float64, error) {
	// Find recordings that overlap with blockquote time range
	recordings, err := e.storage.ListRecordings(ctx, &blockquote.StartTime, &blockquote.EndTime)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list recordings: %w", err)
	}

	// If no recordings found, try a wider time range (blockquote might span recording boundaries)
	// Expand by 1 hour on each side to catch edge cases
	if len(recordings) == 0 {
		expandedStart := blockquote.StartTime.Add(-1 * time.Hour)
		expandedEnd := blockquote.EndTime.Add(1 * time.Hour)
		recordings, err = e.storage.ListRecordings(ctx, &expandedStart, &expandedEnd)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to list recordings (expanded range): %w", err)
		}
	}

	if len(recordings) == 0 {
		return nil, 0, nil // No overlapping recordings
	}

	var bestMatch *storage.Segment
	var bestOverlap float64

	// Check each recording
	for _, recording := range recordings {
		// Convert blockquote times to relative times for this recording
		blockquoteStartRel := blockquote.StartTime.Sub(recording.StartTime).Seconds()
		blockquoteEndRel := blockquote.EndTime.Sub(recording.StartTime).Seconds()

		// Query segments in time range
		segments, err := e.storage.GetSegmentsByTimeRange(
			ctx,
			recording.ID,
			blockquoteStartRel,
			blockquoteEndRel,
		)
		if err != nil {
			log.Printf("Warning: failed to get segments for recording %s: %v", recording.ID, err)
			continue
		}

		// Calculate overlap for each segment
		for _, segment := range segments {
			// Convert segment times to absolute UTC
			segmentStartUTC := recording.StartTime.Add(time.Duration(segment.StartTime) * time.Second)
			segmentEndUTC := recording.StartTime.Add(time.Duration(segment.EndTime) * time.Second)

			// Calculate overlap
			overlapPercentage := calculateOverlap(
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

	return bestMatch, bestOverlap, nil
}

// calculateOverlap calculates the overlap percentage between a blockquote and a segment.
// Uses "Intersection over Minimum" formula for stricter matching.
// Returns a value between 0.0 and 1.0 (0% to 100% overlap).
func calculateOverlap(blockquoteStart, blockquoteEnd, segmentStart, segmentEnd time.Time) float64 {
	// Find intersection
	overlapStart := maxTime(blockquoteStart, segmentStart)
	overlapEnd := minTime(blockquoteEnd, segmentEnd)

	// No overlap if start is after end
	if overlapStart.After(overlapEnd) || overlapStart.Equal(overlapEnd) {
		return 0.0
	}

	// Calculate durations
	overlapDuration := overlapEnd.Sub(overlapStart)
	blockquoteDuration := blockquoteEnd.Sub(blockquoteStart)
	segmentDuration := segmentEnd.Sub(segmentStart)

	// Use minimum duration for stricter matching (Intersection over Minimum)
	minDuration := blockquoteDuration
	if segmentDuration < minDuration {
		minDuration = segmentDuration
	}

	if minDuration <= 0 {
		return 0.0
	}

	// Calculate overlap percentage
	overlapPercentage := float64(overlapDuration) / float64(minDuration)

	// Clamp to [0.0, 1.0]
	if overlapPercentage > 1.0 {
		overlapPercentage = 1.0
	}
	if overlapPercentage < 0.0 {
		overlapPercentage = 0.0
	}

	return overlapPercentage
}

// maxTime returns the later of two times
func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// minTime returns the earlier of two times
func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

// MapSpeakerNamesForLifelog maps speaker names for all blockquotes in a given lifelog.
// This is a convenience function that fetches blockquotes for a lifelog and then maps them.
func (e *Exporter) MapSpeakerNamesForLifelog(ctx context.Context, lifelogID string, reprocess bool) (int, error) {
	// Fetch all blockquotes for this lifelog
	blockquotes, err := e.storage.GetLifelogBlockquotesByLifelog(ctx, lifelogID)
	if err != nil {
		return 0, fmt.Errorf("failed to get blockquotes for lifelog %s: %w", lifelogID, err)
	}

	return e.MapSpeakerNames(ctx, blockquotes, reprocess)
}

// MapSpeakerNamesForTimeRange maps speaker names for all blockquotes in a given time range.
// This is useful for batch processing blockquotes for a specific date or date range.
func (e *Exporter) MapSpeakerNamesForTimeRange(ctx context.Context, startTime, endTime time.Time, reprocess bool) (int, error) {
	// Fetch all blockquotes in the time range
	blockquotes, err := e.storage.GetLifelogBlockquotesByTimeRange(ctx, startTime, endTime)
	if err != nil {
		return 0, fmt.Errorf("failed to get blockquotes for time range: %w", err)
	}

	return e.MapSpeakerNames(ctx, blockquotes, reprocess)
}

// mappingStats tracks statistics during mapping
type mappingStats struct {
	total        int
	alreadyMapped int
	matched      int
	unmatched    int
}

