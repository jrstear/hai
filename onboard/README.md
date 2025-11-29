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

## Project Structure

```
onboard/
├── cmd/
│   └── server/          # Main application entry point
│       └── main.go
├── internal/
│   ├── diarization/     # Diarization subprocess integration
│   ├── docker/          # Docker lifecycle management
│   ├── ingest/          # Audio fetching from Limitless API
│   ├── import/          # Elasticsearch data import
│   └── server/          # HTTP server and handlers
├── templates/           # HTML templates
├── static/              # Static assets (CSS, JS)
├── bin/                 # Build output (gitignored)
├── Taskfile.yml         # Task runner configuration
└── go.mod               # Go module definition
```

## Configuration

The server can be configured via environment variables and CLI flags:

- `--port`: Server port (default: 3000)
- `--no-open`: Don't automatically open browser
- `HF_TOKEN`: Hugging Face token for diarization (required)

## License

MIT

