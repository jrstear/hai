package server

import (
	"sync"
	"time"
)

// JobStatus represents the status of a processing job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// StageStatus represents the status of a processing stage for a date
type StageStatus string

const (
	StageStatusPending StageStatus = "pending"
	StageStatusRunning StageStatus = "running"
	StageStatusDone    StageStatus = "done"
	StageStatusFailed  StageStatus = "failed"
)

// HourProgress represents progress for a single hour
type HourProgress struct {
	Hour      string      `json:"hour"`       // YYYY-MM-DD HH:00 format (UTC, for internal tracking)
	DisplayHour string    `json:"display_hour"` // YYYY-MM-DD HH:00 format (user's timezone, for UI display)
	Date      string      `json:"date"`       // YYYY-MM-DD (for grouping, UTC)
	Lifelog   StageStatus `json:"lifelog"`    // Status: pending, running, done, failed (per-date)
	LifelogProgress int   `json:"lifelog_progress"` // 0-100
	Audio     StageStatus `json:"audio"`      // Status: pending, running, done, failed
	AudioProgress int     `json:"audio_progress"`   // 0-100
	Diarize   StageStatus `json:"diarize"`    // Status: pending, running, done, failed
	DiarizeProgress int   `json:"diarize_progress"` // 0-100
	Error     string      `json:"error,omitempty"`
}

// Job represents a processing job
type Job struct {
	ID        string    `json:"id"`
	Status    JobStatus `json:"status"`
	Progress  int       `json:"progress"` // 0-100 (overall)
	Message   string    `json:"message"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	
	// Job parameters
	APIKey    string    `json:"-"` // Don't expose in JSON
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Timezone  string    `json:"timezone"`
	Reprocess bool      `json:"reprocess"` // If true, remove and re-process existing files
	
	// Per-hour progress (keyed by hour string: "YYYY-MM-DD HH:00")
	HourProgress map[string]*HourProgress `json:"hour_progress"`
	// Per-date lifelog tracking (keyed by date: "YYYY-MM-DD")
	DateLifelogDone map[string]bool `json:"-"`
	
	// Cancellation
	Cancel chan struct{} `json:"-"` // Channel to signal cancellation
	cancelOnce sync.Once `json:"-"` // Ensure cancel channel is only closed once
}

// SubmitRequest represents a job submission request
type SubmitRequest struct {
	APIKey    string `json:"apiKey"`
	StartTime string `json:"startTime"` // RFC3339 format
	EndTime   string `json:"endTime"`   // RFC3339 format
	Timezone  string `json:"timezone"`  // Timezone (e.g., "America/Denver")
	Reprocess bool   `json:"reprocess"` // If true, remove and re-process existing files
}

