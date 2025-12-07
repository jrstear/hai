# Onboarding Server: Go vs Python Analysis

## Overview

The onboarding server needs to:
1. Serve HTTP (web UI on port 3000)
2. Call diarization (Python with pyannote.audio)
3. Manage Docker lifecycle (start services, health checks)
4. Import data to Elasticsearch
5. Coordinate the entire onboarding flow

## Current Code Structure

### Diarization (Python)
- `cmd/diarize/diarize.py` has `run_diarization()` function
- Returns dict with segments, embeddings, metadata
- Can be imported directly or called via subprocess
- Already used via subprocess in `compare_with_lifelogs.py`

### Ingest (Go)
- `cmd/ingest/main.go` - Simple HTTP client
- Clean, straightforward Go code
- Already in your codebase

## Go Approach

### Implementation

```go
// cmd/onboard/main.go
package main

import (
    "encoding/json"
    "net/http"
    "os/exec"
    "context"
)

func runDiarization(audioFile string) (map[string]interface{}, error) {
    // Call Python script as subprocess
    cmd := exec.Command("python3", "cmd/diarize/diarize.py", audioFile)
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }
    
    // Parse JSON output
    var results map[string]interface{}
    json.Unmarshal(output, &results)
    return results, nil
}
```

### Pros

1. **Language Consistency**
   - ✅ Matches your backend API (also Go)
   - ✅ Single language for backend services
   - ✅ Easier for Go developers to maintain

2. **Performance**
   - ✅ Better HTTP server performance
   - ✅ Lower memory footprint
   - ✅ Faster startup time

3. **Deployment**
   - ✅ Single binary (`go build -o hai-onboard`)
   - ✅ No Python dependencies in deployment
   - ✅ Easier distribution

4. **Concurrency**
   - ✅ Excellent goroutine support
   - ✅ Easy to handle multiple jobs concurrently
   - ✅ Better for coordinating async operations

5. **Type Safety**
   - ✅ Compile-time error checking
   - ✅ Better IDE support
   - ✅ Fewer runtime errors

### Cons

1. **Diarization Integration**
   - ❌ Must call Python as subprocess
   - ❌ Need to parse JSON output
   - ❌ Harder to get real-time progress
   - ❌ Error handling more complex

2. **Code Duplication**
   - ❌ Need to reimplement audio fetching (or call Go ingest code)
   - ❌ Two separate codebases for similar logic

3. **Development Experience**
   - ❌ Need to switch between Go and Python
   - ❌ Harder to debug Python subprocess calls

## Python Approach

### Implementation

```python
# cmd/onboard/server.py
from flask import Flask, request, jsonify
from cmd.diarize.diarize import run_diarization
import subprocess

app = Flask(__name__)

@app.route('/api/submit', methods=['POST'])
def submit_job():
    data = request.json
    api_key = data['apiKey']
    start_date = data['startDate']
    end_date = data['endDate']
    
    # Direct function call - no subprocess!
    results = run_diarization(audio_file, hf_token)
    
    return jsonify({'status': 'complete', 'results': results})
```

### Pros

1. **Direct Integration**
   - ✅ Import diarization function directly
   - ✅ No subprocess overhead
   - ✅ Easy to get real-time progress
   - ✅ Share code between diarization and onboarding

2. **Simpler Code**
   - ✅ Reuse existing diarization code
   - ✅ Same language as ML/AI components
   - ✅ Easier to pass complex objects

3. **Development Speed**
   - ✅ Faster iteration (no compilation)
   - ✅ Easier debugging
   - ✅ Better for prototyping

4. **Ecosystem**
   - ✅ Flask/FastAPI for HTTP (simple, mature)
   - ✅ Python Docker client (docker-py)
   - ✅ Python Elasticsearch client (elasticsearch-py)
   - ✅ All libraries you need

### Cons

1. **Language Mixing**
   - ❌ Backend API is Go, onboarding is Python
   - ❌ Two languages to maintain
   - ❌ Different deployment processes

2. **Performance**
   - ❌ Slower HTTP server (though fine for onboarding)
   - ❌ Higher memory usage
   - ❌ Slower startup (imports, etc.)

3. **Deployment**
   - ❌ Need Python environment
   - ❌ Need to manage dependencies
   - ❌ More complex deployment

4. **Type Safety**
   - ❌ Runtime errors more common
   - ❌ Less IDE support

## Hybrid Approach

### Option: Go Server + Python Worker

```go
// Go server coordinates, Python does heavy lifting
func (s *Server) processJob(jobID string) {
    // Start Python worker as subprocess
    cmd := exec.Command("python3", "cmd/onboard/worker.py", jobID)
    cmd.Stdout = os.Stdout  // Stream output
    cmd.Run()
}
```

**Pros**:
- ✅ Go for HTTP server (performance)
- ✅ Python for diarization (direct integration)
- ✅ Clear separation of concerns

**Cons**:
- ❌ More complex architecture
- ❌ Still need subprocess communication

## Recommendation: **Go with Smart Subprocess Handling**

### Why Go?

1. **You're already using Go** for backend services
2. **Better long-term consistency** - all backend in Go
3. **Performance matters** for HTTP server (even if diarization is slow)
4. **Single binary deployment** is cleaner
5. **Better concurrency** for managing multiple jobs

### How to Make It Work Well

#### 1. Use Structured JSON Communication

```go
// Call Python with structured input/output
type DiarizationRequest struct {
    AudioFile string `json:"audio_file"`
    HFToken   string `json:"hf_token"`
    Force     bool   `json:"force"`
}

func runDiarization(req DiarizationRequest) (*DiarizationResult, error) {
    // Write request to stdin
    input, _ := json.Marshal(req)
    cmd := exec.Command("python3", "cmd/diarize/diarize.py", "--json")
    cmd.Stdin = bytes.NewReader(input)
    
    // Read structured output
    output, _ := cmd.Output()
    var result DiarizationResult
    json.Unmarshal(output, &result)
    return &result, nil
}
```

#### 2. Stream Progress Updates

```go
// Stream Python stdout for progress
func runDiarizationWithProgress(audioFile string, progressChan chan<- string) {
    cmd := exec.Command("python3", "cmd/diarize/diarize.py", audioFile)
    cmd.Stdout = os.Stdout  // Stream to Go
    
    // Parse progress from stdout
    scanner := bufio.NewScanner(cmd.Stdout)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.Contains(line, "Progress:") {
            progressChan <- extractProgress(line)
        }
    }
}
```

#### 3. Reuse Go Ingest Code

```go
// Import your existing ingest package
import "yourproject/cmd/ingest"

func fetchAudio(apiKey string, start, end time.Time) error {
    // Reuse existing Go code!
    return ingest.DownloadAudio(apiKey, start, end, outputPath)
}
```

#### 4. Error Handling

```go
func runDiarizationSafe(audioFile string) (*Result, error) {
    cmd := exec.Command("python3", "cmd/diarize/diarize.py", audioFile)
    
    var stderr bytes.Buffer
    cmd.Stderr = &stderr
    
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("diarization failed: %v\n%s", err, stderr.String())
    }
    
    // Parse and return
}
```

## When Python Makes More Sense

Choose Python if:
- ✅ You want fastest development (prototype quickly)
- ✅ You need tight integration with diarization (real-time progress)
- ✅ You're comfortable with mixed-language codebases
- ✅ Deployment complexity isn't a concern

## Final Recommendation

**Use Go** because:

1. **Consistency**: Your backend is Go, keep it consistent
2. **Performance**: Better HTTP server performance
3. **Deployment**: Single binary is cleaner
4. **Maintainability**: One language for backend services
5. **Subprocess is fine**: Python subprocess works well for this use case

The subprocess overhead is minimal compared to diarization time (minutes), and you get better long-term maintainability.

### Implementation Strategy

1. **Start with Go** for onboarding server
2. **Call Python diarization** as subprocess (it's already a CLI tool)
3. **Reuse Go ingest code** for audio fetching
4. **Use structured JSON** for communication
5. **Stream progress** from Python stdout

This gives you the best of both worlds: Go performance and consistency, with Python doing the heavy ML lifting.












