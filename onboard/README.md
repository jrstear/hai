# Hai Onboarding Server

Onboarding server for the Hai audio lifelog processing system. Provides a web interface for users to set up and process their audio data.

## Features

- Web-based onboarding UI
- Audio fetching from Limitless API
- Speaker diarization integration (Python subprocess)
- Docker service management
- Data import to Elasticsearch

## Prerequisites

- Go 1.21+
- Python 3.10+ with pyannote.audio (for diarization)
- Docker and docker-compose (optional, for full stack)

## Installation

### Install Task (Task Runner)

```bash
# macOS
brew install go-task/tap/go-task

# Or via Go
go install github.com/go-task/task/v3/cmd/task@latest
```

### Setup

```bash
# Download dependencies
task deps

# Build
task build

# Run
task run
```

## Usage

### Setup

```bash
# Setup Python environment (one-time)
task setup-python

# Make sure HF_TOKEN is set
export HF_TOKEN='your-huggingface-token-here'
```

### Development

```bash
# Run server (checks Python first, builds, then runs)
task run

# Or with custom port
task run -- --port 8080

# Run with auto-reload (requires air: go install github.com/cosmtrek/air@latest)
task dev
```

### Production

```bash
# Build binary
task build

# Run binary
./bin/hai-onboard --port 3000

# Or with custom output directory
./bin/hai-onboard --port 3000 --output-dir /path/to/audio
```

### Web Interface

1. Start the server: `task run`
2. Browser should open automatically to `http://localhost:3000`
3. Enter your Limitless API key
4. Select date range
5. Click "Start Processing"
6. Watch progress as it fetches audio and runs diarization

## CLI Utilities

### Delete Elasticsearch Indexes

Delete Elasticsearch indexes (useful for schema migrations):

```bash
# Build the command
task build-delete-es-indexes

# Delete default indexes (segments, speakers, speaker_embeddings)
task delete-es-indexes -- -confirm

# Delete specific indexes
task delete-es-indexes -- -confirm -indexes "segments,speakers"

# Or run directly
./bin/delete-es-indexes -confirm -indexes "segments,speakers,speaker_embeddings"
```

**Warning:** This is a destructive operation. All data in the specified indexes will be permanently deleted.

### Create Speakers

Run speaker clustering to create/update speakers from stored embeddings:

```bash
# Build the command
task build-create-speakers

# Run clustering with default settings (eps=0.15, minSamples=2)
task create-speakers

# With custom parameters
task create-speakers -- -eps 0.20 -min-samples 3

# With verbose output
task create-speakers -- -verbose

# Or run directly
./bin/create-speakers -eps 0.15 -min-samples 2 -verbose
```

**What it does:**
- Loads all stored speaker embeddings from Elasticsearch
- Runs DBSCAN clustering to group similar embeddings
- Creates `Speaker` records (centroids) for each cluster
- Updates `SpeakerEmbedding.SpeakerID` and `Segment.SpeakerID` for all clustered embeddings

**Parameters:**
- `-eps`: Cosine distance threshold (default: 0.15, corresponds to similarity >= 0.85)
- `-min-samples`: Minimum points required to form a cluster (default: 2)

## Project Structure

```
onboard/
├── cmd/
│   ├── server/              # Main application entry point
│   ├── load-es/             # Load diarization results to Elasticsearch
│   ├── match-blockquotes-to-segments/  # Match lifelog blockquotes to segments by time overlap
│   ├── delete-es-indexes/   # Delete Elasticsearch indexes
│   ├── create-es-indexes/   # Create Elasticsearch indexes with updated schema
│   └── create-speakers/     # Run speaker clustering to create/update speakers
├── internal/
│   ├── diarization/         # Diarization subprocess integration
│   ├── docker/              # Docker lifecycle management
│   ├── ingest/              # Audio fetching from Limitless API
│   ├── import/              # Elasticsearch data import
│   └── server/              # HTTP server and handlers
├── templates/               # HTML templates
├── static/                  # Static assets (CSS, JS)
├── bin/                     # Build output (gitignored)
├── Taskfile.yml             # Task runner configuration
└── go.mod                   # Go module definition
```

## Configuration

The server can be configured via environment variables and CLI flags:

- `--port`: Server port (default: 3000)
- `--no-open`: Don't automatically open browser
- `HF_TOKEN`: Hugging Face token for diarization (required)

## License

MIT

