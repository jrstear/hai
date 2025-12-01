package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"hai/onboard/internal/export2elastic"
	"hai/storage"
)

var (
	startDate = flag.String("start", "", "Start date (YYYY-MM-DD) or datetime (YYYY-MM-DD HH:MM:SS) (required)")
	endDate   = flag.String("end", "", "End date (YYYY-MM-DD) or datetime (YYYY-MM-DD HH:MM:SS) (required)")
	lifelogID = flag.String("lifelog", "", "Lifelog ID to map (alternative to date range)")
	esURL     = flag.String("es-url", "", "Elasticsearch URL (default: from ELASTICSEARCH_URL env var)")
	reprocess = flag.Bool("reprocess", false, "Reprocess already-mapped blockquotes")
	verbose   = flag.Bool("verbose", false, "Enable verbose logging")
)

func main() {
	flag.Parse()

	// Validate flags
	if *lifelogID == "" && (*startDate == "" || *endDate == "") {
		fmt.Fprintf(os.Stderr, "Error: Either -lifelog or both -start and -end are required\n")
		flag.Usage()
		os.Exit(1)
	}

	if *lifelogID != "" && (*startDate != "" || *endDate != "") {
		fmt.Fprintf(os.Stderr, "Error: Cannot specify both -lifelog and date range\n")
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

	ctx := context.Background()
	var matched int

	// Map by lifelog ID or date range
	if *lifelogID != "" {
		if *verbose {
			log.Printf("Mapping speaker names for lifelog: %s", *lifelogID)
		}
		matched, err = exporter.MapSpeakerNamesForLifelog(ctx, *lifelogID, *reprocess)
		if err != nil {
			log.Fatalf("Failed to map speaker names for lifelog %s: %v", *lifelogID, err)
		}
	} else {
		// Parse date range
		startTime, err := parseDateTime(*startDate)
		if err != nil {
			log.Fatalf("Failed to parse start date: %v", err)
		}
		endTime, err := parseDateTime(*endDate)
		if err != nil {
			log.Fatalf("Failed to parse end date: %v", err)
		}

		if endTime.Before(startTime) || endTime.Equal(startTime) {
			log.Fatal("Error: end date must be after start date")
		}

		if *verbose {
			log.Printf("Mapping speaker names for time range: %s to %s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
		}
		matched, err = exporter.MapSpeakerNamesForTimeRange(ctx, startTime, endTime, *reprocess)
		if err != nil {
			log.Fatalf("Failed to map speaker names for time range: %v", err)
		}
	}

	fmt.Printf("Speaker name mapping complete: %d blockquotes matched\n", matched)
}

// parseDateTime parses a date string in either YYYY-MM-DD or YYYY-MM-DD HH:MM:SS format
func parseDateTime(dateStr string) (time.Time, error) {
	// Try full datetime format first
	if t, err := time.Parse("2006-01-02 15:04:05", dateStr); err == nil {
		return t.UTC(), nil
	}
	// Try date-only format (assume start of day)
	if t, err := time.Parse("2006-01-02", dateStr); err == nil {
		return t.UTC(), nil
	}
	// Try RFC3339 format
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("invalid date format: %s (expected YYYY-MM-DD or YYYY-MM-DD HH:MM:SS)", dateStr)
}

