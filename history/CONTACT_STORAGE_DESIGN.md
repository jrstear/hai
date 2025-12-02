# Contact Storage Design

## Decision: Use Elasticsearch

**Rationale**:
- ✅ **Already a dependency**: Elasticsearch is already set up and in use (`ELASTICSEARCH_URL` environment variable)
- ✅ **No new dependencies**: Don't need to add SQLite or file parsing overhead
- ✅ **Simple integration**: Follows existing pattern (speakers, recordings, segments already in ES)
- ✅ **Future-proof**: Easy to add full-text search, aggregations, and sync with Google/Apple later
- ✅ **Consistent architecture**: All data in one place (ES)

**Alternatives considered**:
- ❌ **Option A (vCard file)**: Parsing overhead on every request, no indexing/search
- ❌ **Option B (Split files)**: No clear advantage, adds complexity
- ❌ **Option C (SQLite)**: Would add new dependency, inconsistent with current architecture

## Schema Design

### Minimal Schema (Start Simple)

**Index**: `contacts`

**Fields**:
```json
{
  "id": "contact_abc123",                    // Internal ID (UUID-based)
  "external_id": "google:xyz789",            // External ID (google:xxx, apple:yyy, vcf:zzz)
  "name": "John Doe",                        // Full name
  "given_name": "John",                      // First name (for sorting/searching)
  "family_name": "Doe",                      // Last name (for sorting/searching)
  "email": "john@example.com",               // Primary email (optional)
  "phone": "+1-555-123-4567",                // Primary phone (optional)
  "picture_url": "/contacts/pictures/john.jpg", // Path to picture (optional)
  "favorite_color": "#4CAF50",               // Hex color code (optional, for color coding)
  "known": true,                             // Whether speaker voice is known (computed)
  "created_at": "2024-12-02T10:00:00Z",     // When contact was created
  "updated_at": "2024-12-02T10:00:00Z",     // Last update time
  "source": "vcf"                            // Source: "vcf", "google", "apple", "manual"
}
```

### Elasticsearch Mapping

```json
{
  "mappings": {
    "properties": {
      "id": {
        "type": "keyword"
      },
      "external_id": {
        "type": "keyword"
      },
      "name": {
        "type": "text",
        "fields": {
          "keyword": {
            "type": "keyword"
          }
        }
      },
      "given_name": {
        "type": "text",
        "fields": {
          "keyword": {
            "type": "keyword"
          }
        }
      },
      "family_name": {
        "type": "text",
        "fields": {
          "keyword": {
            "type": "keyword"
          }
        }
      },
      "email": {
        "type": "keyword"
      },
      "phone": {
        "type": "keyword"
      },
      "picture_url": {
        "type": "keyword"
      },
      "favorite_color": {
        "type": "keyword"
      },
      "known": {
        "type": "boolean"
      },
      "created_at": {
        "type": "date"
      },
      "updated_at": {
        "type": "date"
      },
      "source": {
        "type": "keyword"
      }
    }
  },
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  }
}
```

## Implementation Approach

### Phase 1: Direct ES Access (Simplest)

**Start with direct Elasticsearch access** (don't add to storage interface yet):

1. Create `onboard/internal/contacts/` package
2. Simple CRUD functions:
   - `ListContacts(ctx, filters)` - List all contacts, optionally filtered
   - `GetContact(ctx, id)` - Get contact by ID
   - `CreateContact(ctx, contact)` - Create new contact
   - `UpdateContact(ctx, id, updates)` - Update contact
   - `AssociateSpeaker(ctx, speakerID, contactID)` - Link speaker to contact

3. Use existing ES client pattern (similar to how `storage/elasticsearch.go` works)

**Benefits**:
- ✅ Fastest to implement
- ✅ No interface changes needed
- ✅ Can migrate to storage interface later if needed

### Phase 2: Add to Storage Interface (Optional, Later)

If we want consistency with other entities, add to `storage.Storage` interface:
- `CreateContact(ctx, contact)`
- `GetContact(ctx, id)`
- `ListContacts(ctx, filters)`
- `UpdateContact(ctx, id, updates)`

**But**: Not necessary for MVP. Can add later if we want to support SQLite backend for contacts.

## vCard Import Flow

1. **Parse vCard file** (`data/contacts/contacts.vcf`)
   - Use library: `github.com/emersion/go-vcard`
   - Extract: name, email, phone, photo

2. **Create contact records**:
   - Generate internal ID: `contact_<uuid>`
   - Set external_id: `vcf:<name>_<email>` (or hash)
   - Store in ES `contacts` index

3. **Handle duplicates**:
   - Check by email or phone
   - If exists, update instead of create
   - Track source in `source` field

## Integration with Speakers

**Link via `speakers.contact_id`**:
- When user associates speaker with contact, update `speakers.contact_id = contact.id`
- Query speakers by contact: `GET /api/speakers?contact_id=contact_abc123`
- Query contacts by known status: Check if any speaker has `contact_id` set

**Computed field `known`**:
- Query speakers index for `contact_id = contact.id`
- If any found, `known = true`, else `known = false`
- Can be computed on-the-fly or cached (update on association)

## Future: Google/Apple Integration

**When ready to integrate**:
1. **Google Contacts API**:
   - OAuth 2.0 authentication
   - Fetch contacts via People API
   - Store with `external_id = "google:<resource_name>"`
   - Sync periodically (update existing, add new)

2. **Apple Contacts** (macOS/iOS):
   - Use Contacts framework (Swift/Objective-C)
   - Or export to vCard and import (simpler)
   - Store with `external_id = "apple:<uuid>"`

3. **Sync strategy**:
   - Keep `external_id` to track source
   - Merge duplicates by email/phone
   - Update local ES index from external sources
   - Handle conflicts (last-write-wins or manual resolution)

## API Endpoints (for contacts page)

```
GET    /api/contacts                    # List contacts (with filters: known, search)
GET    /api/contacts/:id                # Get contact details
POST   /api/contacts                    # Create new contact
PUT    /api/contacts/:id                # Update contact (including favorite_color)
DELETE /api/contacts/:id                # Delete contact (optional, for now)

GET    /api/speakers                    # List speakers (with filters)
GET    /api/speakers/:id/recordings     # Get recordings for speaker
POST   /api/speakers/:id/associate      # Associate speaker with contact
```

## File Structure

```
onboard/
├── internal/
│   ├── contacts/
│   │   ├── contacts.go          # ES client and CRUD operations
│   │   ├── vcard.go             # vCard parsing
│   │   └── types.go             # Contact struct
│   └── server/
│       └── handlers.go          # API endpoints (add contact handlers)
```

## Migration Path

1. **Now**: Direct ES access, simple schema
2. **Later**: Add to storage interface if we want SQLite support
3. **Future**: Add Google/Apple sync, enrich with external data

## Benefits of This Approach

- ✅ **Simple**: Uses existing ES infrastructure
- ✅ **Fast**: No parsing overhead, indexed for search
- ✅ **Flexible**: Easy to add fields later (favorite_color, notes, etc.)
- ✅ **Searchable**: Full-text search on names, email search
- ✅ **Scalable**: ES handles large contact lists efficiently
- ✅ **Future-proof**: Easy to add sync with Google/Apple

## Next Steps

1. Create `onboard/internal/contacts/` package
2. Define Contact struct
3. Implement ES CRUD operations
4. Add vCard parsing
5. Create API endpoints
6. Test with sample vCard file

