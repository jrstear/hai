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

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

var (
	esURL       = flag.String("es-url", "", "Elasticsearch URL (default: from ELASTICSEARCH_URL env var)")
	recordingID = flag.String("recording-id", "", "Recording ID to delete data for (required)")
	confirm     = flag.Bool("confirm", false, "Confirm deletion (required to actually delete)")
	verbose     = flag.Bool("verbose", false, "Enable verbose logging")
)

const (
	indexSegments        = "segments"
	indexSpeakerEmbeddings = "speaker_embeddings"
	indexRecordings      = "recordings"
)

func main() {
	flag.Parse()

	if *recordingID == "" {
		log.Fatal("Error: -recording-id is required")
	}

	if !*confirm {
		log.Fatal("Error: -confirm flag is required to delete data. This is a destructive operation.")
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
		log.Printf("Deleting data for recording: %s", *recordingID)
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

	// Delete segments
	if err := deleteByQuery(ctx, client, indexSegments, "recording_id", *recordingID, "segments"); err != nil {
		log.Printf("Error deleting segments: %v", err)
	}

	// Delete speaker embeddings
	if err := deleteByQuery(ctx, client, indexSpeakerEmbeddings, "recording_id", *recordingID, "speaker embeddings"); err != nil {
		log.Printf("Error deleting speaker embeddings: %v", err)
	}

	// Delete recording
	if err := deleteRecording(ctx, client, *recordingID); err != nil {
		log.Printf("Error deleting recording: %v", err)
	}

	fmt.Printf("\nDeletion complete for recording: %s\n", *recordingID)
}

func deleteByQuery(ctx context.Context, client *elasticsearch.Client, index, field, value, description string) error {
	// Build delete-by-query request
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				field: value,
			},
		},
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return fmt.Errorf("failed to marshal query: %w", err)
	}

	if *verbose {
		log.Printf("Deleting %s matching %s=%s...", description, field, value)
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
		// If we can't parse the response, that's okay - deletion might have succeeded
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
		return nil
	}

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to delete recording: %s", string(body))
	}

	fmt.Printf("Deleted recording: %s\n", recordingID)
	return nil
}

