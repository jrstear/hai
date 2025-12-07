# Speaker Mapping Query Capabilities

## After Implementation: What You Can Do

Once the speaker name mapping algorithm is implemented and executed, you will have:

### ✅ **Data Available**

1. **Lifelog Blockquotes with Speaker Mappings:**
   - `SpeakerName`: Original name from Limitless API ("You", "Jon Stearley", "Unknown")
   - `SpeakerID`: Mapped global speaker ID (`spkr_xxxxx`) - **populated after mapping**
   - `Content`: Full transcript text
   - `StartTime` / `EndTime`: Absolute UTC timestamps
   - `RecordingID`: Which recording this overlaps with (if matched)

2. **Segments with Speaker IDs:**
   - `SpeakerID`: Global speaker ID (already populated via cosine similarity)
   - `RecordingID`: Which recording
   - `StartTime` / `EndTime`: Relative to recording start

3. **Speakers:**
   - `ID`: Global speaker ID
   - `Embedding`: Voice embedding vector
   - `FirstSeen` / `LastSeen`: When this speaker was detected

### ✅ **Current Query Capabilities**

**Already Available:**
- ✅ `GetLifelogBlockquotesByLifelog(lifelogID)` - All blockquotes for a lifelog
- ✅ `GetLifelogBlockquotesByTimeRange(startTime, endTime)` - Blockquotes in time range
- ✅ `GetSegmentsBySpeakerID(speakerID)` - All segments for a speaker
- ✅ `GetSegmentsByRecording(recordingID)` - All segments for a recording
- ✅ Full-text search on blockquote `content` field (via Elasticsearch)

**Missing (Need to Add):**
- ❌ `GetLifelogBlockquotesBySpeakerID(speakerID)` - All blockquotes for a speaker

### 🔍 **Review Capabilities**

**Yes, you can review:**

1. **All blockquotes for a given speaker ID:**
   - Need to add `GetLifelogBlockquotesBySpeakerID()` method
   - Or query Elasticsearch directly: `speaker_id.keyword = "spkr_xxxxx"`
   - Or use Kibana to query and visualize

2. **Lifelog blockquotes with their mapped speaker IDs:**
   - Query blockquotes by time range
   - Filter by `speaker_id` field
   - See both `SpeakerName` (from Limitless) and `SpeakerID` (our mapping)

3. **Compare Limitless names vs. our speaker IDs:**
   - Query blockquotes grouped by `SpeakerName`
   - See which `SpeakerID` each name maps to
   - Identify inconsistencies or mapping issues

4. **Full-text search on transcripts:**
   - Search blockquote `content` field
   - Filter by `speaker_id` to see what a specific speaker said
   - Combine with time range filters

### 📊 **Example Queries**

**1. All blockquotes for speaker `spkr_abc123`:**
```go
// Need to add this method:
blockquotes, err := storage.GetLifelogBlockquotesBySpeakerID(ctx, "spkr_abc123")
```

**2. All blockquotes where Limitless said "Jon Stearley":**
```go
// Query Elasticsearch directly or via Kibana:
// speaker_name.keyword = "Jon Stearley"
```

**3. All blockquotes for a speaker in a time range:**
```go
// Get by time range, then filter by speaker_id
blockquotes, err := storage.GetLifelogBlockquotesByTimeRange(ctx, startTime, endTime)
// Filter: blockquotes where SpeakerID == "spkr_abc123"
```

**4. Review unmapped blockquotes:**
```go
// Query Elasticsearch: speaker_id is NULL
// Or: NOT speaker_id:*
```

**5. Compare segments vs. blockquotes for same speaker:**
```go
// Get segments
segments, err := storage.GetSegmentsBySpeakerID(ctx, "spkr_abc123")

// Get blockquotes
blockquotes, err := storage.GetLifelogBlockquotesBySpeakerID(ctx, "spkr_abc123")

// Compare time ranges, transcripts, etc.
```

### 🎯 **What You Can Review**

1. **Mapping Accuracy:**
   - See which Limitless speaker names map to which global speaker IDs
   - Identify cases where "Jon Stearley" maps to multiple speaker IDs (potential issue)
   - Identify cases where multiple names map to same speaker ID (expected)

2. **Unmapped Blockquotes:**
   - Find blockquotes where `SpeakerID` is NULL
   - Review why they weren't matched (no overlap? below threshold?)

3. **Transcript Content:**
   - Read full transcripts for each speaker
   - Search transcripts for specific topics
   - Compare what Limitless captured vs. what we diarized

4. **Time Alignment:**
   - Compare blockquote timestamps with segment timestamps
   - Verify overlap calculations
   - Identify timing discrepancies

5. **Speaker Distribution:**
   - See how many blockquotes each speaker has
   - See time distribution (when each speaker appears)
   - Identify most frequent speakers

### 🛠️ **Implementation Needed**

**Add to Storage Interface:**
```go
// GetLifelogBlockquotesBySpeakerID retrieves all blockquotes for a given speaker ID
// Results are sorted by start_time ascending
GetLifelogBlockquotesBySpeakerID(ctx context.Context, speakerID string) ([]*LifelogBlockquote, error)
```

**Elasticsearch Implementation:**
```go
func (s *ElasticsearchStorage) GetLifelogBlockquotesBySpeakerID(ctx context.Context, speakerID string) ([]*LifelogBlockquote, error) {
    query := map[string]interface{}{
        "query": map[string]interface{}{
            "term": map[string]interface{}{
                "speaker_id.keyword": speakerID,
            },
        },
        "size": 10000,
        "sort": []map[string]interface{}{
            {"start_time": map[string]interface{}{"order": "asc"}},
        },
    }
    return s.searchLifelogBlockquotes(ctx, query)
}
```

### 📈 **Kibana Visualization**

With Kibana (hai-c17), you can:

1. **Create Dashboards:**
   - Table: All blockquotes with `SpeakerName`, `SpeakerID`, `Content`, `StartTime`
   - Bar chart: Blockquotes per speaker ID
   - Timeline: Blockquotes over time, colored by speaker ID
   - Pie chart: Distribution of speaker names

2. **Interactive Queries:**
   - Filter by `speaker_id`
   - Filter by `speaker_name`
   - Filter by time range
   - Full-text search on `content`

3. **Manual Correction:**
   - View blockquotes with their mapped speaker IDs
   - Identify incorrect mappings
   - Update `SpeakerID` field directly (via API or UI)

4. **Data Discovery:**
   - Explore relationships between speakers
   - Find conversations between specific speakers
   - Analyze speaker patterns over time

### ✅ **Summary**

**After mapping is implemented, you will be able to:**

1. ✅ Review all lifelog blockquotes with their mapped speaker IDs
2. ✅ Search for all blockquotes for a given speaker ID (need to add method)
3. ✅ Compare Limitless speaker names with our global speaker IDs
4. ✅ Review unmapped blockquotes
5. ✅ Full-text search on transcripts
6. ✅ Visualize and analyze in Kibana

**The data will all be in place for review!** You just need:
- The mapping algorithm implemented and run
- The `GetLifelogBlockquotesBySpeakerID()` method added (or use Kibana/Elasticsearch directly)













