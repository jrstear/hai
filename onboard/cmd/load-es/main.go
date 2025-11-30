package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"hai/onboard/internal/diarization"
	"hai/onboard/internal/export2elastic"
	"hai/storage"
)

var (
	diarizationFile = flag.String("diarization", "", "Path to diarization JSON file (required)")
	audioFile       = flag.String("audio", "", "Path to audio file (optional, will derive from diarization file if not provided)")
	esURL           = flag.String("es-url", "", "Elasticsearch URL (default: from ELASTICSEARCH_URL env var)")
	verbose         = flag.Bool("verbose", false, "Enable verbose logging")
)

func main() {
	flag.Parse()

	// Validate required flags
	if *diarizationFile == "" {
		fmt.Fprintf(os.Stderr, "Error: -diarization flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Get Elasticsearch URL
	url := *esURL
	if url == "" {
		url = os.Getenv("ELASTICSEARCH_URL")
		if url == "" {
			log.Fatal("Error: Elasticsearch URL not provided. Use -es-url flag or set ELASTICSEARCH_URL environment variable")
		}
	}

	// Derive audio file path if not provided
	audioPath := *audioFile
	if audioPath == "" {
		// Try to derive from diarization file path
		// e.g., data/2025/11/22/15.json -> data/2025/11/22/15.ogg
		diarizationPath := *diarizationFile
		ext := filepath.Ext(diarizationPath)
		baseWithoutExt := diarizationPath[:len(diarizationPath)-len(ext)]
		audioPath = baseWithoutExt + ".ogg"
		if *verbose {
			log.Printf("Derived audio file path: %s", audioPath)
		}
	}

	// Read diarization JSON file
	if *verbose {
		log.Printf("Reading diarization file: %s", *diarizationFile)
	}
	data, err := os.ReadFile(*diarizationFile)
	if err != nil {
		log.Fatalf("Failed to read diarization file: %v", err)
	}

	// Parse diarization result
	var result diarization.Result
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("Failed to parse diarization JSON: %v", err)
	}

	if *verbose {
		log.Printf("Diarization result: %d speakers, %d segments", result.SpeakerCount, result.SegmentCount)
	}

	// Initialize Elasticsearch storage
	if *verbose {
		log.Printf("Connecting to Elasticsearch at: %s", url)
	}
	esStorage, err := storage.NewElasticsearchStorage(url)
	if err != nil {
		log.Fatalf("Failed to create Elasticsearch storage: %v", err)
	}
	defer esStorage.Close()

	// Create exporter
	exporter := export2elastic.NewExporter(esStorage)

	// Export to Elasticsearch
	relPath := export2elastic.ExtractRelativePathForOutput(*diarizationFile)
	
	ctx := context.Background()
	_, _, wasSkipped, err := exporter.ExportResult(ctx, &result, audioPath)
	if err != nil {
		log.Fatalf("Failed to load %s to Elasticsearch: %v", relPath, err)
	}
	
	if wasSkipped {
		fmt.Printf("data/%s already exists - skipping loading Elasticsearch.\n", relPath)
	} else {
		fmt.Printf("Loading %s to Elasticsearch\n", relPath)
	}
}

