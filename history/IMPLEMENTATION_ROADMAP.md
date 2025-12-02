# Implementation Roadmap - Contacts Page & Related Features

## Overview

This document outlines the recommended implementation steps for building the contacts page and related features, based on the design work completed in `history/CONTACTS_PAGE_UI_DESIGN.md`.

## Current State

- ✅ Onboard server exists (`onboard/cmd/server`)
- ✅ Storage layer exists (SQLite + Elasticsearch via `storage` package)
- ✅ Speaker/recording/segment schema implemented
- ✅ Design documents complete for contacts page
- ❌ No contacts storage/schema yet
- ❌ No contacts page UI yet
- ❌ No API endpoints for contacts page

## Implementation Phases

### Phase 1: Backend Foundation (Prerequisites)

**Goal**: Set up data layer and API foundation for contacts page

#### 1.1 Contact Schema & Storage
- **Issue**: `hai-9ke` - Design Contact schema and storage
- **Tasks**:
  - Design contacts table schema (name, picture, email, phone, favorite_color, external_id)
  - Add to `storage/schema.go` and `storage/SCHEMA.md`
  - Create migration in `storage/migrations/`
  - Implement storage interface methods:
    - `CreateContact(ctx, contact)`
    - `GetContact(ctx, id)`
    - `ListContacts(ctx, filters)`
    - `UpdateContact(ctx, id, updates)`
    - `AssociateSpeakerWithContact(ctx, speakerID, contactID)`
  - Implement for both SQLite and Elasticsearch backends
- **Dependencies**: None
- **Estimated effort**: 2-3 days

#### 1.2 vCard Parsing & Import
- **Issue**: `hai-y0d` - Implement vCard parsing and storage
- **Tasks**:
  - Add vCard parsing library (e.g., `github.com/emersion/go-vcard`)
  - Create parser in `onboard/internal/contacts/` or `storage/contacts.go`
  - Parse `data/contacts/contacts.vcf` file
  - Import contacts into database
  - Handle duplicates (by email/phone matching)
  - CLI tool or server endpoint to trigger import
- **Dependencies**: `hai-9ke` (contact schema)
- **Estimated effort**: 1-2 days

#### 1.3 API Endpoints for Contacts Page
- **Issue**: `hai-4be` - Add API endpoints for Contacts page
- **Tasks**:
  - Add endpoints to `onboard/internal/server/handlers.go`:
    - `GET /api/contacts` - List contacts (with filters: known/unknown, search)
    - `GET /api/contacts/:id` - Get contact details
    - `POST /api/contacts` - Create new contact
    - `PUT /api/contacts/:id` - Update contact (including favorite_color)
    - `GET /api/speakers` - List unknown speakers (with filters: date range, sort)
    - `GET /api/speakers/:id/recordings` - Get recordings for a speaker
    - `GET /api/recordings/:id/segments` - Get segments for a recording
    - `POST /api/speakers/:id/associate` - Associate speaker with contact
    - `GET /api/recordings` - List recordings (with filters: speaker, date range)
  - Add request/response types in `onboard/internal/server/types.go`
  - Use storage interface (works with SQLite or Elasticsearch)
- **Dependencies**: `hai-9ke` (contact schema), `hai-y0d` (vCard import)
- **Estimated effort**: 2-3 days

#### 1.4 Server Architecture Refactoring
- **Issue**: `hai-af3` - Refactor server architecture for multi-page support
- **Tasks**:
  - Review current server structure (`onboard/internal/server/`)
  - Add routing for multiple pages (onboard, contacts, future: calendar)
  - Set up static file serving for multiple pages
  - Add navigation/routing structure
  - Ensure clean separation between pages
- **Dependencies**: None (can be done in parallel)
- **Estimated effort**: 1-2 days

### Phase 2: Frontend Foundation

**Goal**: Build the core contacts page UI

#### 2.1 Contacts Page UI Layout
- **Issue**: `hai-8ri` - Design and implement contacts page UI layout
- **Tasks**:
  - Create HTML template in `onboard/templates/contacts.html`
  - Implement responsive layout:
    - Mobile portrait: Side-by-side contacts/speakers (50% each, top 50%), recordings (bottom 50%)
    - Desktop: Three-column horizontal layout
  - Set up CSS framework/styling (consider Tailwind CSS or custom CSS)
  - Implement basic structure (three areas, no functionality yet)
  - Add navigation between onboard and contacts pages
- **Dependencies**: `hai-af3` (server architecture)
- **Estimated effort**: 2-3 days

#### 2.2 Native-Style Contacts List Component
- **Issue**: `hai-vrw` - Implement native-style contacts list component
- **Tasks**:
  - Build contacts list with:
    - Alphabetical section headers (or compact version)
    - Circular profile pictures (32-36px)
    - Name truncation (12-15 chars)
    - Known status indicators (green ✓, red X)
    - Search bar (collapsible on mobile)
    - Filter by known/unknown/both
  - Match iOS/Android native contacts app styling
  - Implement virtual scrolling for long lists
  - Add tap/click handlers (show recordings)
- **Dependencies**: `hai-8ri` (page layout), `hai-4be` (API endpoints)
- **Estimated effort**: 3-4 days

#### 2.3 Unknown Speakers Table Component
- **Tasks**:
  - Build compact speakers table:
    - Columns: ID (color-coded), Latest date
    - Secondary columns: Recordings, Hours (expandable/horizontal scroll)
    - Sortable columns
    - Filter by date range
    - Selected state highlighting
  - Implement color coding (default colors for speakers)
  - Compact design for ~180-215px width
- **Dependencies**: `hai-8ri` (page layout), `hai-4be` (API endpoints)
- **Estimated effort**: 2-3 days

#### 2.4 Recordings/Segments List Component
- **Tasks**:
  - Build recordings list:
    - Play/pause button (44x44px minimum)
    - Duration, date, time
    - Picture and name (if contact associated)
    - Conversation button
    - Transcript preview (expandable)
  - Implement audio playback controls
  - Color coding for visual association
- **Dependencies**: `hai-8ri` (page layout), `hai-4be` (API endpoints), `hai-epq` (audio playback)
- **Estimated effort**: 3-4 days

#### 2.5 Audio Playback
- **Issue**: `hai-epq` - Implement audio playback for recordings
- **Tasks**:
  - Set up audio player (HTML5 Audio API or library)
  - Implement play/pause/resume
  - Support byte offset ranges (HTTP Range requests)
  - Progress bar and time display
  - Long-press to restart
  - Handle multiple audio sources
- **Dependencies**: `hai-4be` (API endpoints for audio URLs)
- **Estimated effort**: 2-3 days

### Phase 3: Core Functionality

**Goal**: Enable the main association workflow

#### 3.1 Drag-and-Drop Association
- **Issue**: `hai-v8a` - Implement drag-and-drop association between contacts and recordings
- **Tasks**:
  - Implement drag initiation (long-press on mobile, click-drag on desktop)
  - Visual feedback during drag (elevation, highlights)
  - Drop zone highlighting
  - API call to associate speaker with contact
  - Haptic/audio feedback on success
  - Support both directions (contact → recording, recording → contact)
- **Dependencies**: `hai-8ri` (page layout), `hai-4be` (API endpoints), `hai-03j` (color coding)
- **Estimated effort**: 3-4 days

#### 3.2 Color Coding System
- **Issue**: `hai-03j` - Implement color coding system for visual association
- **Tasks**:
  - Assign default colors to unknown speakers
  - Store favorite colors for contacts (in contact schema)
  - Apply colors to rows (speakers, contacts, recordings)
  - Update colors when speaker is selected
  - Ensure consistency across app
  - Accessibility: WCAG AA contrast, color-blind friendly
- **Dependencies**: `hai-9ke` (contact schema with favorite_color), `hai-8ri` (page layout)
- **Estimated effort**: 2-3 days

#### 3.3 Create New Contact Feature
- **Issue**: `hai-sw0` - Add create new contact feature
- **Tasks**:
  - Add FAB or "+" button in contacts area
  - Create contact form/modal
  - Form fields: name (required), picture (optional), email, phone
  - Save to database
  - Immediately allow association with speaker
- **Dependencies**: `hai-9ke` (contact schema), `hai-4be` (API endpoints), `hai-8ri` (page layout)
- **Estimated effort**: 1-2 days

#### 3.4 Navigation Between Pages
- **Issue**: `hai-ylh` - Add navigation between Onboard and Contacts pages
- **Tasks**:
  - Add navigation bar/header
  - Links between onboard and contacts pages
  - Active page highlighting
  - Mobile-friendly navigation (hamburger menu?)
- **Dependencies**: `hai-af3` (server architecture), `hai-8ri` (contacts page)
- **Estimated effort**: 1 day

### Phase 4: Enhanced Features

**Goal**: Add polish and advanced features

#### 4.1 Smart Matching Suggestions
- **Issue**: `hai-cv1` - Smart speaker-contact matching suggestions
- **Tasks**:
  - Implement likelihood-based matching algorithm
  - Meeting participant matching
  - Gender inference matching
  - Temporal proximity matching
  - Calculate and display likelihood scores
- **Dependencies**: `hai-8ri` (page layout), calendar integration (future)
- **Estimated effort**: 3-5 days

#### 4.2 Visual Row Alignment
- **Issue**: `hai-znf` - Visual row alignment for suggested matches
- **Tasks**:
  - Align contact and speaker rows when matches suggested
  - Visual indicators (connecting lines, highlights)
  - Show multiple suggestions simultaneously
- **Dependencies**: `hai-cv1` (matching algorithm), `hai-8ri` (side-by-side layout)
- **Estimated effort**: 2-3 days

#### 4.3 Gamified Association Button
- **Issue**: `hai-oxf` - Gamified association button with feedback
- **Tasks**:
  - Add button between aligned rows
  - Visual animation (chain links snapping)
  - Audio feedback (snap sound)
  - Haptic feedback on mobile
- **Dependencies**: `hai-znf` (visual alignment)
- **Estimated effort**: 2-3 days

#### 4.4 DateTime and Location Filters
- **Issue**: `hai-2aq` - Add datetime and location filter bar
- **Tasks**:
  - Add filter bar to top of contacts page
  - DateTime range picker (start/end, side-by-side)
  - Location filter (text input or autocomplete)
  - Clear button
  - Visual indicator when filters active
  - Collapsible on mobile to save space
- **Dependencies**: `hai-8ri` (page layout), `hai-4be` (API endpoints with filters)
- **Estimated effort**: 2-3 days

### Phase 5: Future Features (Later)

- Calendar page (`hai-421`)
- Meeting planning page (`hai-l6d`)
- Person detail/PRM page (`hai-5h6`)
- Voice commands (`hai-pf8`)
- Landscape mode evaluation (`hai-cyr`)

## Recommended Implementation Order

### Sprint 1 (Week 1-2): Backend Foundation
1. `hai-9ke` - Contact schema and storage
2. `hai-y0d` - vCard parsing and import
3. `hai-4be` - API endpoints
4. `hai-af3` - Server architecture refactoring (parallel)

### Sprint 2 (Week 3-4): Frontend Foundation
1. `hai-8ri` - Contacts page UI layout
2. `hai-epq` - Audio playback
3. `hai-vrw` - Native-style contacts list
4. Unknown speakers table (no issue yet, part of `hai-8ri`)

### Sprint 3 (Week 5-6): Core Functionality
1. `hai-03j` - Color coding system
2. `hai-v8a` - Drag-and-drop association
3. `hai-sw0` - Create new contact
4. `hai-ylh` - Navigation between pages

### Sprint 4 (Week 7-8): Enhanced Features
1. `hai-cv1` - Smart matching suggestions
2. `hai-znf` - Visual row alignment
3. `hai-oxf` - Gamified association button
4. `hai-2aq` - DateTime/location filters

## Quick Start (Minimum Viable Product)

If you want to get a working contacts page quickly, focus on:

1. **Backend** (3-4 days):
   - `hai-9ke` - Contact schema (minimal: name, picture, external_id)
   - `hai-4be` - Basic API endpoints (list contacts, list speakers, associate)
   - `hai-af3` - Server routing for contacts page

2. **Frontend** (4-5 days):
   - `hai-8ri` - Basic page layout (three areas, no styling)
   - `hai-vrw` - Simple contacts list (no search/filter yet)
   - Unknown speakers table (basic)
   - Recordings list (basic, no audio playback yet)
   - Simple tap-to-associate (no drag-and-drop yet)

3. **Core Feature** (2-3 days):
   - `hai-03j` - Basic color coding (default colors only)
   - Simple association (tap contact, tap speaker, button to associate)

**Total MVP**: ~10-12 days of focused work

## Technical Decisions Needed

1. **Frontend Framework**: 
   - Vanilla JS + HTML/CSS?
   - React/Vue/Svelte?
   - Flutter (for cross-platform)?
   - Recommendation: Start with vanilla JS for MVP, migrate to framework later if needed

2. **CSS Framework**:
   - Tailwind CSS?
   - Custom CSS?
   - Recommendation: Tailwind for rapid development, or custom CSS for full control

3. **Audio Playback Library**:
   - HTML5 Audio API?
   - Howler.js?
   - Wavesurfer.js?
   - Recommendation: HTML5 Audio for MVP, upgrade to library if needed

4. **State Management**:
   - Simple JavaScript objects?
   - Redux/MobX?
   - Recommendation: Simple objects for MVP, add state management if complexity grows

## Testing Strategy

1. **Unit Tests**: Storage layer methods
2. **Integration Tests**: API endpoints
3. **E2E Tests**: Full association workflow
4. **Manual Testing**: Mobile devices (iOS/Android), desktop browsers

## Documentation

- Update `onboard/README.md` with contacts page info
- API documentation (OpenAPI/Swagger?)
- User guide for contacts page workflow





