# Contacts Page UI Design

## Overview

The contacts page is the primary interface for associating contacts with unknown speakers. This document outlines the UI/UX design with a focus on mobile-first responsive design.

## Design Principles

1. **Familiarity First**: Contacts area should mirror iOS/Android native contacts apps
2. **Mobile-First**: Optimize for phone usage, but support desktop/web
3. **Efficient Workflow**: Minimize taps/actions to complete associations
4. **Progressive Enhancement**: Start simple, add advanced features (landscape mode) later

## Layout Structure

### Three Main Areas

1. **Contacts Area** (left/top)
2. **Unknown Speakers Area** (right/middle)
3. **Recordings/Segments Area** (bottom/remainder)

### Future: Top Filter Bar (Later Feature)

**Planned additions** (will further stress space constraints):
- **DateTime Range Filter**: Start/end datetime picker (side-by-side)
  - Purpose: Constrain time scope of contact/speaker activity
  - Use case: Filter to a specific calendar meeting
  - Example: "2024-12-02 10:00 AM" to "2024-12-02 11:30 AM"
- **Location Filter**: Location name/selector (side-by-side with datetime)
  - Purpose: Filter by location (e.g., "Conference Room A", "Park", "Coffee Shop")
  - Use case: "Show me conversations at the park this morning"
  - Can be text input or dropdown/autocomplete

**Layout Impact**:
- Adds ~40-60px height to top of page
- Reduces available space for Contacts/Speakers/Recordings areas
- May require collapsible filter bar (expand/collapse icon)
- Or overlay/modal for filters on mobile to save space

**Design Considerations**:
- Compact datetime pickers (native mobile pickers)
- Location can be autocomplete/search
- Clear button to reset filters
- Visual indicator when filters are active

## Mobile Layout (Portrait - Primary)

### Side-by-Side Layout (Required for Visual Alignment Feature)

**CRITICAL REQUIREMENT**: Contacts and Speakers must be side-by-side to enable the visual row alignment feature (`hai-znf`) where matching rows can be visually connected.

```
┌──────────────────────────────┐
│ [DateTime Start] [DateTime   │ ← Future: Filter bar
│  End] [Location] [Clear]     │   (collapsible/overlay)
├──────────────┬──────────────┤
│   Contacts   │   Speakers   │
│   (50% width)│  (50% width) │
│   (50% height)              │
│   - Compact  │   - Compact  │
│     list     │     table    │
│   - Search   │   - Sortable │
│   - Filter   │   - Filter   │
├──────────────┴──────────────┤
│   Recordings/Segments       │
│   (100% width, 50% height)  │
│   - Play controls           │
│   - Transcript (if fit)     │
└─────────────────────────────┘
```

**Screen Space Analysis**:
- **Typical phone width**: 375-430px (iPhone SE to Pro Max)
- **Per-column width**: ~187-215px (with minimal margins)
- **Top section height**: ~333-466px (50% of 667-932px screen height)
- **Bottom section height**: ~333-466px (remaining 50%)

**Feasibility**: ✅ **Feasible with compact design optimizations**

### Feasibility Analysis

**Screen Dimensions** (Portrait):
- **iPhone SE (smallest)**: 375px × 667px
  - Per column: ~187px width
  - Top section: ~333px height
- **iPhone 14 Pro (standard)**: 390px × 844px
  - Per column: ~195px width
  - Top section: ~422px height
- **iPhone 14 Pro Max (large)**: 430px × 932px
  - Per column: ~215px width
  - Top section: ~466px height
- **Typical Android**: 360-412px × 640-915px
  - Per column: ~180-206px width
  - Top section: ~320-457px height

**Design Constraints & Solutions**:

1. **Narrow Column Width (~180-215px)**
   - ✅ **Solution**: Aggressive truncation, smaller fonts, minimal padding
   - ✅ **Solution**: Icon-only status indicators, compact profile pictures
   - ✅ **Solution**: Collapsible search/filter (icon button, expand on tap)

2. **Limited Vertical Space (~320-466px for top section)**
   - ✅ **Solution**: Compact row heights (44-48px minimum)
   - ✅ **Solution**: Remove or minimize section headers
   - ✅ **Solution**: Sticky headers that collapse when scrolling

3. **Contact Name Display**
   - ⚠️ **Challenge**: Long names need truncation
   - ✅ **Solution**: Ellipsis after ~12-15 characters
   - ✅ **Solution**: Full name on tap/long-press
   - ✅ **Solution**: Prioritize first name + last initial if space allows

4. **Speaker Table Columns**
   - ⚠️ **Challenge**: 4 columns (ID, Latest, Recordings, Hours) won't fit
   - ✅ **Solution**: Primary view shows only ID + Latest (2 columns)
   - ✅ **Solution**: Secondary columns (Recordings, Hours) via:
     - Horizontal scroll on row
     - Expand on tap
     - Or show in bottom recordings area when speaker selected

5. **Visual Alignment Feature** (`hai-znf`)
   - ✅ **Benefit**: Side-by-side layout enables visual row alignment
   - ✅ **Implementation**: Connect matching rows with visual line/indicator
   - ✅ **UX**: Users can see suggested matches at a glance

**Trade-offs**:
- ❌ Less information visible at once (requires scrolling/tapping)
- ❌ More aggressive truncation than ideal
- ✅ Enables critical visual alignment feature
- ✅ Better for association workflow (see both sides simultaneously)
- ✅ More efficient use of screen space than vertical stack

**Conclusion**: The side-by-side layout is **feasible and necessary** for the visual alignment feature. The compact design requires careful optimization but provides better UX for the core association workflow.

### Contacts Area Design (Mobile Portrait - Side-by-Side)

**CRITICAL CONSTRAINT**: Must fit in ~187-215px width (50% of screen)

**Visual Style**: Match native iOS/Android contacts apps, but optimized for narrow width

**Compact Design Optimizations**:
- **Profile pictures**: Smaller (32-36px instead of 40-48px)
- **Name text**: Truncate with ellipsis, smaller font (14-15pt instead of 17pt)
- **Status indicator**: Icon-only (no text), positioned inline with name
- **Minimal padding**: 8-12px horizontal padding (instead of 16px)
- **No section headers**: Skip alphabetical grouping to save vertical space (or use very compact headers)
- **Search bar**: Collapsible or icon-only button (tap to expand)
- **Filter**: Icon button in header (not full-width bar)

**iOS-style elements** (compact):
- Contact rows with:
  - Small circular profile picture (left, 32-36px)
  - Name (truncated, 14-15pt)
  - Status icon (right, 16-20px)
    - Green checkmark (✓) for known
    - Red X for unknown
    - Gray dash (-) for not checked
- Smooth scrolling with momentum
- Pull-to-refresh support (if space allows)

**Android-style elements** (compact):
- Material Design 3 components (compact variants)
- Contact rows with:
  - Small circular avatar (left, 32-36dp)
  - Name (truncated, 14-15sp)
  - Status icon badge (right)
- Collapsible search bar
- Icon-based filter button

**Common Features**:
- Tap contact → shows their recordings in bottom area
- Long-press → context menu (future: quick actions)
- Swipe gestures (future: swipe to associate)
- **Horizontal scroll**: If contact name is too long, allow horizontal scroll within row (rare)
- **Color coding**: Contact row background = contact's favorite color (if set) or associated speaker's color

**Space-Saving Strategies**:
1. **Hide less essential info**: Show only picture, name, status
2. **Truncate aggressively**: Names longer than ~12-15 chars get ellipsis
3. **Compact row height**: 44-48px (minimum touch target)
4. **Sticky search**: Search bar collapses to icon when scrolling
5. **Filter as overlay**: Filter options appear as bottom sheet/modal (not inline)

### Unknown Speakers Area (Mobile Portrait - Side-by-Side)

**CRITICAL CONSTRAINT**: Must fit in ~187-215px width (50% of screen)

**Ultra-Compact Table Design**:
- **Essential columns only**: `[ID] [Latest]` (primary view)
- **Secondary columns**: `[Recordings] [Hours]` (show on tap/expand or horizontal scroll)
- Row height: ~44-48px (minimum touch-friendly)
- **ID column** (left, ~40-50px):
  - Color-coded badge (pink/blue/purple) - **OR** use row background color instead
  - Short identifier (1-2 chars, large font for visibility)
  - Circular or rounded square badge
  - **Alternative**: Skip badge, use row background color for visual identification
- **Latest column** (right, remaining space):
  - Date in compact format: "12/2" or "Dec 2" or relative "2d ago"
  - Small font (12-13pt)
  - Monospace font for alignment
- Sortable columns (tap header to sort)
- Selected speaker highlighted (background color change, recordings get same color)
- Filter button (date range) as icon in header
- **Color coding**: Each speaker row has unique background color (or contact's favorite color if associated)

**Space-Saving Strategies**:
1. **Two-column primary view**: ID + Latest only
2. **Horizontal scroll**: Swipe left on row to reveal Recordings/Hours
3. **Expandable rows**: Tap to expand and show all columns
4. **Compact date format**: Use relative dates ("2d ago") or short format ("12/2")
5. **Minimal padding**: 4-8px horizontal padding
6. **Compact headers**: Smaller font, icon-based sort indicators
7. **Filter as overlay**: Date range picker appears as bottom sheet/modal

**Alternative: Card View** (if table too cramped):
- Each speaker as a compact card
- Stacked info: ID badge (large) on top, Latest below
- Tap to expand for full details
- Easier to align with contact rows visually

### Recordings/Segments Area (Mobile Portrait)

**Layout**: Full width (100%), bottom 50% of screen (~333-466px height)

**Compact Row Design**:
- Essential info: `[▶] [MM] [Picture] [Name] [Conversation]`
- Secondary info: `[Date] [Time]` (smaller, secondary text)
- Play button: 44x44px (touch-friendly, left side)
- Picture: 32-36px circular (if contact associated)
- Name/Speaker ID: Truncated, color-coded if unknown speaker
- Conversation button: Icon only (branch icon, right side)
- Duration (MM): Small text, monospace

**Space Optimization**:
- **Two-line rows**: Primary info on first line, date/time on second (smaller font)
- **Or single-line**: Truncate aggressively, show date/time on tap
- **Transcript**: Expandable (tap row to expand, shows below)
- **Horizontal scroll**: If needed for very long names

**Audio Controls**:
- Large play/pause button (44x44px minimum, left)
- Progress bar: Thin, below row (when playing)
- Time remaining: Small text next to duration
- Long-press play button → restart from beginning

**Interaction**:
- Tap row → expand to show transcript (pushes other rows down)
- Drag from contact/speaker → drop on recording to associate
- Swipe gestures (future: swipe to associate)
- **Visual alignment**: When contact/speaker rows align, show connecting line/indicator
- **Color coding**: Recording row background = speaker's color (or contact's favorite color if associated)
  - When speaker is selected, all their recordings get same background color for visual association

## Desktop/Web Layout

### Horizontal Split Layout

```
┌──────────┬──────────────┬──────────────────┐
│          │              │                  │
│ Contacts │  Speakers    │   Recordings     │
│          │              │                  │
│ (25%)    │   (25%)      │    (50%)         │
│          │              │                  │
└──────────┴──────────────┴──────────────────┘
```

**Advantages**:
- All three areas visible simultaneously
- More horizontal space for tables
- Better for multi-tasking
- Easier drag-and-drop between areas

**Responsive Breakpoints**:
- Mobile: < 768px (vertical stack)
- Tablet: 768px - 1024px (hybrid layout)
- Desktop: > 1024px (horizontal split)

## Landscape Mode (Future Enhancement)

### Potential Layout

```
┌──────────────────┬──────────────────────────┐
│                  │                          │
│  Contacts        │   Recordings +           │
│  (narrow)        │   Transcript             │
│                  │   (wide, readable)       │
│  Speakers        │                          │
│  (narrow)        │                          │
└──────────────────┴──────────────────────────┘
```

**Use Case**:
- User selects speaker in portrait mode
- Rotates to landscape to read transcript while listening
- Rotates back to portrait to make association

**Considerations**:
- May be unnecessary if portrait mode works well
- Adds complexity (orientation handling, layout switching)
- Test with real users first before implementing

## Contacts Area - Native App Patterns

### iOS Contacts App Reference

**Visual Elements**:
- **List Style**: Grouped table view with section headers
- **Contact Row**:
  ```
  [Picture] Name                    [Status]
  ```
- **Colors**: System blue for selection, gray for text
- **Typography**: SF Pro (system font)
- **Search**: UISearchController with scope buttons
- **Navigation**: Push navigation to detail view

**Key Patterns to Replicate**:
1. Alphabetical grouping with section headers
2. Circular profile pictures (consistent sizing)
3. Smooth scrolling with section headers sticking
4. Search bar that collapses/expands
5. Pull-to-refresh gesture

### Android Contacts App Reference

**Visual Elements**:
- **List Style**: RecyclerView with Material Design cards
- **Contact Row**:
  ```
  [Avatar] Name              [Status Chip]
  ```
- **Colors**: Material Design 3 color system
- **Typography**: Roboto (system font)
- **Search**: Material search bar with suggestions
- **Navigation**: Fragment navigation

**Key Patterns to Replicate**:
1. Material Design 3 components
2. Elevation and shadows for depth
3. Ripple effects on touch
4. Bottom sheet for filters/actions
5. Snackbar for feedback

## Responsive Design Strategy

### Breakpoint Strategy

1. **Mobile Portrait** (< 768px)
   - **Side-by-side top**: Contacts (50% width) + Speakers (50% width) = 50% height
   - **Full-width bottom**: Recordings (100% width) = 50% height
   - Ultra-compact design with aggressive space optimization
   - Touch-optimized (44px minimum touch targets)
   - **Critical**: This layout enables visual row alignment feature

2. **Mobile Landscape / Tablet Portrait** (768px - 1024px)
   - **Option A**: Keep side-by-side (more horizontal space available)
   - **Option B**: Three-column horizontal (Contacts | Speakers | Recordings)
   - More comfortable spacing, less aggressive truncation
   - Larger touch targets

3. **Desktop / Tablet Landscape** (> 1024px)
   - Three-column horizontal layout (25% | 25% | 50%)
   - Comfortable spacing, full information display
   - Hover states for interactions
   - Keyboard shortcuts support
   - Drag-and-drop between columns

### Touch Target Sizes

- **Minimum**: 44x44px (iOS) / 48x48dp (Android)
- **Recommended**: 48x48px for primary actions
- **Play button**: 56x56px (prominent)
- **Table rows**: 48-56px height minimum

### Typography Scaling

- **Mobile**: Base 16px, scale down for secondary text
- **Desktop**: Base 16px, can scale up slightly
- **Headings**: 1.5x - 2x base size
- **Captions**: 0.875x base size

## Interaction Patterns

### Drag and Drop

**Mobile**:
- Long-press to initiate drag
- Visual feedback (elevation, scale)
- Drop zones highlighted
- Haptic feedback on drop

**Desktop**:
- Click and drag
- Visual feedback (cursor change, ghost image)
- Drop zones highlighted
- Keyboard modifier keys for special actions

### Selection States

- **Contact selected**: Highlighted background, checkmark
- **Speaker selected**: Highlighted row, recordings shown
- **Recording playing**: Active state indicator, progress bar

### Color Coding System (Visual Association)

**Purpose**: Provide consistent visual association between speakers, contacts, and recordings throughout the app.

**Color Assignment**:
1. **Default Speaker Color**: Each unknown speaker gets a unique color (light green, light blue, light pink, etc.)
   - Assigned automatically when speaker is first detected
   - Ensures visual distinction between multiple speakers
2. **Contact Favorite Color**: Contacts can have a favorite color (set in person detail page)
   - User picks from color picker/selector
   - Stored in contact/person profile
   - Takes precedence over speaker's default color when associated

**Visual Application**:
- **Speaker Row**: Background color = speaker's color (default or contact's favorite if associated)
- **Contact Row**: Background color = contact's favorite color (if set), or speaker's color (if associated with speaker)
- **Selected Speaker**: When speaker is selected, all their recording rows get the same background color
- **Recording Rows**: 
  - If speaker is associated: Use speaker/contact color
  - If speaker is selected: Use speaker color for visual association
  - If no association: Use speaker's default color or neutral

**Color Consistency**:
- Same person = same color everywhere in the app
- Calendar page: Participant pictures have colored border/background
- Person detail page: Header/theme uses their color
- Timeline/visualizations: Use person's color for their data

**Design Considerations**:
- Use light/pastel colors for backgrounds (readable text contrast)
- Ensure sufficient contrast for accessibility (WCAG AA minimum)
- Color-blind friendly: Consider patterns/icons in addition to color
- Default palette: Light green, light blue, light pink, light yellow, light purple, light orange, etc.

### Feedback

- **Visual**: Animations, color changes, highlights
- **Audio**: Snap sound on association (gamified feature)
- **Haptic**: Vibration on mobile for key actions
- **Toast/Snackbar**: Success/error messages

## Accessibility

### Requirements

1. **Screen Reader Support**:
   - Semantic HTML/ARIA labels
   - Announce contact names, status, actions
   - Describe drag-and-drop actions

2. **Keyboard Navigation**:
   - Tab through all interactive elements
   - Arrow keys to navigate lists
   - Enter/Space to activate
   - Escape to cancel drag

3. **Visual**:
   - High contrast mode support
   - Color not sole indicator (use icons + color)
   - Text size scaling support

4. **Motor**:
   - Large touch targets
   - Alternative to drag-and-drop (tap to select, button to associate)
   - Reduced motion support

## Implementation Considerations

### Technology Stack

**Frontend Framework Options**:
- **React** (if web-focused)
- **Flutter** (if cross-platform mobile + web)
- **Native** (iOS SwiftUI + Android Kotlin) - if mobile-only

**UI Component Libraries**:
- **iOS**: SwiftUI native components
- **Android**: Material Design 3 (Jetpack Compose)
- **Web**: 
  - Material-UI (React)
  - Flutter Material (Flutter)
  - Or custom CSS matching native patterns

### State Management

- Selected contact
- Selected speaker
- Currently playing recording
- Filter states (known/unknown, date range)
- Search query
- Sort order

### Performance

- **Virtual scrolling** for long contact lists (100+ contacts)
- **Lazy loading** of contact pictures
- **Debounced search** (wait for user to stop typing)
- **Optimistic UI updates** for associations (update immediately, sync in background)

## Future Enhancements

1. **Landscape Mode**: As discussed, if needed
2. **Swipe Gestures**: Swipe right to associate, left to dismiss
3. **Batch Operations**: Select multiple speakers, associate to one contact
4. **Voice Commands**: "Associate speaker A with John"
5. **Quick Actions**: Long-press menu with common actions
6. **Keyboard Shortcuts**: Desktop power-user features
7. **Dark Mode**: System preference support

## Testing Strategy

### Device Testing

- **iOS**: iPhone SE (small), iPhone 14 Pro (standard), iPad (tablet)
- **Android**: Pixel 5 (small), Pixel 7 Pro (standard), Pixel Tablet
- **Web**: Chrome, Safari, Firefox on various screen sizes

### User Testing

1. **Task Completion**: Can users successfully associate contacts?
2. **Time to Complete**: How long does it take?
3. **Error Rate**: How many mistakes/frustrations?
4. **Preference**: Portrait vs landscape usage patterns

## Future Features

### Create New Contact
- **Use Case**: User has a speaker but no matching contact exists
- **Trigger**: Button/action in contacts area or speakers area
- **Flow**: 
  - Tap "Create Contact" or "+" button
  - Opens contact creation form/modal
  - User enters name, optionally picture, email, phone
  - New contact is created and can be associated with speaker
- **Location**: Could be FAB (Floating Action Button) or in contacts area header

### Voice Commands (Long-term)
- **Use Case**: "Bring up the conversation I had at the park this morning"
- **Functionality**:
  - Natural language parsing for time/location/context
  - Filters contacts page to matching datetime range and location
  - Shows relevant speakers and recordings
  - User can then quickly match contacts with speakers
- **Integration**: Voice assistant integration (Siri, Google Assistant, or custom)

## Open Questions

1. **Landscape Mode**: Is it necessary, or does portrait work well enough?
2. **Table vs List**: Should speakers area be a table or list view? (Leaning toward compact table for alignment)
3. **Transcript Display**: Always visible, expandable, or separate view?
4. **Drag Direction**: Contact → Recording, Recording → Contact, or both?
5. **Bulk Operations**: How important is batch association?
6. **Speaker Columns**: Which columns are essential in primary view? (ID + Latest confirmed, Recordings/Hours secondary)
7. **Contact Truncation**: What's the optimal character limit before ellipsis? (12-15 chars suggested)
8. **Search Bar**: Always visible, collapsible, or icon-only? (Icon-only recommended for space)
9. **Filter Bar**: Collapsible, always visible, or overlay/modal? (Collapsible recommended to save space)
10. **Create Contact**: FAB, header button, or context menu? (FAB recommended for discoverability)

## Related Documents

- `seed.md` - Original requirements
- `history/CONTACTS_INTEGRATION_NOTES.md` - Contacts data integration
- Beads issues:
  - `hai-cv1` - Smart speaker-contact matching suggestions
  - `hai-znf` - Visual row alignment for suggested matches
  - `hai-oxf` - Gamified association button with feedback

