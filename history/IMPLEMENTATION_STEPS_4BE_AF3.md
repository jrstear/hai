# Implementation Steps: hai-af3 and hai-4be

## Architecture Overview

**New structure with separate API and web servers:**

- **`onboard/`** - Stays unchanged (onboarding server)
- **`api/`** - New backend API server (`hai-api` binary)
- **`web/`** - New web frontend server (`hai-web` binary)

See `history/NEW_ARCHITECTURE_STRUCTURE.md` for complete architecture details.

## hai-af3: Server Architecture Setup

### Goal

Create new `api/` and `web/` directory structure with separate servers for better scaling and separation of concerns.

### Backward Compatibility

**✅ `hai-onboard` binary remains completely unchanged**

- `onboard/` directory stays as-is
- No modifications to onboarding server
- New servers are separate binaries

### Step 1: Create Directory Structure

**Action**: Create new `api/` and `web/` directories

```bash
mkdir -p api/cmd/server
mkdir -p api/internal/{handlers,contacts,server}
mkdir -p web/cmd/server
mkdir -p web/internal/server
mkdir -p web/templates
mkdir -p web/static
```

**Files to create**:
- `api/go.mod` - New Go module for API server
- `web/go.mod` - New Go module for web server

**Initialize Go modules**:
```bash
cd api
go mod init hai/api

cd ../web
go mod init hai/web
```

### Step 2: Add Router Libraries

**Action**: Add chi router to both `api/` and `web/`

```bash
cd api
go get github.com/go-chi/chi/v5
go get github.com/go-chi/chi/v5/middleware
go get github.com/go-chi/cors

cd ../web
go get github.com/go-chi/chi/v5
go get github.com/go-chi/chi/v5/middleware
```

**Files to modify**:
- `api/go.mod` (will be updated by go get)
- `web/go.mod` (will be updated by go get)

### Step 3: Create API Server Structure

**Action**: Create `api/cmd/server/main.go` with basic server setup

**Files to create**:
- `api/cmd/server/main.go` - API server entry point

**Basic structure**:
```go
package main

import (
    "flag"
    "log"
    "net/http"
    "os"
    
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/go-chi/cors"
    
    "hai/api/internal/server"
)

var (
    port = flag.String("port", "8080", "API server port")
)

func main() {
    flag.Parse()
    
    // Initialize storage (Elasticsearch)
    esURL := os.Getenv("ELASTICSEARCH_URL")
    if esURL == "" {
        log.Fatal("ELASTICSEARCH_URL environment variable required")
    }
    
    // Create server instance
    srv := server.NewAPIServer(esURL)
    
    // Setup router
    r := chi.NewRouter()
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(cors.Handler(cors.Options{
        AllowedOrigins: []string{"*"}, // Configure properly for production
        AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
    }))
    
    // API routes (JSON only)
    r.Route("/api", func(r chi.Router) {
        // Health check
        r.Get("/health", srv.HandleHealth)
        
        // Future: Contacts, speakers, recordings endpoints
        // r.Get("/contacts", srv.HandleListContacts)
        // etc.
    })
    
    // Start server
    addr := ":" + *port
    log.Printf("Starting API server on http://localhost%s", addr)
    if err := http.ListenAndServe(addr, r); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
```

### Step 4: Create Web Server Structure

**Action**: Create `web/cmd/server/main.go` with basic server setup

**Files to create**:
- `web/cmd/server/main.go` - Web server entry point

**Basic structure**:
```go
package main

import (
    "flag"
    "log"
    "net/http"
    
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    
    "hai/web/internal/server"
)

var (
    port   = flag.String("port", "3000", "Web server port")
    apiURL = flag.String("api-url", "http://localhost:8080", "API server URL")
)

func main() {
    flag.Parse()
    
    // Create server instance
    srv := server.NewWebServer(*apiURL)
    
    // Setup router
    r := chi.NewRouter()
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    
    // Static assets
    r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
    
    // Page routes (HTML)
    r.Get("/", srv.HandleIndex)
    r.Get("/contacts", srv.HandleContactsPage)
    // Future: r.Get("/calendar", srv.HandleCalendarPage)
    
    // Start server
    addr := ":" + *port
    log.Printf("Starting web server on http://localhost%s", addr)
    log.Printf("API server: %s", *apiURL)
    if err := http.ListenAndServe(addr, r); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
```

### Step 5: Create API Server Package

**Action**: Create `api/internal/server/` package

**Files to create**:
- `api/internal/server/server.go` - API server struct and initialization
- `api/internal/server/handlers.go` - API handlers (JSON responses)
- `api/internal/server/errors.go` - Error response helpers

**Server struct**:
```go
// api/internal/server/server.go
package server

import (
    "hai/storage"
)

type APIServer struct {
    storage storage.Storage
}

func NewAPIServer(esURL string) (*APIServer, error) {
    esStorage, err := storage.NewElasticsearchStorage(esURL)
    if err != nil {
        return nil, err
    }
    
    return &APIServer{
        storage: esStorage,
    }, nil
}

// Health check handler
func (s *APIServer) HandleHealth(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
```

### Step 6: Create Web Server Package

**Action**: Create `web/internal/server/` package

**Files to create**:
- `web/internal/server/server.go` - Web server struct and initialization
- `web/internal/server/handlers.go` - Page handlers (HTML rendering)
- `web/internal/server/templates.go` - Template loading/rendering helper
- `web/internal/server/api_client.go` - HTTP client for calling API server

**Server struct**:
```go
// web/internal/server/server.go
package server

import (
    "net/http"
)

type WebServer struct {
    apiURL    string
    apiClient *http.Client
    templates *template.Template
}

func NewWebServer(apiURL string) (*WebServer, error) {
    // Initialize templates
    templates, err := template.ParseGlob("templates/*.html")
    if err != nil {
        return nil, err
    }
    
    return &WebServer{
        apiURL:    apiURL,
        apiClient: &http.Client{},
        templates: templates,
    }, nil
}

// Index page handler
func (s *WebServer) HandleIndex(w http.ResponseWriter, r *http.Request) {
    s.renderTemplate(w, "index.html", nil)
}

// Contacts page handler
func (s *WebServer) HandleContactsPage(w http.ResponseWriter, r *http.Request) {
    // Fetch data from API
    // Render template with data
    s.renderTemplate(w, "contacts.html", data)
}

func (s *WebServer) renderTemplate(w http.ResponseWriter, name string, data interface{}) error {
    w.Header().Set("Content-Type", "text/html")
    return s.templates.ExecuteTemplate(w, name, data)
}
```

### Step 7: Create Base Templates

**Action**: Create web templates

**Files to create**:
- `web/templates/base.html` - Base template with shared layout
- `web/templates/index.html` - Landing/index page
- `web/templates/contacts.html` - Contacts page (placeholder for now)

**Base template structure**:
```html
<!-- web/templates/base.html -->
<!DOCTYPE html>
<html>
<head>
    <title>{{block "title" .}}Hai{{end}}</title>
    <link rel="stylesheet" href="/static/css/main.css">
</head>
<body>
    <nav>
        <a href="/">Home</a>
        <a href="/contacts">Contacts</a>
    </nav>
    <main>
        {{block "content" .}}{{end}}
    </main>
    <script src="/static/js/main.js"></script>
</body>
</html>
```

### Step 8: Test Basic Setup

**Action**: Verify both servers can start and communicate

**Test Checklist**:
- ✅ API server starts on port 8080
- ✅ Web server starts on port 3000
- ✅ API health check works: `curl http://localhost:8080/api/health`
- ✅ Web index page loads: `curl http://localhost:3000/`
- ✅ Web server can call API server

**Testing Commands**:
```bash
# Terminal 1: Start API server
cd api
go run cmd/server/main.go

# Terminal 2: Start web server
cd web
go run cmd/server/main.go --api-url http://localhost:8080

# Terminal 3: Test
curl http://localhost:8080/api/health
curl http://localhost:3000/
```

**Success Criteria**: Both servers start, API returns JSON, web returns HTML.

---

## hai-4be: API Endpoints for Contacts Page

### Prerequisites
- ✅ hai-af3 (API/web server structure) - **Do this first**
- ✅ hai-9ke (contact schema) - Design done, implementation needed
- ⏳ hai-y0d (vCard parsing) - Can do in parallel

### Step 1: Create Contacts Package

**Action**: Create `api/internal/contacts/` package

**Files to create**:
- `api/internal/contacts/types.go` - Contact struct
- `api/internal/contacts/contacts.go` - ES CRUD operations
- `api/internal/contacts/vcard.go` - vCard parsing (or separate issue)

**Contact struct** (from CONTACT_STORAGE_DESIGN.md):
```go
type Contact struct {
    ID            string    `json:"id"`
    ExternalID    string    `json:"external_id"`
    Name          string    `json:"name"`
    GivenName     string    `json:"given_name"`
    FamilyName    string    `json:"family_name"`
    Email         string    `json:"email,omitempty"`
    Phone         string    `json:"phone,omitempty"`
    PictureURL    string    `json:"picture_url,omitempty"`
    FavoriteColor string    `json:"favorite_color,omitempty"`
    Known         bool      `json:"known"` // Computed
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
    Source        string    `json:"source"` // "vcf", "google", "apple", "manual"
}
```

### Step 2: Implement ES Operations

**Action**: Implement CRUD operations in `contacts.go`

**Functions to implement**:
```go
// Create ES client connection (reuse existing ES client from server)
func NewContactsClient(esURL string) (*ContactsClient, error)

// CRUD operations
func (c *ContactsClient) ListContacts(ctx context.Context, filters *ContactFilters) ([]*Contact, error)
func (c *ContactsClient) GetContact(ctx context.Context, id string) (*Contact, error)
func (c *ContactsClient) CreateContact(ctx context.Context, contact *Contact) error
func (c *ContactsClient) UpdateContact(ctx context.Context, id string, updates *Contact) error
func (c *ContactsClient) AssociateSpeaker(ctx context.Context, speakerID, contactID string) error

// Helper: Compute "known" status
func (c *ContactsClient) computeKnownStatus(ctx context.Context, contactID string) (bool, error)
```

**ContactFilters**:
```go
type ContactFilters struct {
    Known  *bool   // Filter by known/unknown
    Search string  // Search by name/email
    Source string  // Filter by source
}
```

### Step 3: Create ES Index

**Action**: Add contacts index creation to ES setup

**Files to modify**:
- `storage/elasticsearch.go` - Add `indexContacts` constant and mapping
- Or create index in `contacts.go` on first use

**ES Mapping** (from CONTACT_STORAGE_DESIGN.md):
- Add to `ensureIndices()` function in `storage/elasticsearch.go`
- Or create separately in contacts package

### Step 4: Implement API Handlers

**Action**: Create API handlers in `api/internal/server/handlers.go`

**Endpoints to implement**:

1. **GET /api/contacts**
   ```go
   func (s *Server) handleListContacts(w http.ResponseWriter, r *http.Request)
   ```
   - Query params: `?known=true&search=john&source=vcf`
   - Returns: `[]Contact`

2. **GET /api/contacts/:id**
   ```go
   func (s *Server) handleGetContact(w http.ResponseWriter, r *http.Request)
   ```
   - Path param: `id`
   - Returns: `Contact`

3. **POST /api/contacts**
   ```go
   func (s *Server) handleCreateContact(w http.ResponseWriter, r *http.Request)
   ```
   - Request body: `Contact` (without ID, timestamps)
   - Returns: `Contact` (with ID, timestamps)

4. **PUT /api/contacts/:id**
   ```go
   func (s *Server) handleUpdateContact(w http.ResponseWriter, r *http.Request)
   ```
   - Path param: `id`
   - Request body: Partial `Contact` updates
   - Returns: Updated `Contact`

5. **GET /api/speakers**
   ```go
   func (s *Server) handleListSpeakers(w http.ResponseWriter, r *http.Request)
   ```
   - Query params: `?contact_id=xxx&unassociated=true`
   - Returns: `[]Speaker` (with computed duration, last_seen)

6. **GET /api/speakers/:id/recordings**
   ```go
   func (s *Server) handleGetSpeakerRecordings(w http.ResponseWriter, r *http.Request)
   ```
   - Path param: `id`
   - Returns: `[]Recording` or `[]Segment` (need to decide format)

7. **POST /api/speakers/:id/associate**
   ```go
   func (s *Server) handleAssociateSpeaker(w http.ResponseWriter, r *http.Request)
   ```
   - Path param: `id` (speaker ID)
   - Request body: `{"contact_id": "contact_xxx"}`
   - Updates `speakers.contact_id` in ES
   - Returns: Updated speaker and contact

8. **GET /api/recordings**
   ```go
   func (s *Server) handleListRecordings(w http.ResponseWriter, r *http.Request)
   ```
   - Query params: `?speaker_id=xxx&start_time=...&end_time=...`
   - Returns: `[]Recording`

### Step 5: Register API Routes

**Action**: Add API routes to router in `api/cmd/server/main.go`

**Files to modify**:
- `api/cmd/server/main.go` - Add API route registration

**Example**:
```go
r.Route("/api", func(r chi.Router) {
    // Health check
    r.Get("/health", srv.HandleHealth)
    
    // Contacts
    r.Get("/contacts", srv.HandleListContacts)
    r.Get("/contacts/{id}", srv.HandleGetContact)
    r.Post("/contacts", srv.HandleCreateContact)
    r.Put("/contacts/{id}", srv.HandleUpdateContact)
    
    // Speakers
    r.Get("/speakers", srv.HandleListSpeakers)
    r.Get("/speakers/{id}/recordings", srv.HandleGetSpeakerRecordings)
    r.Post("/speakers/{id}/associate", srv.HandleAssociateSpeaker)
    
    // Recordings
    r.Get("/recordings", srv.HandleListRecordings)
})
```

**Note**: CORS is already added in Step 3.

### Step 6: Error Handling

**Action**: Create consistent error response format

**Files to create/modify**:
- `api/internal/server/errors.go` - Error response helpers

**Error format**:
```go
type ErrorResponse struct {
    Error   string `json:"error"`
    Code    string `json:"code,omitempty"`
    Details string `json:"details,omitempty"`
}

func writeError(w http.ResponseWriter, status int, err error) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(ErrorResponse{
        Error: err.Error(),
    })
}
```

### Step 8: Testing

**Action**: Test all endpoints

**Test cases**:
- List contacts (empty, with data)
- Get contact (exists, not found)
- Create contact (valid, invalid)
- Update contact
- List speakers (unassociated, with contact)
- Associate speaker with contact
- List recordings

**Tools**:
- `curl` commands
- Postman/Insomnia
- Browser DevTools
- Write simple test script

---

## Implementation Order

### Phase 1: Architecture (hai-af3)
1. Add chi router
2. Create route structure
3. Organize handlers
4. Template management
5. Test existing functionality

### Phase 2: Contacts API (hai-4be)
1. Create contacts package
2. Implement ES operations
3. Create ES index
4. Implement API handlers
5. Register routes
6. Add CORS
7. Error handling
8. Testing

### Dependencies
- hai-af3 should be done first (or at least router setup)
- hai-4be can start after router is in place
- hai-y0d (vCard parsing) can be done in parallel or after basic API works

## Files Summary

### New Directories to Create
- `api/` - API server
- `web/` - Web server

### New Files to Create (API)
- `api/go.mod`
- `api/cmd/server/main.go`
- `api/internal/server/server.go`
- `api/internal/server/handlers.go`
- `api/internal/server/errors.go`
- `api/internal/contacts/types.go`
- `api/internal/contacts/contacts.go`
- `api/internal/contacts/vcard.go`

### New Files to Create (Web)
- `web/go.mod`
- `web/cmd/server/main.go`
- `web/internal/server/server.go`
- `web/internal/server/handlers.go`
- `web/internal/server/templates.go`
- `web/internal/server/api_client.go`
- `web/templates/base.html`
- `web/templates/index.html`
- `web/templates/contacts.html` (placeholder)
- `web/static/css/main.css` (placeholder)
- `web/static/js/main.js` (placeholder)

### Files to Modify
- `storage/elasticsearch.go` (add contacts index)

### Files to Leave Unchanged
- `onboard/` - Entire directory stays as-is

