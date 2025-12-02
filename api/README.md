# Hai API Server

Backend REST API server for the Hai audio lifelog processing system.

## Overview

The API server provides REST endpoints for:
- Contacts management
- Speakers queries
- Recordings and segments
- Future: Calendar, person details, etc.

## Prerequisites

- Go 1.24.5+
- Elasticsearch (running and accessible)
- `ELASTICSEARCH_URL` environment variable set

## Installation

```bash
# Download dependencies
go mod download

# Build binary
go build -o ../bin/hai-api ./cmd/server
```

## Usage

### Development

```bash
# Set Elasticsearch URL
export ELASTICSEARCH_URL="http://localhost:9200"

# Run server (default port 8080)
go run cmd/server/main.go

# Or with custom port
go run cmd/server/main.go --port 8080
```

### Production

```bash
# Build binary
go build -o ../bin/hai-api ./cmd/server

# Run binary
./bin/hai-api --port 8080
```

## API Endpoints

### Health Check

```
GET /api/health
```

Returns:
```json
{
  "status": "ok"
}
```

### Contacts Endpoints

- `GET /api/contacts` - List contacts (with filters: `?known=true&search=john`)
- `GET /api/contacts/:id` - Get contact by ID
- `POST /api/contacts` - Create new contact
- `PUT /api/contacts/:id` - Update contact
- `POST /api/contacts/upload` - Upload and import vCard file
- `POST /api/contacts/:contactId/associate-speaker` - Associate speaker with contact

### Speakers Endpoints

- `GET /api/speakers/unassociated` - List speakers without associated contacts (with duration and last_seen)
- `GET /api/speakers/:speakerId/recordings` - Get recordings/segments for a speaker (with transcripts, sortable by duration or time)

### Recordings Endpoints

- `GET /api/recordings/:recordingId/audio` - Get Limitless API information for streaming audio (with optional `?start=X&end=Y` time range in seconds, relative to recording start)
  - Returns JSON with Limitless API URL, query parameters, and headers needed
  - Client must call Limitless API directly with their own API key
  - Returns information including:
    - `api_url`: Limitless API endpoint
    - `query_params`: `startMs` and `endMs` (milliseconds since epoch)
    - `headers`: Required headers (client must provide `X-API-Key`)
    - `absolute_start_time` and `absolute_end_time`: RFC3339 formatted times
    - `content_type`: Expected response type (audio/ogg)

### Contact Recordings

- `GET /api/contacts/:contactId/recordings` - Get recordings/segments for a contact (aggregates from all associated speakers, sortable by duration or time)

## Configuration

- `ELASTICSEARCH_URL` (required): Elasticsearch server URL (e.g., "http://localhost:9200")
- `--port` (optional): Server port (default: 8080)

**Note**: The audio streaming endpoint returns Limitless API information. Clients must provide their own `LIMITLESS_API_KEY` when calling the Limitless API directly.

## Architecture

- **Storage**: Uses `hai/storage` package (Elasticsearch backend)
- **Router**: chi router for HTTP routing
- **CORS**: Enabled for cross-origin requests (configure for production)

## Project Structure

```
api/
├── cmd/
│   └── server/          # hai-api binary entry point
├── internal/
│   ├── server/          # Server setup and handlers
│   ├── contacts/        # Contacts package (to be implemented)
│   └── handlers/        # API handlers (to be implemented)
└── go.mod
```

## Development

### Adding New Endpoints

1. Create handler in `internal/server/handlers.go` or separate file
2. Register route in `cmd/server/main.go`
3. Use `writeError()` for error responses
4. Return JSON responses

### Testing

```bash
# Test health endpoint
curl http://localhost:8080/api/health
```

## Dependencies

- `github.com/go-chi/chi/v5` - HTTP router
- `github.com/go-chi/cors` - CORS middleware
- `hai/storage` - Shared storage package

