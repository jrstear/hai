package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	apiBaseURL = "https://api.limitless.ai/v1"
)

func main() {
	apiKey := os.Getenv("LIMITLESS_API_KEY")
	if apiKey == "" {
		// Try to read from .env if not in environment
		// Simple .env parser for this prototype
		if envBytes, err := os.ReadFile(".env"); err == nil {
			envContent := string(envBytes)
			for _, line := range splitLines(envContent) {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "export ") {
					line = strings.TrimPrefix(line, "export ")
				}
				if strings.HasPrefix(line, "LIMITLESS_API_KEY=") {
					apiKey = strings.TrimPrefix(line, "LIMITLESS_API_KEY=")
					apiKey = strings.Trim(apiKey, `"'`)
					break
				}
			}
		}
	}

	if apiKey == "" {
		log.Fatal("LIMITLESS_API_KEY environment variable not set and not found in .env")
	}

	// Default to the requested time: Nov 22, 2025, 3pm-7pm MST
	// MST is UTC-7
	defaultStart := "2025-11-22T15:00:00-07:00"
	defaultDuration := 4 * time.Hour

	startStr := flag.String("start", defaultStart, "Start time (RFC3339)")
	duration := flag.Duration("duration", defaultDuration, "Duration to fetch")
	outDir := flag.String("out", "data/audio", "Output directory")
	flag.Parse()

	startTime, err := time.Parse(time.RFC3339, *startStr)
	if err != nil {
		log.Fatalf("Invalid start time: %v", err)
	}

	endTime := startTime.Add(*duration)

	fmt.Printf("Fetching audio from %s to %s\n", startTime, endTime)

	// Create output directory
	if err := os.MkdirAll(*outDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Chunk into 1-hour segments
	chunkSize := 1 * time.Hour
	for current := startTime; current.Before(endTime); current = current.Add(chunkSize) {
		chunkEnd := current.Add(chunkSize)
		if chunkEnd.After(endTime) {
			chunkEnd = endTime
		}

		// Construct filename: YYYY/MM/DD/HH.ogg
		// We'll use the output directory structure requested: YYYY/MM/DD/HH.ogg
		// Note: The user requested YYYY/MM/DD/HH.ogg.
		// We need to handle the directory creation for each file.
		relPath := filepath.Join(
			current.Format("2006"),
			current.Format("01"),
			current.Format("02"),
			fmt.Sprintf("%s.ogg", current.Format("15")),
		)
		fullPath := filepath.Join(*outDir, relPath)

		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			log.Printf("Failed to create directory for %s: %v", fullPath, err)
			continue
		}

		fmt.Printf("Downloading chunk: %s to %s -> %s\n", current, chunkEnd, fullPath)

		if err := downloadAudio(apiKey, current, chunkEnd, fullPath); err != nil {
			log.Printf("Failed to download chunk %s: %v", current, err)
		}
	}
}

func downloadAudio(apiKey string, start, end time.Time, outputPath string) error {
	// Convert to milliseconds
	startMs := start.UnixMilli()
	endMs := end.UnixMilli()

	url := fmt.Sprintf("%s/download-audio?startMs=%d&endMs=%d", apiBaseURL, startMs, endMs)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-API-Key", apiKey)

	client := &http.Client{Timeout: 5 * time.Minute} // Large timeout for audio download
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func splitLines(s string) []string {
	var lines []string
	var current []rune
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, string(current))
			current = nil
		} else if r != '\r' {
			current = append(current, r)
		}
	}
	if len(current) > 0 {
		lines = append(lines, string(current))
	}
	return lines
}
