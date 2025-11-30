package fetch

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	apiBaseURL = "https://api.limitless.ai/v1"
)

// FetchAudio downloads audio from Limitless API for the given time range
// Returns the file path and a boolean indicating if the file already existed
func FetchAudio(apiKey string, startTime, endTime time.Time, outputDir string, reprocess bool) (string, bool, error) {
	// Ensure times are in UTC for directory structure consistency
	startUTC := startTime.UTC()

	// Create output directory structure: YYYY/MM/DD/HH.ogg (in UTC)
	relPath := filepath.Join(
		startUTC.Format("2006"),
		startUTC.Format("01"),
		startUTC.Format("02"),
		fmt.Sprintf("%s.ogg", startUTC.Format("15")),
	)
	fullPath := filepath.Join(outputDir, relPath)

	// Check if file already exists
	if _, err := os.Stat(fullPath); err == nil {
		if reprocess {
			// Remove existing file to force re-download
			if err := os.Remove(fullPath); err != nil {
				return "", false, fmt.Errorf("failed to remove existing file: %w", err)
			}
			fmt.Printf("Removed existing audio file (reprocess=true): %s\n", fullPath)
		} else {
			// Use relative path for cleaner output
			fmt.Printf("data/%s already exists - skipping download.\n", relPath)
			return fullPath, true, nil // File already exists, skip download
		}
	}

	// Starting download - print relative path for cleaner output (in UTC)
	audioRelPath := filepath.Join(
		startUTC.Format("2006"),
		startUTC.Format("01"),
		startUTC.Format("02"),
		fmt.Sprintf("%s.ogg", startUTC.Format("15")),
	)
	fmt.Printf("Downloading audio %s\n", audioRelPath)

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", false, fmt.Errorf("failed to create directory: %w", err)
	}

	// Convert to milliseconds
	startMs := startTime.UnixMilli()
	endMs := endTime.UnixMilli()

	url := fmt.Sprintf("%s/download-audio?startMs=%d&endMs=%d", apiBaseURL, startMs, endMs)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-Key", apiKey)

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("failed to download audio: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", false, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Save to file
	out, err := os.Create(fullPath)
	if err != nil {
		return "", false, fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", false, fmt.Errorf("failed to write audio file: %w", err)
	}

	return fullPath, false, nil
}

// FetchAudioRange downloads multiple 1-hour chunks for a date range
// Note: This function is deprecated in favor of per-hour fetching with skip logic
func FetchAudioRange(apiKey string, startTime, endTime time.Time, outputDir string, progressCallback func(current time.Time, total int, currentNum int)) ([]string, error) {
	var files []string
	chunkSize := 1 * time.Hour
	totalChunks := int(endTime.Sub(startTime) / chunkSize)
	if totalChunks == 0 {
		totalChunks = 1
	}

	currentNum := 0
	for current := startTime; current.Before(endTime); current = current.Add(chunkSize) {
		currentNum++
		chunkEnd := current.Add(chunkSize)
		if chunkEnd.After(endTime) {
			chunkEnd = endTime
		}

		if progressCallback != nil {
			progressCallback(current, totalChunks, currentNum)
		}

		file, _, err := FetchAudio(apiKey, current, chunkEnd, outputDir, false)
		if err != nil {
			return files, fmt.Errorf("failed to fetch chunk at %s: %w", current, err)
		}

		files = append(files, file)
	}

	return files, nil
}
