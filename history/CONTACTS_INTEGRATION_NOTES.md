# Contacts Integration Notes

## Current Situation
- **macOS**: Local contacts in Contacts app
- **Android**: Google Contacts (syncs with Google account)

## Initial Approach: Manual Export

### Export Format: vCard (.vcf)

**Standard format** supported by both macOS and Google Contacts. Single file can contain multiple contacts.

### macOS Export Steps
1. Open **Contacts** app
2. Select contacts (or All Contacts: Cmd+A)
3. **File** > **Export** > **Export vCard...**
4. Save as `contacts.vcf`

**Command line alternative** (for automation later):
```bash
# Export all contacts to vCard
osascript -e 'tell application "Contacts" to export vCard of every person'
```

### Google Contacts Export Steps
1. Go to [contacts.google.com](https://contacts.google.com)
2. Select contacts (or "Select all" checkbox)
3. Click **Export** button (or three-dot menu > Export)
4. Choose **vCard format**
5. Download `contacts.vcf`

**API alternative** (for automation later):
- Use Google People API with OAuth 2.0
- Endpoint: `https://people.googleapis.com/v1/people:batchGet`
- Requires API credentials and user consent

## Storage Location

**Initial**: `data/contacts/contacts.vcf`

Structure:
```
data/
├── audio/
├── contacts/
│   └── contacts.vcf    # Initial export (merged from macOS + Google)
└── speakers.db
```

## Contact Schema

**For now**: External ID reference only
- `speakers.contact_id` stores external ID like:
  - `google:abc123` (Google Contacts ID)
  - `apple:xyz789` (macOS Contacts UUID)
  - `vcf:contact_name_email` (if using vCard as source)

**Future**: Full contacts table with sync
- Store name, picture, email, phone
- Sync with Google People API
- Sync with macOS Contacts framework
- Handle duplicates/merging

## vCard Format Example

```
BEGIN:VCARD
VERSION:3.0
FN:John Doe
N:Doe;John;;;
EMAIL;TYPE=INTERNET:john@example.com
TEL;TYPE=CELL:+1-555-123-4567
PHOTO;ENCODING=b;TYPE=JPEG:/9j/4AAQSkZJRgABAQAAAQABAAD...
END:VCARD

BEGIN:VCARD
VERSION:3.0
FN:Jane Smith
...
END:VCARD
```

## Parsing vCard

Python libraries:
- `vobject` - Parse/emit vCard
- `vcard` - Simple vCard parser

Example:
```python
import vobject

with open('contacts.vcf', 'r') as f:
    vcards = vobject.readComponents(f)
    for vcard in vcards:
        name = vcard.fn.value
        email = vcard.email.value if hasattr(vcard, 'email') else None
        # Store in contacts table or map to speakers
```

## Merging Strategy

**Manual merge for now**:
1. Export macOS contacts → `contacts_macos.vcf`
2. Export Google contacts → `contacts_google.vcf`
3. Manually merge duplicates (or use tool)
4. Save merged → `data/contacts/contacts.vcf`

**Future automation**:
- Detect duplicates by name/email/phone
- Merge conflicts resolution
- Keep both sources (track origin)

## Integration with Speakers

When user associates speaker with contact:
1. Search contacts by name (fuzzy match)
2. Show matches, user selects
3. Store `contact_id` in `speakers.contact_id`
4. Display contact picture/name in UI

## Beads Issue
- `hai-eog`: Integrate contacts from macOS and Google Contacts

## Next Steps
1. ✅ Export contacts to vCard (user action)
2. 🔲 Create `data/contacts/` directory
3. 🔲 Place `contacts.vcf` file
4. 🔲 Document vCard parsing in codebase
5. 🔲 Implement contact lookup for speaker association (later)

