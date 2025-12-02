package storage

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

// generateSpeakerID generates a new speaker ID in the format spkr_xxxxx
func generateSpeakerID() string {
	id := uuid.New().String()
	// Use first 8 characters of UUID for shorter ID
	return fmt.Sprintf("spkr_%s", id[:8])
}

// cosineDistance computes the cosine distance between two embeddings
// Returns distance in range [0, 2] where 0 = identical, 2 = opposite
// Cosine distance = 1 - cosine similarity
func cosineDistance(a, b []float32) float64 {
	if len(a) != len(b) {
		return 2.0 // Maximum distance for mismatched dimensions
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 2.0 // Maximum distance for zero vectors
	}

	similarity := dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
	return 1.0 - similarity // Convert similarity to distance
}

// cluster represents a group of similar embeddings
type cluster struct {
	Embeddings []*SpeakerEmbedding
}

// dbscan performs DBSCAN clustering on embeddings
// eps: maximum distance for points to be considered neighbors (0.15 for cosine distance)
// minSamples: minimum number of points required to form a cluster
// progress: optional callback function for reporting progress (can be nil)
// Returns: clusters (groups of embeddings) and singletons (noise/outliers)
func dbscan(embeddings []*SpeakerEmbedding, eps float64, minSamples int, progress ProgressFunc) ([]cluster, []*SpeakerEmbedding) {
	if len(embeddings) == 0 {
		return nil, nil
	}

	visited := make([]bool, len(embeddings))
	clustered := make([]bool, len(embeddings))
	var clusters []cluster
	var singletons []*SpeakerEmbedding

	// Find neighbors for each point
	for i := range embeddings {
		if visited[i] {
			continue
		}
		visited[i] = true

		neighbors := findNeighbors(embeddings, i, eps)

		if len(neighbors) < minSamples {
			// Point is noise/outlier (singleton)
			singletons = append(singletons, embeddings[i])
			if progress != nil {
				progress("Found singleton")
			}
			continue
		}

		// Start a new cluster
		cluster := cluster{
			Embeddings: []*SpeakerEmbedding{embeddings[i]},
		}
		clustered[i] = true

		// Expand cluster
		seedSet := neighbors
		for j := 0; j < len(seedSet); j++ {
			neighborIdx := seedSet[j]
			if !visited[neighborIdx] {
				visited[neighborIdx] = true
				neighborNeighbors := findNeighbors(embeddings, neighborIdx, eps)
				if len(neighborNeighbors) >= minSamples {
					// Add neighbors to seed set
					seedSet = append(seedSet, neighborNeighbors...)
				}
			}
			if !clustered[neighborIdx] {
				clustered[neighborIdx] = true
				cluster.Embeddings = append(cluster.Embeddings, embeddings[neighborIdx])
			}
		}

		clusters = append(clusters, cluster)
		if progress != nil {
			progress(fmt.Sprintf("Found cluster with %d embeddings", len(cluster.Embeddings)))
		}
	}

	return clusters, singletons
}

// findNeighbors finds all neighbors of a point within eps distance
func findNeighbors(embeddings []*SpeakerEmbedding, idx int, eps float64) []int {
	var neighbors []int
	point := embeddings[idx].Embedding

	for i, emb := range embeddings {
		if i == idx {
			continue
		}
		dist := cosineDistance(point, emb.Embedding)
		if dist <= eps {
			neighbors = append(neighbors, i)
		}
	}

	return neighbors
}

// ComputeWeightedCentroid computes a duration-weighted centroid from a slice of SpeakerEmbedding.
// The centroid is computed as: centroid[i] = sum(embedding[i] * duration) / sum(durations)
//
// This ensures that embeddings from recordings where the speaker spoke more (longer duration)
// contribute more heavily to the centroid, as they're more reliable/representative.
//
// Parameters:
//   - embeddings: Slice of SpeakerEmbedding to compute centroid from
//
// Returns:
//   - The computed centroid as a []float32 (256 dimensions), or nil if embeddings is empty
//
// Example:
//
//	embeddings := []*SpeakerEmbedding{
//	    {Embedding: []float32{0.1, 0.2, ...}, DurationSeconds: 300.0}, // 5 minutes
//	    {Embedding: []float32{0.15, 0.25, ...}, DurationSeconds: 60.0}, // 1 minute
//	}
//	centroid := ComputeWeightedCentroid(embeddings)
//	// The first embedding contributes 5x more weight than the second
func ComputeWeightedCentroid(embeddings []*SpeakerEmbedding) []float32 {
	if len(embeddings) == 0 {
		return nil
	}

	// Get dimension from first embedding
	dim := len(embeddings[0].Embedding)
	if dim == 0 {
		return nil
	}

	// Initialize centroid and total weight
	centroid := make([]float32, dim)
	totalWeight := 0.0

	// Compute weighted sum: sum(embedding[i] * duration) for each dimension
	for _, emb := range embeddings {
		if len(emb.Embedding) != dim {
			// Skip embeddings with mismatched dimensions
			continue
		}

		weight := emb.DurationSeconds
		if weight <= 0 {
			// Skip embeddings with zero or negative duration
			continue
		}

		totalWeight += weight

		// Add weighted embedding to centroid
		for i, v := range emb.Embedding {
			centroid[i] += float32(weight) * v
		}
	}

	// Normalize by total weight: divide by sum(durations)
	if totalWeight > 0 {
		for i := range centroid {
			centroid[i] /= float32(totalWeight)
		}
	} else {
		// If all weights were zero or negative, return nil
		return nil
	}

	return centroid
}

// ProgressFunc is a function type for reporting clustering progress
// Called during DBSCAN to report clusters and singletons as they are found
type ProgressFunc func(message string)

// ClusterSpeakersConfig holds configuration for clustering
type ClusterSpeakersConfig struct {
	// Eps is the maximum distance for points to be considered neighbors (cosine distance)
	// Default: 0.15 (corresponds to similarity >= 0.85)
	Eps float64

	// MinSamples is the minimum number of points required to form a cluster
	// Default: 2
	MinSamples int

	// Progress is an optional callback function for reporting progress during clustering
	// If nil, no progress messages are printed
	Progress ProgressFunc
}

// DefaultClusterSpeakersConfig returns default clustering configuration
func DefaultClusterSpeakersConfig() ClusterSpeakersConfig {
	return ClusterSpeakersConfig{
		Eps:        0.15, // Cosine distance threshold (similarity >= 0.85)
		MinSamples: 2,    // Need at least 2 points to form a cluster
	}
}

// ClusterSpeakers performs incremental DBSCAN clustering on unclustered embeddings.
// It first tries to match unclustered embeddings to existing centroid speakers, then
// clusters only the unmatched embeddings. This avoids creating duplicate speakers.
//
// Process:
// 1. Load all stored embeddings
// 2. Separate unclustered embeddings (SpeakerID is NULL)
// 3. For each unclustered embedding:
//    - Try to match against existing centroid speakers
//    - If match found: assign to that speaker (update embedding and segments)
//    - If no match: add to list for clustering
// 4. Run DBSCAN clustering only on unmatched embeddings
// 5. For each cluster:
//    - Compute duration-weighted centroid
//    - Create new Speaker (centroid)
//    - Update SpeakerEmbedding.SpeakerID for all embeddings in cluster
//    - Update Segment.SpeakerID for all segments with those embeddings
// 6. Handle singletons (noise/outliers):
//    - Create Speaker for each singleton
//    - Update SpeakerEmbedding.SpeakerID
//    - Update Segment.SpeakerID
//
// Note: Segments without stored embeddings (SpeakerEmbeddingID is NULL) won't be
// updated during clustering. They'll only get SpeakerID if a match is found during export.
//
// Returns the clustering results with counts of embeddings processed, clusters found, etc.
func ClusterSpeakers(ctx context.Context, storage Storage, config ClusterSpeakersConfig) (*ClusterSpeakersResult, error) {
	// 1. Load all stored embeddings
	allEmbeddings, err := storage.ListAllEmbeddings(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load embeddings: %w", err)
	}

	result := &ClusterSpeakersResult{
		EmbeddingsProcessed: len(allEmbeddings),
	}

	if len(allEmbeddings) == 0 {
		// No embeddings to cluster
		return result, nil
	}

	// 2. Separate unclustered embeddings (SpeakerID is NULL)
	var unclusteredEmbeddings []*SpeakerEmbedding
	for _, emb := range allEmbeddings {
		if emb.SpeakerID == nil {
			unclusteredEmbeddings = append(unclusteredEmbeddings, emb)
		}
	}

	if len(unclusteredEmbeddings) == 0 {
		// All embeddings are already clustered (matched during export or previous clustering)
		if config.Progress != nil {
			config.Progress(fmt.Sprintf("All %d embeddings are already clustered (matched during export or previous clustering)", len(allEmbeddings)))
		}
		return result, nil
	}

	// 3. Get count of existing speakers for progress message
	existingSpeakers, err := storage.ListSpeakers(ctx, nil)
	existingSpeakerCount := 0
	if err == nil {
		existingSpeakerCount = len(existingSpeakers)
	}

	// 4. Try to match unclustered embeddings to existing centroids
	var embeddingsToCluster []*SpeakerEmbedding
	matchedCount := 0
	now := time.Now().UTC()

	for _, embedding := range unclusteredEmbeddings {
		// Validate embedding
		if err := ValidateEmbedding(embedding.Embedding); err != nil {
			if err == ErrZeroMagnitudeEmbedding {
				// Skip zero-magnitude embeddings
				continue
			}
			return nil, fmt.Errorf("invalid embedding %s: %w", embedding.ID, err)
		}

		// Try to match against existing centroids
		// Get top 5 matches to see best similarity scores (for debugging)
		matches, err := storage.FindSimilarSpeakers(
			ctx,
			embedding.Embedding,
			0.0, // Don't filter by threshold in query - filter in code instead
			5,   // Get top 5 to see best matches
			true, // onlyCentroids: only match against centroid speakers
		)
		if err != nil {
			return nil, fmt.Errorf("failed to find similar speakers for embedding %s: %w", embedding.ID, err)
		}

		// Find the best match above threshold
		var bestMatch *SpeakerMatch
		var bestScoreBelowThreshold float64 = 0.0
		for i := range matches {
			if matches[i].Similarity >= SimilarityThreshold {
				if bestMatch == nil || matches[i].Similarity > bestMatch.Similarity {
					bestMatch = &matches[i]
				}
			} else if matches[i].Similarity > bestScoreBelowThreshold {
				bestScoreBelowThreshold = matches[i].Similarity
			}
		}

		// Debug: log if we have matches but none above threshold
		if config.Progress != nil && len(matches) > 0 && bestMatch == nil {
			config.Progress(fmt.Sprintf("  Embedding %s: best similarity=%.4f (below threshold %.2f)", embedding.ID, bestScoreBelowThreshold, SimilarityThreshold))
		}

		if bestMatch != nil {
			// Found a match - assign to existing speaker
			speakerID := bestMatch.Speaker.ID
			embedding.SpeakerID = &speakerID

			// Update embedding
			if err := storage.UpdateSpeakerEmbedding(ctx, embedding); err != nil {
				return nil, fmt.Errorf("failed to update embedding %s: %w", embedding.ID, err)
			}

			// Update segments
			segments, err := storage.GetSegmentsBySpeakerEmbedding(ctx, embedding.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get segments for embedding %s: %w", embedding.ID, err)
			}

			for _, segment := range segments {
				segment.SpeakerID = &speakerID
				if err := storage.UpdateSegment(ctx, segment); err != nil {
					return nil, fmt.Errorf("failed to update segment %d: %w", segment.ID, err)
				}
			}

			// Update speaker's last_seen timestamp
			update := &Speaker{
				ID:       speakerID,
				LastSeen: now,
			}
			if err := storage.UpdateSpeaker(ctx, update); err != nil {
				return nil, fmt.Errorf("failed to update speaker %s: %w", speakerID, err)
			}

			matchedCount++
		} else {
			// No match found - add to list for clustering
			embeddingsToCluster = append(embeddingsToCluster, embedding)
		}
	}

	if config.Progress != nil {
		config.Progress(fmt.Sprintf("Matched %d embeddings to %d existing speakers, %d remaining for clustering", matchedCount, existingSpeakerCount, len(embeddingsToCluster)))
	}

	if len(embeddingsToCluster) == 0 {
		// All unclustered embeddings were matched to existing speakers
		return result, nil
	}

	// 5. Run DBSCAN clustering only on unmatched embeddings
	clusters, singletons := dbscan(embeddingsToCluster, config.Eps, config.MinSamples, config.Progress)

	// 6. Process clusters
	result.ClustersFound = len(clusters)
	result.EmbeddingsMatched = matchedCount
	for _, cluster := range clusters {
		// Compute duration-weighted centroid
		centroid := ComputeWeightedCentroid(cluster.Embeddings)
		if centroid == nil {
			// Skip clusters with invalid centroids
			continue
		}

		// Find first and last seen times
		var firstSeen, lastSeen time.Time
		if len(cluster.Embeddings) > 0 {
			firstSeen = cluster.Embeddings[0].CreatedAt
			lastSeen = cluster.Embeddings[0].CreatedAt
			for _, emb := range cluster.Embeddings {
				if emb.CreatedAt.Before(firstSeen) {
					firstSeen = emb.CreatedAt
				}
				if emb.CreatedAt.After(lastSeen) {
					lastSeen = emb.CreatedAt
				}
			}
		}

		// Create or update speaker (centroid)
		// For now, always create new speaker - re-clustering logic can be added later
		speakerID := generateSpeakerID()
		speaker := &Speaker{
			ID:        speakerID,
			Embedding: centroid,
			FirstSeen: firstSeen,
			LastSeen:  lastSeen,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		if err := storage.CreateSpeaker(ctx, speaker); err != nil {
			// If speaker already exists, try to update it
			if err == ErrDuplicateKey {
				// Update existing speaker
				if err := storage.UpdateSpeaker(ctx, speaker); err != nil {
					return nil, fmt.Errorf("failed to update speaker %s: %w", speakerID, err)
				}
			} else {
				return nil, fmt.Errorf("failed to create speaker: %w", err)
			}
		} else {
			result.SpeakersCreated++
		}

		// Update embeddings and segments
		for _, embedding := range cluster.Embeddings {
			// Update embedding
			embedding.SpeakerID = &speakerID
			if err := storage.UpdateSpeakerEmbedding(ctx, embedding); err != nil {
				return nil, fmt.Errorf("failed to update embedding %s: %w", embedding.ID, err)
			}

			// Update segments with this embedding
			segments, err := storage.GetSegmentsBySpeakerEmbedding(ctx, embedding.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get segments for embedding %s: %w", embedding.ID, err)
			}

			for _, segment := range segments {
				segment.SpeakerID = &speakerID
				if err := storage.UpdateSegment(ctx, segment); err != nil {
					return nil, fmt.Errorf("failed to update segment %d: %w", segment.ID, err)
				}
			}
		}
	}

	// 4. Handle singletons (noise/outliers)
	result.SingletonsFound = len(singletons)
	for _, singleton := range singletons {
		// Create speaker for singleton (single-point cluster)
		speakerID := generateSpeakerID()
		speaker := &Speaker{
			ID:        speakerID,
			Embedding: singleton.Embedding, // Single point = its own centroid
			FirstSeen: singleton.CreatedAt,
			LastSeen:  singleton.CreatedAt,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		if err := storage.CreateSpeaker(ctx, speaker); err != nil {
			if err == ErrDuplicateKey {
				if err := storage.UpdateSpeaker(ctx, speaker); err != nil {
					return nil, fmt.Errorf("failed to update singleton speaker %s: %w", speakerID, err)
				}
			} else {
				return nil, fmt.Errorf("failed to create singleton speaker: %w", err)
			}
		} else {
			result.SpeakersCreated++
		}

		// Update embedding
		singleton.SpeakerID = &speakerID
		if err := storage.UpdateSpeakerEmbedding(ctx, singleton); err != nil {
			return nil, fmt.Errorf("failed to update singleton embedding %s: %w", singleton.ID, err)
		}

		// Update segments
		segments, err := storage.GetSegmentsBySpeakerEmbedding(ctx, singleton.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get segments for singleton embedding %s: %w", singleton.ID, err)
		}

		for _, segment := range segments {
			segment.SpeakerID = &speakerID
			if err := storage.UpdateSegment(ctx, segment); err != nil {
				return nil, fmt.Errorf("failed to update singleton segment %d: %w", segment.ID, err)
			}
		}
	}

	return result, nil
}

// ReclusterAllSpeakers performs a full re-clustering of all stored embeddings.
// This is useful for periodic re-clustering (e.g., monthly) to improve cluster quality
// and handle cases where clusters should be merged or split.
//
// Process:
// 1. Load ALL embeddings (including already clustered)
// 2. Run DBSCAN clustering on all embeddings
// 3. For each cluster:
//    - Compute duration-weighted centroid
//    - Try to match cluster to existing speaker by centroid similarity
//    - If match found: update existing speaker (recompute centroid, update timestamps)
//    - If no match: create new speaker
//    - Update SpeakerEmbedding.SpeakerID for all embeddings in cluster
//    - Update Segment.SpeakerID for all segments with those embeddings
// 4. Handle singletons (noise/outliers):
//    - Try to match singleton to existing speaker
//    - If match found: assign to existing speaker
//    - If no match: create new speaker
//    - Update embedding and segments
//
// Note: This will reassign embeddings to different speakers if clusters change.
// Use this periodically (e.g., monthly) rather than on every run.
func ReclusterAllSpeakers(ctx context.Context, storage Storage, config ClusterSpeakersConfig) (*ClusterSpeakersResult, error) {
	// 1. Load ALL embeddings (including already clustered)
	allEmbeddings, err := storage.ListAllEmbeddings(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load embeddings: %w", err)
	}

	result := &ClusterSpeakersResult{
		EmbeddingsProcessed: len(allEmbeddings),
	}

	if len(allEmbeddings) == 0 {
		return result, nil
	}

	if config.Progress != nil {
		config.Progress(fmt.Sprintf("Re-clustering %d embeddings...", len(allEmbeddings)))
	}

	// 2. Run DBSCAN clustering on all embeddings
	clusters, singletons := dbscan(allEmbeddings, config.Eps, config.MinSamples, config.Progress)

	// 3. Process clusters
	result.ClustersFound = len(clusters)
	now := time.Now().UTC()

	for _, cluster := range clusters {
		// Compute duration-weighted centroid
		centroid := ComputeWeightedCentroid(cluster.Embeddings)
		if centroid == nil {
			continue
		}

		// Find first and last seen times
		var firstSeen, lastSeen time.Time
		if len(cluster.Embeddings) > 0 {
			firstSeen = cluster.Embeddings[0].CreatedAt
			lastSeen = cluster.Embeddings[0].CreatedAt
			for _, emb := range cluster.Embeddings {
				if emb.CreatedAt.Before(firstSeen) {
					firstSeen = emb.CreatedAt
				}
				if emb.CreatedAt.After(lastSeen) {
					lastSeen = emb.CreatedAt
				}
			}
		}

		// Try to match cluster to existing speaker by centroid similarity
		matches, err := storage.FindSimilarSpeakers(
			ctx,
			centroid,
			SimilarityThreshold,
			1, // Only need the best match
			true, // onlyCentroids
		)
		if err != nil {
			return nil, fmt.Errorf("failed to find similar speakers for cluster centroid: %w", err)
		}

		var speakerID string
		if len(matches) > 0 && matches[0].Similarity >= SimilarityThreshold {
			// Match found - update existing speaker
			speakerID = matches[0].Speaker.ID
			update := &Speaker{
				ID:        speakerID,
				Embedding: centroid, // Update centroid with new weighted average
				FirstSeen: firstSeen,
				LastSeen:  lastSeen,
				UpdatedAt: now,
			}
			if err := storage.UpdateSpeaker(ctx, update); err != nil {
				return nil, fmt.Errorf("failed to update speaker %s: %w", speakerID, err)
			}
		} else {
			// No match - create new speaker
			speakerID = generateSpeakerID()
			speaker := &Speaker{
				ID:        speakerID,
				Embedding: centroid,
				FirstSeen: firstSeen,
				LastSeen:  lastSeen,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := storage.CreateSpeaker(ctx, speaker); err != nil {
				return nil, fmt.Errorf("failed to create speaker: %w", err)
			}
			result.SpeakersCreated++
		}

		// Update embeddings and segments
		for _, embedding := range cluster.Embeddings {
			embedding.SpeakerID = &speakerID
			if err := storage.UpdateSpeakerEmbedding(ctx, embedding); err != nil {
				return nil, fmt.Errorf("failed to update embedding %s: %w", embedding.ID, err)
			}

			segments, err := storage.GetSegmentsBySpeakerEmbedding(ctx, embedding.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get segments for embedding %s: %w", embedding.ID, err)
			}

			for _, segment := range segments {
				segment.SpeakerID = &speakerID
				if err := storage.UpdateSegment(ctx, segment); err != nil {
					return nil, fmt.Errorf("failed to update segment %d: %w", segment.ID, err)
				}
			}
		}
	}

	// 4. Handle singletons
	result.SingletonsFound = len(singletons)
	for _, singleton := range singletons {
		// Try to match singleton to existing speaker
		matches, err := storage.FindSimilarSpeakers(
			ctx,
			singleton.Embedding,
			SimilarityThreshold,
			1,
			true, // onlyCentroids
		)
		if err != nil {
			return nil, fmt.Errorf("failed to find similar speakers for singleton: %w", err)
		}

		var speakerID string
		if len(matches) > 0 && matches[0].Similarity >= SimilarityThreshold {
			// Match found - assign to existing speaker
			speakerID = matches[0].Speaker.ID
			update := &Speaker{
				ID:       speakerID,
				LastSeen: singleton.CreatedAt,
				UpdatedAt: now,
			}
			if err := storage.UpdateSpeaker(ctx, update); err != nil {
				return nil, fmt.Errorf("failed to update speaker %s: %w", speakerID, err)
			}
		} else {
			// No match - create new speaker
			speakerID = generateSpeakerID()
			speaker := &Speaker{
				ID:        speakerID,
				Embedding: singleton.Embedding,
				FirstSeen: singleton.CreatedAt,
				LastSeen:  singleton.CreatedAt,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := storage.CreateSpeaker(ctx, speaker); err != nil {
				return nil, fmt.Errorf("failed to create singleton speaker: %w", err)
			}
			result.SpeakersCreated++
		}

		// Update embedding
		singleton.SpeakerID = &speakerID
		if err := storage.UpdateSpeakerEmbedding(ctx, singleton); err != nil {
			return nil, fmt.Errorf("failed to update singleton embedding %s: %w", singleton.ID, err)
		}

		// Update segments
		segments, err := storage.GetSegmentsBySpeakerEmbedding(ctx, singleton.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get segments for singleton embedding %s: %w", singleton.ID, err)
		}

		for _, segment := range segments {
			segment.SpeakerID = &speakerID
			if err := storage.UpdateSegment(ctx, segment); err != nil {
				return nil, fmt.Errorf("failed to update singleton segment %d: %w", segment.ID, err)
			}
		}
	}

	return result, nil
}

// ClusterSpeakersResult holds the results of clustering
type ClusterSpeakersResult struct {
	EmbeddingsProcessed int
	EmbeddingsMatched   int // Number of embeddings matched to existing speakers
	ClustersFound       int
	SingletonsFound     int
	SpeakersCreated     int
}

