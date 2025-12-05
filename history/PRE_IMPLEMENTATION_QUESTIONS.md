# Pre-Implementation Questions for Onboarding

## Overview

Before starting implementation of the onboarding server, we need to answer several key questions to ensure a smooth development process.

## 1. Project Structure & Organization

### Questions:
- **Where should onboarding code live?**
  - ✅ **Decision**: `cmd/onboard/` (matches existing structure: `cmd/ingest/`, `cmd/diarize/`)
  
- **How to organize Go packages?**
  - Should we create shared packages (e.g., `internal/ingest`, `internal/diarization`)?
  - ✅ **Decision**: Start with `cmd/onboard/` as self-contained, extract shared code later if needed

- **File structure?**
  ```
  cmd/onboard/
  ├── main.go              # HTTP server, routing
  ├── diarization.go       # Python subprocess integration
  ├── docker.go            # Docker lifecycle management
  ├── import.go            # ES data import (future)
  ├── templates/           # HTML templates
  │   └── index.html
  └── static/              # CSS, JS (if needed)
  ```

## 2. Dependencies & Environment

### Questions:
- **Go dependencies needed?**
  - HTTP server: `net/http` (stdlib) or `gorilla/mux`?
  - ✅ **Decision**: Start with `net/http` (stdlib), add router if needed
  - Docker client: `docker/docker/client` or exec.Command?
  - ✅ **Decision**: Use `exec.Command` for docker-compose (simpler)
  - Elasticsearch client: `olivere/elastic` or `elastic/go-elasticsearch`?
  - ✅ **Decision**: `elastic/go-elasticsearch` (official, maintained)

- **Python environment?**
  - How to ensure Python/conda environment is available?
  - ✅ **Decision**: Check for `python3` in PATH, document requirements
  - How to pass HF_TOKEN to Python subprocess?
  - ✅ **Decision**: Pass via environment variable

- **Docker requirements?**
  - How to check if Docker is installed?
  - ✅ **Decision**: Check for `docker` and `docker-compose` in PATH
  - What if Docker isn't available?
  - ✅ **Decision**: Show helpful error message, offer fallback mode

## 3. Configuration & Secrets

### Questions:
- **How to handle API keys?**
  - User provides Limitless API key via web form
  - ✅ **Decision**: Store in memory only (session-based), never persist
  - HF_TOKEN for diarization?
  - ✅ **Decision**: Read from environment variable (user must set before starting)

- **Configuration file?**
  - Do we need a config file?
  - ✅ **Decision**: No config file initially, use environment variables and CLI flags

- **Port configuration?**
  - Hardcode port 3000 or make configurable?
  - ✅ **Decision**: CLI flag `--port` with default 3000

## 4. Audio File Storage

### Questions:
- **Where to store downloaded audio?**
  - ✅ **Decision**: `data/audio/{user_id}/{date}/` (user_id from session)
  - For single-user onboarding, use `data/audio/onboarding/`

- **Temporary vs permanent storage?**
  - ✅ **Decision**: Store permanently (user might want to re-process)
  - Can add cleanup later if needed

- **Storage limits?**
  - Should we check disk space?
  - ✅ **Decision**: Not initially, add later if needed

## 5. Job Management

### Questions:
- **How to track jobs?**
  - In-memory map or persistent storage?
  - ✅ **Decision**: In-memory map initially (simple), add persistence later if needed
  - Job IDs: UUID or simple counter?
  - ✅ **Decision**: UUID (better for multi-user later)

- **Job status states?**
  - ✅ **Decision**: `pending`, `fetching`, `diarizing`, `importing`, `completed`, `failed`

- **Progress reporting?**
  - How granular should progress be?
  - ✅ **Decision**: Percentage (0-100) with status message
  - How to stream progress from Python?
  - ✅ **Decision**: Parse Python stdout for progress messages

## 6. Error Handling

### Questions:
- **What errors can occur?**
  - Limitless API failures
  - Python subprocess failures
  - Docker startup failures
  - Disk space issues
  - ✅ **Decision**: Handle all gracefully with user-friendly messages

- **Retry logic?**
  - Should we retry failed operations?
  - ✅ **Decision**: Retry API calls (3 attempts), don't retry diarization (too expensive)

- **Error logging?**
  - Where to log errors?
  - ✅ **Decision**: Log to stderr, optionally to file if `--log-file` specified

## 7. User Experience

### Questions:
- **Browser auto-open?**
  - Should we automatically open browser?
  - ✅ **Decision**: Yes, with `--no-open` flag to disable

- **Progress updates?**
  - Polling interval for status?
  - ✅ **Decision**: 2 seconds (balance between responsiveness and server load)

- **Timeout handling?**
  - How long to wait for diarization?
  - ✅ **Decision**: No timeout (diarization can take hours), but show progress

## 8. Integration Points

### Questions:
- **How to call existing ingest code?**
  - Import as package or duplicate logic?
  - ✅ **Decision**: Extract shared code to `internal/ingest` package, import it

- **How to call diarization?**
  - ✅ **Decision**: Call `cmd/diarize/diarize.py` as subprocess
  - Pass audio file path, read JSON output

- **Docker integration?**
  - ✅ **Decision**: Use `exec.Command` to run `docker-compose up -d`
  - Poll health endpoints to detect readiness

## 9. Testing Strategy

### Questions:
- **How to test onboarding flow?**
  - Mock Limitless API?
  - ✅ **Decision**: Use test fixtures for small audio files
  - Mock Python subprocess?
  - ✅ **Decision**: Test with real Python script, use small test files

- **Integration tests?**
  - ✅ **Decision**: Test full flow with test audio file (< 1 minute)

## 10. Public Sharing Strategy

### Questions:
- **What to share publicly?**
  - ✅ **Decision**: Onboarding code, diarization code, basic structure
  - ❌ **Don't share**: Personal data, API keys, private history docs

- **How to structure for sharing?**
  - ✅ **Decision**: Use git subtree (see below)

## Decisions Summary

### ✅ Confirmed Decisions:
1. **Structure**: `cmd/onboard/` with separate files for concerns
2. **Dependencies**: stdlib HTTP, exec.Command for Docker, official ES client
3. **Configuration**: Environment variables + CLI flags
4. **Storage**: `data/audio/onboarding/` for single-user
5. **Jobs**: In-memory map with UUIDs
6. **Progress**: Percentage with status messages
7. **Errors**: Graceful handling with user-friendly messages
8. **Integration**: Extract shared ingest code, call Python as subprocess
9. **Testing**: Real Python script with test fixtures

### 🔲 Still Need Decisions:
- None! Ready to start implementation.

## Next Steps

1. ✅ Answer all questions (done above)
2. 🔲 Create `cmd/onboard/` directory structure
3. 🔲 Set up Go module dependencies
4. 🔲 Implement basic HTTP server
5. 🔲 Add diarization subprocess integration
6. 🔲 Add Docker management
7. 🔲 Create HTML template
8. 🔲 Test end-to-end flow











