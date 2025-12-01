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
// date should be in UTC for directory structure consistency
// timezone is used for the API call to determine which date to fetch in that timezone
func FetchLifelogs(apiKey string, date time.Time, timezone string, outputDir string, reprocess bool) (string, bool, error) {
	// Ensure date is in UTC for directory structure
	dateUTC := date.UTC()

	// Load timezone for API call
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return "", false, fmt.Errorf("invalid timezone: %w", err)
	}

	// Convert UTC date to user's timezone to get the date string for API call
	// The API expects the date in the specified timezone
	dateInTZ := dateUTC.In(loc)
	dateStr := dateInTZ.Format("2006-01-02")

	// Create directory structure: YYYY/MM/DD/lifelog.json (in UTC)
	relPath := filepath.Join(
		dateUTC.Format("2006"),
		dateUTC.Format("01"),
		dateUTC.Format("02"),
		"lifelog.json",
	)
	outputPath := filepath.Join(outputDir, relPath)

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", false, fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if file already exists
	if _, err := os.Stat(outputPath); err == nil {
		if reprocess {
			// Remove existing file to force re-fetch
			if err := os.Remove(outputPath); err != nil {
				return "", false, fmt.Errorf("failed to remove existing file: %w", err)
			}
			fmt.Printf("Removed existing lifelog file (reprocess=true): %s\n", outputPath)
		} else {
			// Use relative path for cleaner output
			relPathForMsg := filepath.Join(
				dateUTC.Format("2006"),
				dateUTC.Format("01"),
				dateUTC.Format("02"),
				"lifelog.json",
			)
			fmt.Printf("data/%s already exists - skipping download.\n", relPathForMsg)
			return outputPath, true, nil // File already exists, skip fetch
		}
	}

	// Starting download - include date path for clarity
	relPathForMsg := filepath.Join(
		dateUTC.Format("2006"),
		dateUTC.Format("01"),
		dateUTC.Format("02"),
		"lifelog.json",
	)
	fmt.Printf("Downloading %s\n", relPathForMsg)

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

// ParseLifelogForAudioHours parses a lifelog JSON file and returns the set of hours (in UTC) that have audio
// Returns:
//   - hoursWithAudio: map[string]bool where key is "YYYY-MM-DD HH:00" format (UTC)
//   - hasAudio: true if any hours have audio, false if lifelog is null/empty/no blockquotes
//   - error: if file cannot be read or parsed
func ParseLifelogForAudioHours(lifelogPath string, timezone string) (map[string]bool, bool, error) {
	// Read file
	data, err := os.ReadFile(lifelogPath)
	if err != nil {
		return nil, false, err
	}

	// Handle null case
	var lifelogs []Lifelog
	trimmed := string(data)
	if trimmed == "null\n" || trimmed == "null" || trimmed == "null\r\n" {
		return make(map[string]bool), false, nil
	}

	// Parse JSON
	if err := json.Unmarshal(data, &lifelogs); err != nil {
		return nil, false, fmt.Errorf("failed to parse lifelog: %w", err)
	}

	// Handle empty array
	if len(lifelogs) == 0 {
		return make(map[string]bool), false, nil
	}

	// Load timezone for conversion (though we'll work in UTC for hour keys)
	_, err = time.LoadLocation(timezone)
	if err != nil {
		return nil, false, fmt.Errorf("invalid timezone: %w", err)
	}

	hoursWithAudio := make(map[string]bool)
	hasAnyAudio := false

	// Process each lifelog
	for _, lifelog := range lifelogs {
		// Check if lifelog has any blockquotes
		hasBlockquotes := false
		for _, content := range lifelog.Contents {
			if content.Type == "blockquote" {
				hasBlockquotes = true
				break
			}
		}

		if !hasBlockquotes {
			continue // Skip lifelogs without audio
		}

		hasAnyAudio = true

		// Lifelog times are already parsed as time.Time from JSON
		// Convert to UTC for hour calculation
		startUTC := lifelog.StartTime.UTC()
		endUTC := lifelog.EndTime.UTC()

		// Generate all hours between start and end (inclusive)
		// Round start time down to the hour
		startHour := time.Date(
			startUTC.Year(), startUTC.Month(), startUTC.Day(),
			startUTC.Hour(), 0, 0, 0,
			time.UTC,
		)

		// Round end time down to the hour (inclusive)
		endHour := time.Date(
			endUTC.Year(), endUTC.Month(), endUTC.Day(),
			endUTC.Hour(), 0, 0, 0,
			time.UTC,
		)

		// Add all hours from start to end (inclusive)
		current := startHour
		for current.Before(endHour) || current.Equal(endHour) {
			hourKey := current.Format("2006-01-02 15:00")
			hoursWithAudio[hourKey] = true
			current = current.Add(1 * time.Hour)
		}
	}

	return hoursWithAudio, hasAnyAudio, nil
}
