package main

import (
	"flag"
	"log"
	"net/http"

	"hai/web/internal/server"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var (
	port   = flag.String("port", "3000", "Web server port")
	apiURL = flag.String("api-url", "http://localhost:8080", "API server URL")
)

func main() {
	flag.Parse()

	// Create server instance
	srv, err := server.NewWebServer(*apiURL)
	if err != nil {
		log.Fatalf("Failed to initialize web server: %v", err)
	}

	// Setup router
	r := chi.NewRouter()
	
	// Logger that skips very long paths (likely base64 image URLs in 404s)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip logging for very long paths (likely base64 image data)
			if len(r.URL.Path) > 200 {
				next.ServeHTTP(w, r)
				return
			}
			middleware.Logger(next).ServeHTTP(w, r)
		})
	})
	r.Use(middleware.Recoverer)

	// Static assets
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Page routes (HTML)
	r.Get("/", srv.HandleIndex)
	r.Get("/contacts", srv.HandleContactsPage)
	r.Get("/lifelog", srv.HandleLifelogPage)
	// Future: r.Get("/calendar", srv.HandleCalendarPage)

	// Proxy endpoint for Limitless API (uses LIMITLESS_API_KEY env var)
	r.Get("/api/limitless/audio", srv.HandleLimitlessAudio)

	// Start server
	addr := ":" + *port
	log.Printf("Starting web server on http://localhost%s", addr)
	log.Printf("API server: %s", *apiURL)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

