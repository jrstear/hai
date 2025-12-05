# Filter System Implementation Order

## Overview
This document outlines the logical order for implementing the filter system with **page-scoped** state management, filter bar UI, people selector, and related features.

## Implementation Order

### Phase 1: Foundation - Page-Scoped Filter State
**Bead: hai-yr85 - Implement page-scoped filter state management**

**Tasks:**
1. Create `filter_provider.dart` with:
   - `calendarDateFilterProvider` - StateProvider<DateTime?> for calendar page only
   - `calendarPeopleFilterProvider` - StateProvider<List<String>> for calendar page only
   - Helper functions to add/remove people, set date, clear filters
   - State should persist across navigation (Riverpod StateProvider survives navigation)
   - Future: `todoDateFilterProvider`, `todoPeopleFilterProvider` for todo page

**Why first:** Everything else depends on being able to store and retrieve filter state. This is the foundation.

**Key Insight:** Filters are page-specific, not global. Each page has its own filter state that persists independently.

---

### Phase 2: Filter Bar Framework
**Bead: hai-ro1v - Implement filter bar framework with time and people filters**

**Tasks:**
1. Create `filter_bar.dart` widget:
   - Reusable component that appears at top of screens
   - Contains time filter on left, people filter on right
   - Handles layout and spacing
   - Framework for adding more filters later

2. Integrate filter bar into:
   - Calendar screen (replaces current date selector)
   - Conversation screen
   - People screen (optional for now)
   - Todo screen (optional for now)

**Depends on:** Phase 1 (needs filter state providers)

---

### Phase 3: Time/Date Filter Component
**Bead: hai-ro1v (part 2) - Time filter implementation**

**Tasks:**
1. Create `time_filter.dart` widget:
   - Date display with < > buttons for prev/next day
   - Swipe gestures support (left/right)
   - Click date to open date picker
   - Connects to `dateFilterProvider` from Phase 1

2. Integrate into filter bar (left side)

**Depends on:** Phase 1, Phase 2

---

### Phase 4: People Filter Display
**Bead: hai-67tl - Implement people filter display with alphabetical avatars**

**Tasks:**
1. Create `people_filter_display.dart` widget:
   - Reads from `calendarPeopleFilterProvider` (or conversation participants for conversation page)
   - Shows contact avatars in alphabetical order (first,family name)
   - Starts at right edge, expands leftward
   - Always shows + button at far right
   - Handles wrapping (smart wrapping on conversation page)
   - Shows X button on each avatar for removal (future)

2. Integrate into filter bar (right side)

3. Handle conversation page special case:
   - Smart wrapping logic (wrap title first up to 5 people)
   - Dynamic boundary based on space

**Depends on:** Phase 1, Phase 2, ContactAvatar widget (already exists)

---

### Phase 5: People Selector Drawer
**Bead: hai-hxyh - Implement people selector drawer/slide-in component**

**Tasks:**
1. Create `people_selector.dart` widget:
   - Slide-in drawer from right (1/3 screen width)
   - Three sections:
     - Top bar: close button (left) + search bar (right)
     - People in filter section (sorted, pre-checked)
     - All other people section (sorted)
   - Three columns: checkbox, picture, name
   - Checkbox adds immediately without closing
   - Row click adds and closes
   - Search filters by name substring

2. Open/close animations (slide in/out from right)

3. Integrate with filter state:
   - **On Calendar/Todo pages:** Read current filter to show "people in filter" section
   - **On Conversation page:** Read conversation participants to show "conversation participants" section
   - Update filter/participants when checkbox/row clicked

**Depends on:** Phase 1, ContactsProvider (already exists)

---

### Phase 6: Add/Remove Interactions
**Bead: hai-3ty0 - Implement add/remove people from filter interactions**

**Tasks:**
1. Connect + button in people filter → opens people selector

2. Implement add people logic:
   - Checkbox click → add to filter immediately
   - Row click → add to filter + close selector

3. Implement remove people logic (basic):
   - X button on avatar → remove from filter
   - Update filter state provider

**Depends on:** Phase 4, Phase 5

---

### Phase 7: Conversation-Specific Filter Display
**Bead: hai-hik4 - Implement conversation-specific people filter display**

**Tasks:**
1. Modify conversation screen:
   - Move title and people list into filter bar
   - Title on left, people on right (same line)
   - Shows conversation participants (data, not a filter)

2. Smart wrapping logic:
   - Wrap title first (up to 5 people)
   - After 5 people, wrap title then allow people to wrap
   - Dynamic boundary calculation

3. Allow adding people via + button:
   - Adds to conversation participants list
   - Added people appear in "conversation participants" section of people selector
   - Makes it easier to associate unknown speakers (they're at top, no scrolling)

**Note:** Conversation page uses conversation participants (conversation-specific data), NOT filter state.

**Depends on:** Phase 2, Phase 4

---

### Phase 8: Unknown Speaker Association
**Bead: hai-zod3 - Associate unknown speaker from conversation view**

**Tasks:**
1. Make unknown speaker icon clickable in blockquotes

2. Open people selector on click

3. Association logic:
   - If person in conversation participants: just associate blockquote
   - If person not in conversation participants: associate + add to conversation participants

4. API integration for associating speaker with blockquote

**Note:** On conversation page, this works with conversation participants (not filter state).

**Depends on:** Phase 5, Phase 6

---

## Summary of Dependencies

```
Phase 1 (Page-Scoped State) ← Foundation
    ↓
Phase 2 (Filter Bar Framework)
    ↓
Phase 3 (Time Filter) ────┐
    ↓                     │
Phase 4 (People Display) ─┤
    ↓                     │
Phase 5 (Selector) ───────┤ All depend on Filter Bar
    ↓                     │
Phase 6 (Interactions) ───┤
    ↓                     │
Phase 7 (Conversation) ───┘
    ↓
Phase 8 (Unknown Speaker)
```

## Key Implementation Notes

1. **Page-Scoped State First**: Must be able to read/write filter state before any UI can display or modify it. Each page has its own filter state that persists independently.

2. **Filter Bar Framework**: Provides the container structure - everything else plugs into it

3. **Time Filter**: Can be built alongside people filter, but time is simpler (single value vs list)

4. **People Display**: Needs state + framework, but can start as read-only display

5. **People Selector**: Complex component, but can be built independently and integrated later

6. **Conversation Special Case**: Builds on existing filter display but adds conversation-specific logic

7. **Unknown Speaker**: Final integration piece that ties everything together

## Testing Strategy

- Phase 1: Test state persistence across navigation
- Phase 2: Test filter bar appears on all screens
- Phase 3: Test date selection, navigation, persistence
- Phase 4: Test avatar display, alphabetical ordering, wrapping
- Phase 5: Test drawer animations, search, checkbox/row clicks
- Phase 6: Test adding/removing updates filter state
- Phase 7: Test conversation page layout, wrapping logic
- Phase 8: Test unknown speaker association workflow

## Future Enhancements (Separate Beads)

- API filtering support (hai-8klp) - Server-side filtering optimization
- Remove people from conversation filter (hai-owbz) - UX enhancement
- AND condition for multi-person filter (hai-1s84) - Advanced filtering
- Hide filter bar on certain pages (hai-6xg8) - Layout optimization

