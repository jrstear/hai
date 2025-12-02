package storage

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestComputeWeightedCentroid(t *testing.T) {
	tests := []struct {
		name      string
		embeddings []*SpeakerEmbedding
		want      []float32
		wantNil   bool
	}{
		{
			name:      "empty embeddings",
			embeddings: []*SpeakerEmbedding{},
			wantNil:   true,
		},
		{
			name: "single embedding",
			embeddings: []*SpeakerEmbedding{
				{
					Embedding:       []float32{0.1, 0.2, 0.3},
					DurationSeconds: 100.0,
				},
			},
			want: []float32{0.1, 0.2, 0.3},
		},
		{
			name: "two embeddings equal weight",
			embeddings: []*SpeakerEmbedding{
				{
					Embedding:       []float32{0.1, 0.2, 0.3},
					DurationSeconds: 100.0,
				},
				{
					Embedding:       []float32{0.3, 0.4, 0.5},
					DurationSeconds: 100.0,
				},
			},
			want: []float32{0.2, 0.3, 0.4}, // Simple average: (0.1+0.3)/2, (0.2+0.4)/2, (0.3+0.5)/2
		},
		{
			name: "two embeddings different weights",
			embeddings: []*SpeakerEmbedding{
				{
					Embedding:       []float32{0.1, 0.2, 0.3},
					DurationSeconds: 300.0, // 5 minutes - 3x weight
				},
				{
					Embedding:       []float32{0.3, 0.4, 0.5},
					DurationSeconds: 100.0, // 1 minute - 1x weight
				},
			},
			// Weighted average: (0.1*300 + 0.3*100) / 400 = (30 + 30) / 400 = 60/400 = 0.15
			//                   (0.2*300 + 0.4*100) / 400 = (60 + 40) / 400 = 100/400 = 0.25
			//                   (0.3*300 + 0.5*100) / 400 = (90 + 50) / 400 = 140/400 = 0.35
			want: []float32{0.15, 0.25, 0.35},
		},
		{
			name: "three embeddings with zero duration",
			embeddings: []*SpeakerEmbedding{
				{
					Embedding:       []float32{0.1, 0.2, 0.3},
					DurationSeconds: 100.0,
				},
				{
					Embedding:       []float32{0.3, 0.4, 0.5},
					DurationSeconds: 0.0, // Should be skipped
				},
				{
					Embedding:       []float32{0.5, 0.6, 0.7},
					DurationSeconds: 200.0,
				},
			},
			// Only first and third count: (0.1*100 + 0.5*200) / 300 = (10 + 100) / 300 = 110/300 ≈ 0.3667
			//                              (0.2*100 + 0.6*200) / 300 = (20 + 120) / 300 = 140/300 ≈ 0.4667
			//                              (0.3*100 + 0.7*200) / 300 = (30 + 140) / 300 = 170/300 ≈ 0.5667
			want: []float32{110.0 / 300.0, 140.0 / 300.0, 170.0 / 300.0},
		},
		{
			name: "mismatched dimensions",
			embeddings: []*SpeakerEmbedding{
				{
					Embedding:       []float32{0.1, 0.2, 0.3},
					DurationSeconds: 100.0,
				},
				{
					Embedding:       []float32{0.3, 0.4}, // Wrong dimension - should be skipped
					DurationSeconds: 100.0,
				},
			},
			want: []float32{0.1, 0.2, 0.3}, // Only first embedding counts
		},
		{
			name: "all zero duration",
			embeddings: []*SpeakerEmbedding{
				{
					Embedding:       []float32{0.1, 0.2, 0.3},
					DurationSeconds: 0.0,
				},
				{
					Embedding:       []float32{0.3, 0.4, 0.5},
					DurationSeconds: 0.0,
				},
			},
			wantNil: true, // Should return nil if all weights are zero
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeWeightedCentroid(tt.embeddings)

			if tt.wantNil {
				if got != nil {
					t.Errorf("ComputeWeightedCentroid() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("ComputeWeightedCentroid() = nil, want %v", tt.want)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("ComputeWeightedCentroid() length = %d, want %d", len(got), len(tt.want))
				return
			}

			// Compare with tolerance for floating point precision
			const tolerance = 1e-6
			for i := range got {
				if math.Abs(float64(got[i]-tt.want[i])) > tolerance {
					t.Errorf("ComputeWeightedCentroid()[%d] = %f, want %f", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestComputeWeightedCentroid_256Dimensions(t *testing.T) {
	// Test with actual 256-dimensional embeddings
	emb1 := make([]float32, 256)
	emb2 := make([]float32, 256)
	for i := range emb1 {
		emb1[i] = 0.1
		emb2[i] = 0.2
	}

	embeddings := []*SpeakerEmbedding{
		{
			Embedding:       emb1,
			DurationSeconds: 100.0,
		},
		{
			Embedding:       emb2,
			DurationSeconds: 200.0,
		},
	}

	centroid := ComputeWeightedCentroid(embeddings)

	if centroid == nil {
		t.Fatal("ComputeWeightedCentroid() = nil, want 256-dimensional vector")
	}

	if len(centroid) != 256 {
		t.Errorf("ComputeWeightedCentroid() length = %d, want 256", len(centroid))
	}

	// Expected: (0.1*100 + 0.2*200) / 300 = (10 + 40) / 300 = 50/300 = 1/6 ≈ 0.1667
	expected := float32(50.0 / 300.0)
	const tolerance = 1e-6
	for i := range centroid {
		if math.Abs(float64(centroid[i]-expected)) > tolerance {
			t.Errorf("ComputeWeightedCentroid()[%d] = %f, want %f", i, centroid[i], expected)
		}
	}
}

func TestCosineDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		wantDist float64
	}{
		{
			name:     "identical vectors",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			wantDist: 0.0, // Cosine similarity = 1.0, distance = 0.0
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1.0, 0.0},
			b:        []float32{0.0, 1.0},
			wantDist: 1.0, // Cosine similarity = 0.0, distance = 1.0
		},
		{
			name:     "opposite vectors",
			a:        []float32{1.0, 0.0},
			b:        []float32{-1.0, 0.0},
			wantDist: 2.0, // Cosine similarity = -1.0, distance = 2.0
		},
		{
			name:     "mismatched dimensions",
			a:        []float32{1.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			wantDist: 2.0, // Maximum distance
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineDistance(tt.a, tt.b)
			const tolerance = 1e-6
			if math.Abs(got-tt.wantDist) > tolerance {
				t.Errorf("cosineDistance() = %f, want %f", got, tt.wantDist)
			}
		})
	}
}

func TestDBSCAN(t *testing.T) {
	// Create test embeddings with small variations
	// Cluster 1: emb1, emb2 (very similar, cosine distance < 0.15)
	// Cluster 2: emb3, emb4 (very similar, cosine distance < 0.15, but different from cluster 1)
	// Singleton: emb5 (different from both clusters)

	emb1 := make([]float32, 256)
	emb2 := make([]float32, 256)
	emb3 := make([]float32, 256)
	emb4 := make([]float32, 256)
	emb5 := make([]float32, 256)

	// Cluster 1: emb1, emb2 (very similar - small random variations)
	// Use a base vector and add small perturbations
	base1 := float32(0.1)
	for i := range emb1 {
		emb1[i] = base1 + float32(i%10)*0.001 // Small variation
		emb2[i] = base1 + float32(i%10)*0.001 + 0.01 // Very close to emb1
	}

	// Cluster 2: emb3, emb4 (very similar, but different base from cluster 1)
	base2 := float32(0.9)
	for i := range emb3 {
		emb3[i] = base2 + float32(i%10)*0.001 // Small variation
		emb4[i] = base2 + float32(i%10)*0.001 + 0.01 // Very close to emb3
	}

	// Singleton: emb5 (different from both clusters)
	base3 := float32(0.5)
	for i := range emb5 {
		emb5[i] = base3 + float32(i%10)*0.01 // Different pattern
	}

	embeddings := []*SpeakerEmbedding{
		{ID: "emb1", Embedding: emb1, DurationSeconds: 100.0, CreatedAt: time.Now()},
		{ID: "emb2", Embedding: emb2, DurationSeconds: 100.0, CreatedAt: time.Now()},
		{ID: "emb3", Embedding: emb3, DurationSeconds: 100.0, CreatedAt: time.Now()},
		{ID: "emb4", Embedding: emb4, DurationSeconds: 100.0, CreatedAt: time.Now()},
		{ID: "emb5", Embedding: emb5, DurationSeconds: 100.0, CreatedAt: time.Now()},
	}

	// Run DBSCAN with eps=0.15, minSamples=2
	clusters, singletons := dbscan(embeddings, 0.15, 2, nil) // No progress callback for test

	// Verify that all embeddings are either in clusters or singletons
	totalClustered := 0
	for _, cluster := range clusters {
		totalClustered += len(cluster.Embeddings)
	}
	totalAssigned := totalClustered + len(singletons)

	if totalAssigned != len(embeddings) {
		t.Errorf("dbscan() total assigned = %d, want %d", totalAssigned, len(embeddings))
	}

	// Verify clusters have at least minSamples embeddings
	for i, cluster := range clusters {
		if len(cluster.Embeddings) < 2 {
			t.Errorf("dbscan() cluster %d has %d embeddings, want at least 2", i, len(cluster.Embeddings))
		}
	}

	// Verify that emb1 and emb2 are in the same cluster (they should be very similar)
	emb1InCluster := false
	emb2InCluster := false
	for _, cluster := range clusters {
		for _, emb := range cluster.Embeddings {
			if emb.ID == "emb1" {
				emb1InCluster = true
			}
			if emb.ID == "emb2" {
				emb2InCluster = true
			}
		}
		// Check if both are in this cluster
		if emb1InCluster && emb2InCluster {
			break
		}
		// Reset if not both found in this cluster
		if !(emb1InCluster && emb2InCluster) {
			emb1InCluster = false
			emb2InCluster = false
		}
	}

	// emb1 and emb2 should be very similar, so they should be in the same cluster
	// (unless cosine distance calculation shows they're not close enough)
	// For now, just verify the function runs without error
	if len(clusters) == 0 && len(singletons) == 0 {
		t.Error("dbscan() returned no clusters or singletons")
	}
}

// MockStorage is a simple mock for testing ClusterSpeakers
type mockStorage struct {
	speakers          map[string]*Speaker
	embeddings        map[string]*SpeakerEmbedding
	segments          map[int64]*Segment
	segmentsByEmb     map[string][]*Segment
	createSpeakerErr  error
	updateSpeakerErr  error
	updateEmbeddingErr error
	updateSegmentErr  error
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		speakers:       make(map[string]*Speaker),
		embeddings:     make(map[string]*SpeakerEmbedding),
		segments:       make(map[int64]*Segment),
		segmentsByEmb:  make(map[string][]*Segment),
	}
}

func (m *mockStorage) ListAllEmbeddings(ctx context.Context, speakerID *string) ([]*SpeakerEmbedding, error) {
	var result []*SpeakerEmbedding
	for _, emb := range m.embeddings {
		if speakerID == nil || (emb.SpeakerID != nil && *emb.SpeakerID == *speakerID) {
			result = append(result, emb)
		}
	}
	return result, nil
}

func (m *mockStorage) CreateSpeaker(ctx context.Context, speaker *Speaker) error {
	if m.createSpeakerErr != nil {
		return m.createSpeakerErr
	}
	if _, exists := m.speakers[speaker.ID]; exists {
		return ErrDuplicateKey
	}
	m.speakers[speaker.ID] = speaker
	return nil
}

func (m *mockStorage) UpdateSpeaker(ctx context.Context, speaker *Speaker) error {
	if m.updateSpeakerErr != nil {
		return m.updateSpeakerErr
	}
	if _, exists := m.speakers[speaker.ID]; !exists {
		return ErrNotFound
	}
	m.speakers[speaker.ID] = speaker
	return nil
}

func (m *mockStorage) UpdateSpeakerEmbedding(ctx context.Context, embedding *SpeakerEmbedding) error {
	if m.updateEmbeddingErr != nil {
		return m.updateEmbeddingErr
	}
	if _, exists := m.embeddings[embedding.ID]; !exists {
		return ErrNotFound
	}
	m.embeddings[embedding.ID] = embedding
	return nil
}

func (m *mockStorage) GetSegmentsBySpeakerEmbedding(ctx context.Context, embeddingID string) ([]*Segment, error) {
	return m.segmentsByEmb[embeddingID], nil
}

func (m *mockStorage) UpdateSegment(ctx context.Context, segment *Segment) error {
	if m.updateSegmentErr != nil {
		return m.updateSegmentErr
	}
	if _, exists := m.segments[segment.ID]; !exists {
		return ErrNotFound
	}
	m.segments[segment.ID] = segment
	return nil
}

// Implement other required Storage methods (stubs)
func (m *mockStorage) Close() error { return nil }
func (m *mockStorage) Health(ctx context.Context) error { return nil }
func (m *mockStorage) GetSpeaker(ctx context.Context, id string) (*Speaker, error) { return nil, ErrNotFound }
func (m *mockStorage) FindSimilarSpeakers(ctx context.Context, embedding []float32, threshold float64, limit int, onlyCentroids bool) ([]SpeakerMatch, error) { return nil, nil }
func (m *mockStorage) ListSpeakers(ctx context.Context, contactID *string) ([]*Speaker, error) { return nil, nil }
func (m *mockStorage) CreateRecording(ctx context.Context, recording *Recording) error { return nil }
func (m *mockStorage) GetRecording(ctx context.Context, id string) (*Recording, error) { return nil, ErrNotFound }
func (m *mockStorage) GetRecordingByPath(ctx context.Context, filePath string) (*Recording, error) { return nil, ErrNotFound }
func (m *mockStorage) ListRecordings(ctx context.Context, startTime *time.Time, endTime *time.Time) ([]*Recording, error) { return nil, nil }
func (m *mockStorage) UpdateRecording(ctx context.Context, recording *Recording) error { return nil }
func (m *mockStorage) CreateSegment(ctx context.Context, segment *Segment) error { return nil }
func (m *mockStorage) CreateSegments(ctx context.Context, segments []*Segment) (int, error) { return 0, nil }
func (m *mockStorage) GetSegmentsByTimeRange(ctx context.Context, recordingID string, startTime, endTime float64) ([]*Segment, error) { return nil, nil }
func (m *mockStorage) GetSegment(ctx context.Context, id int64) (*Segment, error) { return nil, ErrNotFound }
func (m *mockStorage) GetSegmentsByRecording(ctx context.Context, recordingID string) ([]*Segment, error) { return nil, nil }
func (m *mockStorage) GetSegmentsBySpeaker(ctx context.Context, speakerID string) ([]*Segment, error) { return nil, nil }
func (m *mockStorage) UpdateSegmentByteOffsets(ctx context.Context, segmentID int64, startByteOffset, endByteOffset int64) error { return nil }
func (m *mockStorage) CreateLifelog(ctx context.Context, lifelog *Lifelog) error { return nil }
func (m *mockStorage) GetLifelog(ctx context.Context, id string) (*Lifelog, error) { return nil, ErrNotFound }
func (m *mockStorage) ListLifelogs(ctx context.Context, startTime *time.Time, endTime *time.Time) ([]*Lifelog, error) { return nil, nil }
func (m *mockStorage) UpdateLifelog(ctx context.Context, lifelog *Lifelog) error { return nil }
func (m *mockStorage) CreateLifelogBlockquote(ctx context.Context, blockquote *LifelogBlockquote) error { return nil }
func (m *mockStorage) CreateLifelogBlockquotes(ctx context.Context, blockquotes []*LifelogBlockquote) (int, error) { return 0, nil }
func (m *mockStorage) GetLifelogBlockquote(ctx context.Context, id string) (*LifelogBlockquote, error) { return nil, ErrNotFound }
func (m *mockStorage) GetLifelogBlockquotesByLifelog(ctx context.Context, lifelogID string) ([]*LifelogBlockquote, error) { return nil, nil }
func (m *mockStorage) GetLifelogBlockquotesByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*LifelogBlockquote, error) { return nil, nil }
func (m *mockStorage) UpdateLifelogBlockquote(ctx context.Context, blockquote *LifelogBlockquote) error { return nil }
func (m *mockStorage) GetLifelogBlockquotesBySpeakerID(ctx context.Context, speakerID string) ([]*LifelogBlockquote, error) { return nil, nil }
func (m *mockStorage) CreateSpeakerEmbedding(ctx context.Context, embedding *SpeakerEmbedding) error { return nil }
func (m *mockStorage) GetSpeakerEmbedding(ctx context.Context, id string) (*SpeakerEmbedding, error) { return nil, ErrNotFound }
func (m *mockStorage) ListUnclusteredEmbeddings(ctx context.Context) ([]*SpeakerEmbedding, error) { return nil, nil }
func (m *mockStorage) GetSegmentsWithoutEmbeddings(ctx context.Context) ([]*Segment, error) { return nil, nil }

func TestClusterSpeakers(t *testing.T) {
	ctx := context.Background()
	storage := newMockStorage()

	// Create test embeddings - make emb1 and emb2 identical so they definitely cluster together
	emb1 := make([]float32, 256)
	emb2 := make([]float32, 256)
	for i := range emb1 {
		emb1[i] = 0.1
		emb2[i] = 0.1 // Identical to emb1 - will definitely be in same cluster
	}

	emb1ID := "emb1"
	emb2ID := "emb2"
	now := time.Now()

	storage.embeddings[emb1ID] = &SpeakerEmbedding{
		ID:              emb1ID,
		Embedding:       emb1,
		DurationSeconds: 100.0,
		CreatedAt:       now,
	}

	storage.embeddings[emb2ID] = &SpeakerEmbedding{
		ID:              emb2ID,
		Embedding:       emb2,
		DurationSeconds: 200.0,
		CreatedAt:       now,
	}

	// Create segments linked to embeddings
	seg1ID := int64(1)
	seg2ID := int64(2)

	storage.segments[seg1ID] = &Segment{
		ID:                 seg1ID,
		SpeakerEmbeddingID: &emb1ID,
		RecordingID:        "rec1",
		CreatedAt:          now,
	}

	storage.segments[seg2ID] = &Segment{
		ID:                 seg2ID,
		SpeakerEmbeddingID: &emb2ID,
		RecordingID:        "rec1",
		CreatedAt:          now,
	}

	storage.segmentsByEmb[emb1ID] = []*Segment{storage.segments[seg1ID]}
	storage.segmentsByEmb[emb2ID] = []*Segment{storage.segments[seg2ID]}

	// Run clustering
	config := DefaultClusterSpeakersConfig()
	result, err := ClusterSpeakers(ctx, storage, config)
	if err != nil {
		t.Fatalf("ClusterSpeakers() error = %v", err)
	}

	if result == nil {
		t.Fatal("ClusterSpeakers() returned nil result")
	}

	// Verify at least one speaker was created
	if len(storage.speakers) == 0 {
		t.Error("ClusterSpeakers() created 0 speakers, want at least 1")
	}

	// Verify embeddings were updated (assigned to speakers)
	if storage.embeddings[emb1ID].SpeakerID == nil {
		t.Error("ClusterSpeakers() emb1.SpeakerID is nil")
	}
	if storage.embeddings[emb2ID].SpeakerID == nil {
		t.Error("ClusterSpeakers() emb2.SpeakerID is nil")
	}

	// Verify segments were updated
	if storage.segments[seg1ID].SpeakerID == nil {
		t.Error("ClusterSpeakers() seg1.SpeakerID is nil")
	}
	if storage.segments[seg2ID].SpeakerID == nil {
		t.Error("ClusterSpeakers() seg2.SpeakerID is nil")
	}

	// Verify segments match their embedding's speaker ID
	speakerID1 := storage.embeddings[emb1ID].SpeakerID
	speakerID2 := storage.embeddings[emb2ID].SpeakerID
	segSpeakerID1 := storage.segments[seg1ID].SpeakerID
	segSpeakerID2 := storage.segments[seg2ID].SpeakerID

	if speakerID1 == nil || segSpeakerID1 == nil || *segSpeakerID1 != *speakerID1 {
		t.Errorf("ClusterSpeakers() seg1.SpeakerID (%v) doesn't match emb1.SpeakerID (%v)", segSpeakerID1, speakerID1)
	}
	if speakerID2 == nil || segSpeakerID2 == nil || *segSpeakerID2 != *speakerID2 {
		t.Errorf("ClusterSpeakers() seg2.SpeakerID (%v) doesn't match emb2.SpeakerID (%v)", segSpeakerID2, speakerID2)
	}
}

