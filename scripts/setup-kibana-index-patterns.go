package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	kibanaURL = flag.String("kibana-url", "", "Kibana URL (default: http://localhost:5601)")
	verbose   = flag.Bool("verbose", false, "Enable verbose logging")
)

// IndexPatternConfig defines an index pattern to create
type IndexPatternConfig struct {
	Name      string
	TimeField string
}

// IndexPatternRequest is the request body for creating an index pattern
type IndexPatternRequest struct {
	Attributes struct {
		Title       string `json:"title"`
		TimeFieldName string `json:"timeFieldName"`
	} `json:"attributes"`
}

// IndexPatternResponse is the response from Kibana API
type IndexPatternResponse struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Title       string `json:"title"`
		TimeFieldName string `json:"timeFieldName"`
	} `json:"attributes"`
	Error *struct {
		Type     string `json:"type"`
		Message  string `json:"message"`
	} `json:"error,omitempty"`
}

// Index patterns to create
var indexPatterns = []IndexPatternConfig{
	{Name: "speakers", TimeField: "created_at"},
	{Name: "recordings", TimeField: "start_time"},
	{Name: "segments", TimeField: "created_at"},
	{Name: "lifelogs", TimeField: "start_time"},
	{Name: "lifelog_blockquotes", TimeField: "start_time"},
}

func main() {
	flag.Parse()

	// Get Kibana URL
	url := *kibanaURL
	if url == "" {
		url = os.Getenv("KIBANA_URL")
		if url == "" {
			url = "http://localhost:5601"
		}
	}

	if *verbose {
		log.Printf("Kibana URL: %s", url)
	}

	// Wait for Kibana to be ready
	if err := waitForKibana(url); err != nil {
		log.Fatalf("Failed to wait for Kibana: %v", err)
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create index patterns
	successCount := 0
	skipCount := 0
	errorCount := 0

	for _, pattern := range indexPatterns {
		created, err := createIndexPattern(client, url, pattern)
		if err != nil {
			log.Printf("Error creating index pattern %s: %v", pattern.Name, err)
			errorCount++
			continue
		}

		if created {
			successCount++
			if *verbose {
				log.Printf("Created index pattern: %s (time field: %s)", pattern.Name, pattern.TimeField)
			} else {
				fmt.Printf("Created index pattern: %s\n", pattern.Name)
			}
		} else {
			skipCount++
			if *verbose {
				log.Printf("Index pattern %s already exists, skipping", pattern.Name)
			}
		}
	}

	// Summary
	fmt.Printf("\nSummary: %d created, %d already existed, %d errors\n", successCount, skipCount, errorCount)

	if errorCount > 0 {
		os.Exit(1)
	}
}

// waitForKibana waits for Kibana to be ready by checking the status endpoint
func waitForKibana(url string) error {
	maxAttempts := 60
	attempt := 0

	for attempt < maxAttempts {
		resp, err := http.Get(url + "/api/status")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				if *verbose {
					log.Printf("Kibana is ready")
				}
				return nil
			}
		}

		attempt++
		if *verbose {
			log.Printf("Waiting for Kibana... (attempt %d/%d)", attempt, maxAttempts)
		}
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("Kibana did not become ready after %d attempts", maxAttempts)
}

// createIndexPattern creates an index pattern in Kibana
// Returns true if created, false if already exists, error on failure
func createIndexPattern(client *http.Client, kibanaURL string, config IndexPatternConfig) (bool, error) {
	// Check if index pattern already exists
	exists, err := indexPatternExists(client, kibanaURL, config.Name)
	if err != nil {
		return false, fmt.Errorf("failed to check if index pattern exists: %w", err)
	}
	if exists {
		return false, nil // Already exists, skip
	}

	// Create index pattern
	reqBody := IndexPatternRequest{}
	reqBody.Attributes.Title = config.Name
	reqBody.Attributes.TimeFieldName = config.TimeField

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return false, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/saved_objects/index-pattern/%s", kibanaURL, config.Name)
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("kbn-xsrf", "true")

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		// Success
		var result IndexPatternResponse
		if err := json.Unmarshal(body, &result); err == nil && result.Error != nil {
			return false, fmt.Errorf("Kibana API error: %s", result.Error.Message)
		}
		return true, nil
	}

	// Check if it's a conflict (already exists)
	if resp.StatusCode == 409 {
		return false, nil // Already exists
	}

	// Other error
	return false, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
}

// indexPatternExists checks if an index pattern already exists
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

