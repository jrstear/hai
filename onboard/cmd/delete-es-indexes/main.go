package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
)

var (
	esURL   = flag.String("es-url", "", "Elasticsearch URL (default: from ELASTICSEARCH_URL env var)")
	indexes = flag.String("indexes", "", "Comma-separated list of indexes to delete (default: all indexes)")
	confirm = flag.Bool("confirm", false, "Confirm deletion (required to actually delete)")
	verbose = flag.Bool("verbose", false, "Enable verbose logging")
)

// All indexes that can be deleted
var allIndexes = []string{
	"speakers",
	"recordings",
	"segments",
	"speaker_embeddings",
	"lifelogs",
	"lifelog_blockquotes",
}

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

	if !*confirm {
		log.Fatal("Error: -confirm flag is required to delete indexes. This is a destructive operation.")
	}

	// Parse indexes to delete
	var indexList []string
	if *indexes == "" {
		// Default: delete all indexes
		indexList = allIndexes
		if *verbose {
			log.Printf("No indexes specified, will delete all indexes: %v", indexList)
		}
	} else {
		// Parse comma-separated list
		indexList = strings.Split(*indexes, ",")
		for i := range indexList {
			indexList[i] = strings.TrimSpace(indexList[i])
		}
	}

	if *verbose {
		log.Printf("Connecting to Elasticsearch at: %s", url)
		log.Printf("Indexes to delete: %v", indexList)
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

	// Delete each index
	for _, indexName := range indexList {
		if indexName == "" {
			continue
		}

		// Check if index exists
		existsRes, err := client.Indices.Exists([]string{indexName})
		if err != nil {
			log.Printf("Warning: Failed to check if index %s exists: %v", indexName, err)
			continue
		}
		existsRes.Body.Close()

		if existsRes.StatusCode == 404 {
			log.Printf("Index %s does not exist - skipping", indexName)
			continue
		}

		// Delete the index
		if *verbose {
			log.Printf("Deleting index: %s", indexName)
		}

		deleteRes, err := client.Indices.Delete(
			[]string{indexName},
			client.Indices.Delete.WithContext(ctx),
		)
		if err != nil {
			log.Printf("Error: Failed to delete index %s: %v", indexName, err)
			continue
		}
		defer deleteRes.Body.Close()

		if deleteRes.IsError() {
			body, _ := io.ReadAll(deleteRes.Body)
			log.Printf("Error: Failed to delete index %s: %s", indexName, string(body))
			continue
		}

		fmt.Printf("Deleted index: %s\n", indexName)
	}

	fmt.Println("\nIndex deletion complete. New indexes will be created automatically when storage is initialized.")
}

