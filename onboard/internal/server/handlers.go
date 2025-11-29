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

	"github.com/google/uuid"
	"hai/onboard/internal/diarization"
	"hai/onboard/internal/fetch"
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

	// Parse start and end times
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid start time: %v", err), http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid end time: %v", err), http.StatusBadRequest)
		return
	}

	if endTime.Before(startTime) || endTime.Equal(startTime) {
		http.Error(w, "End time must be after start time", http.StatusBadRequest)
		return
	}

	// Set timezone
	timezone := req.Timezone
	if timezone == "" {
		timezone = s.timezone
	}

	// Generate list of hours to process
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
	uniqueDates := make(map[string]bool)
	for _, hour := range hours {
		hourKey := hour.Format("2006-01-02 15:04")
		dateKey := hour.Format("2006-01-02")
		uniqueDates[dateKey] = true
		
		job.HourProgress[hourKey] = &HourProgress{
			Hour:            hourKey,
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
		"jobId": jobID,
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

	// Generate hours to process
	hours := generateHours(job.StartTime, job.EndTime)
	
	// Get unique dates for lifelog fetching
	uniqueDates := make(map[string]bool)
	for _, hour := range hours {
		dateKey := hour.Format("2006-01-02")
		uniqueDates[dateKey] = true
	}
	
	// Create channels for pipeline (work flows: lifelog -> audio -> diarize)
	lifelogChan := make(chan string, len(uniqueDates))  // Dates ready for lifelog fetching
	audioChan := make(chan time.Time, len(hours))       // Hours ready for audio fetching
	diarizeChan := make(chan string, len(hours))        // Audio files ready for diarization

	// Send all unique dates to lifelog channel
	dateList := make([]string, 0, len(uniqueDates))
	for date := range uniqueDates {
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
			// Update all hours for this date
			for hourKey, hp := range job.HourProgress {
				if hp.Date == date {
					s.updateHourStage(job, hourKey, "lifelog", StageStatusRunning, 0, "")
				}
			}
			
			dateTime, err := time.Parse("2006-01-02", date)
			if err != nil {
				for hourKey, hp := range job.HourProgress {
					if hp.Date == date {
						s.updateHourStage(job, hourKey, "lifelog", StageStatusFailed, 0, fmt.Sprintf("Invalid date: %v", err))
					}
				}
				continue
			}
			
			_, wasCached, err := fetch.FetchLifelogs(job.APIKey, dateTime, job.Timezone, s.outputDir, job.Reprocess)
			
			if err != nil {
				// Only show error on first hour of the date to avoid duplicate error messages
				// Since lifelogs are per-day, we don't need to show the error for every hour
				firstHourKey := ""
				for hourKey, hp := range job.HourProgress {
					if hp.Date == date {
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
			
			// Mark lifelog as done for all hours of this date
			job.DateLifelogDone[date] = true
			for hourKey, hp := range job.HourProgress {
				if hp.Date == date {
					if wasCached {
						// Already existed, mark as done immediately
						s.updateHourStage(job, hourKey, "lifelog", StageStatusDone, 100, "")
					} else {
						// Just fetched, mark as done
						s.updateHourStage(job, hourKey, "lifelog", StageStatusDone, 100, "")
					}
				}
			}
			
			// Send all hours for this date to audio channel
			for _, hour := range hours {
				select {
				case <-job.Cancel:
					return
				default:
					if hour.Format("2006-01-02") == date {
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
			hourKey := hour.Format("2006-01-02 15:04")
			
			chunkEnd := hour.Add(1 * time.Hour)
			if chunkEnd.After(job.EndTime) {
				chunkEnd = job.EndTime
			}
			
			// Fetch this hour of audio (checks if already exists)
			audioFile, wasCached, err := fetch.FetchAudio(job.APIKey, hour, chunkEnd, s.outputDir, job.Reprocess)
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
				log.Printf("Using cached diarization for %s: %d speakers, %d segments", audioFile, result.SpeakerCount, result.SegmentCount)
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

// generateHours generates a list of hour start times between start and end
func generateHours(startTime, endTime time.Time) []time.Time {
	var hours []time.Time
	chunkSize := 1 * time.Hour
	
	// Round start time down to the hour
	start := time.Date(
		startTime.Year(), startTime.Month(), startTime.Day(),
		startTime.Hour(), 0, 0, 0,
		startTime.Location(),
	)
	
	for current := start; current.Before(endTime); current = current.Add(chunkSize) {
		hours = append(hours, current)
	}
	
	return hours
}

func extractHourFromPath(path string) string {
	// Extract hour from path like: data/2025/11/22/15.ogg
	// Return: 2025-11-22 15:04
	// Normalize path separators
	normalized := filepath.ToSlash(path)
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
				return fmt.Sprintf("%s-%s-%s %s:04", year, month, day, hour)
			}
		}
	}
	
	// Fallback: try to parse from directory structure
	dir := filepath.Dir(path)
	base := filepath.Base(path) // filename like "15.ogg"
	hour := strings.TrimSuffix(base, filepath.Ext(base))
	
	// Try to extract date from parent directories
	dayDir := filepath.Base(dir)
	monthDir := filepath.Base(filepath.Dir(dir))
	yearDir := filepath.Base(filepath.Dir(filepath.Dir(dir)))
	
	// Validate we have reasonable values
	if len(yearDir) == 4 && len(monthDir) <= 2 && len(dayDir) <= 2 {
		// Pad hour with leading zero if needed
		if len(hour) == 1 {
			hour = "0" + hour
		}
		return fmt.Sprintf("%s-%s-%s %s:04", yearDir, monthDir, dayDir, hour)
	}
	
	return ""
}
