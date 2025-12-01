# Conversation Schema Design

## Terminology Decision

**Use "conversation"** (consistent with Limitless) for groups of segments/blockquotes.

**Definition:** A conversation is a contiguous series of segments/blockquotes by one or more distinct speakers.

- Limitless uses "conversation" for both single and multi-speaker groups
- We'll use the same term for consistency
- The distinction between monolog (1 speaker) and conversation (2+ speakers) is semantic/display, not structural
- Contiguity means segments are adjacent in time (may have small gaps, but no large breaks)

### Terminology Origins

**"Segment"** - Inherited from pyannote.audio (our diarization library)
- Standard term in speaker diarization literature
- Refers to a time period during which a single speaker speaks
- Used throughout our codebase and schema

**"Blockquote"** - Limitless API terminology
- Used by Limitless for transcribed speech segments
- Likely named because speech appears as markdown blockquotes in their UI
- Not standard in speech processing, but we reference it when discussing Limitless data

**See:** `history/TERMINOLOGY_ORIGINS.md` for detailed investigation

## Schema Design Principles

### Don't Store Derived Fields

**Rationale:** Store the data, derive the metadata. Elasticsearch is mature and efficient at aggregations.

**Fields NOT to store:**
1. **`speaker_count`** - Can be derived by counting unique `speaker_id` values in segments
2. **`speaker_list`** - Can be derived using Elasticsearch `terms` aggregation to get unique speaker IDs
3. **`type`** (monolog/conversation) - Can be derived from speaker count > 1
   - This is a client/display concern, not a storage concern

**Elasticsearch capabilities:**
- **Unique values**: Use `terms` aggregation on `speaker_id` field
- **Count unique**: Use `cardinality` aggregation on `speaker_id` field
- **Filter by count**: Use `bucket_selector` aggregation to filter buckets by count

**Example query to get speaker count:**
```json
{
  "aggs": {
    "unique_speakers": {
      "cardinality": {
        "field": "speaker_id"
      }
    }
  }
}
```

**Example query to get unique speaker list:**
```json
{
  "aggs": {
    "speakers": {
      "terms": {
        "field": "speaker_id",
        "size": 100
      }
    }
  }
}
```

## Conversation Schema (Proposed)

### Core Fields

```go
type Conversation struct {
    ID          string    `json:"id"`           // Unique conversation ID (e.g., "conv_xxxxx")
    Title       string    `json:"title"`        // Auto-generated or user-provided title
    StartTime   time.Time `json:"start_time"`   // UTC start time (from earliest segment)
    EndTime     time.Time `json:"end_time"`     // UTC end time (from latest segment)
    CreatedAt   time.Time `json:"created_at"`   // When conversation was created
    UpdatedAt   time.Time `json:"updated_at"`   // Last update timestamp
    
    // Relationships
    // Note: RecordingIDs not stored - can be derived by matching conversation time range
    // with recording time ranges (recordings have start_time and duration)
    
    // Grouping metadata
    // Flexible approach: Store grouping factors as key-value pairs
    // This allows new grouping factors to be added without schema changes
    GroupingFactors map[string]interface{} `json:"grouping_factors,omitempty"` // Flexible grouping metadata
    // Examples:
    //   "type": "conversation" | "calendar_event" | "location" | "time_period"
    //   "calendar_event_id": "event_123"
    //   "location_id": "loc_456"
    //   "parent_id": "conv_789"  // For hierarchical grouping
    //   "auto_grouped": true     // If automatically grouped vs user-created
    //   "confidence": 0.95       // For auto-grouping confidence scores
}
```

### Derived Fields (Not Stored)

These are calculated at query time:

- **`speaker_count`**: Count of unique `speaker_id` values in segments
- **`speaker_ids`**: List of unique `speaker_id` values (via `terms` aggregation)
- **`type`**: "monolog" if speaker_count == 1, "conversation" if speaker_count > 1
- **`duration`**: `end_time - start_time`
- **`segment_count`**: Count of segments in conversation
- **`recording_ids`**: List of recordings that contain segments in this conversation
  - Derived by matching conversation time range with recording time ranges
  - Recordings have `start_time` and `duration`, so trivial to compute overlap

### Relationship to Segments

**Option A: Separate index with references**
- Store conversations in `conversations` index
- Segments reference conversation via `conversation_id` field
- Pros: Clean separation, easy to query conversations independently
- Cons: Need to maintain consistency, join queries

**Option B: Nested documents**
- Store conversations with nested segment documents
- Pros: Segments travel with conversations, simple queries
- Cons: Duplication if segments need to be queried independently

**Option C: Hybrid (Recommended)**
- Store conversations in `conversations` index
- Add `conversation_id` field to segments (optional, can be NULL)
- Segments can belong to multiple conversations (e.g., calendar event + conversation grouping)
- Pros: Flexible, supports multiple grouping strategies
- Cons: More complex queries

## Grouping Factors Design

### Flexible vs Specific Fields

**Decision: Use flexible `GroupingFactors` map**

**Rationale:**
- New grouping factors will be added over time (calendar, location, topic, etc.)
- Schema changes are expensive (migrations, reindexing)
- Flexible structure allows evolution without breaking changes
- Elasticsearch supports `object` type with dynamic mapping

**Structure:**
```go
GroupingFactors map[string]interface{} `json:"grouping_factors,omitempty"`
```

**Common keys:**
- `"type"`: Primary grouping type ("conversation", "calendar_event", "location", "time_period")
- `"calendar_event_id"`: If grouped by calendar event
- `"location_id"`: If grouped by location
- `"parent_id"`: For hierarchical grouping
- `"auto_grouped"`: Boolean - automatically grouped vs user-created
- `"confidence"`: Float (0.0-1.0) - confidence score for auto-grouping
- `"grouping_algorithm"`: String - which algorithm created this grouping

**Benefits:**
- Extensible: Add new factors without schema changes
- Flexible: Support multiple grouping strategies simultaneously
- Queryable: Elasticsearch can query nested object fields
- Backward compatible: Old conversations without new factors still work

**Example:**
```json
{
  "grouping_factors": {
    "type": "conversation",
    "auto_grouped": true,
    "confidence": 0.95,
    "grouping_algorithm": "silence_threshold_1000ms",
    "calendar_event_id": "event_123",
    "location_id": "loc_456"
  }
}
```

## Questions for Further Investigation

### UTC Boundary Handling

**See bead:** `hai-6qw` (Investigate UTC date boundary handling for conversations)

**Key question:** How do we handle conversations that span UTC date boundaries?

**References:**
- `history/LIFELOG_DEEP_ANALYSIS.md` - Initial investigation showing no UTC boundary spans in sample data
- `history/CONVERSATION_SCHEMA_DESIGN.md` - This document

**Investigation needed:**
- Test with actual data that spans UTC boundaries
- Determine if API splits conversations at UTC boundaries
- Design query patterns for cross-boundary conversations
- Consider storing conversations with UTC timestamps but querying by local time

## Implementation Considerations

### Query Patterns

**Get conversation with speaker count:**
```go
// Query segments for conversation
segments := GetSegmentsByConversation(conversationID)

// Use aggregation to get unique speakers
speakerCount := CardinalityAggregation(segments, "speaker_id")
speakerIDs := TermsAggregation(segments, "speaker_id")

// Derive type
conversationType := "monolog"
if speakerCount > 1 {
    conversationType = "conversation"
}
```

**Filter conversations by type:**
```go
// Get all conversations, then filter by speaker count
conversations := GetAllConversations()
for _, conv := range conversations {
    segments := GetSegmentsByConversation(conv.ID)
    speakerCount := CountUniqueSpeakers(segments)
    if speakerCount == 1 {
        // It's a monolog
    } else {
        // It's a conversation
    }
}
```

### Performance Considerations

- **Aggregations are fast** in Elasticsearch (especially with proper field types)
- **Cardinality aggregation** is approximate but very fast (can be exact with `precision_threshold`)
- **Terms aggregation** is efficient for getting unique values
- **Consider caching** derived values if queries become a bottleneck (premature optimization)

## Next Steps

1. ✅ Terminology: Use "conversation" (consistent with Limitless)
2. ✅ Schema: Don't store derived fields (speaker_count, speaker_list, type)
3. ❓ UTC boundary handling: Investigate and design query patterns
4. ❓ Relationship model: Choose between separate index, nested, or hybrid
5. ❓ Grouping strategies: Design how multiple grouping strategies interact

