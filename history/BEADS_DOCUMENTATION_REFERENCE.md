# Beads Issue Documentation References

## Standard Practice

**All beads issues should reference relevant documentation files** using comments. This creates a clear link between:
- Issue tracking (what needs to be done)
- Design documents (how it should be done)
- Planning discussions (why decisions were made)

## How to Add References

### Option 1: Comments (Recommended)
Add a comment with references to relevant markdown files:

```bash
bd comment <issue-id> "Documentation: history/FILE_NAME.md covers design, history/OTHER.md has decisions" --json
```

### Option 2: Description Updates
Update the issue description to include references at the end:

```bash
bd update <issue-id> --description "Original description... See history/FILE_NAME.md for details." --json
```

## Current Issue → Documentation Mapping

| Issue ID | Title | Documentation Files |
|----------|-------|---------------------|
| `hai-tw5` | Design and implement normalized speaker database schema | `history/SPEAKER_DATABASE_SCHEMA.md`, `history/REMAINING_CONSIDERATIONS.md`, `history/IMPLEMENTATION_READY_SUMMARY.md` |
| `hai-645` | Build OGG byte offset indexer | `history/BYTE_OFFSET_EFFICIENCY_CHECK.md` |
| `hai-d8z` | Migrate existing diarization results to database | `history/SPEAKER_DATABASE_SCHEMA.md`, `history/REMAINING_CONSIDERATIONS.md` |
| `hai-eog` | Integrate contacts from macOS and Google Contacts | `history/CONTACTS_INTEGRATION_NOTES.md` |
| `hai-a2z` | Plan for scaling speaker matching beyond 1k speakers | `history/REMAINING_CONSIDERATIONS.md` (section 6) |

## Benefits

1. **Quick Navigation**: From issue → design docs in one click/comment
2. **Context Preservation**: Design decisions linked to implementation work
3. **Onboarding**: New contributors can see all relevant docs immediately
4. **History**: Comments show when documentation was created/referenced

## Maintenance

- When creating new planning documents, add a comment to the relevant issue
- When updating design docs, add a comment with the change summary
- Keep this mapping document updated as new issues/docs are created

## Example

```bash
# After creating a new design doc
bd comment hai-xxx "New design document: history/NEW_DESIGN.md covers schema design for feature Y" --json
```

