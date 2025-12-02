package main

import (
	"context"
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
	esURL     = flag.String("es", "", "Elasticsearch URL (default: ELASTICSEARCH_URL env var)")
	index     = flag.String("index", "", "Index name to manage (contacts, or 'all' for all indexes)")
	action    = flag.String("action", "rebuild", "Action: delete, create, or rebuild (default: rebuild)")
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
	default:
		return fmt.Errorf("index %s not yet supported in this utility (only 'contacts' is supported)", indexName)
	}
}

