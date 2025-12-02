package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"hai/storage"
)

var (
	esURL   = flag.String("es-url", "", "Elasticsearch URL (default: from ELASTICSEARCH_URL env var)")
	verbose = flag.Bool("verbose", false, "Enable verbose logging")
)

func main() {
	flag.Parse()

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
	}

	// Create Elasticsearch storage - this will automatically create indexes with proper mappings
	ctx := context.Background()
	esStorage, err := storage.NewElasticsearchStorage(url)
	if err != nil {
		log.Fatalf("Failed to create Elasticsearch storage: %v", err)
	}
	defer esStorage.Close()

	// Check health
	if err := esStorage.Health(ctx); err != nil {
		log.Fatalf("Elasticsearch health check failed: %v", err)
	}

	fmt.Println("Elasticsearch indexes created successfully:")
	fmt.Println("  - speakers")
	fmt.Println("  - recordings")
	fmt.Println("  - segments (with speaker_embedding_id and optional speaker_id)")
	fmt.Println("  - speaker_embeddings (new)")
	fmt.Println("  - lifelogs")
	fmt.Println("  - lifelog_blockquotes")
	fmt.Println("\nAll indexes are ready for data.")
}









