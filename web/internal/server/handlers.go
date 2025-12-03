package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// HandleIndex handles GET /
func (s *WebServer) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := map[string]interface{}{
		"Title": "Hai - Audio Lifelog",
		"APIURL": s.apiURL,
	}

	// Try to render index.html, fallback to base if not found
	if err := s.renderTemplate(w, "index.html", data); err != nil {
		// Fallback: render base template or simple HTML
		s.renderSimplePage(w, "Hai - Audio Lifelog", `
			<h1>Welcome to Hai</h1>
			<p>Audio Lifelog Processing System</p>
			<nav>
				<a href="/contacts">Contacts</a>
			</nav>
		`)
	}
}

// HandleContactsPage handles GET /contacts
func (s *WebServer) HandleContactsPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pageStart := time.Now()

	// Fetch contacts and speakers from API (timing is logged in fetch functions)
	contacts, err := s.fetchContacts()
	if err != nil {
		log.Printf("[ERROR] Failed to fetch contacts: %v", err)
		contacts = []map[string]interface{}{}
	}

	speakers, err := s.fetchSpeakers()
	if err != nil {
		log.Printf("[ERROR] Failed to fetch speakers: %v", err)
		speakers = []map[string]interface{}{}
	}

	templateStart := time.Now()
	data := map[string]interface{}{
		"Title":          "Contacts - Hai",
		"APIURL":         s.apiURL,
		"LimitlessAPIURL": "/api/limitless/audio", // Web server proxy endpoint
		"Contacts":       contacts,
		"Speakers":       speakers,
	}

	// Try to render contacts.html, fallback to base if not found
	if err := s.renderTemplate(w, "contacts.html", data); err != nil {
		log.Printf("[ERROR] Failed to render template: %v", err)
		// Fallback: render simple page
		s.renderSimplePage(w, "Contacts - Hai", `
			<h1>Contacts</h1>
			<p>Contacts page coming soon...</p>
			<nav>
				<a href="/">Home</a>
			</nav>
		`)
	}

	templateElapsed := time.Since(templateStart)
	totalElapsed := time.Since(pageStart)
	log.Printf("[TIMING] HandleContactsPage - Template render: %v, Total: %v", templateElapsed, totalElapsed)
}

// HandleLimitlessAudio proxies audio requests to Limitless API using LIMITLESS_API_KEY env var
// This makes the server single-user only (one API key for all requests)
func (s *WebServer) HandleLimitlessAudio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.limitlessAPIKey == "" {
		http.Error(w, "LIMITLESS_API_KEY not configured", http.StatusInternalServerError)
		return
	}

	// Get query parameters from request
	queryParams := r.URL.Query()
	startMs := queryParams.Get("startMs")
	endMs := queryParams.Get("endMs")

	if startMs == "" || endMs == "" {
		http.Error(w, "startMs and endMs query parameters are required", http.StatusBadRequest)
		return
	}

	// Build Limitless API URL
	limitlessURL := "https://api.limitless.ai/v1/download-audio"
	reqURL, err := url.Parse(limitlessURL)
	if err != nil {
		http.Error(w, "Invalid API URL", http.StatusInternalServerError)
		return
	}

	// Add query parameters
	reqURL.RawQuery = url.Values{
		"startMs": []string{startMs},
		"endMs":   []string{endMs},
	}.Encode()

	// Create request to Limitless API
	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Add API key header
	req.Header.Set("X-API-Key", s.limitlessAPIKey)

	// Make request to Limitless API
	resp, err := s.apiClient.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch audio: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Copy status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("[ERROR] Failed to copy audio response: %v", err)
		return
	}
}

// HandleLifelogPage handles GET /lifelog
func (s *WebServer) HandleLifelogPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pageStart := time.Now()

	// Get date from query parameter or use default
	date := r.URL.Query().Get("date")
	if date == "" {
		date = "2025-11-22" // Default date
	}

	// Fetch blockquotes for the date
	lifelogData, err := s.fetchLifelogs(date)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch lifelogs: %v", err)
		// Continue with empty data rather than failing
		lifelogData = map[string]interface{}{
			"date":        date,
			"blockquotes": []interface{}{},
			"grouped":     map[string]interface{}{},
			"total":       0,
		}
	}

	data := map[string]interface{}{
		"Title":     "Lifelog - Hai",
		"APIURL":    s.apiURL,
		"Date":      date,
		"LifelogData": lifelogData,
	}

	// Try to render lifelog.html
	if err := s.renderTemplate(w, "lifelog.html", data); err != nil {
		log.Printf("[ERROR] Failed to render lifelog template: %v", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}

	elapsed := time.Since(pageStart)
	log.Printf("[TIMING] HandleLifelogPage: %v", elapsed)
}

// renderSimplePage renders a simple HTML page (fallback when templates not found)
func (s *WebServer) renderSimplePage(w http.ResponseWriter, title, content string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	html := `<!DOCTYPE html>
<html>
<head>
	<title>` + title + `</title>
	<style>
		body { font-family: system-ui, -apple-system, sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; }
		nav { margin: 20px 0; padding: 10px; background: #f0f0f0; }
		nav a { margin-right: 20px; text-decoration: none; color: #0066cc; }
		nav a:hover { text-decoration: underline; }
	</style>
</head>
<body>
	` + content + `
</body>
</html>`
	w.Write([]byte(html))
}

