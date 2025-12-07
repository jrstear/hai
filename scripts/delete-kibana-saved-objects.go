package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	kibanaURL = flag.String("kibana-url", "", "Kibana URL (default: http://localhost:5601)")
	objects   = flag.String("objects", "segments,speakers,speaker_embeddings", "Comma-separated list of index pattern names to delete")
	confirm   = flag.Bool("confirm", false, "Confirm deletion (required to actually delete)")
	verbose   = flag.Bool("verbose", false, "Enable verbose logging")
)

func main() {
	flag.Parse()

	if !*confirm {
		log.Fatal("Error: -confirm flag is required to delete saved objects. This is a destructive operation.")
	}

	// Get Kibana URL
	url := *kibanaURL
	if url == "" {
		url = os.Getenv("KIBANA_URL")
		if url == "" {
			url = "http://localhost:5601"
		}
	}

	// Parse objects to delete
	objectList := strings.Split(*objects, ",")
	for i := range objectList {
		objectList[i] = strings.TrimSpace(objectList[i])
	}

	if *verbose {
		log.Printf("Kibana URL: %s", url)
		log.Printf("Index patterns to delete: %v", objectList)
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Check if Kibana is ready
	if err := waitForKibana(client, url); err != nil {
		log.Fatalf("Failed to connect to Kibana: %v", err)
	}

	// Delete each index pattern
	successCount := 0
	skipCount := 0
	errorCount := 0

	for _, patternName := range objectList {
		if patternName == "" {
			continue
		}

		deleted, err := deleteIndexPattern(client, url, patternName)
		if err != nil {
			log.Printf("Error deleting index pattern %s: %v", patternName, err)
			errorCount++
			continue
		}

		if deleted {
			successCount++
			fmt.Printf("Deleted index pattern: %s\n", patternName)
		} else {
			skipCount++
			if *verbose {
				log.Printf("Index pattern %s does not exist, skipping", patternName)
			}
		}
	}

	// Summary
	fmt.Printf("\nSummary: %d deleted, %d not found, %d errors\n", successCount, skipCount, errorCount)

	if errorCount > 0 {
		os.Exit(1)
	}
}

// waitForKibana waits for Kibana to be ready
func waitForKibana(client *http.Client, url string) error {
	maxAttempts := 10
	for attempt := 0; attempt < maxAttempts; attempt++ {
		resp, err := client.Get(url + "/api/status")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		if attempt < maxAttempts-1 {
			time.Sleep(1 * time.Second)
		}
	}
	return fmt.Errorf("Kibana not ready at %s", url)
}

// deleteIndexPattern deletes an index pattern from Kibana
// Returns true if deleted, false if not found, error on failure
func deleteIndexPattern(client *http.Client, kibanaURL, name string) (bool, error) {
	// Check if index pattern exists
	exists, err := indexPatternExists(client, kibanaURL, name)
	if err != nil {
		return false, fmt.Errorf("failed to check if index pattern exists: %w", err)
	}
	if !exists {
		return false, nil // Doesn't exist, skip
	}

	// Delete the index pattern
	url := fmt.Sprintf("%s/api/saved_objects/index-pattern/%s", kibanaURL, name)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("kbn-xsrf", "true")

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		return true, nil // Successfully deleted
	}

	if resp.StatusCode == 404 {
		return false, nil // Doesn't exist
	}

	// Other error
	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
}

// indexPatternExists checks if an index pattern exists
func indexPatternExists(client *http.Client, kibanaURL, name string) (bool, error) {
	url := fmt.Sprintf("%s/api/saved_objects/index-pattern/%s", kibanaURL, name)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("kbn-xsrf", "true")

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return true, nil // Exists
	}
	if resp.StatusCode == 404 {
		return false, nil // Doesn't exist
	}

	// Other status code - treat as error
	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
}












