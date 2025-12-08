package export2elastic

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"hai/onboard/internal/fetch"
	"hai/storage"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/google/uuid"
)

// ExportLifelogs exports lifelogs from a JSON file to Elasticsearch
// Returns the number of lifelogs and blockquotes indexed, a boolean indicating if it was skipped, and any error
// If lifelogs already exist for the date, they are skipped (not re-indexed)
func (e *Exporter) ExportLifelogs(ctx context.Context, lifelogFilePath string) (int, int, bool, error) {
	// Read the lifelog JSON file
	data, err := os.ReadFile(lifelogFilePath)
	if err != nil {
		return 0, 0, false, fmt.Errorf("failed to read lifelog file: %w", err)
	}

	// Handle "null" response from API
	if string(data) == "null" || len(data) == 0 {
		return 0, 0, false, nil // No lifelogs to export
	}

	var lifelogs []fetch.Lifelog
	if err := json.Unmarshal(data, &lifelogs); err != nil {
		return 0, 0, false, fmt.Errorf("failed to parse lifelog JSON: %w", err)
	}

	if len(lifelogs) == 0 {
		return 0, 0, false, nil // No lifelogs to export
	}

	// Extract date from file path to check if lifelogs already exist
	// Path format: data/YYYY/MM/DD/lifelog.json
	dateStr, err := extractDateFromLifelogPath(lifelogFilePath)
	if err != nil {
		return 0, 0, false, fmt.Errorf("failed to extract date from path: %w", err)
	}

	// Check if any lifelogs already exist for this date
	// We'll check by querying for lifelogs that start on this date
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 0, 0, false, fmt.Errorf("failed to parse date: %w", err)
	}
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	existingLifelogs, err := e.storage.ListLifelogs(ctx, &startOfDay, &endOfDay)
	if err != nil {
		return 0, 0, false, fmt.Errorf("failed to check existing lifelogs: %w", err)
	}

	// If lifelogs already exist for this date, skip
	if len(existingLifelogs) > 0 {
		// Count blockquotes for existing lifelogs
		totalBlockquotes := 0
		for _, ll := range existingLifelogs {
			blockquotes, err := e.storage.GetLifelogBlockquotesByLifelog(ctx, ll.ID)
			if err == nil {
				totalBlockquotes += len(blockquotes)
			}
		}
		return len(existingLifelogs), totalBlockquotes, true, nil
	}

	// Load contacts for name matching (optional - contacts may not be loaded yet)
	var contacts []Contact
	var esClient *elasticsearch.Client

	// Check if we need to replace "You" - if so, we MUST have esClient
	needsReplacement := false
	for _, ll := range lifelogs {
		for _, content := range ll.Contents {
			if content.Type == "blockquote" {
				if strings.ToLower(strings.TrimSpace(content.SpeakerName)) == "you" {
					needsReplacement = true
					break
				}
			}
		}
		if needsReplacement {
			break
		}
	}

	// Create ES client if needed (for replacing "You" or loading contacts)
	if e.esURL != "" {
		var err error
		esClient, err = e.createESClient(e.esURL)
		if err == nil {
			// Load contacts for name matching
			loadedContacts, err := e.loadContacts(ctx, esClient)
			if err == nil {
				contacts = loadedContacts
			}
		} else if needsReplacement {
			// Log warning if we need replacement but can't create client
			log.Printf("Warning: Need to replace 'You' but failed to create ES client: %v. 'You' will not be replaced.", err)
		}
	} else if needsReplacement {
		// Log warning if we need replacement but no esURL is configured
		log.Printf("Warning: Need to replace 'You' but esURL is not configured. 'You' will not be replaced. Set esURL when creating exporter to enable replacement.")
	}

	// Export lifelogs and blockquotes
	lifelogCount := 0
	blockquoteCount := 0

	for _, ll := range lifelogs {
		// Convert fetch.Lifelog to storage.Lifelog
		storageLifelog := &storage.Lifelog{
			ID:        ll.ID,
			Title:     ll.Title,
			Markdown:  ll.Markdown,
			StartTime: ll.StartTime.UTC(),
			EndTime:   ll.EndTime.UTC(),
			CreatedAt: time.Now().UTC(),
		}

		// Create lifelog in Elasticsearch
		if err := e.storage.CreateLifelog(ctx, storageLifelog); err != nil {
			if err == storage.ErrDuplicateKey {
				// Already exists, skip
				continue
			}
			return lifelogCount, blockquoteCount, false, fmt.Errorf("failed to create lifelog %s: %w", ll.ID, err)
		}
		lifelogCount++

		// Extract blockquotes from contents
		blockquotes := make([]*storage.LifelogBlockquote, 0)
		for _, content := range ll.Contents {
			if content.Type == "blockquote" {
				// Parse start and end times from content
				var startTime, endTime time.Time
				if content.StartTime != "" {
					startTime, err = time.Parse(time.RFC3339, content.StartTime)
					if err != nil {
						// Try parsing as ISO 8601 without timezone
						startTime, err = time.Parse("2006-01-02T15:04:05", content.StartTime)
						if err != nil {
							// Fallback: calculate from lifelog start + offset
							startTime = ll.StartTime.Add(time.Duration(content.StartOffsetMs) * time.Millisecond)
						}
					}
				} else {
					// Use offset if no absolute time
					startTime = ll.StartTime.Add(time.Duration(content.StartOffsetMs) * time.Millisecond)
				}

				if content.EndTime != "" {
					endTime, err = time.Parse(time.RFC3339, content.EndTime)
					if err != nil {
						endTime, err = time.Parse("2006-01-02T15:04:05", content.EndTime)
						if err != nil {
							endTime = ll.EndTime.Add(time.Duration(content.EndOffsetMs) * time.Millisecond)
						}
					}
				} else {
					endTime = ll.EndTime.Add(time.Duration(content.EndOffsetMs) * time.Millisecond)
				}

				// Generate blockquote ID
				blockquoteID := fmt.Sprintf("bq_%s", uuid.New().String()[:8])

				// Replace "You" with actual user name from settings if available
				// This MUST happen before storage - "You" should never be stored in blockquotes
				speakerName := content.SpeakerName
				if strings.ToLower(strings.TrimSpace(speakerName)) == "you" {
					if esClient != nil {
						userName, err := e.loadUserName(ctx, esClient)
						if err == nil && userName != "" {
							speakerName = userName
							// log.Printf("Replaced 'You' with '%s' in blockquote %s", userName, blockquoteID)
						} else if err != nil {
							log.Printf("Warning: Failed to load user_name from settings: %v. Keeping 'You' as speaker name.", err)
						} else {
							log.Printf("Warning: user_name setting not found in settings. Keeping 'You' as speaker name.")
						}
					} else {
						// This should not happen if needsReplacement check worked, but handle gracefully
						log.Printf("Warning: Cannot replace 'You' - ES client not available. Speaker name will remain 'You'.")
					}
				}

				blockquote := &storage.LifelogBlockquote{
					ID:            blockquoteID,
					LifelogID:     ll.ID,
					Content:       content.Content,
					SpeakerName:   speakerName,
					StartOffsetMs: content.StartOffsetMs,
					EndOffsetMs:   content.EndOffsetMs,
					StartTime:     startTime.UTC(),
					EndTime:       endTime.UTC(),
					CreatedAt:     time.Now().UTC(),
				}

				// Set SpeakerIdentifier if available
				if content.SpeakerIdentifier != nil {
					blockquote.SpeakerID = content.SpeakerIdentifier
				}

				// Auto-associate with contact if speaker_name matches a contact name
				// Note: speakerName has already been replaced with user_name if it was "You"
				if len(contacts) > 0 {
					matchedContactID := e.matchContactByName(ctx, speakerName, contacts, nil)
					if matchedContactID != nil {
						blockquote.ContactID = matchedContactID
					}
				}

				blockquotes = append(blockquotes, blockquote)
			}
		}

		// Bulk create blockquotes
		if len(blockquotes) > 0 {
			count, err := e.storage.CreateLifelogBlockquotes(ctx, blockquotes)
			if err != nil {
				return lifelogCount, blockquoteCount, false, fmt.Errorf("failed to create blockquotes for lifelog %s: %w", ll.ID, err)
			}
			blockquoteCount += count
		}
	}

	return lifelogCount, blockquoteCount, false, nil
}

// extractDateFromLifelogPath extracts the date (YYYY-MM-DD) from a lifelog file path
// Path format: data/YYYY/MM/DD/lifelog.json
func extractDateFromLifelogPath(filePath string) (string, error) {
	// Normalize path separators
	normalized := filepath.ToSlash(filePath)
	parts := strings.Split(normalized, "/")

	// Look for YYYY/MM/DD pattern
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		// Look for year (4 digits starting with 2)
		if len(part) == 4 && part[0] == '2' {
			// Check if we have enough parts: YYYY/MM/DD/lifelog.json
			if i+3 < len(parts) {
				year := part
				month := parts[i+1]
				day := parts[i+2]
				// Pad month and day with leading zeros if needed
				if len(month) == 1 {
					month = "0" + month
				}
				if len(day) == 1 {
					day = "0" + day
				}
				return fmt.Sprintf("%s-%s-%s", year, month, day), nil
			}
		}
	}

	return "", fmt.Errorf("could not extract date from path: %s", filePath)
}
