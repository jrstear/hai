package export2elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
)

// createESClient creates an Elasticsearch client from URL
func (e *Exporter) createESClient(esURL string) (*elasticsearch.Client, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{esURL},
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}
	return client, nil
}

const (
	indexContacts = "contacts"
	indexSettings = "settings"
)

// Contact represents a contact for name matching
type Contact struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// matchContactByName matches a speaker name to a contact by name
// Also checks if speaker name matches the "You" name from settings
// Returns contact ID if single match found, nil if no match or multiple matches
func (e *Exporter) matchContactByName(ctx context.Context, speakerName string, contacts []Contact, esClient *elasticsearch.Client) *string {
	// Normalize speaker name
	normalizedSpeaker := normalizeName(speakerName)

	// Skip "Unknown" - it shouldn't auto-match
	if normalizedSpeaker == "unknown" {
		return nil
	}

	// Check if speaker name matches "You" name from settings
	if esClient != nil {
		userName, err := e.loadUserName(ctx, esClient)
		if err == nil && userName != "" {
			normalizedUserName := normalizeName(userName)
			// If speaker name matches "You" name, find the contact with that name
			if namesMatch(normalizedSpeaker, normalizedUserName) {
				// Find contact that matches the user name
				for _, contact := range contacts {
					normalizedContact := normalizeName(contact.Name)
					if namesMatch(normalizedContact, normalizedUserName) {
						// Found the contact that matches "You" name
						return &contact.ID
					}
				}
			}
		}
	}

	// Also handle "You" as a special case (even if no user_name in settings)
	if normalizedSpeaker == "you" {
		// Try to find a contact that might be the user
		// This is a fallback if user_name setting isn't set
		// We'll still try to match against contacts, but won't force it
	}

	// Find matching contacts
	var matches []Contact
	for _, contact := range contacts {
		normalizedContact := normalizeName(contact.Name)
		if namesMatch(normalizedSpeaker, normalizedContact) {
			matches = append(matches, contact)
		}
	}

	// Only return contact ID if exactly one match
	if len(matches) == 1 {
		return &matches[0].ID
	}

	// No match or multiple matches - return nil
	return nil
}

// loadContacts loads all contacts from Elasticsearch
// Returns empty slice and no error if contacts index doesn't exist or is empty
func (e *Exporter) loadContacts(ctx context.Context, esClient *elasticsearch.Client) ([]Contact, error) {
	if esClient == nil {
		// No ES client available - return empty (contacts matching will be skipped)
		return []Contact{}, nil
	}

	// Query all contacts from Elasticsearch
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
		"size": 1000, // TODO: Support pagination if needed
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	res, err := esClient.Search(
		esClient.Search.WithIndex(indexContacts),
		esClient.Search.WithBody(bytes.NewReader(queryJSON)),
		esClient.Search.WithContext(ctx),
	)
	if err != nil {
		// If index doesn't exist or query fails, return empty (not an error - contacts may not be loaded yet)
		return []Contact{}, nil
	}
	defer res.Body.Close()

	if res.IsError() {
		if res.StatusCode == 404 {
			// Contacts index doesn't exist yet - that's ok
			return []Contact{}, nil
		}
		// Error querying contacts - return empty (not an error - contacts matching is optional)
		_, _ = io.ReadAll(res.Body) // Drain response body
		return []Contact{}, nil
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode contacts response: %w", err)
	}

	contacts := make([]Contact, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		contactID, ok := hit.Source["id"].(string)
		if !ok {
			continue
		}
		contactName, ok := hit.Source["name"].(string)
		if !ok {
			continue
		}
		contacts = append(contacts, Contact{
			ID:   contactID,
			Name: contactName,
		})
	}

	return contacts, nil
}

// normalizeName normalizes a name for matching (lowercase, trim whitespace)
func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// namesMatch checks if two normalized names match
// Uses same logic as Flutter app: exact match or substring match (if shorter >= 3 chars)
func namesMatch(name1, name2 string) bool {
	// Exact match
	if name1 == name2 {
		return true
	}

	// Check if one name contains the other (for partial matches)
	if strings.Contains(name1, name2) || strings.Contains(name2, name1) {
		// Only allow if the shorter name is at least 3 characters
		// to avoid false matches like "a" matching "alice"
		shorter := name1
		if len(name2) < len(name1) {
			shorter = name2
		}
		if len(shorter) >= 3 {
			return true
		}
	}

	return false
}

// loadUserName loads the user_name setting from the settings index
// Returns empty string and no error if setting doesn't exist or index doesn't exist
func (e *Exporter) loadUserName(ctx context.Context, esClient *elasticsearch.Client) (string, error) {
	if esClient == nil {
		return "", nil
	}

	// Query for user_name setting
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"key": "user_name",
			},
		},
		"size": 1,
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return "", fmt.Errorf("failed to marshal query: %w", err)
	}

	res, err := esClient.Search(
		esClient.Search.WithIndex(indexSettings),
		esClient.Search.WithBody(bytes.NewReader(queryJSON)),
		esClient.Search.WithContext(ctx),
	)
	if err != nil {
		// If index doesn't exist or query fails, return empty (not an error - settings may not be set yet)
		return "", nil
	}
	defer res.Body.Close()

	if res.IsError() {
		if res.StatusCode == 404 {
			// Settings index doesn't exist yet - that's ok
			return "", nil
		}
		// Error querying settings - return empty (not an error - settings matching is optional)
		_, _ = io.ReadAll(res.Body) // Drain response body
		return "", nil
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode settings response: %w", err)
	}

	if len(result.Hits.Hits) == 0 {
		return "", nil
	}

	return result.Hits.Hits[0].Source.Value, nil
}
