package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)

const (
	indexSpeakers          = "speakers"
	indexRecordings        = "recordings"
	indexSegments          = "segments"
	indexLifelogs          = "lifelogs"
	indexLifelogBlockquotes = "lifelog_blockquotes"
	indexSpeakerEmbeddings = "speaker_embeddings"
)

// ElasticsearchStorage implements the Storage interface using Elasticsearch
type ElasticsearchStorage struct {
	client      *elasticsearch.Client
	segmentIDCounter int64 // Counter for generating segment IDs
}

// NewElasticsearchStorage creates a new Elasticsearch storage instance
// url is the Elasticsearch URL (e.g., "http://localhost:9200")
func NewElasticsearchStorage(url string) (*ElasticsearchStorage, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{url},
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	storage := &ElasticsearchStorage{
		client: client,
	}

	// Ensure indices exist with proper mappings
	if err := storage.ensureIndices(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to create indices: %w", err)
	}

	return storage, nil
}

// Close closes the Elasticsearch connection
func (s *ElasticsearchStorage) Close() error {
	// Elasticsearch client doesn't need explicit closing
	return nil
}

// Health checks if Elasticsearch is healthy and accessible
func (s *ElasticsearchStorage) Health(ctx context.Context) error {
	res, err := s.client.Cluster.Health(s.client.Cluster.Health.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("health check error: %s", string(body))
	}

	return nil
}

// ensureIndices creates indices if they don't exist with proper mappings
func (s *ElasticsearchStorage) ensureIndices(ctx context.Context) error {
	indices := []struct {
		name    string
		mapping map[string]interface{}
	}{
		{
			name: indexSpeakers,
			mapping: map[string]interface{}{
				"mappings": map[string]interface{}{
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "keyword",
						},
						"embedding": map[string]interface{}{
							"type":     "dense_vector",
							"dims":     256,
							"index":    true,
							"similarity": "cosine",
						},
						"first_seen": map[string]interface{}{
							"type": "date",
						},
						"last_seen": map[string]interface{}{
							"type": "date",
						},
						"contact_id": map[string]interface{}{
							"type":  "keyword",
							"index": false,
						},
						"created_at": map[string]interface{}{
							"type": "date",
						},
						"updated_at": map[string]interface{}{
							"type": "date",
						},
					},
				},
				"settings": map[string]interface{}{
					"number_of_shards":   1,
					"number_of_replicas": 0,
				},
			},
		},
		{
			name: indexRecordings,
			mapping: map[string]interface{}{
				"mappings": map[string]interface{}{
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "keyword",
						},
						"file_path": map[string]interface{}{
							"type": "keyword",
						},
						"start_time": map[string]interface{}{
							"type": "date",
						},
						"duration_seconds": map[string]interface{}{
							"type": "float",
						},
						"sample_rate": map[string]interface{}{
							"type": "integer",
						},
						"format": map[string]interface{}{
							"type": "keyword",
						},
						"diarized_at": map[string]interface{}{
							"type": "date",
						},
						"processing_time": map[string]interface{}{
							"type": "float",
						},
						"rtf": map[string]interface{}{
							"type": "float",
						},
						"device": map[string]interface{}{
							"type": "keyword",
						},
						"created_at": map[string]interface{}{
							"type": "date",
						},
					},
				},
				"settings": map[string]interface{}{
					"number_of_shards":   1,
					"number_of_replicas": 0,
				},
			},
		},
		{
			name: indexSegments,
			mapping: map[string]interface{}{
				"mappings": map[string]interface{}{
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "keyword",
						},
						"speaker_embedding_id": map[string]interface{}{
							"type": "keyword",
						},
						"speaker_id": map[string]interface{}{
							"type": "keyword",
						},
						"recording_id": map[string]interface{}{
							"type": "keyword",
						},
					"local_speaker_id": map[string]interface{}{
						"type": "keyword",
					},
					"blockquote_id": map[string]interface{}{
						"type": "keyword",
					},
					"start_time": map[string]interface{}{
						"type": "float",
					},
						"end_time": map[string]interface{}{
							"type": "float",
						},
						"duration": map[string]interface{}{
							"type": "float",
						},
						"start_byte_offset": map[string]interface{}{
							"type": "long",
						},
						"end_byte_offset": map[string]interface{}{
							"type": "long",
						},
						"created_at": map[string]interface{}{
							"type": "date",
						},
					},
				},
				"settings": map[string]interface{}{
					"number_of_shards":   1,
					"number_of_replicas": 0,
				},
			},
		},
		{
			name: indexLifelogs,
			mapping: map[string]interface{}{
				"mappings": map[string]interface{}{
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "keyword",
						},
						"title": map[string]interface{}{
							"type": "text",
						},
						"markdown": map[string]interface{}{
							"type": "text",
						},
						"start_time": map[string]interface{}{
							"type": "date",
						},
						"end_time": map[string]interface{}{
							"type": "date",
						},
						"created_at": map[string]interface{}{
							"type": "date",
						},
					},
				},
				"settings": map[string]interface{}{
					"number_of_shards":   1,
					"number_of_replicas": 0,
				},
			},
		},
		{
			name: indexLifelogBlockquotes,
			mapping: map[string]interface{}{
				"mappings": map[string]interface{}{
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "keyword",
						},
						"lifelog_id": map[string]interface{}{
							"type": "keyword",
						},
						"recording_id": map[string]interface{}{
							"type": "keyword",
						},
						"content": map[string]interface{}{
							"type": "text",
						},
						"speaker_name": map[string]interface{}{
							"type": "keyword",
						},
						"speaker_id": map[string]interface{}{
							"type": "keyword",
						},
						"contact_id": map[string]interface{}{
							"type": "keyword",
						},
						"start_offset_ms": map[string]interface{}{
							"type": "integer",
						},
						"end_offset_ms": map[string]interface{}{
							"type": "integer",
						},
						"start_time": map[string]interface{}{
							"type": "date",
						},
						"end_time": map[string]interface{}{
							"type": "date",
						},
						"created_at": map[string]interface{}{
							"type": "date",
						},
					},
				},
				"settings": map[string]interface{}{
					"number_of_shards":   1,
					"number_of_replicas": 0,
				},
			},
		},
		{
			name: indexSpeakerEmbeddings,
			mapping: map[string]interface{}{
				"mappings": map[string]interface{}{
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "keyword",
						},
						"speaker_id": map[string]interface{}{
							"type": "keyword",
						},
						"recording_id": map[string]interface{}{
							"type": "keyword",
						},
						"local_speaker_id": map[string]interface{}{
							"type": "keyword",
						},
						"embedding": map[string]interface{}{
							"type":     "dense_vector",
							"dims":     256,
							"index":    true,
							"similarity": "cosine",
						},
						"duration_seconds": map[string]interface{}{
							"type": "float",
						},
						"created_at": map[string]interface{}{
							"type": "date",
						},
					},
				},
				"settings": map[string]interface{}{
					"number_of_shards":   1,
					"number_of_replicas": 0,
				},
			},
		},
	}

	for _, idx := range indices {
		// Check if index exists
		res, err := s.client.Indices.Exists([]string{idx.name})
		if err != nil {
			return fmt.Errorf("failed to check index existence: %w", err)
		}
		res.Body.Close()

		if res.StatusCode == 200 {
			// Index exists, skip
			continue
		}

		// Create index with mapping
		mappingJSON, err := json.Marshal(idx.mapping)
		if err != nil {
			return fmt.Errorf("failed to marshal mapping: %w", err)
		}

		res, err = s.client.Indices.Create(
			idx.name,
			s.client.Indices.Create.WithBody(bytes.NewReader(mappingJSON)),
			s.client.Indices.Create.WithContext(ctx),
		)
		if err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.name, err)
		}
		defer res.Body.Close()

		if res.IsError() {
			body, _ := io.ReadAll(res.Body)
			return fmt.Errorf("failed to create index %s: %s", idx.name, string(body))
		}
	}

	return nil
}

// Speaker operations

func (s *ElasticsearchStorage) CreateSpeaker(ctx context.Context, speaker *Speaker) error {
	if err := ValidateEmbedding(speaker.Embedding); err != nil {
		return err
	}

	// Check if speaker already exists
	_, err := s.GetSpeaker(ctx, speaker.ID)
	if err == nil {
		return ErrDuplicateKey
	}
	if err != ErrNotFound {
		return err
	}

	doc := s.speakerToDoc(speaker)
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal speaker: %w", err)
	}

	res, err := s.client.Index(
		indexSpeakers,
		bytes.NewReader(docJSON),
		s.client.Index.WithDocumentID(speaker.ID),
		s.client.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to index speaker: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to index speaker: %s", string(body))
	}

	return nil
}

func (s *ElasticsearchStorage) GetSpeaker(ctx context.Context, id string) (*Speaker, error) {
	res, err := s.client.Get(indexSpeakers, id, s.client.Get.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get speaker: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, ErrNotFound
	}

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("failed to get speaker: %s", string(body))
	}

	var result struct {
		Source map[string]interface{} `json:"_source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode speaker: %w", err)
	}

	return s.docToSpeaker(result.Source)
}

func (s *ElasticsearchStorage) FindSimilarSpeakers(ctx context.Context, embedding []float32, threshold float64, limit int, onlyCentroids bool) ([]SpeakerMatch, error) {
	if err := ValidateEmbedding(embedding); err != nil {
		return nil, err
	}

	// Convert float32 to float64 for Elasticsearch
	embedding64 := make([]float64, len(embedding))
	for i, v := range embedding {
		embedding64[i] = float64(v)
	}

	// Build kNN query
	// Note: min_score doesn't work reliably with kNN queries in Elasticsearch
	// Instead, we fetch more candidates and filter by threshold in application code
	query := map[string]interface{}{
		"knn": map[string]interface{}{
			"field":         "embedding",
			"query_vector":  embedding64,
			"k":             100, // Get more candidates, filter by threshold in code
			"num_candidates": 100,
		},
	}

	// Set size to get enough results to filter (or use limit if specified)
	if limit > 0 {
		query["size"] = limit
	} else {
		query["size"] = 100 // Get enough results to filter by threshold
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	res, err := s.client.Search(
		s.client.Search.WithIndex(indexSpeakers),
		s.client.Search.WithBody(bytes.NewReader(queryJSON)),
		s.client.Search.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search speakers: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("failed to search speakers: %s", string(body))
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
				Score  float64                `json:"_score"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	matches := make([]SpeakerMatch, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		speaker, err := s.docToSpeaker(hit.Source)
		if err != nil {
			continue // Skip invalid documents
		}

		// Elasticsearch returns similarity as _score
		// For cosine similarity with dense_vector, the score is the cosine similarity
		// Range: -1.0 to 1.0, but typically 0.0 to 1.0 for normalized vectors
		// However, Elasticsearch may return scores in a different range
		// We'll use the score as-is and let the caller filter by threshold
		similarity := hit.Score
		
		// Only filter by threshold if threshold > 0 (caller wants filtering)
		// If threshold is 0, return all matches for caller to filter
		if threshold <= 0 || similarity >= threshold {
			matches = append(matches, SpeakerMatch{
				Speaker:    speaker,
				Similarity: similarity,
			})
		}
	}

	// Apply limit if specified
	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}

	return matches, nil
}

func (s *ElasticsearchStorage) UpdateSpeaker(ctx context.Context, speaker *Speaker) error {
	// Check if speaker exists
	existing, err := s.GetSpeaker(ctx, speaker.ID)
	if err != nil {
		return err
	}

	// Merge updates (only non-zero fields)
	if speaker.Embedding != nil {
		if err := ValidateEmbedding(speaker.Embedding); err != nil {
			return err
		}
		existing.Embedding = speaker.Embedding
	}
	if !speaker.FirstSeen.IsZero() {
		existing.FirstSeen = speaker.FirstSeen
	}
	if !speaker.LastSeen.IsZero() {
		existing.LastSeen = speaker.LastSeen
	}
	if speaker.ContactID != nil {
		existing.ContactID = speaker.ContactID
	}
	existing.UpdatedAt = time.Now()

	doc := s.speakerToDoc(existing)
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal speaker: %w", err)
	}

	res, err := s.client.Index(
		indexSpeakers,
		bytes.NewReader(docJSON),
		s.client.Index.WithDocumentID(speaker.ID),
		s.client.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to update speaker: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to update speaker: %s", string(body))
	}

	return nil
}

func (s *ElasticsearchStorage) ListSpeakers(ctx context.Context, contactID *string) ([]*Speaker, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
		"size": 10000, // TODO: Implement pagination if needed
	}

	if contactID != nil {
		query["query"] = map[string]interface{}{
			"term": map[string]interface{}{
				"contact_id": *contactID,
			},
		}
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	res, err := s.client.Search(
		s.client.Search.WithIndex(indexSpeakers),
		s.client.Search.WithBody(bytes.NewReader(queryJSON)),
		s.client.Search.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search speakers: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("failed to search speakers: %s", string(body))
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	speakers := make([]*Speaker, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		speaker, err := s.docToSpeaker(hit.Source)
		if err != nil {
			continue // Skip invalid documents
		}
		speakers = append(speakers, speaker)
	}

	return speakers, nil
}

// Recording operations

func (s *ElasticsearchStorage) CreateRecording(ctx context.Context, recording *Recording) error {
	// Check if recording already exists
	_, err := s.GetRecording(ctx, recording.ID)
	if err == nil {
		return ErrDuplicateKey
	}
	if err != ErrNotFound {
		return err
	}

	doc := s.recordingToDoc(recording)
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal recording: %w", err)
	}

	res, err := s.client.Index(
		indexRecordings,
		bytes.NewReader(docJSON),
		s.client.Index.WithDocumentID(recording.ID),
		s.client.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to index recording: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to index recording: %s", string(body))
	}

	return nil
}

func (s *ElasticsearchStorage) GetRecording(ctx context.Context, id string) (*Recording, error) {
	res, err := s.client.Get(indexRecordings, id, s.client.Get.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get recording: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, ErrNotFound
	}

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("failed to get recording: %s", string(body))
	}

	var result struct {
		Source map[string]interface{} `json:"_source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode recording: %w", err)
	}

	return s.docToRecording(result.Source)
}

func (s *ElasticsearchStorage) GetRecordingByPath(ctx context.Context, filePath string) (*Recording, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"file_path": filePath,
			},
		},
		"size": 1,
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	res, err := s.client.Search(
		s.client.Search.WithIndex(indexRecordings),
		s.client.Search.WithBody(bytes.NewReader(queryJSON)),
		s.client.Search.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search recording: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("failed to search recording: %s", string(body))
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	if len(result.Hits.Hits) == 0 {
		return nil, ErrNotFound
	}

	return s.docToRecording(result.Hits.Hits[0].Source)
}

func (s *ElasticsearchStorage) ListRecordings(ctx context.Context, startTime *time.Time, endTime *time.Time) ([]*Recording, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
		"size": 10000, // TODO: Implement pagination if needed
		"sort": []map[string]interface{}{
			{"start_time": map[string]interface{}{"order": "asc"}},
		},
	}

	if startTime != nil || endTime != nil {
		rangeQuery := map[string]interface{}{}
		if startTime != nil {
			rangeQuery["gte"] = startTime.Format(time.RFC3339)
		}
		if endTime != nil {
			rangeQuery["lt"] = endTime.Format(time.RFC3339)
		}
		query["query"] = map[string]interface{}{
			"range": map[string]interface{}{
				"start_time": rangeQuery,
			},
		}
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	res, err := s.client.Search(
		s.client.Search.WithIndex(indexRecordings),
		s.client.Search.WithBody(bytes.NewReader(queryJSON)),
		s.client.Search.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search recordings: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("failed to search recordings: %s", string(body))
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	recordings := make([]*Recording, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		recording, err := s.docToRecording(hit.Source)
		if err != nil {
			continue // Skip invalid documents
		}
		recordings = append(recordings, recording)
	}

	return recordings, nil
}

func (s *ElasticsearchStorage) UpdateRecording(ctx context.Context, recording *Recording) error {
	// Check if recording exists
	existing, err := s.GetRecording(ctx, recording.ID)
	if err != nil {
		return err
	}

	// Merge updates (only non-zero fields)
	if recording.FilePath != "" {
		existing.FilePath = recording.FilePath
	}
	if !recording.StartTime.IsZero() {
		existing.StartTime = recording.StartTime
	}
	if recording.Duration != 0 {
		existing.Duration = recording.Duration
	}
	if recording.SampleRate != nil {
		existing.SampleRate = recording.SampleRate
	}
	if recording.Format != nil {
		existing.Format = recording.Format
	}
	if recording.DiarizedAt != nil {
		existing.DiarizedAt = recording.DiarizedAt
	}
	if recording.ProcessingTime != nil {
		existing.ProcessingTime = recording.ProcessingTime
	}
	if recording.RTF != nil {
		existing.RTF = recording.RTF
	}
	if recording.Device != nil {
		existing.Device = recording.Device
	}

	doc := s.recordingToDoc(existing)
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal recording: %w", err)
	}

	res, err := s.client.Index(
		indexRecordings,
		bytes.NewReader(docJSON),
		s.client.Index.WithDocumentID(recording.ID),
		s.client.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to update recording: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to update recording: %s", string(body))
	}

	return nil
}

// Segment operations

func (s *ElasticsearchStorage) CreateSegment(ctx context.Context, segment *Segment) error {
	// Generate ID if not provided
	if segment.ID == 0 {
		segment.ID = atomic.AddInt64(&s.segmentIDCounter, 1)
		// Use timestamp as base to avoid collisions
		if segment.ID < time.Now().UnixNano()/1000 {
			segment.ID = time.Now().UnixNano() / 1000
			atomic.StoreInt64(&s.segmentIDCounter, segment.ID)
		}
	}

	doc := s.segmentToDoc(segment)
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal segment: %w", err)
	}

	segmentID := strconv.FormatInt(segment.ID, 10)
	res, err := s.client.Index(
		indexSegments,
		bytes.NewReader(docJSON),
		s.client.Index.WithDocumentID(segmentID),
		s.client.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to index segment: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to index segment: %s", string(body))
	}

	return nil
}

func (s *ElasticsearchStorage) CreateSegments(ctx context.Context, segments []*Segment) (int, error) {
	if len(segments) == 0 {
		return 0, nil
	}

	// Generate IDs for segments that don't have one
	baseID := time.Now().UnixNano() / 1000
	for i := range segments {
		if segments[i].ID == 0 {
			segments[i].ID = baseID + int64(i)
		}
	}
	// Update counter to avoid collisions
	if len(segments) > 0 {
		lastID := segments[len(segments)-1].ID
		atomic.StoreInt64(&s.segmentIDCounter, lastID)
	}

	// Build bulk request
	var buf bytes.Buffer
	for _, segment := range segments {
		// Action line
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": indexSegments,
				"_id":    strconv.FormatInt(segment.ID, 10),
			},
		}
		actionJSON, _ := json.Marshal(action)
		buf.Write(actionJSON)
		buf.WriteString("\n")

		// Document line
		doc := s.segmentToDoc(segment)
		docJSON, err := json.Marshal(doc)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal segment: %w", err)
		}
		buf.Write(docJSON)
		buf.WriteString("\n")
	}

	res, err := s.client.Bulk(bytes.NewReader(buf.Bytes()), s.client.Bulk.WithContext(ctx))
	if err != nil {
		return 0, fmt.Errorf("failed to bulk index segments: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return 0, fmt.Errorf("failed to bulk index segments: %s", string(body))
	}

	// Parse response to count successful inserts
	var bulkResult struct {
		Items []struct {
			Index struct {
				Status int `json:"status"`
			} `json:"index"`
		} `json:"items"`
	}

	if err := json.NewDecoder(res.Body).Decode(&bulkResult); err != nil {
		// If we can't parse, assume all succeeded
		return len(segments), nil
	}

	successCount := 0
	for _, item := range bulkResult.Items {
		if item.Index.Status >= 200 && item.Index.Status < 300 {
			successCount++
		}
	}

	return successCount, nil
}

func (s *ElasticsearchStorage) GetSegment(ctx context.Context, id int64) (*Segment, error) {
	segmentID := strconv.FormatInt(id, 10)
	res, err := s.client.Get(indexSegments, segmentID, s.client.Get.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get segment: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, ErrNotFound
	}

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("failed to get segment: %s", string(body))
	}

	var result struct {
		Source map[string]interface{} `json:"_source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode segment: %w", err)
	}

	return s.docToSegment(result.Source)
}

func (s *ElasticsearchStorage) GetSegmentsBySpeaker(ctx context.Context, speakerID string) ([]*Segment, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"speaker_id": speakerID,
			},
		},
		"size": 10000, // TODO: Implement pagination if needed
		"sort": []map[string]interface{}{
			{"start_time": map[string]interface{}{"order": "asc"}},
		},
	}

	return s.searchSegments(ctx, query)
}

func (s *ElasticsearchStorage) GetSegmentsByRecording(ctx context.Context, recordingID string) ([]*Segment, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"recording_id": recordingID,
			},
		},
		"size": 10000, // TODO: Implement pagination if needed
		"sort": []map[string]interface{}{
			{"start_time": map[string]interface{}{"order": "asc"}},
		},
	}

	return s.searchSegments(ctx, query)
}

func (s *ElasticsearchStorage) GetSegmentsByTimeRange(ctx context.Context, recordingID string, startTime, endTime float64) ([]*Segment, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"recording_id": recordingID,
						},
					},
					{
						"range": map[string]interface{}{
							"start_time": map[string]interface{}{
								"gte": startTime,
								"lt":  endTime,
							},
						},
					},
				},
			},
		},
		"size": 10000, // TODO: Implement pagination if needed
		"sort": []map[string]interface{}{
			{"start_time": map[string]interface{}{"order": "asc"}},
		},
	}

	return s.searchSegments(ctx, query)
}

func (s *ElasticsearchStorage) UpdateSegmentByteOffsets(ctx context.Context, segmentID int64, startByteOffset, endByteOffset int64) error {
	// Get existing segment
	segment, err := s.GetSegment(ctx, segmentID)
	if err != nil {
		return err
	}

	// Update byte offsets
	segment.StartByteOffset = &startByteOffset
	segment.EndByteOffset = &endByteOffset

	// Update document
	doc := s.segmentToDoc(segment)
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal segment: %w", err)
	}

	idStr := strconv.FormatInt(segmentID, 10)
	res, err := s.client.Index(
		indexSegments,
		bytes.NewReader(docJSON),
		s.client.Index.WithDocumentID(idStr),
		s.client.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to update segment: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to update segment: %s", string(body))
	}

	return nil
}

// Lifelog operations

func (s *ElasticsearchStorage) CreateLifelog(ctx context.Context, lifelog *Lifelog) error {
	// Check if lifelog already exists
	_, err := s.GetLifelog(ctx, lifelog.ID)
	if err == nil {
		return ErrDuplicateKey
	}
	if err != ErrNotFound {
		return err
	}

	doc := s.lifelogToDoc(lifelog)
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal lifelog: %w", err)
	}

	res, err := s.client.Index(
		indexLifelogs,
		bytes.NewReader(docJSON),
		s.client.Index.WithDocumentID(lifelog.ID),
		s.client.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to index lifelog: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to index lifelog: %s", string(body))
	}

	return nil
}

func (s *ElasticsearchStorage) GetLifelog(ctx context.Context, id string) (*Lifelog, error) {
	res, err := s.client.Get(indexLifelogs, id, s.client.Get.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get lifelog: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, ErrNotFound
	}

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("failed to get lifelog: %s", string(body))
	}

	var result struct {
		Source map[string]interface{} `json:"_source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode lifelog: %w", err)
	}

	return s.docToLifelog(result.Source)
}

func (s *ElasticsearchStorage) ListLifelogs(ctx context.Context, startTime *time.Time, endTime *time.Time) ([]*Lifelog, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
		"size": 10000, // TODO: Implement pagination if needed
		"sort": []map[string]interface{}{
			{"start_time": map[string]interface{}{"order": "asc"}},
		},
	}

	if startTime != nil || endTime != nil {
		rangeQuery := map[string]interface{}{}
		if startTime != nil {
			rangeQuery["gte"] = startTime.Format(time.RFC3339)
		}
		if endTime != nil {
			rangeQuery["lt"] = endTime.Format(time.RFC3339)
		}
		query["query"] = map[string]interface{}{
			"range": map[string]interface{}{
				"start_time": rangeQuery,
			},
		}
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	res, err := s.client.Search(
		s.client.Search.WithIndex(indexLifelogs),
		s.client.Search.WithBody(bytes.NewReader(queryJSON)),
		s.client.Search.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search lifelogs: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("failed to search lifelogs: %s", string(body))
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	lifelogs := make([]*Lifelog, len(result.Hits.Hits))
	for i, hit := range result.Hits.Hits {
		lifelog, err := s.docToLifelog(hit.Source)
		if err != nil {
			log.Printf("Warning: failed to parse lifelog from search hit: %v", err)
			continue
		}
		lifelogs[i] = lifelog
	}

	return lifelogs, nil
}

func (s *ElasticsearchStorage) UpdateLifelog(ctx context.Context, lifelog *Lifelog) error {
	// Check if lifelog exists
	existing, err := s.GetLifelog(ctx, lifelog.ID)
	if err != nil {
		return err
	}

	// Merge updates (only non-zero fields)
	if lifelog.Title != "" {
		existing.Title = lifelog.Title
	}
	if lifelog.Markdown != "" {
		existing.Markdown = lifelog.Markdown
	}
	if !lifelog.StartTime.IsZero() {
		existing.StartTime = lifelog.StartTime
	}
	if !lifelog.EndTime.IsZero() {
		existing.EndTime = lifelog.EndTime
	}

	doc := s.lifelogToDoc(existing)
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal updated lifelog: %w", err)
	}

	res, err := s.client.Update(
		indexLifelogs,
		lifelog.ID,
		bytes.NewReader([]byte(fmt.Sprintf(`{"doc":%s}`, docJSON))),
		s.client.Update.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to update lifelog: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to update lifelog: %s", string(body))
	}

	return nil
}

func (s *ElasticsearchStorage) CreateLifelogBlockquote(ctx context.Context, blockquote *LifelogBlockquote) error {
	// Check if blockquote already exists
	_, err := s.GetLifelogBlockquote(ctx, blockquote.ID)
	if err == nil {
		return ErrDuplicateKey
	}
	if err != ErrNotFound {
		return err
	}

	doc := s.lifelogBlockquoteToDoc(blockquote)
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal blockquote: %w", err)
	}

	res, err := s.client.Index(
		indexLifelogBlockquotes,
		bytes.NewReader(docJSON),
		s.client.Index.WithDocumentID(blockquote.ID),
		s.client.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to index blockquote: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to index blockquote: %s", string(body))
	}

	return nil
}

func (s *ElasticsearchStorage) CreateLifelogBlockquotes(ctx context.Context, blockquotes []*LifelogBlockquote) (int, error) {
	if len(blockquotes) == 0 {
		return 0, nil
	}

	// Build bulk request
	var buf bytes.Buffer
	for _, blockquote := range blockquotes {
		// Action line
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": indexLifelogBlockquotes,
				"_id":    blockquote.ID,
			},
		}
		actionJSON, _ := json.Marshal(action)
		buf.Write(actionJSON)
		buf.WriteString("\n")

		// Document line
		doc := s.lifelogBlockquoteToDoc(blockquote)
		docJSON, err := json.Marshal(doc)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal blockquote: %w", err)
		}
		buf.Write(docJSON)
		buf.WriteString("\n")
	}

	res, err := s.client.Bulk(bytes.NewReader(buf.Bytes()), s.client.Bulk.WithContext(ctx))
	if err != nil {
		return 0, fmt.Errorf("failed to bulk index blockquotes: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return 0, fmt.Errorf("failed to bulk index blockquotes: %s", string(body))
	}

	// Parse response to count successful inserts
	var bulkResult struct {
		Items []struct {
			Index struct {
				Status int `json:"status"`
			} `json:"index"`
		} `json:"items"`
	}

	if err := json.NewDecoder(res.Body).Decode(&bulkResult); err != nil {
		// If we can't parse, assume all succeeded
		return len(blockquotes), nil
	}

	successCount := 0
	for _, item := range bulkResult.Items {
		if item.Index.Status >= 200 && item.Index.Status < 300 {
			successCount++
		}
	}

	return successCount, nil
}

func (s *ElasticsearchStorage) GetLifelogBlockquote(ctx context.Context, id string) (*LifelogBlockquote, error) {
	res, err := s.client.Get(indexLifelogBlockquotes, id, s.client.Get.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get blockquote: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, ErrNotFound
	}

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("failed to get blockquote: %s", string(body))
	}

	var result struct {
		Source map[string]interface{} `json:"_source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode blockquote: %w", err)
	}

	return s.docToLifelogBlockquote(result.Source)
}

func (s *ElasticsearchStorage) GetLifelogBlockquotesByLifelog(ctx context.Context, lifelogID string) ([]*LifelogBlockquote, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"lifelog_id.keyword": lifelogID,
			},
		},
		"size": 10000, // TODO: Implement pagination if needed
		"sort": []map[string]interface{}{
			{"start_time": map[string]interface{}{"order": "asc"}},
		},
	}
	return s.searchLifelogBlockquotes(ctx, query)
}

func (s *ElasticsearchStorage) GetLifelogBlockquotesByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*LifelogBlockquote, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"range": map[string]interface{}{
				"start_time": map[string]interface{}{
					"gte": startTime.Format(time.RFC3339),
					"lt":  endTime.Format(time.RFC3339),
				},
			},
		},
		"size": 10000, // TODO: Implement pagination if needed
		"sort": []map[string]interface{}{
			{"start_time": map[string]interface{}{"order": "asc"}},
		},
	}
	return s.searchLifelogBlockquotes(ctx, query)
}

func (s *ElasticsearchStorage) UpdateLifelogBlockquote(ctx context.Context, blockquote *LifelogBlockquote) error {
	// Check if blockquote exists
	existing, err := s.GetLifelogBlockquote(ctx, blockquote.ID)
	if err != nil {
		return err
	}

	// Merge updates (only non-zero fields)
	if blockquote.LifelogID != "" {
		existing.LifelogID = blockquote.LifelogID
	}
	if blockquote.RecordingID != nil {
		existing.RecordingID = blockquote.RecordingID
	}
	if blockquote.Content != "" {
		existing.Content = blockquote.Content
	}
	if blockquote.SpeakerName != "" {
		existing.SpeakerName = blockquote.SpeakerName
	}
	if blockquote.SpeakerID != nil {
		existing.SpeakerID = blockquote.SpeakerID
	}
	if blockquote.ContactID != nil {
		existing.ContactID = blockquote.ContactID
	}
	if blockquote.StartOffsetMs != 0 {
		existing.StartOffsetMs = blockquote.StartOffsetMs
	}
	if blockquote.EndOffsetMs != 0 {
		existing.EndOffsetMs = blockquote.EndOffsetMs
	}
	if !blockquote.StartTime.IsZero() {
		existing.StartTime = blockquote.StartTime
	}
	if !blockquote.EndTime.IsZero() {
		existing.EndTime = blockquote.EndTime
	}

	doc := s.lifelogBlockquoteToDoc(existing)
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal updated blockquote: %w", err)
	}

	res, err := s.client.Update(
		indexLifelogBlockquotes,
		blockquote.ID,
		bytes.NewReader([]byte(fmt.Sprintf(`{"doc":%s}`, docJSON))),
		s.client.Update.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to update blockquote: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to update blockquote: %s", string(body))
	}

	return nil
}

// Helper functions

func (s *ElasticsearchStorage) searchSegments(ctx context.Context, query map[string]interface{}) ([]*Segment, error) {
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	res, err := s.client.Search(
		s.client.Search.WithIndex(indexSegments),
		s.client.Search.WithBody(bytes.NewReader(queryJSON)),
		s.client.Search.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search segments: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("failed to search segments: %s", string(body))
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	segments := make([]*Segment, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		segment, err := s.docToSegment(hit.Source)
		if err != nil {
			continue // Skip invalid documents
		}
		segments = append(segments, segment)
	}

	return segments, nil
}

func (s *ElasticsearchStorage) speakerToDoc(speaker *Speaker) map[string]interface{} {
	// Convert float32 embedding to float64 for Elasticsearch
	embedding64 := make([]float64, len(speaker.Embedding))
	for i, v := range speaker.Embedding {
		embedding64[i] = float64(v)
	}

	doc := map[string]interface{}{
		"id":         speaker.ID,
		"embedding":  embedding64,
		"first_seen": speaker.FirstSeen.Format(time.RFC3339),
		"last_seen":  speaker.LastSeen.Format(time.RFC3339),
		"created_at": speaker.CreatedAt.Format(time.RFC3339),
		"updated_at": speaker.UpdatedAt.Format(time.RFC3339),
	}

	if speaker.ContactID != nil {
		doc["contact_id"] = *speaker.ContactID
	}

	return doc
}

func (s *ElasticsearchStorage) recordingToDoc(recording *Recording) map[string]interface{} {
	doc := map[string]interface{}{
		"id":               recording.ID,
		"file_path":        recording.FilePath,
		"start_time":       recording.StartTime.Format(time.RFC3339),
		"duration_seconds": recording.Duration,
		"created_at":       recording.CreatedAt.Format(time.RFC3339),
	}

	if recording.SampleRate != nil {
		doc["sample_rate"] = *recording.SampleRate
	}
	if recording.Format != nil {
		doc["format"] = *recording.Format
	}
	if recording.DiarizedAt != nil {
		doc["diarized_at"] = recording.DiarizedAt.Format(time.RFC3339)
	}
	if recording.ProcessingTime != nil {
		doc["processing_time"] = *recording.ProcessingTime
	}
	if recording.RTF != nil {
		doc["rtf"] = *recording.RTF
	}
	if recording.Device != nil {
		doc["device"] = *recording.Device
	}

	return doc
}

func (s *ElasticsearchStorage) segmentToDoc(segment *Segment) map[string]interface{} {
	doc := map[string]interface{}{
		"id":          strconv.FormatInt(segment.ID, 10),
		"recording_id": segment.RecordingID,
		"start_time":  segment.StartTime,
		"end_time":    segment.EndTime,
		"duration":    segment.Duration,
		"created_at":  segment.CreatedAt.Format(time.RFC3339),
	}

	// Optional fields
	if segment.SpeakerEmbeddingID != nil {
		doc["speaker_embedding_id"] = *segment.SpeakerEmbeddingID
	}
	if segment.SpeakerID != nil {
		doc["speaker_id"] = *segment.SpeakerID
	}
	if segment.LocalSpeakerID != nil {
		doc["local_speaker_id"] = *segment.LocalSpeakerID
	}
	if segment.BlockquoteID != nil {
		doc["blockquote_id"] = *segment.BlockquoteID
	}
	if segment.StartByteOffset != nil {
		doc["start_byte_offset"] = *segment.StartByteOffset
	}
	if segment.EndByteOffset != nil {
		doc["end_byte_offset"] = *segment.EndByteOffset
	}

	return doc
}

// Helper functions to parse documents from Elasticsearch

func (s *ElasticsearchStorage) docToSpeaker(doc map[string]interface{}) (*Speaker, error) {
	speaker := &Speaker{}

	if id, ok := doc["id"].(string); ok {
		speaker.ID = id
	}

	// Parse embedding (float64[] -> float32[])
	if embedding, ok := doc["embedding"].([]interface{}); ok {
		speaker.Embedding = make([]float32, len(embedding))
		for i, v := range embedding {
			if f, ok := v.(float64); ok {
				speaker.Embedding[i] = float32(f)
			}
		}
	}

	// Parse dates
	if firstSeen, ok := doc["first_seen"].(string); ok {
		if t, err := time.Parse(time.RFC3339, firstSeen); err == nil {
			speaker.FirstSeen = t
		}
	}
	if lastSeen, ok := doc["last_seen"].(string); ok {
		if t, err := time.Parse(time.RFC3339, lastSeen); err == nil {
			speaker.LastSeen = t
		}
	}
	if createdAt, ok := doc["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			speaker.CreatedAt = t
		}
	}
	if updatedAt, ok := doc["updated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			speaker.UpdatedAt = t
		}
	}

	// Parse optional contact_id
	if contactID, ok := doc["contact_id"].(string); ok && contactID != "" {
		speaker.ContactID = &contactID
	}

	return speaker, nil
}

func (s *ElasticsearchStorage) docToRecording(doc map[string]interface{}) (*Recording, error) {
	recording := &Recording{}

	if id, ok := doc["id"].(string); ok {
		recording.ID = id
	}
	if filePath, ok := doc["file_path"].(string); ok {
		recording.FilePath = filePath
	}
	if duration, ok := doc["duration_seconds"].(float64); ok {
		recording.Duration = duration
	}

	// Parse dates
	if startTime, ok := doc["start_time"].(string); ok {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			recording.StartTime = t
		}
	}
	if createdAt, ok := doc["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			recording.CreatedAt = t
		}
	}
	if diarizedAt, ok := doc["diarized_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, diarizedAt); err == nil {
			recording.DiarizedAt = &t
		}
	}

	// Parse optional fields
	if sampleRate, ok := doc["sample_rate"].(float64); ok {
		sr := int(sampleRate)
		recording.SampleRate = &sr
	}
	if format, ok := doc["format"].(string); ok {
		recording.Format = &format
	}
	if processingTime, ok := doc["processing_time"].(float64); ok {
		recording.ProcessingTime = &processingTime
	}
	if rtf, ok := doc["rtf"].(float64); ok {
		recording.RTF = &rtf
	}
	if device, ok := doc["device"].(string); ok {
		recording.Device = &device
	}

	return recording, nil
}

func (s *ElasticsearchStorage) docToSegment(doc map[string]interface{}) (*Segment, error) {
	segment := &Segment{}

	// Parse ID (stored as string in ES, convert to int64)
	if idStr, ok := doc["id"].(string); ok {
		if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			segment.ID = id
		}
	}

	// Parse optional speaker fields
	if speakerEmbeddingID, ok := doc["speaker_embedding_id"].(string); ok {
		segment.SpeakerEmbeddingID = &speakerEmbeddingID
	}
	if speakerID, ok := doc["speaker_id"].(string); ok {
		segment.SpeakerID = &speakerID
	}
	if recordingID, ok := doc["recording_id"].(string); ok {
		segment.RecordingID = recordingID
	}
	if startTime, ok := doc["start_time"].(float64); ok {
		segment.StartTime = startTime
	}
	if endTime, ok := doc["end_time"].(float64); ok {
		segment.EndTime = endTime
	}
	if duration, ok := doc["duration"].(float64); ok {
		segment.Duration = duration
	}

	// Parse dates
	if createdAt, ok := doc["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			segment.CreatedAt = t
		}
	}

	// Parse optional fields
	if localSpeakerID, ok := doc["local_speaker_id"].(string); ok {
		segment.LocalSpeakerID = &localSpeakerID
	}
	if blockquoteID, ok := doc["blockquote_id"].(string); ok {
		segment.BlockquoteID = &blockquoteID
	}
	if startByteOffset, ok := doc["start_byte_offset"].(float64); ok {
		offset := int64(startByteOffset)
		segment.StartByteOffset = &offset
	}
	if endByteOffset, ok := doc["end_byte_offset"].(float64); ok {
		offset := int64(endByteOffset)
		segment.EndByteOffset = &offset
	}

	return segment, nil
}

func (s *ElasticsearchStorage) searchLifelogBlockquotes(ctx context.Context, query map[string]interface{}) ([]*LifelogBlockquote, error) {
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	res, err := s.client.Search(
		s.client.Search.WithIndex(indexLifelogBlockquotes),
		s.client.Search.WithBody(bytes.NewReader(queryJSON)),
		s.client.Search.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search blockquotes: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("failed to search blockquotes: %s", string(body))
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	blockquotes := make([]*LifelogBlockquote, len(result.Hits.Hits))
	for i, hit := range result.Hits.Hits {
		blockquote, err := s.docToLifelogBlockquote(hit.Source)
		if err != nil {
			log.Printf("Warning: failed to parse blockquote from search hit: %v", err)
			continue
		}
		blockquotes[i] = blockquote
	}

	return blockquotes, nil
}

func (s *ElasticsearchStorage) lifelogToDoc(lifelog *Lifelog) map[string]interface{} {
	return map[string]interface{}{
		"id":         lifelog.ID,
		"title":      lifelog.Title,
		"markdown":   lifelog.Markdown,
		"start_time": lifelog.StartTime.Format(time.RFC3339),
		"end_time":   lifelog.EndTime.Format(time.RFC3339),
		"created_at": lifelog.CreatedAt.Format(time.RFC3339),
	}
}

func (s *ElasticsearchStorage) lifelogBlockquoteToDoc(blockquote *LifelogBlockquote) map[string]interface{} {
	doc := map[string]interface{}{
		"id":              blockquote.ID,
		"lifelog_id":      blockquote.LifelogID,
		"content":         blockquote.Content,
		"speaker_name":    blockquote.SpeakerName,
		"start_offset_ms": blockquote.StartOffsetMs,
		"end_offset_ms":   blockquote.EndOffsetMs,
		"start_time":      blockquote.StartTime.Format(time.RFC3339),
		"end_time":        blockquote.EndTime.Format(time.RFC3339),
		"created_at":      blockquote.CreatedAt.Format(time.RFC3339),
	}

	if blockquote.RecordingID != nil {
		doc["recording_id"] = *blockquote.RecordingID
	}
	if blockquote.SpeakerID != nil {
		doc["speaker_id"] = *blockquote.SpeakerID
	}
	if blockquote.ContactID != nil {
		doc["contact_id"] = *blockquote.ContactID
	}

	return doc
}

func (s *ElasticsearchStorage) docToLifelog(doc map[string]interface{}) (*Lifelog, error) {
	lifelog := &Lifelog{}

	if id, ok := doc["id"].(string); ok {
		lifelog.ID = id
	}
	if title, ok := doc["title"].(string); ok {
		lifelog.Title = title
	}
	if markdown, ok := doc["markdown"].(string); ok {
		lifelog.Markdown = markdown
	}

	// Parse dates
	if startTime, ok := doc["start_time"].(string); ok {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			lifelog.StartTime = t
		}
	}
	if endTime, ok := doc["end_time"].(string); ok {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			lifelog.EndTime = t
		}
	}
	if createdAt, ok := doc["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			lifelog.CreatedAt = t
		}
	}

	return lifelog, nil
}

func (s *ElasticsearchStorage) docToLifelogBlockquote(doc map[string]interface{}) (*LifelogBlockquote, error) {
	blockquote := &LifelogBlockquote{}

	if id, ok := doc["id"].(string); ok {
		blockquote.ID = id
	}
	if lifelogID, ok := doc["lifelog_id"].(string); ok {
		blockquote.LifelogID = lifelogID
	}
	if content, ok := doc["content"].(string); ok {
		blockquote.Content = content
	}
	if speakerName, ok := doc["speaker_name"].(string); ok {
		blockquote.SpeakerName = speakerName
	}
	if startOffsetMs, ok := doc["start_offset_ms"].(float64); ok {
		blockquote.StartOffsetMs = int(startOffsetMs)
	}
	if endOffsetMs, ok := doc["end_offset_ms"].(float64); ok {
		blockquote.EndOffsetMs = int(endOffsetMs)
	}

	// Parse dates
	if startTime, ok := doc["start_time"].(string); ok {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			blockquote.StartTime = t
		}
	}
	if endTime, ok := doc["end_time"].(string); ok {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			blockquote.EndTime = t
		}
	}
	if createdAt, ok := doc["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			blockquote.CreatedAt = t
		}
	}

	// Parse optional fields
	if recordingID, ok := doc["recording_id"].(string); ok {
		blockquote.RecordingID = &recordingID
	}
	if speakerID, ok := doc["speaker_id"].(string); ok {
		blockquote.SpeakerID = &speakerID
	}
	if contactID, ok := doc["contact_id"].(string); ok && contactID != "" {
		blockquote.ContactID = &contactID
	}

	return blockquote, nil
}

// Additional segment methods

func (s *ElasticsearchStorage) GetSegmentsBySpeakerEmbedding(ctx context.Context, embeddingID string) ([]*Segment, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"speaker_embedding_id": embeddingID,
			},
		},
		"size": 10000, // TODO: Implement pagination if needed
		"sort": []map[string]interface{}{
			{"start_time": map[string]interface{}{"order": "asc"}},
		},
	}

	return s.searchSegments(ctx, query)
}

func (s *ElasticsearchStorage) GetSegmentsWithoutEmbeddings(ctx context.Context) ([]*Segment, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must_not": map[string]interface{}{
					"exists": map[string]interface{}{
						"field": "speaker_embedding_id",
					},
				},
			},
		},
		"size": 10000, // TODO: Implement pagination if needed
		"sort": []map[string]interface{}{
			{"start_time": map[string]interface{}{"order": "asc"}},
		},
	}

	return s.searchSegments(ctx, query)
}

func (s *ElasticsearchStorage) UpdateSegment(ctx context.Context, segment *Segment) error {
	// Check if segment exists
	_, err := s.GetSegment(ctx, segment.ID)
	if err != nil {
		return err
	}

	doc := s.segmentToDoc(segment)
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal segment: %w", err)
	}

	segmentID := strconv.FormatInt(segment.ID, 10)
	res, err := s.client.Update(
		indexSegments,
		segmentID,
		bytes.NewReader([]byte(fmt.Sprintf(`{"doc":%s}`, string(docJSON)))),
		s.client.Update.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to update segment: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to update segment: %s", string(body))
	}

	return nil
}

// SpeakerEmbedding operations

func (s *ElasticsearchStorage) CreateSpeakerEmbedding(ctx context.Context, embedding *SpeakerEmbedding) error {
	if err := ValidateEmbedding(embedding.Embedding); err != nil {
		return err
	}

	// Check if embedding already exists
	_, err := s.GetSpeakerEmbedding(ctx, embedding.ID)
	if err == nil {
		return ErrDuplicateKey
	}
	if err != ErrNotFound {
		return err
	}

	doc := s.speakerEmbeddingToDoc(embedding)
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal speaker embedding: %w", err)
	}

	res, err := s.client.Index(
		indexSpeakerEmbeddings,
		bytes.NewReader(docJSON),
		s.client.Index.WithDocumentID(embedding.ID),
		s.client.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to index speaker embedding: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to index speaker embedding: %s", string(body))
	}

	return nil
}

func (s *ElasticsearchStorage) GetSpeakerEmbedding(ctx context.Context, id string) (*SpeakerEmbedding, error) {
	res, err := s.client.Get(indexSpeakerEmbeddings, id, s.client.Get.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get speaker embedding: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, ErrNotFound
	}

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("failed to get speaker embedding: %s", string(body))
	}

	var result struct {
		Source map[string]interface{} `json:"_source"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode speaker embedding: %w", err)
	}

	return s.docToSpeakerEmbedding(result.Source)
}

func (s *ElasticsearchStorage) ListUnclusteredEmbeddings(ctx context.Context) ([]*SpeakerEmbedding, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must_not": map[string]interface{}{
					"exists": map[string]interface{}{
						"field": "speaker_id",
					},
				},
			},
		},
		"size": 10000, // TODO: Implement pagination if needed
		"sort": []map[string]interface{}{
			{"created_at": map[string]interface{}{"order": "asc"}},
		},
	}

	return s.searchSpeakerEmbeddings(ctx, query)
}

func (s *ElasticsearchStorage) ListAllEmbeddings(ctx context.Context, speakerID *string) ([]*SpeakerEmbedding, error) {
	var query map[string]interface{}

	if speakerID != nil {
		query = map[string]interface{}{
			"query": map[string]interface{}{
				"term": map[string]interface{}{
					"speaker_id": *speakerID,
				},
			},
			"size": 10000,
			"sort": []map[string]interface{}{
				{"created_at": map[string]interface{}{"order": "asc"}},
			},
		}
	} else {
		query = map[string]interface{}{
			"query": map[string]interface{}{
				"match_all": map[string]interface{}{},
			},
			"size": 10000,
			"sort": []map[string]interface{}{
				{"created_at": map[string]interface{}{"order": "asc"}},
			},
		}
	}

	return s.searchSpeakerEmbeddings(ctx, query)
}

func (s *ElasticsearchStorage) UpdateSpeakerEmbedding(ctx context.Context, embedding *SpeakerEmbedding) error {
	// Check if embedding exists
	_, err := s.GetSpeakerEmbedding(ctx, embedding.ID)
	if err != nil {
		return err
	}

	doc := s.speakerEmbeddingToDoc(embedding)
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal speaker embedding: %w", err)
	}

	res, err := s.client.Update(
		indexSpeakerEmbeddings,
		embedding.ID,
		bytes.NewReader([]byte(fmt.Sprintf(`{"doc":%s}`, string(docJSON)))),
		s.client.Update.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to update speaker embedding: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to update speaker embedding: %s", string(body))
	}

	return nil
}

// Helper functions for SpeakerEmbedding

func (s *ElasticsearchStorage) speakerEmbeddingToDoc(embedding *SpeakerEmbedding) map[string]interface{} {
	// Convert float32 to float64 for Elasticsearch
	embedding64 := make([]float64, len(embedding.Embedding))
	for i, v := range embedding.Embedding {
		embedding64[i] = float64(v)
	}

	doc := map[string]interface{}{
		"id":               embedding.ID,
		"recording_id":     embedding.RecordingID,
		"local_speaker_id": embedding.LocalSpeakerID,
		"embedding":        embedding64,
		"duration_seconds": embedding.DurationSeconds,
		"created_at":       embedding.CreatedAt.Format(time.RFC3339),
	}

	if embedding.SpeakerID != nil {
		doc["speaker_id"] = *embedding.SpeakerID
	}

	return doc
}

func (s *ElasticsearchStorage) docToSpeakerEmbedding(doc map[string]interface{}) (*SpeakerEmbedding, error) {
	embedding := &SpeakerEmbedding{}

	if id, ok := doc["id"].(string); ok {
		embedding.ID = id
	}
	if recordingID, ok := doc["recording_id"].(string); ok {
		embedding.RecordingID = recordingID
	}
	if localSpeakerID, ok := doc["local_speaker_id"].(string); ok {
		embedding.LocalSpeakerID = localSpeakerID
	}
	if durationSeconds, ok := doc["duration_seconds"].(float64); ok {
		embedding.DurationSeconds = durationSeconds
	}

	// Parse embedding (float64[] -> float32[])
	if embeddingData, ok := doc["embedding"].([]interface{}); ok {
		embedding.Embedding = make([]float32, len(embeddingData))
		for i, v := range embeddingData {
			if f, ok := v.(float64); ok {
				embedding.Embedding[i] = float32(f)
			}
		}
	}

	// Parse dates
	if createdAt, ok := doc["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			embedding.CreatedAt = t
		}
	}

	// Parse optional fields
	if speakerID, ok := doc["speaker_id"].(string); ok {
		embedding.SpeakerID = &speakerID
	}

	return embedding, nil
}

func (s *ElasticsearchStorage) searchSpeakerEmbeddings(ctx context.Context, query map[string]interface{}) ([]*SpeakerEmbedding, error) {
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	res, err := s.client.Search(
		s.client.Search.WithIndex(indexSpeakerEmbeddings),
		s.client.Search.WithBody(bytes.NewReader(queryJSON)),
		s.client.Search.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search speaker embeddings: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("failed to search speaker embeddings: %s", string(body))
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	embeddings := make([]*SpeakerEmbedding, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		embedding, err := s.docToSpeakerEmbedding(hit.Source)
		if err != nil {
			log.Printf("Warning: failed to parse speaker embedding from search hit: %v", err)
			continue
		}
		embeddings = append(embeddings, embedding)
	}

	return embeddings, nil
}

