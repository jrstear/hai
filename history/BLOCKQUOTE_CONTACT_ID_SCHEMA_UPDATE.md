# Blockquote Schema Update: Adding contact_id Field

**Date**: 2025-12-05
**Related Beads**: `hai-l1mm` (highlight matching speaker_name), future API endpoint for storing associations

## Summary

Added `contact_id` field to the `LifelogBlockquote` schema to allow direct association of blockquotes with contacts from the contacts index. This is a short-term solution while building the app primarily on Limitless data, with plans to connect to diarization/speaker system in the future.

## Context

The app is currently built primarily on Limitless API data (blockquotes with `speaker_name` from Limitless), augmented by contacts. The long-term goal is to connect with our custom diarization system (which uses `speaker_id`), but that work has been deferred.

## Changes Made

### Schema Updates

1. **storage/SCHEMA.md**:
   - Added `contact_id` field to `LifelogBlockquote` documentation
   - Added redundancy note explaining that `speaker_name`, `contact_id`, and `speaker_id` all identify "who is speaking" but from different sources:
     - `speaker_name`: From Limitless API (external, cannot be modified)
     - `contact_id`: User-assigned from contacts index (short-term solution)
     - `speaker_id`: From custom diarization system (long-term goal)

2. **storage/schema.go**:
   - Added `ContactID *string` field to `LifelogBlockquote` struct

3. **storage/elasticsearch.go**:
   - Added `contact_id` keyword field to Elasticsearch mapping for `lifelog_blockquotes` index
   - Updated `lifelogBlockquoteToDoc()` to include `contact_id` when present
   - Updated `docToLifelogBlockquote()` to parse `contact_id` from Elasticsearch documents
   - Updated `UpdateLifelogBlockquote()` to handle `contact_id` updates

4. **api/internal/server/lifelogs_handlers.go**:
   - Added `ContactID *string` field to `BlockquoteResponse` struct
   - Updated handler to populate `contact_id` from blockquote when building response

5. **pida/lib/models/lifelog.dart**:
   - Added `contactId` field to `Blockquote` model with `@JsonKey(name: 'contact_id')`
   - Updated constructor to include `contactId` parameter

## Data Flow

**Current (Temporary)**:
- User associates blockquote with contact → stored only in local memory (`blockquoteContactAssociationProvider`)
- Lost on app restart

**Next Step** (after API endpoint is implemented):
- User associates blockquote with contact → API call to update blockquote's `contact_id` in Elasticsearch
- Persists across app restarts
- Blockquote now has direct link to contact

## Relationship Paths

**Current Short-Term Path**:
```
Blockquote → Contact (via contact_id)
```

**Long-Term Goal**:
```
Blockquote → Speaker (via speaker_id) → Contact (via speaker.contact_id)
```

**Current Temporary Local State**:
```
Blockquote → Local State (blockquoteContactAssociationProvider) → Contact
```

## Next Steps

1. ✅ Schema updated
2. ⏭️ **Create API endpoint** to update blockquote's `contact_id`:
   - `PUT /api/blockquotes/{blockquoteId}/contact`
   - Body: `{"contact_id": "contact_xxx"}` or `{"contact_id": null}` to clear
3. ⏭️ Update Flutter app to call this API when user associates/disassociates
4. ⏭️ Load stored `contact_id` from API response (already in model)
5. ⏭️ Remove local-only `blockquoteContactAssociationProvider` (or keep as cache)

## Notes

- `contact_id` is what's behind the people page, people filter, and people selector - everything uses contact IDs from the contacts index
- The redundancy between `speaker_name`, `contact_id`, and `speaker_id` is intentional - they serve different purposes and will coexist
- Future enhancement (`hai-l1mm`): Highlight blockquote person name when `contact_id` matches Limitless `speaker_name` for comparison/migration purposes

