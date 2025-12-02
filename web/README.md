# Hai Web Server

Web frontend server for the Hai audio lifelog processing system. Serves HTML pages and static assets, calling the API server for data.

## Overview

The web server provides:
- Multi-page web application (HTML/JS/CSS)
- Server-rendered HTML templates
- Static asset serving
- API integration (calls `hai-api` server)

## Prerequisites

- Go 1.21+
- `hai-api` server running (default: http://localhost:8080)

## Installation

```bash
# Download dependencies
go mod download

# Build binary
go build -o ../bin/hai-web ./cmd/server
```

## Usage

### Development

```bash
# Set API server URL (default: http://localhost:8080)
export API_URL="http://localhost:8080"

# Run server (default port 3000)
go run cmd/server/main.go

# Or with custom port and API URL
go run cmd/server/main.go --port 3000 --api-url http://localhost:8080
```

### Production

```bash
# Build binary
go build -o ../bin/hai-web ./cmd/server

# Run binary
./bin/hai-web --port 3000 --api-url http://localhost:8080
```

## Pages

- `/` - Home/landing page
- `/contacts` - Contacts page (displays contacts from API)

## Configuration

- `--port` (optional): Web server port (default: 3000)
- `--api-url` (optional): API server URL (default: http://localhost:8080)
- `LIMITLESS_API_KEY` (required for audio playback): Limitless API key for proxying audio requests (single-user mode)

## Architecture

- **Templates**: Server-rendered HTML using Go templates
- **Static Assets**: CSS, JS, images served from `/static/`
- **API Client**: HTTP client for calling `hai-api` server
- **Router**: chi router for HTTP routing

## Project Structure

```
web/
├── cmd/
│   └── server/          # hai-web binary entry point
├── internal/
│   └── server/          # Web server setup and handlers
├── templates/           # HTML templates
│   ├── base.html        # Base template with shared layout
│   ├── index.html       # Home page
│   └── contacts.html    # Contacts page
├── static/              # Static assets
│   ├── css/             # Stylesheets
│   └── js/              # JavaScript
└── go.mod
```

## Development

### Adding New Pages

1. Create template in `templates/` (extends `base.html`)
2. Add handler in `internal/server/handlers.go`
3. Register route in `cmd/server/main.go`

### Template Structure

Templates use Go's template system with block inheritance:
- `base.html` - Base template with shared layout
- Other templates define blocks: `title`, `content`, etc.

## Dependencies

- `github.com/go-chi/chi/v5` - HTTP router
- `github.com/go-chi/chi/v5/middleware` - Middleware

