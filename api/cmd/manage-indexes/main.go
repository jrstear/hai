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
	"strings"

	"hai/api/internal/contacts"

	"github.com/elastic/go-elasticsearch/v8"
)

var (
	esURL  = flag.String("es", "", "Elasticsearch URL (default: ELASTICSEARCH_URL env var)")
	index  = flag.String("index", "", "Index name to manage (contacts, or 'all' for all indexes)")
	action = flag.String("action", "rebuild", "Action: delete, create, or rebuild (default: rebuild)")
)

func main() {
	flag.Parse()

	// Get Elasticsearch URL
	esURLValue := *esURL
	if esURLValue == "" {
		esURLValue = os.Getenv("ELASTICSEARCH_URL")
		if esURLValue == "" {
			log.Fatal("Error: ELASTICSEARCH_URL environment variable or -es flag is required")
		}
	}

	if *index == "" {
		log.Fatal("Error: -index flag is required (contacts or all)")
	}

	// Create Elasticsearch client
	cfg := elasticsearch.Config{
		Addresses: []string{esURLValue},
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error: Failed to create Elasticsearch client: %v", err)
	}

	ctx := context.Background()

	// Determine which indexes to manage
	var indexes []string
	if *index == "all" {
		indexes = []string{
			"contacts",
			"speakers",
			"recordings",
			"segments",
			"lifelogs",
			"lifelog_blockquotes",
			"speaker_embeddings",
			"settings",
		}
	} else {
		indexes = []string{*index}
	}

	// Perform action
	switch *action {
	case "delete":
		for _, idx := range indexes {
			if err := deleteIndex(ctx, client, idx); err != nil {
				log.Printf("Warning: Failed to delete index %s: %v", idx, err)
			} else {
				fmt.Printf("✓ Deleted index: %s\n", idx)
			}
		}
	case "create":
		for _, idx := range indexes {
			if err := createIndex(ctx, client, idx); err != nil {
				log.Printf("Error: Failed to create index %s: %v", idx, err)
			} else {
				fmt.Printf("✓ Created index: %s\n", idx)
			}
		}
	case "rebuild":
		for _, idx := range indexes {
			fmt.Printf("Rebuilding index: %s\n", idx)
			// Delete if exists
			if err := deleteIndex(ctx, client, idx); err != nil {
				// Ignore "index not found" errors
				if !strings.Contains(err.Error(), "index_not_found_exception") {
					log.Printf("Warning: Failed to delete index %s: %v", idx, err)
				}
			} else {
				fmt.Printf("  ✓ Deleted existing index\n")
			}
			// Create
			if err := createIndex(ctx, client, idx); err != nil {
				log.Fatalf("Error: Failed to create index %s: %v", idx, err)
			}
			fmt.Printf("  ✓ Created index\n")
		}
		fmt.Println("\n✓ All indexes rebuilt successfully")
	default:
		log.Fatalf("Error: Unknown action: %s (must be delete, create, or rebuild)", *action)
	}
}

func deleteIndex(ctx context.Context, client *elasticsearch.Client, indexName string) error {
	res, err := client.Indices.Delete(
		[]string{indexName},
		client.Indices.Delete.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() && res.StatusCode != 404 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to delete index: status=%d, body=%s", res.StatusCode, string(body))
	}

	return nil
}

func createIndex(ctx context.Context, client *elasticsearch.Client, indexName string) error {
	switch indexName {
	case "contacts":
		contactsHandler := contacts.NewElasticsearchContacts(client)
		return contactsHandler.EnsureIndex(ctx)
	case "speakers", "recordings", "segments", "lifelogs", "lifelog_blockquotes", "speaker_embeddings", "settings":
		// Use storage package mapping definitions to create these indexes
		return createStorageIndex(ctx, client, indexName)
	default:
		return fmt.Errorf("index %s not supported", indexName)
	}
}

// createStorageIndex creates an index using the storage package's mapping definitions
func createStorageIndex(ctx context.Context, client *elasticsearch.Client, indexName string) error {
	// Get the mapping for this index from storage package
	mapping, err := getIndexMapping(indexName)
	if err != nil {
		return err
	}

	// Marshal mapping to JSON
	mappingJSON, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %w", err)
	}

	// Create index with mapping
	res, err := client.Indices.Create(
		indexName,
		client.Indices.Create.WithBody(bytes.NewReader(mappingJSON)),
		client.Indices.Create.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to create index: status=%d, body=%s", res.StatusCode, string(body))
	}

	return nil
}

// getIndexMapping returns the mapping definition for a given index name
// This matches the mappings defined in storage/elasticsearch.go ensureIndices
func getIndexMapping(indexName string) (map[string]interface{}, error) {
	baseSettings := map[string]interface{}{
		"number_of_shards":   1,
		"number_of_replicas": 0,
	}

	switch indexName {
	case "speakers":
		return map[string]interface{}{
			"mappings": map[string]interface{}{
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "keyword"},
					"embedding": map[string]interface{}{
						"type":       "dense_vector",
						"dims":       256,
						"index":      true,
						"similarity": "cosine",
					},
					"first_seen": map[string]interface{}{"type": "date"},
					"last_seen":  map[string]interface{}{"type": "date"},
					"contact_id": map[string]interface{}{
						"type":  "keyword",
						"index": false,
					},
					"created_at": map[string]interface{}{"type": "date"},
					"updated_at": map[string]interface{}{"type": "date"},
				},
			},
			"settings": baseSettings,
		}, nil

	case "recordings":
		return map[string]interface{}{
			"mappings": map[string]interface{}{
				"properties": map[string]interface{}{
					"id":               map[string]interface{}{"type": "keyword"},
					"file_path":        map[string]interface{}{"type": "keyword"},
					"start_time":       map[string]interface{}{"type": "date"},
					"duration_seconds": map[string]interface{}{"type": "float"},
					"sample_rate":      map[string]interface{}{"type": "integer"},
					"format":           map[string]interface{}{"type": "keyword"},
					"diarized_at":      map[string]interface{}{"type": "date"},
					"processing_time":  map[string]interface{}{"type": "float"},
					"rtf":              map[string]interface{}{"type": "float"},
					"device":           map[string]interface{}{"type": "keyword"},
					"created_at":       map[string]interface{}{"type": "date"},
				},
			},
			"settings": baseSettings,
		}, nil

	case "segments":
		return map[string]interface{}{
			"mappings": map[string]interface{}{
				"properties": map[string]interface{}{
					"id":                   map[string]interface{}{"type": "keyword"},
					"speaker_embedding_id": map[string]interface{}{"type": "keyword"},
					"speaker_id":           map[string]interface{}{"type": "keyword"},
					"recording_id":         map[string]interface{}{"type": "keyword"},
					"local_speaker_id":     map[string]interface{}{"type": "keyword"},
					"blockquote_id":        map[string]interface{}{"type": "keyword"},
					"start_time":           map[string]interface{}{"type": "float"},
					"end_time":             map[string]interface{}{"type": "float"},
					"duration":             map[string]interface{}{"type": "float"},
					"start_byte_offset":    map[string]interface{}{"type": "long"},
					"end_byte_offset":      map[string]interface{}{"type": "long"},
					"created_at":           map[string]interface{}{"type": "date"},
				},
			},
			"settings": baseSettings,
		}, nil

	case "lifelogs":
		return map[string]interface{}{
			"mappings": map[string]interface{}{
				"properties": map[string]interface{}{
					"id":         map[string]interface{}{"type": "keyword"},
					"title":      map[string]interface{}{"type": "text"},
					"markdown":   map[string]interface{}{"type": "text"},
					"start_time": map[string]interface{}{"type": "date"},
					"end_time":   map[string]interface{}{"type": "date"},
					"created_at": map[string]interface{}{"type": "date"},
				},
			},
			"settings": baseSettings,
		}, nil

	case "lifelog_blockquotes":
		return map[string]interface{}{
			"mappings": map[string]interface{}{
				"properties": map[string]interface{}{
					"id":              map[string]interface{}{"type": "keyword"},
					"lifelog_id":      map[string]interface{}{"type": "keyword"},
					"recording_id":    map[string]interface{}{"type": "keyword"},
					"content":         map[string]interface{}{"type": "text"},
					"speaker_name":    map[string]interface{}{"type": "keyword"},
					"speaker_id":      map[string]interface{}{"type": "keyword"},
					"start_offset_ms": map[string]interface{}{"type": "integer"},
					"end_offset_ms":   map[string]interface{}{"type": "integer"},
					"start_time":      map[string]interface{}{"type": "date"},
					"end_time":        map[string]interface{}{"type": "date"},
					"created_at":      map[string]interface{}{"type": "date"},
				},
			},
			"settings": baseSettings,
		}, nil

	case "speaker_embeddings":
		return map[string]interface{}{
			"mappings": map[string]interface{}{
				"properties": map[string]interface{}{
					"id":               map[string]interface{}{"type": "keyword"},
					"speaker_id":       map[string]interface{}{"type": "keyword"},
					"recording_id":     map[string]interface{}{"type": "keyword"},
					"local_speaker_id": map[string]interface{}{"type": "keyword"},
					"embedding": map[string]interface{}{
						"type":       "dense_vector",
						"dims":       256,
						"index":      true,
						"similarity": "cosine",
					},
					"duration_seconds": map[string]interface{}{"type": "float"},
					"created_at":       map[string]interface{}{"type": "date"},
				},
			},
			"settings": baseSettings,
		}, nil

	case "settings":
		return map[string]interface{}{
			"mappings": map[string]interface{}{
				"properties": map[string]interface{}{
					"key": map[string]interface{}{"type": "keyword"},
					"value": map[string]interface{}{
						"type": "text",
						"fields": map[string]interface{}{
							"keyword": map[string]interface{}{
								"type": "keyword",
							},
						},
					},
					"created_at": map[string]interface{}{"type": "date"},
					"updated_at": map[string]interface{}{"type": "date"},
				},
			},
			"settings": baseSettings,
		}, nil

	default:
		return nil, fmt.Errorf("unknown index: %s", indexName)
	}
}
