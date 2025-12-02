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
	esURL       = flag.String("es-url", "", "Elasticsearch URL (default: from ELASTICSEARCH_URL env var)")
	eps         = flag.Float64("eps", 0.15, "DBSCAN eps parameter (cosine distance threshold, default: 0.15)")
	minSamples  = flag.Int("min-samples", 2, "DBSCAN minSamples parameter (minimum points to form cluster, default: 2)")
	recluster   = flag.Bool("recluster-all", false, "Re-cluster all embeddings (including already clustered). Use for periodic full re-clustering.")
	verbose     = flag.Bool("verbose", false, "Enable verbose logging")
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
		log.Printf("Clustering config: eps=%.3f, minSamples=%d", *eps, *minSamples)
	}

	// Create Elasticsearch storage
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

	// Check if there are any embeddings
	embeddings, err := esStorage.ListAllEmbeddings(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to load embeddings: %v", err)
	}

	if len(embeddings) == 0 {
		fmt.Println("No embeddings found. Nothing to cluster.")
		return
	}

	if *verbose {
		log.Println("Running DBSCAN clustering...")
	}

	// Configure clustering with progress callback
	config := storage.ClusterSpeakersConfig{
		Eps:        *eps,
		MinSamples: *minSamples,
	}

	// Add progress callback if verbose
	if *verbose {
		config.Progress = func(message string) {
			fmt.Println(message)
		}
	}

	// Run clustering (incremental or full re-cluster)
	var result *storage.ClusterSpeakersResult
	if *recluster {
		if *verbose {
			log.Println("Running full re-clustering (all embeddings)...")
		}
		result, err = storage.ReclusterAllSpeakers(ctx, esStorage, config)
		if err != nil {
			log.Fatalf("Re-clustering failed: %v", err)
		}
		fmt.Printf("Re-clustering complete! Processed %d embeddings. Found %d clusters and %d singletons, created %d new speakers.\n",
			result.EmbeddingsProcessed,
			result.ClustersFound,
			result.SingletonsFound,
			result.SpeakersCreated)
	} else {
		result, err = storage.ClusterSpeakers(ctx, esStorage, config)
		if err != nil {
			log.Fatalf("Clustering failed: %v", err)
		}

		// Print enhanced summary
		if result.EmbeddingsMatched > 0 {
			fmt.Printf("Clustering complete! Processed %d embeddings. Matched %d to existing speakers, found %d clusters and %d singletons, created %d new speakers.\n",
				result.EmbeddingsProcessed,
				result.EmbeddingsMatched,
				result.ClustersFound,
				result.SingletonsFound,
				result.SpeakersCreated)
		} else {
			fmt.Printf("Clustering complete! Processed %d embeddings. Found %d clusters and %d singletons, created %d speakers.\n",
				result.EmbeddingsProcessed,
				result.ClustersFound,
				result.SingletonsFound,
				result.SpeakersCreated)
		}
	}
}

