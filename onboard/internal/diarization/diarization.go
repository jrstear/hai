package diarization

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Result represents diarization results
type Result struct {
	AudioFile        string             `json:"audio_file"`
	Timestamp        string             `json:"timestamp"`
	AudioDuration    float64            `json:"audio_duration"`
	ProcessingTime   float64            `json:"processing_time"`
	RTF              float64            `json:"rtf"`
	Device           string             `json:"device"`
	SpeakerCount     int                `json:"speaker_count"`
	Speakers         []string           `json:"speakers"`
	SegmentCount     int                `json:"segment_count"`
	Segments         []Segment          `json:"segments"`
	SpeakerEmbeddings map[string][]float64 `json:"speaker_embeddings"`
}

// Segment represents a single speaker segment
type Segment struct {
	Speaker string  `json:"speaker"`
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Duration float64 `json:"duration"`
}

// RunDiarization runs the Python diarization script on an audio file
// Returns the result and a boolean indicating if it was loaded from cache
func RunDiarization(audioFile string, hfToken string, reprocess bool, progressCallback func(message string)) (*Result, bool, error) {
	// Check if result file already exists
	resultPath := strings.TrimSuffix(audioFile, filepath.Ext(audioFile)) + ".json"
	if _, err := os.Stat(resultPath); err == nil {
		if reprocess {
			// Remove existing result file to force re-processing
			if err := os.Remove(resultPath); err != nil {
				return nil, false, fmt.Errorf("failed to remove existing result file: %w", err)
			}
		} else {
			// Result file exists, load it
			if progressCallback != nil {
				progressCallback("Using cached diarization results")
			}
			
			data, err := os.ReadFile(resultPath)
			if err != nil {
				return nil, false, fmt.Errorf("failed to read cached result: %w", err)
			}
			
			var result Result
			if err := json.Unmarshal(data, &result); err != nil {
				return nil, false, fmt.Errorf("failed to parse cached result: %w", err)
			}
			
			return &result, true, nil
		}
	}
	
	// Find the diarization script (relative to project root)
	// Try multiple possible paths
	possiblePaths := []string{
		filepath.Join("..", "..", "..", "cmd", "diarize", "diarize.py"),
		filepath.Join("cmd", "diarize", "diarize.py"),
		"../cmd/diarize/diarize.py",
	}
	
	var scriptPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			scriptPath = path
			break
		}
	}
	
	if scriptPath == "" {
		return nil, false, fmt.Errorf("diarization script not found. Tried: %v", possiblePaths)
	}

	// Get Python executable (try python3 first, then python)
	pythonCmd := "python3"
	if _, err := exec.LookPath(pythonCmd); err != nil {
		pythonCmd = "python"
		if _, err := exec.LookPath(pythonCmd); err != nil {
			return nil, false, fmt.Errorf("python not found in PATH")
		}
	}

	// Activate conda environment if available
	cmd := exec.Command("bash", "-c", fmt.Sprintf(
		"source $(conda info --base)/etc/profile.d/conda.sh && conda activate hai && %s %s %s",
		pythonCmd, scriptPath, audioFile,
	))

	// Set environment variables
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("HF_TOKEN=%s", hfToken))

	// Capture stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Stream progress from stdout
	if progressCallback != nil {
		progressCallback("Starting diarization...")
	}

	// Run the command
	err := cmd.Run()
	if err != nil {
		return nil, false, fmt.Errorf("diarization failed: %w\nstderr: %s", err, stderr.String())
	}

	// Parse progress messages from stdout
	output := stdout.String()
	if progressCallback != nil {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.Contains(line, "Using cached results") {
				progressCallback(line)
			}
		}
	}
	
	// Read the JSON result
	data, err := os.ReadFile(resultPath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read diarization result: %w", err)
	}

	// Parse JSON
	var result Result
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, false, fmt.Errorf("failed to parse diarization result: %w", err)
	}

	return &result, false, nil
}

