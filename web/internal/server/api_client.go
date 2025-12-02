package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// fetchContacts fetches contacts from the API server
func (s *WebServer) fetchContacts() ([]map[string]interface{}, error) {
	start := time.Now()
	url := fmt.Sprintf("%s/api/contacts", s.apiURL)

	resp, err := s.apiClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contacts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Contacts []map[string]interface{} `json:"contacts"`
		Total    int                      `json:"total"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	elapsed := time.Since(start)
	log.Printf("[TIMING] fetchContacts: %v (%d contacts)", elapsed, result.Total)

	return result.Contacts, nil
}

// fetchSpeakers fetches unassociated speakers from the API server
func (s *WebServer) fetchSpeakers() ([]map[string]interface{}, error) {
	start := time.Now()
	url := fmt.Sprintf("%s/api/speakers/unassociated", s.apiURL)

	resp, err := s.apiClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch speakers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Speakers []map[string]interface{} `json:"speakers"`
		Total    int                      `json:"total"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	elapsed := time.Since(start)
	log.Printf("[TIMING] fetchSpeakers: %v (%d speakers)", elapsed, result.Total)

	return result.Speakers, nil
}

