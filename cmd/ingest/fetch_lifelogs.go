package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	apiBaseURL = "https://api.limitless.ai/v1"
)

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

type Lifelog struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Markdown  string    `json:"markdown"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	Contents  []Content `json:"contents"`
}

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

func main() {
	apiKey := os.Getenv("LIMITLESS_API_KEY")
	if apiKey == "" {
		if envBytes, err := os.ReadFile(".env"); err == nil {
			envContent := string(envBytes)
			for _, line := range splitLines(envContent) {
				line = strings.TrimSpace(line)
				line = strings.TrimPrefix(line, "export ")
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

	// Default to Nov 22, 2025
	defaultDate := "2025-11-22"
	timezone := "America/Denver" // MST

	dateStr := flag.String("date", defaultDate, "Date to fetch (YYYY-MM-DD)")
	tz := flag.String("timezone", timezone, "Timezone")
	outFile := flag.String("out", "data/lifelogs", "Output file prefix")
	flag.Parse()

	fmt.Printf("Fetching lifelogs for %s (%s timezone)\n", *dateStr, *tz)

	lifelogs, err := fetchLifelogs(apiKey, *dateStr, *tz)
	if err != nil {
		log.Fatalf("Failed to fetch lifelogs: %v", err)
	}

	fmt.Printf("Fetched %d lifelogs\n", len(lifelogs))

	// Save to JSON file
	outputPath := fmt.Sprintf("%s_%s.json", *outFile, *dateStr)
	if err := saveLifelogs(lifelogs, outputPath); err != nil {
		log.Fatalf("Failed to save lifelogs: %v", err)
	}

	fmt.Printf("Saved to %s\n", outputPath)

	// Print summary
	fmt.Println("\nSummary:")
	for _, ll := range lifelogs {
		fmt.Printf("- %s to %s: %s\n",
			ll.StartTime.Format("15:04"),
			ll.EndTime.Format("15:04"),
			ll.Title)
	}
}

func fetchLifelogs(apiKey, date, timezone string) ([]Lifelog, error) {
	var allLifelogs []Lifelog
	var cursor *string

	for {
		params := url.Values{}
		params.Set("date", date)
		params.Set("timezone", timezone)
		params.Set("includeMarkdown", "true")
		params.Set("includeContents", "true")

		if cursor != nil {
			params.Set("cursor", *cursor)
		}

		reqURL := fmt.Sprintf("%s/lifelogs?%s", apiBaseURL, params.Encode())

		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("X-API-Key", apiKey)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		var result LifelogResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}

		allLifelogs = append(allLifelogs, result.Data.Lifelogs...)

		if result.Meta.Lifelogs.NextCursor == nil {
			break
		}
		cursor = result.Meta.Lifelogs.NextCursor
	}

	return allLifelogs, nil
}

func saveLifelogs(lifelogs []Lifelog, path string) error {
	data, err := json.MarshalIndent(lifelogs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
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
