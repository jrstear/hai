package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"hai/onboard/internal/diarization"
	"hai/onboard/internal/fetch"

	"github.com/google/uuid"
)

// Server handles HTTP requests
type Server struct {
	jobs      map[string]*Job
	mu        sync.RWMutex
	outputDir string
	timezone  string
}

// NewServer creates a new server instance
func NewServer(outputDir string) *Server {
	return &Server{
		jobs:      make(map[string]*Job),
		outputDir: outputDir,
		timezone:  "America/Denver", // Default timezone
	}
}

// HandleSubmit handles job submission
func (s *Server) HandleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Set timezone
	timezone := req.Timezone
	if timezone == "" {
		timezone = s.timezone
	}

	// Load timezone location for display
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid timezone: %v", err), http.StatusBadRequest)
		return
	}

	// Parse start and end times (they come as ISO strings in UTC from browser)
	// Parse as RFC3339 (includes timezone), then convert to UTC for internal processing
	startTimeUTC, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid start time: %v", err), http.StatusBadRequest)
		return
	}
	// Convert to UTC for internal processing (all file paths and data use UTC)
	startTime := startTimeUTC.UTC()

	endTimeUTC, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid end time: %v", err), http.StatusBadRequest)
		return
	}
	// Convert to UTC for internal processing
	endTime := endTimeUTC.UTC()

	if endTime.Before(startTime) || endTime.Equal(startTime) {
		http.Error(w, "End time must be after start time (non-inclusive)", http.StatusBadRequest)
		return
	}

	// Generate list of hours to process (in UTC)
	// End time is non-inclusive (processes up to but not including the end hour)
	hours := generateHours(startTime, endTime)

	// Create job
	jobID := uuid.New().String()
	job := &Job{
		ID:              jobID,
		Status:          JobStatusPending,
		Progress:        0,
		Message:         "Job created",
		APIKey:          req.APIKey,
		StartTime:       startTime,
		EndTime:         endTime,
		Timezone:        timezone,
		Reprocess:       req.Reprocess,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		HourProgress:    make(map[string]*HourProgress),
		DateLifelogDone: make(map[string]bool),
		Cancel:          make(chan struct{}),
	}

	// Initialize hour progress and track unique dates for lifelog fetching
	// All times are in UTC for consistency, but display hour is in user's timezone
	uniqueDates := make(map[string]bool)
	for _, hour := range hours {
		// Format as "2006-01-02 15:00" in UTC (hours are always on the hour, minutes are 00)
		hourKey := hour.UTC().Format("2006-01-02 15:00")
		dateKey := hour.UTC().Format("2006-01-02")
		uniqueDates[dateKey] = true

		// Convert to user's timezone for display
		hourInTZ := hour.In(loc)
		displayHour := hourInTZ.Format("2006-01-02 15:00")

		job.HourProgress[hourKey] = &HourProgress{
			Hour:            hourKey,
			DisplayHour:     displayHour,
			Date:            dateKey,
			Lifelog:         StageStatusPending,
			LifelogProgress: 0,
			Audio:           StageStatusPending,
			AudioProgress:   0,
			Diarize:         StageStatusPending,
			DiarizeProgress: 0,
		}
	}

	s.mu.Lock()
	s.jobs[jobID] = job
	s.mu.Unlock()

	// Start processing in background
	go s.processJob(job)

	// Return job ID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"jobId":  jobID,
		"status": "processing",
	})
}

// HandleStatus returns job status
func (s *Server) HandleStatus(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("jobId")
	if jobID == "" {
		http.Error(w, "jobId parameter required", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	job, exists := s.jobs[jobID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// HandleCancel cancels a running job
func (s *Server) HandleCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobID := r.URL.Query().Get("jobId")
	if jobID == "" {
		http.Error(w, "jobId parameter required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	job, exists := s.jobs[jobID]
	s.mu.Unlock()

	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	// Cancel the job
	job.cancelOnce.Do(func() {
		close(job.Cancel)
		s.updateJob(job, JobStatusCancelled, job.Progress, "Job cancelled by user", "")
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "cancelled",
		"jobId":  jobID,
	})
}

// processJob processes a job with concurrent goroutines for each stage
func (s *Server) processJob(job *Job) {
	defer func() {
		if r := recover(); r != nil {
			s.updateJob(job, JobStatusFailed, 0, fmt.Sprintf("Panic: %v", r), "")
		}
	}()

	s.updateJob(job, JobStatusProcessing, 0, "Starting processing...", "")

	hfToken := os.Getenv("HF_TOKEN")
	if hfToken == "" {
		s.updateJob(job, JobStatusFailed, 0, "HF_TOKEN not set", "HF_TOKEN environment variable is required for diarization")
		return
	}

	// Generate hours to process (all times are in UTC)
	hours := generateHours(job.StartTime.UTC(), job.EndTime.UTC())

	// Get unique dates for lifelog fetching
	// We need dates in the user's timezone for API calls, but store in UTC directories
	loc, err := time.LoadLocation(job.Timezone)
	if err != nil {
		s.updateJob(job, JobStatusFailed, 0, "Invalid timezone", fmt.Sprintf("Failed to load timezone: %v", err))
		return
	}

	// Get unique dates in user's timezone (for API calls)
	uniqueDatesInTZ := make(map[string]bool) // key: date in user's timezone (YYYY-MM-DD)
	dateToUTC := make(map[string]time.Time)  // map from TZ date to UTC date for storage

	// Convert start/end times to user's timezone to determine which dates to fetch
	startInTZ := job.StartTime.UTC().In(loc)
	endInTZ := job.EndTime.UTC().In(loc)

	// Get all dates between start and end in user's timezone
	current := time.Date(startInTZ.Year(), startInTZ.Month(), startInTZ.Day(), 0, 0, 0, 0, loc)
	endDate := time.Date(endInTZ.Year(), endInTZ.Month(), endInTZ.Day(), 0, 0, 0, 0, loc)

	for current.Before(endDate) || current.Equal(endDate) {
		dateStr := current.Format("2006-01-02")
		uniqueDatesInTZ[dateStr] = true
		// Convert to UTC for storage (use start of day in TZ, convert to UTC)
		dateToUTC[dateStr] = current.UTC()
		current = current.AddDate(0, 0, 1)
	}

	// Create channels for pipeline (work flows: lifelog -> audio -> diarize)
	lifelogChan := make(chan string, len(uniqueDatesInTZ)) // Dates in user's timezone for API calls
	audioChan := make(chan time.Time, len(hours))          // Hours ready for audio fetching (UTC)
	diarizeChan := make(chan string, len(hours))           // Audio files ready for diarization

	// Send all unique dates to lifelog channel (in user's timezone)
	dateList := make([]string, 0, len(uniqueDatesInTZ))
	for date := range uniqueDatesInTZ {
		dateList = append(dateList, date)
	}
	sort.Strings(dateList)
	for _, date := range dateList {
		lifelogChan <- date
	}
	close(lifelogChan)

	// Start goroutines for each stage
	var wg sync.WaitGroup
	wg.Add(3)

	// Lifelog fetcher goroutine - processes dates sequentially
	// Note: Lifelogs are fetched per-day (not per-hour) because:
	// 1. The Limitless API accepts a date parameter (YYYY-MM-DD)
	// 2. Lifelogs can span multiple hours (they have StartTime/EndTime)
	// 3. One API call returns all lifelogs for the entire day
	go func() {
		defer wg.Done()
		defer close(audioChan)

		for {
			select {
			case <-job.Cancel:
				return
			case date, ok := <-lifelogChan:
				if !ok {
					return
				}
				// date is in user's timezone format (YYYY-MM-DD)
				// Get the UTC date for storage
				utcDate, exists := dateToUTC[date]
				if !exists {
					// Fallback: parse in user's timezone and convert to UTC
					dateTimeInTZ, err := time.ParseInLocation("2006-01-02", date, loc)
					if err != nil {
						// Find matching UTC dates for error reporting
						for hourKey := range job.HourProgress {
							s.updateHourStage(job, hourKey, "lifelog", StageStatusFailed, 0, fmt.Sprintf("Invalid date: %v", err))
						}
						continue
					}
					utcDate = dateTimeInTZ.UTC()
				}

				// Update all hours that might be affected by this lifelog fetch
				// Since lifelogs are per-day in user's timezone, we need to find matching UTC dates
				utcDateStr := utcDate.Format("2006-01-02")
				for hourKey, hp := range job.HourProgress {
					if hp.Date == utcDateStr {
						s.updateHourStage(job, hourKey, "lifelog", StageStatusRunning, 0, "")
					}
				}

				// FetchLifelogs: date parameter is in user's timezone (for API), utcDate is for directory structure
				_, wasCached, err := fetch.FetchLifelogs(job.APIKey, utcDate, job.Timezone, s.outputDir, job.Reprocess)

				if err != nil {
					// Only show error on first hour of the UTC date to avoid duplicate error messages
					utcDateStr := utcDate.Format("2006-01-02")
					firstHourKey := ""
					for hourKey, hp := range job.HourProgress {
						if hp.Date == utcDateStr {
							if firstHourKey == "" {
								firstHourKey = hourKey
							}
							// Mark all hours as failed, but only show error on first one
							if hourKey == firstHourKey {
								s.updateHourStage(job, hourKey, "lifelog", StageStatusFailed, 0, err.Error())
							} else {
								s.updateHourStage(job, hourKey, "lifelog", StageStatusFailed, 0, "")
							}
						}
					}
					continue
				}

				// Mark lifelog as done for all hours of the UTC date (utcDateStr already declared above)
				job.DateLifelogDone[utcDateStr] = true
				for hourKey, hp := range job.HourProgress {
					if hp.Date == utcDateStr {
						if wasCached {
							// Already existed, mark as done immediately
							s.updateHourStage(job, hourKey, "lifelog", StageStatusDone, 100, "")
						} else {
							// Just fetched, mark as done
							s.updateHourStage(job, hourKey, "lifelog", StageStatusDone, 100, "")
						}
					}
				}

				// Send all hours for the UTC date to audio channel (all times in UTC)
				utcDateStrForAudio := utcDateStr
				for _, hour := range hours {
					select {
					case <-job.Cancel:
						return
					default:
						if hour.UTC().Format("2006-01-02") == utcDateStrForAudio {
							audioChan <- hour
						}
					}
				}
			}
		}
	}()

	// Audio fetcher goroutine - processes hours sequentially, streams to diarization
	go func() {
		defer wg.Done()
		defer close(diarizeChan)

		for {
			select {
			case <-job.Cancel:
				return
			case hour, ok := <-audioChan:
				if !ok {
					return
				}
				// Format hour in UTC for the key (hours are always on the hour, minutes are 00)
				hourKey := hour.UTC().Format("2006-01-02 15:00")

				// Calculate chunk end: normally hour + 1 hour, but cap at job end time - 1 second (non-inclusive)
				chunkEnd := hour.Add(1 * time.Hour)
				jobEndUTC := job.EndTime.UTC()
				if chunkEnd.After(jobEndUTC) {
					// Cap at end time minus 1 second to make it non-inclusive
					chunkEnd = jobEndUTC.Add(-1 * time.Second)
				}

				// Fetch this hour of audio (checks if already exists)
				// Ensure times are in UTC for directory structure
				audioFile, wasCached, err := fetch.FetchAudio(job.APIKey, hour.UTC(), chunkEnd.UTC(), s.outputDir, job.Reprocess)
				if err != nil {
					s.updateHourStage(job, hourKey, "audio", StageStatusFailed, 0, fmt.Sprintf("Failed to fetch: %v", err))
					continue
				}

				if wasCached {
					// File already existed, mark as done immediately
					s.updateHourStage(job, hourKey, "audio", StageStatusDone, 100, "")
				} else {
					// Just downloaded, update progress
					s.updateHourStage(job, hourKey, "audio", StageStatusRunning, 0, "")
					s.updateHourStage(job, hourKey, "audio", StageStatusDone, 100, "")
				}

				// Stream to diarization immediately (hourly chaining)
				select {
				case <-job.Cancel:
					return
				case diarizeChan <- audioFile:
				}
			}
		}
	}()

	// Diarization goroutine - processes audio files sequentially as they become available
	go func() {
		defer wg.Done()

		for {
			select {
			case <-job.Cancel:
				return
			case audioFile, ok := <-diarizeChan:
				if !ok {
					return
				}
				// Extract hour from audio file path
				hourKey := extractHourFromPath(audioFile)
				if hourKey == "" {
					log.Printf("Warning: Could not extract hour from path: %s", audioFile)
					// Try to continue anyway - the file was processed even if we can't track it
					continue
				}

				// Verify the hourKey exists in our progress map
				s.mu.RLock()
				_, exists := job.HourProgress[hourKey]
				s.mu.RUnlock()

				if !exists {
					log.Printf("Warning: hourKey '%s' not found in progress map for file: %s", hourKey, audioFile)
					s.mu.RLock()
					keys := make([]string, 0, len(job.HourProgress))
					for k := range job.HourProgress {
						keys = append(keys, k)
					}
					s.mu.RUnlock()
					log.Printf("Available keys: %v", keys)
					// Continue anyway - we'll process the file but can't track progress
					continue
				}

				// Run diarization (checks if result already exists)
				result, wasCached, err := diarization.RunDiarization(audioFile, hfToken, job.Reprocess, func(message string) {
					// Update progress while processing
					s.updateHourStage(job, hourKey, "diarize", StageStatusRunning, 50, "")
				})

				if err != nil {
					s.updateHourStage(job, hourKey, "diarize", StageStatusFailed, 0, err.Error())
					continue
				}

				if wasCached {
					// Result was loaded from cache, mark as done immediately
					// Don't print extra messages - the "already exists" message from RunDiarization is sufficient
					s.updateHourStage(job, hourKey, "diarize", StageStatusDone, 100, "")
				} else {
					// Just diarized, mark as done
					log.Printf("Diarized %s: %d speakers, %d segments", audioFile, result.SpeakerCount, result.SegmentCount)
					s.updateHourStage(job, hourKey, "diarize", StageStatusDone, 100, "")
				}
			}
		}
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	// Check if all hours are complete
	allDone := true
	for _, hp := range job.HourProgress {
		if hp.Diarize != StageStatusDone {
			allDone = false
			break
		}
	}

	if allDone {
		s.updateJob(job, JobStatusCompleted, 100, "Processing complete!", "")
	} else {
		s.updateJob(job, JobStatusFailed, 0, "Some hours failed", "")
	}
}

func (s *Server) updateJob(job *Job, status JobStatus, progress int, message, errorMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job.Status = status
	job.Progress = progress
	job.Message = message
	job.Error = errorMsg
	job.UpdatedAt = time.Now()
}

func (s *Server) updateHourStage(job *Job, hourKey string, stage string, status StageStatus, progress int, errorMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	hp, exists := job.HourProgress[hourKey]
	if !exists {
		return
	}

	switch stage {
	case "lifelog":
		hp.Lifelog = status
		hp.LifelogProgress = progress
	case "audio":
		hp.Audio = status
		hp.AudioProgress = progress
	case "diarize":
		hp.Diarize = status
		hp.DiarizeProgress = progress
	}

	if errorMsg != "" {
		hp.Error = errorMsg
	}

	// Update overall progress
	totalProgress := 0
	count := 0
	for _, h := range job.HourProgress {
		totalProgress += h.LifelogProgress + h.AudioProgress + h.DiarizeProgress
		count += 3
	}
	if count > 0 {
		job.Progress = totalProgress / count
	}

	job.UpdatedAt = time.Now()
}

// generateHours generates a list of hour start times between start and end (in UTC)
// End time is non-inclusive: processes from start hour up to (but not including) the end hour
func generateHours(startTime, endTime time.Time) []time.Time {
	var hours []time.Time
	chunkSize := 1 * time.Hour

	// Ensure times are in UTC
	startUTC := startTime.UTC()
	endUTC := endTime.UTC()

	// Round start time down to the hour in UTC
	start := time.Date(
		startUTC.Year(), startUTC.Month(), startUTC.Day(),
		startUTC.Hour(), 0, 0, 0,
		time.UTC,
	)

	// Round end time down to the hour in UTC (non-inclusive)
	// If end time has minutes/seconds, round up to next hour to include that hour
	// If end time is exactly on the hour, don't include that hour
	end := time.Date(
		endUTC.Year(), endUTC.Month(), endUTC.Day(),
		endUTC.Hour(), 0, 0, 0,
		time.UTC,
	)

	// If end time has any minutes/seconds, round up to next hour
	// This means if user selects 2:30, we process up to (but not including) hour 3
	if endUTC.After(end) {
		end = end.Add(chunkSize)
	}
	// If end time is exactly on the hour, end stays as-is (that hour won't be included)

	for current := start; current.Before(end); current = current.Add(chunkSize) {
		hours = append(hours, current)
	}

	return hours
}

func extractHourFromPath(path string) string {
	// Extract hour from path like: data/2025/11/22/15.ogg or /absolute/path/data/2025/11/22/15.ogg
	// Paths are in UTC. Return: 2025-11-22 15:00 (in UTC format matching hourKey)
	// Normalize path separators and make it absolute to handle both relative and absolute paths
	absPath, err := filepath.Abs(path)
	if err != nil {
		// If we can't get absolute path, use as-is
		absPath = path
	}
	normalized := filepath.ToSlash(absPath)
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
				// Pad hour with leading zero if needed
				if len(hour) == 1 {
					hour = "0" + hour
				}
				// Pad month and day with leading zeros if needed
				if len(month) == 1 {
					month = "0" + month
				}
				if len(day) == 1 {
					day = "0" + day
				}
				// Format matches hourKey format: "2006-01-02 15:00"
				return fmt.Sprintf("%s-%s-%s %s:00", year, month, day, hour)
			}
		}
	}

	// Fallback: try to parse from directory structure (works with absolute paths)
	dir := filepath.Dir(path)
	base := filepath.Base(path) // filename like "15.ogg"
	hour := strings.TrimSuffix(base, filepath.Ext(base))

	// Try to extract date from parent directories
	dayDir := filepath.Base(dir)
	monthDir := filepath.Base(filepath.Dir(dir))
	yearDir := filepath.Base(filepath.Dir(filepath.Dir(dir)))

	// Validate we have reasonable values
	if len(yearDir) == 4 && len(monthDir) <= 2 && len(dayDir) <= 2 {
		// Pad hour, month, day with leading zeros if needed
		if len(hour) == 1 {
			hour = "0" + hour
		}
		if len(monthDir) == 1 {
			monthDir = "0" + monthDir
		}
		if len(dayDir) == 1 {
			dayDir = "0" + dayDir
		}
		// Format matches hourKey format: "2006-01-02 15:00"
		return fmt.Sprintf("%s-%s-%s %s:00", yearDir, monthDir, dayDir, hour)
	}

	return ""
}
