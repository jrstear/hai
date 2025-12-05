package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// HandleLimitlessAudioProxy proxies audio requests to the Limitless API.
// This is necessary for web clients which cannot set custom headers (like X-API-Key)
// on <audio> or AudioPlayer requests directly due to browser security models.
// The API key is read from the server's environment variables.
//
// GET /api/limitless/audio?startMs=X&endMs=Y
func (s *APIServer) HandleLimitlessAudioProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed: %s", r.Method))
		return
	}

	limitlessAPIKey := os.Getenv("LIMITLESS_API_KEY")
	if limitlessAPIKey == "" {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("LIMITLESS_API_KEY not configured on server"))
		return
	}

	// Extract query parameters for startMs and endMs
	startMsStr := r.URL.Query().Get("startMs")
	endMsStr := r.URL.Query().Get("endMs")

	if startMsStr == "" || endMsStr == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("startMs and endMs parameters are required"))
		return
	}

	startMs, err := strconv.ParseInt(startMsStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid startMs parameter: %w", err))
		return
	}
	endMs, err := strconv.ParseInt(endMsStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid endMs parameter: %w", err))
		return
	}

	// Construct the Limitless API URL
	limitlessURL := fmt.Sprintf("https://api.limitless.ai/v1/download-audio?startMs=%d&endMs=%d", startMs, endMs)

	// Create a new HTTP request to the Limitless API
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, limitlessURL, nil)
	if err != nil {
		log.Printf("[ERROR] Failed to create Limitless API request: %v", err)
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to create upstream request: %w", err))
		return
	}

	// Add the X-API-Key header
	req.Header.Set("X-API-Key", limitlessAPIKey)
	req.Header.Set("Accept", "audio/ogg") // Request OGG format

	// Forward Range header if present (for seeking)
	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}

	// Execute the request
	client := &http.Client{
		Timeout: 60 * time.Second, // Set a reasonable timeout for audio streaming
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[ERROR] Failed to proxy request to Limitless API: %v", err)
		writeError(w, http.StatusBadGateway, fmt.Errorf("failed to connect to upstream audio service: %w", err))
		return
	}
	defer resp.Body.Close()

	// Copy Limitless API response headers to our response
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set Content-Type if not already set by Limitless API
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "audio/ogg")
	}

	// Set our own CORS headers for the proxy endpoint
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, Range")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, Content-Type")
	w.Header().Set("Access-Control-Max-Age", "300")

	// Write the status code from the upstream response
	w.WriteHeader(resp.StatusCode)

	// Stream the response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("[ERROR] Failed to stream audio response: %v", err)
		// Note: Cannot write error header after writing body, so just log
	}
}
