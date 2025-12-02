package server

import (
	"log"
	"os"

	"hai/api/internal/contacts"
	"hai/storage"

	"github.com/elastic/go-elasticsearch/v8"
)

// APIServer handles HTTP requests for the API server
type APIServer struct {
	storage  storage.Storage
	esClient *elasticsearch.Client
	contacts *contacts.ElasticsearchContacts
}

// NewAPIServer creates a new API server instance
// Initializes Elasticsearch storage from ELASTICSEARCH_URL environment variable
func NewAPIServer() (*APIServer, error) {
	esURL := os.Getenv("ELASTICSEARCH_URL")
	if esURL == "" {
		return nil, &ErrElasticsearchNotConfigured{}
	}

	log.Printf("Initializing Elasticsearch storage at: %s", esURL)
	esStorage, err := storage.NewElasticsearchStorage(esURL)
	if err != nil {
		return nil, err
	}

	// Create ES client for contacts (shares connection pool with storage)
	cfg := elasticsearch.Config{
		Addresses: []string{esURL},
	}
	esClient, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	// Initialize contacts handler
	contactsHandler := contacts.NewElasticsearchContacts(esClient)

	log.Printf("Elasticsearch storage initialized successfully")

	return &APIServer{
		storage:  esStorage,
		esClient: esClient,
		contacts: contactsHandler,
	}, nil
}

// Close closes the storage connection
func (s *APIServer) Close() error {
	if s.storage != nil {
		return s.storage.Close()
	}
	return nil
}

