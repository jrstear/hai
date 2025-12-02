package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

var (
	esURL     = flag.String("es-url", "", "Elasticsearch URL (default: from ELASTICSEARCH_URL env var)")
	startDate = flag.String("start-date", "", "Start date (YYYY-MM-DD format, inclusive)")
	endDate   = flag.String("end-date", "", "End date (YYYY-MM-DD format, exclusive, defaults to start-date + 1 day)")
	confirm   = flag.Bool("confirm", false, "Confirm deletion (required to actually delete)")
	verbose   = flag.Bool("verbose", false, "Enable verbose logging")
	deleteSpeakers = flag.Bool("delete-speakers", false, "Also delete speakers that were created from these recordings (WARNING: may delete speakers used by other recordings)")
)

const (
	indexSegments        = "segments"
	indexSpeakerEmbeddings = "speaker_embeddings"
	indexRecordings      = "recordings"
	indexSpeakers        = "speakers"
)

func main() {
	flag.Parse()

	if *startDate == "" {
		log.Fatal("Error: -start-date is required (format: YYYY-MM-DD)")
	}

	if !*confirm {
		log.Fatal("Error: -confirm flag is required to delete data. This is a destructive operation.")
	}

	// Parse dates
	start, err := time.Parse("2006-01-02", *startDate)
	if err != nil {
		log.Fatalf("Error: Invalid start-date format. Use YYYY-MM-DD: %v", err)
	}
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)

	end := start.AddDate(0, 0, 1) // Default: start + 1 day
	if *endDate != "" {
		end, err = time.Parse("2006-01-02", *endDate)
		if err != nil {
			log.Fatalf("Error: Invalid end-date format. Use YYYY-MM-DD: %v", err)
		}
		end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	}

	if end.Before(start) || end.Equal(start) {
		log.Fatal("Error: end-date must be after start-date")
	}

	// Get Elasticsearch URL
	url := *esURL
	if url == "" {
		url = os.Getenv("ELASTICSEARCH_URL")
		if url == "" {
			log.Fatal("Error: Elasticsearch URL not provided. Use -es-url flag or set ELASTICSEARCH_URL environment variable")
		}
	}

	if *verbose {
		log.Printf("Connecting to Elasticsearch at: %s", url)
		log.Printf("Deleting data for date range: %s to %s", start.Format("2006-01-02"), end.Format("2006-01-02"))
	}

	// Create Elasticsearch client
	cfg := elasticsearch.Config{
		Addresses: []string{url},
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create Elasticsearch client: %v", err)
	}

	// Check health first
	ctx := context.Background()
	res, err := client.Cluster.Health(client.Cluster.Health.WithContext(ctx))
	if err != nil {
		log.Fatalf("Failed to check Elasticsearch health: %v", err)
	}
	res.Body.Close()

	if *verbose {
		log.Println("Elasticsearch connection successful")
	}

	// 1. Find recordings in date range
	recordings, err := findRecordingsInRange(ctx, client, start, end)
	if err != nil {
		log.Fatalf("Failed to find recordings: %v", err)
	}

	if len(recordings) == 0 {
		fmt.Printf("No recordings found in date range %s to %s\n", start.Format("2006-01-02"), end.Format("2006-01-02"))
		return
	}

	fmt.Printf("Found %d recordings in date range:\n", len(recordings))
	for _, rec := range recordings {
		fmt.Printf("  - %s (start: %s)\n", rec.ID, rec.StartTime.Format("2006-01-02 15:04:05"))
	}

	recordingIDs := make([]string, len(recordings))
	for i, rec := range recordings {
		recordingIDs[i] = rec.ID
	}

	// 2. Delete segments for these recordings
	if err := deleteByRecordingIDs(ctx, client, indexSegments, "recording_id", recordingIDs, "segments"); err != nil {
		log.Printf("Error deleting segments: %v", err)
	}

	// 3. Delete speaker embeddings for these recordings
	if err := deleteByRecordingIDs(ctx, client, indexSpeakerEmbeddings, "recording_id", recordingIDs, "speaker embeddings"); err != nil {
		log.Printf("Error deleting speaker embeddings: %v", err)
	}

	// 4. Delete recordings
	deletedRecordings := 0
	for _, recID := range recordingIDs {
		if err := deleteRecording(ctx, client, recID); err != nil {
			log.Printf("Error deleting recording %s: %v", recID, err)
		} else {
			deletedRecordings++
		}
	}
	fmt.Printf("Deleted %d recordings\n", deletedRecordings)

	// 5. Optionally delete speakers (WARNING: may affect other recordings)
	if *deleteSpeakers {
		if *verbose {
			log.Println("WARNING: Deleting speakers - this may affect other recordings!")
		}
		// Find speakers that were created from embeddings in these recordings
		// This is tricky - we'd need to track which speakers came from which recordings
		// For now, we'll skip this and let the user manually delete speakers if needed
		log.Println("Note: Speaker deletion by date range is not yet implemented. Speakers are shared across recordings, so manual deletion may be needed.")
	}

	fmt.Printf("\nDeletion complete for date range: %s to %s\n", start.Format("2006-01-02"), end.Format("2006-01-02"))
}

type Recording struct {
	ID        string    `json:"id"`
	StartTime time.Time `json:"start_time"`
}

func findRecordingsInRange(ctx context.Context, client *elasticsearch.Client, start, end time.Time) ([]Recording, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"range": map[string]interface{}{
				"start_time": map[string]interface{}{
					"gte": start.Format(time.RFC3339),
					"lt":  end.Format(time.RFC3339),
				},
			},
		},
		"size": 10000, // TODO: Implement pagination if needed
		"sort": []map[string]interface{}{
			{"start_time": map[string]interface{}{"order": "asc"}},
		},
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	res, err := client.Search(
		client.Search.WithIndex(indexRecordings),
		client.Search.WithBody(bytes.NewReader(queryJSON)),
		client.Search.WithContext(ctx),
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
				Source Recording `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	recordings := make([]Recording, len(result.Hits.Hits))
	for i, hit := range result.Hits.Hits {
		recordings[i] = hit.Source
	}

	return recordings, nil
}

func deleteByRecordingIDs(ctx context.Context, client *elasticsearch.Client, index, field string, recordingIDs []string, description string) error {
	if len(recordingIDs) == 0 {
		return nil
	}

	// Build terms query for multiple recording IDs
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"terms": map[string]interface{}{
				field: recordingIDs,
			},
		},
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return fmt.Errorf("failed to marshal query: %w", err)
	}

	if *verbose {
		log.Printf("Deleting %s matching %d recording IDs...", description, len(recordingIDs))
	}

	// Execute delete-by-query
	req := esapi.DeleteByQueryRequest{
		Index: []string{index},
		Body:  bytes.NewReader(queryJSON),
	}

	res, err := req.Do(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to execute delete-by-query: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("delete-by-query failed: %s", string(body))
	}

	// Parse response to get deleted count
	var result struct {
		Deleted int64 `json:"deleted"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		if *verbose {
			log.Printf("Warning: Could not parse delete-by-query response: %v", err)
		}
	} else {
		fmt.Printf("Deleted %d %s\n", result.Deleted, description)
	}

	return nil
}

func deleteRecording(ctx context.Context, client *elasticsearch.Client, recordingID string) error {
	if *verbose {
		log.Printf("Deleting recording: %s...", recordingID)
	}

	res, err := client.Delete(indexRecordings, recordingID, client.Delete.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to delete recording: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		if *verbose {
			log.Printf("Recording %s does not exist, skipping", recordingID)
		}
		return fmt.Errorf("not found") // Return error so caller knows it wasn't deleted
	}

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to delete recording: %s", string(body))
	}

	return nil // Success
}

