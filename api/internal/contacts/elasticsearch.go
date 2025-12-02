package contacts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/google/uuid"
)

const (
	indexContacts = "contacts"
)

// ElasticsearchContacts handles contact operations in Elasticsearch
type ElasticsearchContacts struct {
	client *elasticsearch.Client
}

// NewElasticsearchContacts creates a new Elasticsearch contacts handler
func NewElasticsearchContacts(client *elasticsearch.Client) *ElasticsearchContacts {
	return &ElasticsearchContacts{
		client: client,
	}
}

// EnsureIndex creates the contacts index if it doesn't exist (public method)
func (c *ElasticsearchContacts) EnsureIndex(ctx context.Context) error {
	return c.ensureIndex(ctx)
}

// ensureIndex creates the contacts index if it doesn't exist
func (c *ElasticsearchContacts) ensureIndex(ctx context.Context) error {
	// Check if index exists
	res, err := c.client.Indices.Exists([]string{indexContacts})
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}
	res.Body.Close()

	if res.StatusCode == 200 {
		// Index exists, skip
		return nil
	}

	// Create index with mapping
	mapping := map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type": "keyword",
				},
				"external_id": map[string]interface{}{
					"type": "keyword",
				},
				"name": map[string]interface{}{
					"type": "text",
					"fields": map[string]interface{}{
						"keyword": map[string]interface{}{
							"type": "keyword",
						},
					},
				},
				"given_name": map[string]interface{}{
					"type": "text",
					"fields": map[string]interface{}{
						"keyword": map[string]interface{}{
							"type": "keyword",
						},
					},
				},
				"family_name": map[string]interface{}{
					"type": "text",
					"fields": map[string]interface{}{
						"keyword": map[string]interface{}{
							"type": "keyword",
						},
					},
				},
				"email": map[string]interface{}{
					"type": "keyword",
				},
				"phone": map[string]interface{}{
					"type": "keyword",
				},
				"picture_url": map[string]interface{}{
					"type": "keyword",
				},
				"favorite_color": map[string]interface{}{
					"type": "keyword",
				},
				"known": map[string]interface{}{
					"type": "boolean",
				},
				"created_at": map[string]interface{}{
					"type": "date",
				},
				"updated_at": map[string]interface{}{
					"type": "date",
				},
				"source": map[string]interface{}{
					"type": "keyword",
				},
			},
		},
		"settings": map[string]interface{}{
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
	}

	mappingJSON, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %w", err)
	}

	res, err = c.client.Indices.Create(
		indexContacts,
		c.client.Indices.Create.WithBody(bytes.NewReader(mappingJSON)),
		c.client.Indices.Create.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to create index: %s", string(body))
	}

	return nil
}

// CreateContact creates a new contact
func (c *ElasticsearchContacts) CreateContact(ctx context.Context, contact *Contact) error {
	if err := c.ensureIndex(ctx); err != nil {
		return err
	}

	// Generate ID if not set
	if contact.ID == "" {
		contact.ID = fmt.Sprintf("contact_%s", uuid.New().String())
	}

	// Set timestamps
	now := time.Now()
	if contact.CreatedAt.IsZero() {
		contact.CreatedAt = now
	}
	contact.UpdatedAt = now

	// Check if contact already exists
	_, err := c.GetContact(ctx, contact.ID)
	if err == nil {
		return fmt.Errorf("contact already exists: %s", contact.ID)
	}

	doc := c.contactToDoc(contact)
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal contact: %w", err)
	}

	res, err := c.client.Index(
		indexContacts,
		bytes.NewReader(docJSON),
		c.client.Index.WithDocumentID(contact.ID),
		c.client.Index.WithContext(ctx),
		c.client.Index.WithRefresh("true"), // Force refresh for immediate searchability
	)
	if err != nil {
		return fmt.Errorf("failed to index contact: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to index contact: %s", string(body))
	}

	return nil
}

// GetContact retrieves a contact by ID
func (c *ElasticsearchContacts) GetContact(ctx context.Context, id string) (*Contact, error) {
	res, err := c.client.Get(indexContacts, id, c.client.Get.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, fmt.Errorf("contact not found: %s", id)
	}

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("failed to get contact: %s", string(body))
	}

	var result struct {
		Source map[string]interface{} `json:"_source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode contact: %w", err)
	}

	return c.docToContact(result.Source)
}

// ListContacts lists all contacts, optionally filtered
func (c *ElasticsearchContacts) ListContacts(ctx context.Context, filters *ContactFilters) ([]Contact, int, error) {
	if err := c.ensureIndex(ctx); err != nil {
		return nil, 0, err
	}

	query := map[string]interface{}{
		"match_all": map[string]interface{}{},
	}

	if filters != nil {
		if filters.Known != nil {
			query = map[string]interface{}{
				"term": map[string]interface{}{
					"known": *filters.Known,
				},
			}
		}
		if filters.Search != "" {
			query = map[string]interface{}{
				"multi_match": map[string]interface{}{
					"query":  filters.Search,
					"fields": []string{"name", "given_name", "family_name", "email"},
				},
			}
		}
	}

	searchBody := map[string]interface{}{
		"query": query,
		"size":  1000, // TODO: Add pagination
	}

	searchJSON, err := json.Marshal(searchBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal search: %w", err)
	}

	res, err := c.client.Search(
		c.client.Search.WithIndex(indexContacts),
		c.client.Search.WithBody(bytes.NewReader(searchJSON)),
		c.client.Search.WithContext(ctx),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search contacts: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, 0, fmt.Errorf("failed to search contacts: %s", string(body))
	}

	var result struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("failed to decode search results: %w", err)
	}

	contacts := make([]Contact, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		contact, err := c.docToContact(hit.Source)
		if err != nil {
			continue // Skip invalid contacts
		}
		contacts = append(contacts, *contact)
	}

	return contacts, result.Hits.Total.Value, nil
}

// RefreshIndex refreshes the contacts index to make recently indexed documents searchable
func (c *ElasticsearchContacts) RefreshIndex(ctx context.Context) error {
	res, err := c.client.Indices.Refresh(
		c.client.Indices.Refresh.WithIndex(indexContacts),
		c.client.Indices.Refresh.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to refresh index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to refresh index: %s", string(body))
	}

	return nil
}

// UpdateContact updates an existing contact
func (c *ElasticsearchContacts) UpdateContact(ctx context.Context, id string, updates *Contact) error {
	// Get existing contact
	existing, err := c.GetContact(ctx, id)
	if err != nil {
		return err
	}

	// Apply updates
	if updates.Name != "" {
		existing.Name = updates.Name
	}
	if updates.GivenName != "" {
		existing.GivenName = updates.GivenName
	}
	if updates.FamilyName != "" {
		existing.FamilyName = updates.FamilyName
	}
	if updates.Email != "" {
		existing.Email = updates.Email
	}
	if updates.Phone != "" {
		existing.Phone = updates.Phone
	}
	if updates.PictureURL != "" {
		existing.PictureURL = updates.PictureURL
	}
	if updates.FavoriteColor != "" {
		existing.FavoriteColor = updates.FavoriteColor
	}
	if updates.Source != "" {
		existing.Source = updates.Source
	}

	existing.UpdatedAt = time.Now()

	// Update in ES
	doc := c.contactToDoc(existing)
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal contact: %w", err)
	}

	res, err := c.client.Index(
		indexContacts,
		bytes.NewReader(docJSON),
		c.client.Index.WithDocumentID(id),
		c.client.Index.WithContext(ctx),
		c.client.Index.WithRefresh("true"), // Force refresh for immediate searchability
	)
	if err != nil {
		return fmt.Errorf("failed to update contact: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to update contact: %s", string(body))
	}

	return nil
}

// ContactFilters represents filters for listing contacts
type ContactFilters struct {
	Known   *bool  // Filter by known status
	Search  string // Search by name/email
}

// Helper functions for document conversion

func (c *ElasticsearchContacts) contactToDoc(contact *Contact) map[string]interface{} {
	doc := map[string]interface{}{
		"id":         contact.ID,
		"external_id": contact.ExternalID,
		"name":       contact.Name,
		"given_name": contact.GivenName,
		"family_name": contact.FamilyName,
		"known":      contact.Known,
		"created_at": contact.CreatedAt.Format(time.RFC3339),
		"updated_at": contact.UpdatedAt.Format(time.RFC3339),
		"source":     contact.Source,
	}

	if contact.Email != "" {
		doc["email"] = contact.Email
	}
	if contact.Phone != "" {
		doc["phone"] = contact.Phone
	}
	if contact.PictureURL != "" {
		doc["picture_url"] = contact.PictureURL
	}
	if contact.FavoriteColor != "" {
		doc["favorite_color"] = contact.FavoriteColor
	}

	return doc
}

func (c *ElasticsearchContacts) docToContact(doc map[string]interface{}) (*Contact, error) {
	contact := &Contact{}

	if id, ok := doc["id"].(string); ok {
		contact.ID = id
	}
	if externalID, ok := doc["external_id"].(string); ok {
		contact.ExternalID = externalID
	}
	if name, ok := doc["name"].(string); ok {
		contact.Name = name
	}
	if givenName, ok := doc["given_name"].(string); ok {
		contact.GivenName = givenName
	}
	if familyName, ok := doc["family_name"].(string); ok {
		contact.FamilyName = familyName
	}
	if email, ok := doc["email"].(string); ok {
		contact.Email = email
	}
	if phone, ok := doc["phone"].(string); ok {
		contact.Phone = phone
	}
	if pictureURL, ok := doc["picture_url"].(string); ok {
		contact.PictureURL = pictureURL
	}
	if favoriteColor, ok := doc["favorite_color"].(string); ok {
		contact.FavoriteColor = favoriteColor
	}
	if known, ok := doc["known"].(bool); ok {
		contact.Known = known
	}
	if source, ok := doc["source"].(string); ok {
		contact.Source = source
	}

	// Parse timestamps
	if createdAtStr, ok := doc["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			contact.CreatedAt = t
		}
	}
	if updatedAtStr, ok := doc["updated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
			contact.UpdatedAt = t
		}
	}

	return contact, nil
}

