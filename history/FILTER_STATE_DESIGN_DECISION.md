# Filter State Design Decision: Page-Specific vs Global

## Analysis

### User's Key Insight

Filters are **PAGE-SPECIFIC**, not global. Each page has different filter needs and contexts.

## Page-by-Page Filter Requirements

### Calendar Day View
**Needs:**
- ✅ **Date filter**: Shows conversations for selected date
  - Should remember when navigating away and coming back
  - Current implementation: `_selectedDate` state in CalendarScreen
  
- ✅ **People filter**: Filter conversations to only those involving certain people
  - Example: Show only conversations with Alice & Bob
  - OR logic: Show conversations where ANY of the selected people participated

**Should NOT share with:**
- ❌ Todo page date filter (different purpose - todos vs conversations)
- ❌ Conversation page (no date filter needed, shows all participants)

---

### Conversation Page
**Needs:**
- ❌ **NO date filter**: Date comes from route params (already in URL)
  - Conversation is tied to a specific date/time
  - No need to filter by date
  
- ❌ **NO people filter**: The people list is **conversation data**, not a filter
  - Shows ALL participants in THAT specific conversation
  - This is displaying conversation participants, not filtering
  - Can add people (for associating unknowns), but that's conversation-specific data
  - Adding a person in conversation shouldn't affect calendar filter

**Should NOT share with:**
- ❌ Calendar filters (different context entirely)

---

### People Page
**Needs:**
- ❌ **NO date filter**: Shows all contacts, not filtered by date
- ❌ **NO people filter**: Filtering people by people doesn't make sense

---

### Todo Page (Future)
**Needs:**
- ❓ **Date filter**: BUT for different purpose
  - "What's due/done on X date"
  - Different from calendar date (reviewing yesterday's conversations → want today's todos, not yesterday's)
  
- ❓ **People filter**: BUT for different purpose
  - "Show todos assigned to Alice"
  - Different from calendar people filter (conversations vs todos)

**Should NOT share with:**
- ❌ Calendar filters (completely different data type - todos vs conversations)

---

## User's Examples (Why Global Doesn't Work)

1. **Date Filter Conflict:**
   - User viewing yesterday's conversations on calendar
   - Goes to Todo page
   - Should see what's due TODAY or in future, NOT yesterday
   - ❌ Global date filter would be wrong here

2. **People Filter Conflict:**
   - Calendar filtered to show only conversations with Alice & Bob
   - Opens a conversation (Alice, Bob, Charlie participated)
   - Conversation page should show ALL participants (Alice, Bob, Charlie)
   - NOT filtered to just Alice & Bob
   - ❌ Global people filter would hide Charlie

3. **Adding People Conflict:**
   - Calendar: filtered to Alice & Bob
   - Conversation: Add Charlie to associate unknown speaker
   - Going back to Calendar: Should still be filtered to Alice & Bob (not include Charlie)
   - ❌ Global filter would incorrectly add Charlie to calendar filter

---

## Proposed Solution: Page-Scoped Persistent State

### Architecture

Instead of **global filter state**, use **page-scoped filter providers**:

```dart
// Calendar page filters
final calendarDateFilterProvider = StateProvider<DateTime?>((ref) => null);
final calendarPeopleFilterProvider = StateProvider<List<String>>((ref) => []);

// Todo page filters (future)
final todoDateFilterProvider = StateProvider<DateTime?>((ref) => null);
final todoPeopleFilterProvider = StateProvider<List<String>>((ref) => []);

// Conversation page: NO filter providers
// Uses conversation data directly from route params and API
```

### Key Properties

1. **Page-Scoped**: Each page has its own filter state
   - Calendar filters only affect calendar page
   - Todo filters only affect todo page
   - No cross-page contamination

2. **Persistent**: State persists when navigating away and coming back
   - Calendar date selected → navigate to People → navigate back to Calendar
   - Calendar should remember the selected date
   - This is automatic with Riverpod StateProvider (survives navigation)

3. **Independent**: Filters on different pages don't interfere
   - Calendar date filter independent from Todo date filter
   - Calendar people filter independent from Todo people filter

---

## Conversation Page Special Case

**Important Distinction:**

The conversation page's "people list" is **NOT a filter** - it's conversation data:
- Shows ALL participants in that conversation
- Derived from the conversation's blockquotes
- This is displaying conversation participants, not filtering conversations

Adding people via + button:
- Adds them to the conversation (for associating unknown speakers)
- This is conversation-specific data modification
- Should NOT affect calendar or todo filters

---

## Implementation Plan

### Phase 1: Page-Scoped Filter State (Updated hai-yr85)

Create page-scoped filter providers:
- `calendarDateFilterProvider` - only for calendar page
- `calendarPeopleFilterProvider` - only for calendar page
- Future: `todoDateFilterProvider`, `todoPeopleFilterProvider` for todo page

### Calendar Page Integration:
- Replace `_selectedDate` state with `calendarDateFilterProvider`
- Use `calendarPeopleFilterProvider` for people filter
- Filter conversations by date + people (OR logic for multiple people)

### Conversation Page:
- NO filter providers needed
- People list comes from conversation data (blockquotes)
- Display conversation participants (not a filter)

---

## Questions to Confirm

1. ✅ Calendar date filter should persist when navigating away/back?
2. ✅ Calendar people filter should persist when navigating away/back?
3. ✅ Conversation page people list is data (not filter) - confirmed
4. ✅ Each page has independent filters - confirmed
5. ✅ No sharing between pages - confirmed

---

## Benefits of Page-Scoped Approach

1. **Clear Separation**: Each page's filters are independent
2. **No Conflicts**: Todo date filter doesn't interfere with Calendar date filter
3. **Correct Behavior**: Conversation shows all participants, not filtered list
4. **Persistent**: Filters remembered when navigating away/back (Riverpod feature)
5. **Flexible**: Each page can have different filter types/behavior

---

## Conclusion

**Filters should be PAGE-SPECIFIC with PERSISTENCE**, not global. This matches the user's requirements and examples perfectly.

