# Progressive Onboarding Architecture

## Overview

A smart architecture that starts with a native macOS process (for MPS acceleration), then progressively launches Docker services for full functionality. This provides the best of both worlds: fast diarization and containerized services.

## Architecture Flow

```
┌─────────────────────────────────────────────────────────────┐
│  Phase 1: Native macOS Process (Onboarding)                 │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  Native Web Server (Go/Python)                        │  │
│  │  Port: 3000                                           │  │
│  │  ├── Simple onboarding UI                             │  │
│  │  ├── API key input                                    │  │
│  │  ├── Date range selection                             │  │
│  │  └── Job submission                                   │  │
│  └───────────────────────────────────────────────────────┘  │
│           ↓                                                  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  Native Diarization Worker                            │  │
│  │  ✅ Uses MPS acceleration                             │  │
│  │  ✅ Fetches audio from Limitless API                  │  │
│  │  ✅ Processes with pyannote                           │  │
│  └───────────────────────────────────────────────────────┘  │
│           ↓                                                  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  Background: Start Docker Services                    │  │
│  │  ├── docker-compose up -d                             │  │
│  │  ├── Wait for services to be healthy                  │  │
│  │  └── Import data to Elasticsearch                     │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│  Phase 2: Docker Services (Full App)                        │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  Frontend (React/Vue)                                 │  │
│  │  Port: 3001                                           │  │
│  │  ├── Full-featured UI                                 │  │
│  │  ├── Query interface                                  │  │
│  │  ├── Audio playback                                   │  │
│  │  └── Analytics dashboard                              │  │
│  └───────────────────────────────────────────────────────┘  │
│           ↕                                                  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  Backend API (Go)                                     │  │
│  │  Port: 8080                                           │  │
│  │  ├── Elasticsearch queries                            │  │
│  │  ├── Audio serving                                    │  │
│  │  └── Job management                                   │  │
│  └───────────────────────────────────────────────────────┘  │
│           ↕                                                  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  Elasticsearch                                        │  │
│  │  Port: 9200                                           │  │
│  │  └── Full data storage & querying                     │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## User Experience Flow

### Step 1: Launch Native App
```bash
# User runs native macOS app
./hai-onboard
# or
python cmd/onboard/main.py
```

### Step 2: Onboarding UI (Native Server)
- Browser opens to `http://localhost:3000`
- Simple form:
  - Limitless API key input
  - Date range picker
  - "Start Processing" button
- Shows: "Setting up your audio lifelog..."

### Step 3: Processing (Native)
- Native process:
  1. Fetches audio from Limitless API
  2. Diarizes using MPS acceleration
  3. Saves results to temporary storage
  4. Shows progress: "Processing audio... 45% complete"

### Step 4: Docker Launch (Background)
- Native process starts Docker in background:
  ```bash
  docker-compose up -d
  ```
- Polls for service health:
  - Elasticsearch: `http://localhost:9200/_cluster/health`
  - Backend: `http://localhost:8080/health`
  - Frontend: `http://localhost:3001`

### Step 5: Data Import
- Once Docker services are ready:
  - Import diarization results to Elasticsearch
  - Index speakers, segments, recordings
  - Show: "Importing data... 80% complete"

### Step 6: Switch to Full App
- Native UI detects Docker services ready
- Shows: "Setup complete! Opening full app..."
- Redirects to: `http://localhost:3001` (Docker-served frontend)
- Or: Opens new tab with full app

## Implementation Details

### Native Onboarding Server

**Technology**: Go (matches your existing stack)

**File**: `cmd/onboard/main.go`

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os/exec"
    "time"
    "context"
)

type OnboardingServer struct {
    dockerStarted bool
    dockerReady   bool
    processing    bool
    jobID         string
}

func main() {
    server := &OnboardingServer{}
    
    // Start native web server
    http.HandleFunc("/", server.handleIndex)
    http.HandleFunc("/api/submit", server.handleSubmit)
    http.HandleFunc("/api/status", server.handleStatus)
    http.HandleFunc("/api/docker-status", server.handleDockerStatus)
    
    fmt.Println("Starting onboarding server on http://localhost:3000")
    fmt.Println("Open your browser to get started!")
    
    log.Fatal(http.ListenAndServe(":3000", nil))
}

func (s *OnboardingServer) handleIndex(w http.ResponseWriter, r *http.Request) {
    // Serve simple HTML onboarding page
    html := `
    <!DOCTYPE html>
    <html>
    <head>
        <title>Hai Audio Lifelog - Setup</title>
        <style>
            body { font-family: system-ui; max-width: 600px; margin: 50px auto; }
            input, button { padding: 10px; margin: 5px; width: 100%; }
            .status { margin: 20px 0; padding: 15px; background: #f0f0f0; }
        </style>
    </head>
    <body>
        <h1>Welcome to Hai Audio Lifelog</h1>
        <div id="onboarding">
            <h2>Step 1: Enter Your Details</h2>
            <input type="text" id="apiKey" placeholder="Limitless API Key" />
            <input type="date" id="startDate" />
            <input type="date" id="endDate" />
            <button onclick="submitJob()">Start Processing</button>
        </div>
        <div id="status" class="status" style="display:none;">
            <h3>Status</h3>
            <div id="statusText">Initializing...</div>
            <div id="progress"></div>
        </div>
        <script>
            let jobId = null;
            
            async function submitJob() {
                const apiKey = document.getElementById('apiKey').value;
                const startDate = document.getElementById('startDate').value;
                const endDate = document.getElementById('endDate').value;
                
                const response = await fetch('/api/submit', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ apiKey, startDate, endDate })
                });
                
                const data = await response.json();
                jobId = data.jobId;
                
                document.getElementById('onboarding').style.display = 'none';
                document.getElementById('status').style.display = 'block';
                
                pollStatus();
            }
            
            async function pollStatus() {
                const statusRes = await fetch('/api/status?jobId=' + jobId);
                const status = await statusRes.json();
                
                document.getElementById('statusText').textContent = status.message;
                if (status.progress) {
                    document.getElementById('progress').textContent = 
                        `Progress: ${status.progress}%`;
                }
                
                // Check if Docker is ready
                const dockerRes = await fetch('/api/docker-status');
                const docker = await dockerRes.json();
                
                if (docker.ready && status.complete) {
                    // Switch to full app
                    setTimeout(() => {
                        window.location.href = 'http://localhost:3001';
                    }, 2000);
                    return;
                }
                
                if (!status.complete) {
                    setTimeout(pollStatus, 2000);
                }
            }
        </script>
    </body>
    </html>
    `
    w.Header().Set("Content-Type", "text/html")
    w.Write([]byte(html))
}

func (s *OnboardingServer) handleSubmit(w http.ResponseWriter, r *http.Request) {
    var req struct {
        APIKey    string `json:"apiKey"`
        StartDate string `json:"startDate"`
        EndDate   string `json:"endDate"`
    }
    
    json.NewDecoder(r.Body).Decode(&req)
    
    // Generate job ID
    jobID := fmt.Sprintf("job_%d", time.Now().Unix())
    s.jobID = jobID
    s.processing = true
    
    // Start processing in background
    go s.processJob(req.APIKey, req.StartDate, req.EndDate)
    
    // Start Docker in background
    go s.startDocker()
    
    json.NewEncoder(w).Encode(map[string]string{
        "jobId": jobID,
        "status": "processing",
    })
}

func (s *OnboardingServer) processJob(apiKey, startDate, endDate string) {
    // 1. Fetch audio (call existing Go ingest code)
    // 2. Run diarization (call Python script with MPS)
    // 3. Save results to temp location
    // 4. Mark as complete
    s.processing = false
}

func (s *OnboardingServer) startDocker() {
    // Start Docker Compose
    cmd := exec.Command("docker-compose", "up", "-d")
    cmd.Run()
    
    s.dockerStarted = true
    
    // Poll for Docker services to be ready
    s.waitForDocker()
}

func (s *OnboardingServer) waitForDocker() {
    client := &http.Client{Timeout: 2 * time.Second}
    
    for {
        // Check Elasticsearch
        resp, err := client.Get("http://localhost:9200/_cluster/health")
        if err == nil && resp.StatusCode == 200 {
            // Check Backend
            resp, err = client.Get("http://localhost:8080/health")
            if err == nil && resp.StatusCode == 200 {
                // Check Frontend
                resp, err = client.Get("http://localhost:3001")
                if err == nil && resp.StatusCode == 200 {
                    s.dockerReady = true
                    return
                }
            }
        }
        
        time.Sleep(2 * time.Second)
    }
}

func (s *OnboardingServer) handleStatus(w http.ResponseWriter, r *http.Request) {
    status := map[string]interface{}{
        "processing": s.processing,
        "complete":   !s.processing,
        "message":    "Processing audio...",
    }
    
    if s.processing {
        status["progress"] = 45 // Calculate actual progress
    }
    
    json.NewEncoder(w).Encode(status)
}

func (s *OnboardingServer) handleDockerStatus(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(map[string]bool{
        "started": s.dockerStarted,
        "ready":   s.dockerReady,
    })
}
```

### Docker Compose Configuration

**File**: `docker-compose.yml`

```yaml
version: '3.8'

services:
  frontend:
    build: ./web
    ports:
      - "3001:3000"  # Different port from native server
    environment:
      - API_URL=http://backend:8080
    depends_on:
      - backend

  backend:
    build: ./cmd/api
    ports:
      - "8080:8080"
    environment:
      - ELASTICSEARCH_URL=http://elasticsearch:9200
      - REDIS_URL=redis:6379
    depends_on:
      - elasticsearch
      - redis
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 5s
      timeout: 3s
      retries: 3

  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.11.0
    ports:
      - "9200:9200"
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
      - "ES_JAVA_OPTS=-Xms1g -Xmx1g"
    healthcheck:
      test: ["CMD-SHELL", "curl -f http://localhost:9200/_cluster/health || exit 1"]
      interval: 5s
      timeout: 3s
      retries: 3

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
```

### Data Import Script

**File**: `cmd/onboard/import_to_docker.go`

```go
package main

import (
    "encoding/json"
    "io/ioutil"
    "net/http"
    "path/filepath"
)

func importDiarizationResults(resultsPath string) error {
    // Read diarization results
    data, err := ioutil.ReadFile(resultsPath)
    if err != nil {
        return err
    }
    
    var results DiarizationResults
    json.Unmarshal(data, &results)
    
    // Import to Elasticsearch
    esClient := elasticsearch.NewClient(...)
    
    // Index speakers
    for speakerID, embedding := range results.SpeakerEmbeddings {
        // Match speaker or create new
        // Index to ES
    }
    
    // Index segments
    for _, segment := range results.Segments {
        // Index to ES
    }
    
    return nil
}
```

## Alternative: Python-Based Onboarding

If you prefer Python for the onboarding server:

**File**: `cmd/onboard/server.py`

```python
from flask import Flask, render_template, request, jsonify
import subprocess
import threading
import time
import requests
from diarize import run_diarization

app = Flask(__name__)

docker_ready = False
processing = False

@app.route('/')
def index():
    return render_template('onboarding.html')

@app.route('/api/submit', methods=['POST'])
def submit():
    data = request.json
    api_key = data['apiKey']
    start_date = data['startDate']
    end_date = data['endDate']
    
    # Start processing in background
    thread = threading.Thread(
        target=process_job,
        args=(api_key, start_date, end_date)
    )
    thread.start()
    
    # Start Docker in background
    docker_thread = threading.Thread(target=start_docker)
    docker_thread.start()
    
    return jsonify({'status': 'processing', 'jobId': 'job_123'})

def process_job(api_key, start_date, end_date):
    global processing
    processing = True
    
    # 1. Fetch audio
    # 2. Run diarization (uses MPS)
    results = run_diarization(audio_file, hf_token)
    
    # 3. Save results
    # 4. Import to Docker (once ready)
    processing = False

def start_docker():
    global docker_ready
    
    # Start Docker Compose
    subprocess.run(['docker-compose', 'up', '-d'])
    
    # Wait for services
    while not docker_ready:
        try:
            # Check Elasticsearch
            resp = requests.get('http://localhost:9200/_cluster/health', timeout=2)
            if resp.status_code == 200:
                # Check backend
                resp = requests.get('http://localhost:8080/health', timeout=2)
                if resp.status_code == 200:
                    docker_ready = True
                    # Import data
                    import_to_elasticsearch()
        except:
            pass
        time.sleep(2)

if __name__ == '__main__':
    app.run(port=3000, debug=True)
```

## Benefits of This Architecture

### ✅ Advantages

1. **Fast Onboarding**: No Docker setup needed initially
2. **MPS Acceleration**: Native diarization uses full M1 power
3. **Progressive Enhancement**: Starts simple, upgrades to full app
4. **Easy Sharing**: Others can use full Docker if preferred
5. **Better UX**: User sees progress, smooth transition
6. **Flexible**: Can run native-only or Docker-only modes

### ⚠️ Considerations

1. **Port Management**: Need to coordinate ports (3000 native, 3001 Docker)
2. **Data Migration**: Import from native temp storage to Docker
3. **Error Handling**: What if Docker fails to start?
4. **Resource Usage**: Running both native and Docker simultaneously

## Error Handling

### Docker Fails to Start

```go
func (s *OnboardingServer) startDocker() {
    cmd := exec.Command("docker-compose", "up", "-d")
    if err := cmd.Run(); err != nil {
        // Fallback: Continue with native-only mode
        // Show message: "Docker unavailable, using native mode"
        s.fallbackToNative()
        return
    }
    // ... continue with Docker setup
}
```

### Services Don't Become Ready

```go
func (s *OnboardingServer) waitForDocker() {
    timeout := time.After(5 * time.Minute)
    ticker := time.NewTicker(2 * time.Second)
    
    for {
        select {
        case <-timeout:
            // Timeout: Show error, offer native-only mode
            s.handleDockerTimeout()
            return
        case <-ticker.C:
            // Check services
            if s.checkServices() {
                s.dockerReady = true
                return
            }
        }
    }
}
```

## Deployment Options

### Option 1: Single Binary (Go)

Build everything into one binary:
```bash
go build -o hai-onboard ./cmd/onboard
```

User runs:
```bash
./hai-onboard
```

### Option 2: Python Script

Simple Python script:
```bash
python cmd/onboard/server.py
```

### Option 3: macOS App Bundle

Package as macOS app:
```
Hai.app/
├── Contents/
│   ├── MacOS/hai-onboard
│   └── Resources/
└── ...
```

## Next Steps

1. **Create onboarding server** (`cmd/onboard/main.go`)
2. **Create simple HTML template** for onboarding UI
3. **Implement Docker management** (start, health checks)
4. **Implement data import** (native → Docker)
5. **Add error handling** and fallbacks
6. **Test on M1 Mac** with full flow

## Recommendation

**This architecture is excellent!** It provides:
- ✅ Best performance (MPS acceleration)
- ✅ Easy onboarding (no Docker knowledge needed)
- ✅ Full features (containerized services)
- ✅ Great UX (progressive enhancement)

Start with the Go-based onboarding server since you're already using Go for the backend. The native server can be simple and focused, then hand off to the full Docker-based app once ready.













