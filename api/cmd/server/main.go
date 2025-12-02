package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"hai/api/internal/server"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

var (
	port = flag.String("port", "8080", "API server port")
)

func main() {
	flag.Parse()

	// Create API server instance
	srv, err := server.NewAPIServer()
	if err != nil {
		log.Fatalf("Failed to initialize API server: %v", err)
	}
	defer srv.Close()

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // Configure properly for production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", srv.HandleHealth)

		// Contacts endpoints
		r.Route("/contacts", func(r chi.Router) {
			r.Get("/", srv.HandleListContacts)
			r.Post("/", srv.HandleCreateContact)
			r.Post("/upload", srv.HandleUploadVCards)
			r.Get("/{id}", srv.HandleGetContact)
			r.Put("/{id}", srv.HandleUpdateContact)
			r.Post("/{contactId}/associate-speaker", srv.HandleAssociateSpeaker)
			r.Get("/{contactId}/recordings", srv.HandleGetContactRecordings)
		})

		// Speakers endpoints
		r.Route("/speakers", func(r chi.Router) {
			r.Get("/unassociated", srv.HandleListUnassociatedSpeakers)
			r.Get("/{speakerId}/recordings", srv.HandleGetSpeakerRecordings)
		})

		// Recordings endpoints
		r.Route("/recordings", func(r chi.Router) {
			r.Get("/{recordingId}/audio", srv.HandleGetRecordingAudio)
		})
	})

	// Start server
	addr := fmt.Sprintf(":%s", *port)
	log.Printf("Starting API server on http://localhost%s", addr)
	log.Printf("API endpoints available at http://localhost%s/api/*", addr)

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

