package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"hai/api/internal/contacts"

	"github.com/elastic/go-elasticsearch/v8"
)

var (
	vcardFile = flag.String("vcard", "", "Path to vCard file (.vcf)")
	esURL     = flag.String("es", "", "Elasticsearch URL (default: ELASTICSEARCH_URL env var)")
)

func main() {
	flag.Parse()

	if *vcardFile == "" {
		log.Fatal("Error: -vcard flag is required")
	}

	// Get Elasticsearch URL
	esURLValue := *esURL
	if esURLValue == "" {
		esURLValue = os.Getenv("ELASTICSEARCH_URL")
		if esURLValue == "" {
			log.Fatal("Error: ELASTICSEARCH_URL environment variable or -es flag is required")
		}
	}

	// Check if vCard file exists
	if _, err := os.Stat(*vcardFile); os.IsNotExist(err) {
		log.Fatalf("Error: vCard file not found: %s", *vcardFile)
	}

	fmt.Printf("Testing vCard import:\n")
	fmt.Printf("  vCard file: %s\n", *vcardFile)
	fmt.Printf("  Elasticsearch: %s\n", esURLValue)
	fmt.Println()

	// Create Elasticsearch client
	cfg := elasticsearch.Config{
		Addresses: []string{esURLValue},
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error: Failed to create Elasticsearch client: %v", err)
	}

	// Create contacts handler
	contactsHandler := contacts.NewElasticsearchContacts(client)

	// Import vCard
	ctx := context.Background()
	fmt.Println("Step 1: Importing vCard file...")
	stats := &contacts.ImportStats{}
	imported, err := contactsHandler.ImportVCardsWithStats(ctx, *vcardFile, stats)
	if err != nil {
		log.Fatalf("Error: Failed to import vCard: %v", err)
	}

	// Display import statistics
	fmt.Printf("  Import Statistics:\n")
	fmt.Printf("    Total vCards parsed: %d\n", stats.TotalParsed)
	fmt.Printf("    Invalid cards: %d\n", stats.InvalidCards)
	fmt.Printf("    Duplicates in file: %d\n", stats.DuplicateInFile)
	fmt.Printf("    Already exists in ES: %d\n", stats.AlreadyExists)
	fmt.Printf("    Created: %d\n", stats.Created)
	fmt.Printf("    Updated: %d\n", stats.Updated)
	fmt.Printf("    Failed: %d\n", stats.Failed)
	fmt.Printf("    Total imported/updated: %d\n", len(imported))
	
	if stats.Failed > 0 && len(stats.Errors) > 0 {
		fmt.Printf("\n  Errors (showing first 10):\n")
		maxErrors := 10
		if len(stats.Errors) < maxErrors {
			maxErrors = len(stats.Errors)
		}
		for i := 0; i < maxErrors; i++ {
			fmt.Printf("    - %s\n", stats.Errors[i])
		}
		if len(stats.Errors) > maxErrors {
			fmt.Printf("    ... and %d more errors\n", len(stats.Errors)-maxErrors)
		}
	}

	// Refresh index to make documents immediately searchable
	fmt.Println("\n  Refreshing index for immediate searchability...")
	if err := contactsHandler.RefreshIndex(ctx); err != nil {
		log.Printf("Warning: Failed to refresh index: %v", err)
	} else {
		fmt.Println("  ✓ Index refreshed")
	}
	fmt.Println()

	// Display imported contacts (limit to first 10 if many)
	if len(imported) > 0 {
		fmt.Println("Imported contacts:")
		maxDisplay := 10
		if len(imported) > maxDisplay {
			fmt.Printf("  (Showing first %d of %d contacts)\n\n", maxDisplay, len(imported))
		}
		displayCount := len(imported)
		if displayCount > maxDisplay {
			displayCount = maxDisplay
		}
		for i := 0; i < displayCount; i++ {
			contact := imported[i]
			fmt.Printf("  [%d] %s (ID: %s)\n", i+1, contact.Name, contact.ID)
			if contact.Email != "" {
				fmt.Printf("      Email: %s\n", contact.Email)
			}
			if contact.Phone != "" {
				fmt.Printf("      Phone: %s\n", contact.Phone)
			}
			if contact.GivenName != "" || contact.FamilyName != "" {
				fmt.Printf("      Name: %s %s\n", contact.GivenName, contact.FamilyName)
			}
			fmt.Printf("      External ID: %s\n", contact.ExternalID)
			fmt.Printf("      Source: %s\n", contact.Source)
			fmt.Println()
		}
		if len(imported) > maxDisplay {
			fmt.Printf("  ... and %d more contacts\n\n", len(imported)-maxDisplay)
		}
	}

	// Read back from Elasticsearch
	fmt.Println("Step 2: Reading contacts back from Elasticsearch...")
	readBack, total, err := contactsHandler.ListContacts(ctx, nil)
	if err != nil {
		log.Fatalf("Error: Failed to list contacts: %v", err)
	}

	fmt.Printf("  ✓ Found %d total contact(s) in Elasticsearch\n", total)
	fmt.Println()

	// Compare results
	fmt.Println("Step 3: Comparing imported vs stored contacts...")
	if len(imported) == 0 {
		fmt.Println("  ⚠ No contacts were imported, skipping comparison")
		return
	}

	// Create a map of imported contact IDs
	importedMap := make(map[string]contacts.Contact)
	for _, contact := range imported {
		importedMap[contact.ID] = contact
	}

	// Check if all imported contacts are in the read-back list
	foundCount := 0
	for _, stored := range readBack {
		if imported, exists := importedMap[stored.ID]; exists {
			foundCount++
			fmt.Printf("  ✓ Found imported contact: %s (ID: %s)\n", stored.Name, stored.ID)

			// Compare fields
			matches := true
			if stored.Name != imported.Name {
				fmt.Printf("    ⚠ Name mismatch: stored=%q, imported=%q\n", stored.Name, imported.Name)
				matches = false
			}
			if stored.Email != imported.Email {
				fmt.Printf("    ⚠ Email mismatch: stored=%q, imported=%q\n", stored.Email, imported.Email)
				matches = false
			}
			if stored.Phone != imported.Phone {
				fmt.Printf("    ⚠ Phone mismatch: stored=%q, imported=%q\n", stored.Phone, imported.Phone)
				matches = false
			}
			if stored.ExternalID != imported.ExternalID {
				fmt.Printf("    ⚠ ExternalID mismatch: stored=%q, imported=%q\n", stored.ExternalID, imported.ExternalID)
				matches = false
			}
			if stored.Source != imported.Source {
				fmt.Printf("    ⚠ Source mismatch: stored=%q, imported=%q\n", stored.Source, imported.Source)
				matches = false
			}

			if matches {
				fmt.Printf("    ✓ All fields match\n")
			}
		}
	}

	fmt.Println()
	if foundCount == len(imported) {
		fmt.Printf("✓ SUCCESS: All %d imported contact(s) were found in Elasticsearch\n", foundCount)
	} else {
		fmt.Printf("⚠ WARNING: Only %d of %d imported contact(s) were found in Elasticsearch\n", foundCount, len(imported))
		fmt.Printf("  Missing: %d contact(s)\n", len(imported)-foundCount)
		fmt.Printf("\n  Breakdown:\n")
		fmt.Printf("    - Total vCards in file: %d\n", stats.TotalParsed)
		fmt.Printf("    - Invalid cards (skipped): %d\n", stats.InvalidCards)
		fmt.Printf("    - Duplicates in file (skipped): %d\n", stats.DuplicateInFile)
		fmt.Printf("    - Already existed in ES (updated): %d\n", stats.AlreadyExists)
		fmt.Printf("    - Successfully created: %d\n", stats.Created)
		fmt.Printf("    - Successfully updated: %d\n", stats.Updated)
		fmt.Printf("    - Failed to import: %d\n", stats.Failed)
		fmt.Printf("    - Total returned (created + updated): %d\n", len(imported))
		fmt.Printf("\n  Note: The comparison checks if imported contacts are in the ES list.\n")
		fmt.Printf("  If contacts were updated (not created), they should still be found.\n")
		if stats.Failed > 0 {
			fmt.Printf("  Check errors above for details on failed imports.\n")
		}
	}

	// Test individual retrieval
	if len(imported) > 0 {
		fmt.Println()
		fmt.Println("Step 4: Testing individual contact retrieval...")
		testContact := imported[0]
		retrieved, err := contactsHandler.GetContact(ctx, testContact.ID)
		if err != nil {
			fmt.Printf("  ✗ Failed to retrieve contact %s: %v\n", testContact.ID, err)
		} else {
			fmt.Printf("  ✓ Successfully retrieved contact: %s\n", retrieved.Name)
			if retrieved.ID == testContact.ID && retrieved.Name == testContact.Name {
				fmt.Printf("  ✓ Contact data matches\n")
			} else {
				fmt.Printf("  ⚠ Contact data mismatch\n")
			}
		}
	}

	fmt.Println()
	fmt.Println("Test complete!")
}

