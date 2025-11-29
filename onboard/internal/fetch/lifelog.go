package fetch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// Lifelog represents a lifelog entry from Limitless API
type Lifelog struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Markdown  string    `json:"markdown"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	Contents  []Content `json:"contents"`
}

// Content represents content within a lifelog
type Content struct {
	Type              string  `json:"type"`
	Content           string  `json:"content"`
	StartTime         string  `json:"startTime,omitempty"`
	EndTime           string  `json:"endTime,omitempty"`
	StartOffsetMs     int     `json:"startOffsetMs,omitempty"`
	EndOffsetMs       int     `json:"endOffsetMs,omitempty"`
	SpeakerName       string  `json:"speakerName,omitempty"`
	SpeakerIdentifier *string `json:"speakerIdentifier,omitempty"`
}

// LifelogResponse represents the API response
type LifelogResponse struct {
	Data struct {
		Lifelogs []Lifelog `json:"lifelogs"`
	} `json:"data"`
	Meta struct {
		Lifelogs struct {
			NextCursor *string `json:"nextCursor"`
			Count      int     `json:"count"`
		} `json:"lifelogs"`
	} `json:"meta"`
}

// FetchLifelogs fetches lifelogs for a specific date
// Returns the file path and a boolean indicating if the file already existed
func FetchLifelogs(apiKey string, date time.Time, timezone string, outputDir string, reprocess bool) (string, bool, error) {
	dateStr := date.Format("2006-01-02")
	
	// Create directory structure: YYYY/MM/DD/lifelogs.json
	relPath := filepath.Join(
		date.Format("2006"),
		date.Format("01"),
		date.Format("02"),
		"lifelogs.json",
	)
	outputPath := filepath.Join(outputDir, relPath)
	
	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", false, fmt.Errorf("failed to create directory: %w", err)
	}
	if _, err := os.Stat(outputPath); err == nil {
		if reprocess {
			// Remove existing file to force re-fetch
			if err := os.Remove(outputPath); err != nil {
				return "", false, fmt.Errorf("failed to remove existing file: %w", err)
			}
		} else {
			return outputPath, true, nil // File already exists, skip fetch
		}
	}
	
	var allLifelogs []Lifelog
	var cursor *string

	for {
		params := url.Values{}
		params.Set("date", dateStr)
		params.Set("timezone", timezone)
		params.Set("includeMarkdown", "true")
		params.Set("includeContents", "true")

		if cursor != nil {
			params.Set("cursor", *cursor)
		}

		reqURL := fmt.Sprintf("%s/lifelogs?%s", apiBaseURL, params.Encode())

		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			return "", false, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("X-API-Key", apiKey)

		// Use a longer timeout for lifelog fetches (they can be large)
		// Also set a context with timeout to allow cancellation
		client := &http.Client{
			Timeout: 5 * time.Minute, // 5 minutes for large lifelog fetches
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", false, fmt.Errorf("failed to fetch lifelogs: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return "", false, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		var result LifelogResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return "", false, fmt.Errorf("failed to decode response: %w", err)
		}

		allLifelogs = append(allLifelogs, result.Data.Lifelogs...)

		if result.Meta.Lifelogs.NextCursor == nil {
			break
		}
		cursor = result.Meta.Lifelogs.NextCursor
	}

	// Save to file
	data, err := json.MarshalIndent(allLifelogs, "", "  ")
	if err != nil {
		return "", false, fmt.Errorf("failed to marshal lifelogs: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return "", false, fmt.Errorf("failed to write lifelog file: %w", err)
	}

	return outputPath, false, nil
}

