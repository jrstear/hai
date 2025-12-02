# New Architecture Structure

## Overview

Separate the main app from the onboarding server with clean API/web separation:

- **`onboard/`** - Stays as-is, onboarding-specific (one-time setup)
- **`api/`** - Backend API server (REST API, JSON responses)
- **`web/`** - Web frontend server (serves HTML, static assets, calls API)

## Directory Structure

```
hai/
├── onboard/              # Onboarding server (unchanged)
│   ├── cmd/server/      # hai-onboard binary
│   ├── internal/        # Onboarding-specific code
│   └── templates/       # Onboarding UI
│
├── api/                  # Backend API server (NEW)
│   ├── cmd/
│   │   └── server/      # hai-api binary
│   ├── internal/
│   │   ├── handlers/    # API handlers
│   │   │   ├── contacts.go
│   │   │   ├── speakers.go
│   │   │   └── recordings.go
│   │   ├── contacts/    # Contacts package
│   │   │   ├── contacts.go
│   │   │   ├── vcard.go
│   │   │   └── types.go
│   │   └── server/      # Server setup
│   │       ├── router.go
│   │       └── middleware.go
│   └── go.mod
│
├── web/                  # Web frontend server (NEW)
│   ├── cmd/
│   │   └── server/      # hai-web binary
│   ├── internal/
│   │   └── server/      # Web server setup
│   │       ├── router.go
│   │       └── templates.go
│   ├── static/          # Static assets (CSS, JS, images)
│   ├── templates/       # HTML templates
│   │   ├── base.html
│   │   ├── contacts.html
│   │   ├── calendar.html
│   │   └── ...
│   └── go.mod
│
├── storage/              # Shared storage layer (unchanged)
├── cmd/                  # Shared CLI tools (unchanged)
└── ...
```

## Binary Names

- **`hai-onboard`** - Onboarding server (from `onboard/cmd/server/`)
- **`hai-api`** - Backend API server (from `api/cmd/server/`)
- **`hai-web`** - Web frontend server (from `web/cmd/server/`)

## Server Responsibilities

### `onboard/` (unchanged)
- One-time setup flow
- Fetch audio from Limitless API
- Run diarization
- Import to Elasticsearch
- Simple UI for initial setup

### `api/` (new)
- **Backend API server** (REST API, JSON only)
- Contacts management endpoints
- Speakers/recordings queries
- Calendar integration (future)
- Person detail data (future)
- **No HTML rendering** - pure API
- CORS enabled for cross-origin requests
- Can be consumed by web, mobile, or other clients

### `web/` (new)
- **Web frontend server** (serves HTML, static assets)
- Multi-page web app
- Contacts page
- Calendar page (future)
- Person detail pages (future)
- Static assets (CSS, JS, images)
- Templates (server-rendered HTML)
- **Calls `api/` server** for data (via HTTP)

## Implementation Plan

### Phase 1: Create Structure
1. Create `api/` directory
2. Create `web/` directory
3. Set up `api/go.mod` and `web/go.mod`
4. Move/copy shared code as needed

### Phase 2: Build API Server
1. Create `api/cmd/server/main.go` (hai-api binary)
2. Set up router (chi)
3. Create API handlers (JSON responses only)
4. Create contacts package
5. Add CORS middleware
6. Test API endpoints

### Phase 3: Build Web Server
1. Create `web/cmd/server/main.go` (hai-web binary)
2. Set up router (chi)
3. Create template rendering
4. Create base template
5. Create contacts page
6. Add static asset serving
7. Connect to API server (HTTP client)

### Phase 4: Integration
1. Configure API URL in web server
2. Test full flow (web → API → storage)
3. Add error handling
4. Add development/production configs

## Shared Code

**What stays shared:**
- `storage/` - Used by onboard, api, and potentially web
- `cmd/diarize/` - Used by onboard
- `cmd/ingest/` - Could be used by both

**What goes where:**
- Contacts code → `api/internal/contacts/`
- API handlers → `api/internal/handlers/`
- Web templates → `web/templates/`
- Web server code → `web/internal/server/`

## Benefits

1. **Clear separation**: Onboarding vs main app vs API vs web
2. **Independent scaling**: API and web can scale independently
3. **Independent deployment**: Deploy API and web separately
4. **API-first**: API can be consumed by web, mobile, or other clients
5. **Technology flexibility**: Web server could be replaced with SPA + CDN later
6. **Future mobile**: Mobile app can call same API as web
7. **Cleaner structure**: Each directory has clear purpose

## Migration Strategy

1. **Create new structure** alongside existing
2. **Build API server** with endpoints
3. **Build web server** that calls API
4. **Move web templates** to `web/`
5. **Test everything** works
6. **Keep onboard unchanged** throughout

## Scaling Benefits

**Separate API and Web servers enable:**

1. **Independent scaling**:
   - API server: Scale based on API load
   - Web server: Scale based on web traffic
   - Can run multiple API instances behind load balancer
   - Can run web server as CDN/static hosting

2. **Different deployment strategies**:
   - API: Deploy to API gateway, Kubernetes, etc.
   - Web: Deploy to CDN, static hosting, or separate servers

3. **Technology flexibility**:
   - API: Always Go (consistent)
   - Web: Could be Go now, SPA + CDN later, or both

4. **Multi-client support**:
   - Web app calls API
   - Mobile app calls same API
   - Future: CLI tools, integrations, etc.

5. **Development flexibility**:
   - Develop API and web independently
   - Test API with curl/Postman
   - Test web with mock API or real API

## Configuration

**API Server** (`hai-api`):
- Port: 8080 (default)
- Environment: `API_PORT`, `ELASTICSEARCH_URL`

**Web Server** (`hai-web`):
- Port: 3000 (default)
- Environment: `WEB_PORT`, `API_URL` (points to API server)

**Development**:
```bash
# Terminal 1: Start API server
./bin/hai-api --port 8080

# Terminal 2: Start web server
./bin/hai-web --port 3000 --api-url http://localhost:8080
```

**Production**:
- API: Multiple instances behind load balancer
- Web: Single instance or CDN

