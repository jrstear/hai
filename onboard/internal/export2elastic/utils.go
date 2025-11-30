package export2elastic

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// generateSpeakerID generates a new speaker ID in the format spkr_xxxxx
func generateSpeakerID() string {
	id := uuid.New().String()
	// Use first 8 characters of UUID for shorter ID
	return fmt.Sprintf("spkr_%s", id[:8])
}

// generateRecordingID generates a recording ID from start time
// Format: rec_YYYY_MM_DD_HH
func generateRecordingID(startTime time.Time) string {
	utc := startTime.UTC()
	return fmt.Sprintf("rec_%04d_%02d_%02d_%02d",
		utc.Year(),
		utc.Month(),
		utc.Day(),
		utc.Hour(),
	)
}

// extractRecordingStartTime extracts the recording start time from file path
// Path format: data/YYYY/MM/DD/HH.ogg (in UTC)
// Returns the start time (hour boundary in UTC)
func extractRecordingStartTime(filePath string) (time.Time, error) {
	// Normalize path separators
	normalized := filepath.ToSlash(filePath)
	parts := strings.Split(normalized, "/")

	// Look for YYYY/MM/DD/HH.ogg pattern
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		// Look for year (4 digits starting with 2)
		if len(part) == 4 && part[0] == '2' {
			// Check if we have enough parts: YYYY/MM/DD/filename
			if i+3 < len(parts) {
				year := part
				month := parts[i+1]
				day := parts[i+2]
				hourFile := parts[i+3]
				
				// Extract hour from filename like "15.ogg"
				hour := strings.TrimSuffix(hourFile, filepath.Ext(hourFile))
				
				// Parse components
				var y, m, d, h int
				if _, err := fmt.Sscanf(year, "%d", &y); err != nil {
					continue
				}
				if _, err := fmt.Sscanf(month, "%d", &m); err != nil {
					continue
				}
				if _, err := fmt.Sscanf(day, "%d", &d); err != nil {
					continue
				}
				if _, err := fmt.Sscanf(hour, "%d", &h); err != nil {
					continue
				}

				// Create time at hour boundary in UTC
				return time.Date(y, time.Month(m), d, h, 0, 0, 0, time.UTC), nil
			}
		}
	}

	// Fallback: try to parse from directory structure
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath) // filename like "15.ogg"
	hour := strings.TrimSuffix(base, filepath.Ext(base))

	// Try to extract date from parent directories
	dayDir := filepath.Base(dir)
	monthDir := filepath.Base(filepath.Dir(dir))
	yearDir := filepath.Base(filepath.Dir(filepath.Dir(dir)))

	// Validate we have reasonable values
	if len(yearDir) == 4 && len(monthDir) <= 2 && len(dayDir) <= 2 {
		var y, m, d, h int
		if _, err := fmt.Sscanf(yearDir, "%d", &y); err == nil {
			if _, err := fmt.Sscanf(monthDir, "%d", &m); err == nil {
				if _, err := fmt.Sscanf(dayDir, "%d", &d); err == nil {
					if _, err := fmt.Sscanf(hour, "%d", &h); err == nil {
						return time.Date(y, time.Month(m), d, h, 0, 0, 0, time.UTC), nil
					}
				}
			}
		}
	}

	return time.Time{}, fmt.Errorf("could not extract recording start time from path: %s", filePath)
}

// getFileExtension extracts the file extension without the dot
func getFileExtension(filePath string) string {
	ext := filepath.Ext(filePath)
	if ext == "" {
		return ""
	}
	// Remove leading dot
	return ext[1:]
}

