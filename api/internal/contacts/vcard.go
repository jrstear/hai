package contacts

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-vcard"
)

const (
	VCardFilePath = "data/contacts/contacts.vcf"
)

// ImportStats tracks statistics about vCard import
type ImportStats struct {
	TotalParsed    int
	InvalidCards   int
	DuplicateInFile int
	AlreadyExists  int
	Created        int
	Updated        int
	Failed         int
	Errors         []string
}

// ImportVCards imports contacts from a vCard file
func (c *ElasticsearchContacts) ImportVCards(ctx context.Context, filePath string) ([]Contact, error) {
	return c.ImportVCardsWithStats(ctx, filePath, nil)
}

// ImportVCardsWithStats imports contacts from a vCard file and returns statistics
func (c *ElasticsearchContacts) ImportVCardsWithStats(ctx context.Context, filePath string, stats *ImportStats) ([]Contact, error) {
	// Ensure index exists
	if err := c.ensureIndex(ctx); err != nil {
		return nil, err
	}

	// Initialize stats if provided
	if stats == nil {
		stats = &ImportStats{}
	}

	// Open vCard file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open vCard file: %w", err)
	}
	defer file.Close()

	// Parse vCards
	decoder := vcard.NewDecoder(bufio.NewReader(file))
	imported := make([]Contact, 0)
	seen := make(map[string]bool) // Track by email/phone to avoid duplicates

	for {
		card, err := decoder.Decode()
		if err == io.EOF {
			break
		}
		if err != nil {
			stats.TotalParsed++
			stats.InvalidCards++
			stats.Errors = append(stats.Errors, fmt.Sprintf("parse error: %v", err))
			continue
		}

		stats.TotalParsed++

		contact := c.vcardToContact(card)
		if contact == nil {
			stats.InvalidCards++
			continue // Skip invalid cards
		}

		// Check for duplicates by email or phone within the file
		key := c.contactKey(contact)
		if seen[key] {
			stats.DuplicateInFile++
			continue // Skip duplicate
		}
		seen[key] = true

		// Check if contact already exists in ES
		existing, err := c.findContactByEmailOrPhone(ctx, contact.Email, contact.Phone)
		if err == nil && existing != nil {
			// Update existing contact
			stats.AlreadyExists++
			contact.ID = existing.ID
			contact.CreatedAt = existing.CreatedAt
			if err := c.UpdateContact(ctx, existing.ID, contact); err != nil {
				stats.Failed++
				stats.Errors = append(stats.Errors, fmt.Sprintf("failed to update contact %s: %v", contact.ID, err))
				continue
			}
			stats.Updated++
			imported = append(imported, *contact)
			continue
		}

		// Create new contact
		if err := c.CreateContact(ctx, contact); err != nil {
			stats.Failed++
			stats.Errors = append(stats.Errors, fmt.Sprintf("failed to create contact %s: %v", contact.Name, err))
			continue
		}

		stats.Created++
		imported = append(imported, *contact)
	}

	return imported, nil
}

// ImportVCardsFromDefault imports contacts from the default vCard file location
func (c *ElasticsearchContacts) ImportVCardsFromDefault(ctx context.Context) ([]Contact, error) {
	// Ensure directory exists
	dir := filepath.Dir(VCardFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create contacts directory: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(VCardFilePath); os.IsNotExist(err) {
		// File doesn't exist, return empty list
		return []Contact{}, nil
	}

	return c.ImportVCards(ctx, VCardFilePath)
}

// ImportVCardsFromDefaultWithStats imports contacts from the default vCard file location with statistics
func (c *ElasticsearchContacts) ImportVCardsFromDefaultWithStats(ctx context.Context, stats *ImportStats) ([]Contact, error) {
	// Ensure directory exists
	dir := filepath.Dir(VCardFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create contacts directory: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(VCardFilePath); os.IsNotExist(err) {
		// File doesn't exist, return empty list
		return []Contact{}, nil
	}

	return c.ImportVCardsWithStats(ctx, VCardFilePath, stats)
}

// AppendVCards appends new vCards to the existing file
func AppendVCards(filePath string, vcardData []byte) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create contacts directory: %w", err)
	}

	// Open file in append mode
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open vCard file: %w", err)
	}
	defer file.Close()

	// Write vCard data
	if _, err := file.Write(vcardData); err != nil {
		return fmt.Errorf("failed to write vCard data: %w", err)
	}

	// Add newline if not present
	if len(vcardData) > 0 && vcardData[len(vcardData)-1] != '\n' {
		if _, err := file.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to append newline: %w", err)
		}
	}

	return nil
}

// vcardToContact converts a vCard to a Contact
func (c *ElasticsearchContacts) vcardToContact(card vcard.Card) *Contact {
	contact := &Contact{
		Source:    "vcf",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Extract name
	if fn := card.Value(vcard.FieldFormattedName); fn != "" {
		contact.Name = fn
	}

	// Extract name components from N field
	if name := card.Name(); name != nil {
		if name.GivenName != "" {
			contact.GivenName = name.GivenName
		}
		if name.FamilyName != "" {
			contact.FamilyName = name.FamilyName
		}
	}

	// If name is empty, try to construct from given/family
	if contact.Name == "" {
		parts := []string{}
		if contact.GivenName != "" {
			parts = append(parts, contact.GivenName)
		}
		if contact.FamilyName != "" {
			parts = append(parts, contact.FamilyName)
		}
		if len(parts) > 0 {
			contact.Name = strings.Join(parts, " ")
		}
	}

	// Extract email (prefer pref, then first)
	emails := card.Values(vcard.FieldEmail)
	if len(emails) > 0 {
		// Find pref email
		prefEmail := ""
		for _, email := range emails {
			if card.PreferredValue(vcard.FieldEmail) == email {
				prefEmail = email
				break
			}
		}
		if prefEmail != "" {
			contact.Email = prefEmail
		} else {
			contact.Email = emails[0]
		}
	}

	// Extract phone (prefer pref, then first)
	phones := card.Values(vcard.FieldTelephone)
	if len(phones) > 0 {
		// Find pref phone
		prefPhone := ""
		for _, phone := range phones {
			if card.PreferredValue(vcard.FieldTelephone) == phone {
				prefPhone = phone
				break
			}
		}
		if prefPhone != "" {
			contact.Phone = prefPhone
		} else {
			contact.Phone = phones[0]
		}
	}

	// Extract photo (if present)
	if photo := card.Value(vcard.FieldPhoto); photo != "" {
		// Store photo URL (could be data URI or file path)
		contact.PictureURL = photo
	}

	// Generate external_id from name and email
	if contact.Name != "" || contact.Email != "" {
		key := fmt.Sprintf("%s_%s", contact.Name, contact.Email)
		hash := sha256.Sum256([]byte(key))
		contact.ExternalID = fmt.Sprintf("vcf:%s", hex.EncodeToString(hash[:])[:16])
	} else {
		// Fallback: use phone if available
		if contact.Phone != "" {
			hash := sha256.Sum256([]byte(contact.Phone))
			contact.ExternalID = fmt.Sprintf("vcf:%s", hex.EncodeToString(hash[:])[:16])
		} else {
			// Skip contacts without name, email, or phone
			return nil
		}
	}

	return contact
}

// contactKey generates a unique key for duplicate detection
func (c *ElasticsearchContacts) contactKey(contact *Contact) string {
	if contact.Email != "" {
		return "email:" + strings.ToLower(contact.Email)
	}
	if contact.Phone != "" {
		return "phone:" + strings.ToLower(contact.Phone)
	}
	return "name:" + strings.ToLower(contact.Name)
}

// findContactByEmailOrPhone finds an existing contact by email or phone
func (c *ElasticsearchContacts) findContactByEmailOrPhone(ctx context.Context, email, phone string) (*Contact, error) {
	// Search by email
	if email != "" {
		query := map[string]interface{}{
			"query": map[string]interface{}{
				"term": map[string]interface{}{
					"email": strings.ToLower(email),
				},
			},
		}
		contacts, _, err := c.searchContacts(ctx, query)
		if err == nil && len(contacts) > 0 {
			return &contacts[0], nil
		}
	}

	// Search by phone
	if phone != "" {
		query := map[string]interface{}{
			"query": map[string]interface{}{
				"term": map[string]interface{}{
					"phone": strings.ToLower(phone),
				},
			},
		}
		contacts, _, err := c.searchContacts(ctx, query)
		if err == nil && len(contacts) > 0 {
			return &contacts[0], nil
		}
	}

	return nil, fmt.Errorf("contact not found")
}

// searchContacts is a helper to search contacts with a custom query
func (c *ElasticsearchContacts) searchContacts(ctx context.Context, query map[string]interface{}) ([]Contact, int, error) {
	searchBody := map[string]interface{}{
		"query": query,
		"size":  10,
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
			continue
		}
		contacts = append(contacts, *contact)
	}

	return contacts, result.Hits.Total.Value, nil
}

