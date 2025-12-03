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

// fetchLifelogs fetches blockquotes for a specific date from the API server
func (s *WebServer) fetchLifelogs(date string) (map[string]interface{}, error) {
	start := time.Now()
	url := fmt.Sprintf("%s/api/lifelogs?date=%s", s.apiURL, date)

	resp, err := s.apiClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch lifelogs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	elapsed := time.Since(start)
	total := 0
	if t, ok := result["total"].(float64); ok {
		total = int(t)
	}
	log.Printf("[TIMING] fetchLifelogs: %v (%d blockquotes)", elapsed, total)

	return result, nil
}

