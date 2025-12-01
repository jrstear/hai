# Lifelog-Based Audio Filtering: Investigation & Recommendation

## Problem Statement

Currently, the onboarding process requests audio for **all hours** in the time range, even if some hours don't have any audio recordings. This wastes API calls and hits rate limits unnecessarily.

**Goal**: Use lifelog data to determine which hours actually have audio, then only request those hours from the Limitless API.

## Current Behavior

1. Generate all hours in time range (e.g., 2pm-6pm = 4 hours)
2. Fetch lifelog for each date
3. **Send ALL hours to audio channel** (regardless of whether they have audio)
4. Try to fetch audio for every hour
5. If API returns 404 or no audio, mark as failed

## Investigation Results

### 1. Null Lifelog Case

**Example**: `data/2025/11/16/lifelog.json` contains just `null`

- **Meaning**: API returned `null` (no lifelogs for that day)
- **Implication**: No hours have audio for that day
- **Action**: Mark all hours for that date as N/A

### 2. Lifelog Structure

**Example**: `data/2025/11/20/lifelog.json` contains an array of lifelog objects:

```json
[
  {
    "id": "wwimzBqooRRtmt2XIKWf",
    "title": "Ruth Stearley and Unknown discuss various topics",
    "startTime": "2025-11-20T21:20:12-07:00",
    "endTime": "2025-11-20T21:25:25-07:00",
    "contents": [
      {
        "type": "blockquote",
        "content": "you can't it's not appropriate...",
        "startTime": "2025-11-20T21:20:12-07:00",
        "endTime": "2025-11-20T21:20:16-07:00",
        "startOffsetMs": 0,
        "endOffsetMs": 4000,
        "speakerName": "Ruth Stearley"
      },
      // ... more blockquotes
    ]
  }
]
```

**Key Observations**:
- Lifelogs have `startTime` and `endTime` (full timestamps with timezone)
- `contents` array contains items with `type: "blockquote"` for audio segments
- Blockquotes have `startTime`/`endTime` OR `startOffsetMs`/`endOffsetMs`
- A lifelog can span multiple hours (e.g., 9:20 PM - 9:25 PM)
- Not all hours in a day may have blockquotes

### 3. Determining Hours with Audio

**Logic**:
1. Parse lifelog JSON (handle `null` case)
2. For each lifelog in the array:
   - Check if it has any `contents` with `type: "blockquote"`
   - If yes, extract the hour(s) from `startTime` and `endTime`
3. Return a set of hours (in UTC) that have audio

**Edge Cases**:
- Null lifelog → no hours have audio
- Empty array `[]` → no hours have audio
- Lifelog with no blockquotes → no hours have audio
- Lifelog spanning multiple hours → mark all hours as having audio
- Partial hour coverage → mark the hour as having audio if any blockquote exists

## Recommended Implementation

### 1. Add New Status Type

**File**: `onboard/internal/server/types.go`

```go
const (
    StageStatusPending     StageStatus = "pending"
    StageStatusRunning     StageStatus = "running"
    StageStatusDone        StageStatus = "done"
    StageStatusFailed      StageStatus = "failed"
    StageStatusSkipped     StageStatus = "skipped"  // Already exists
    StageStatusNotAvailable StageStatus = "not_available"  // NEW: No audio for this hour
)
```

### 2. Create Lifelog Parser Function

**File**: `onboard/internal/fetch/lifelog.go` (new function)

```go
// ParseLifelogForAudioHours parses a lifelog JSON file and returns the set of hours (in UTC) that have audio
// Returns:
//   - hoursWithAudio: map[string]bool where key is "YYYY-MM-DD HH:00" format (UTC)
//   - hasAudio: true if any hours have audio, false if lifelog is null/empty/no blockquotes
//   - error: if file cannot be read or parsed
func ParseLifelogForAudioHours(lifelogPath string, timezone string) (map[string]bool, bool, error) {
    // Read file
    data, err := os.ReadFile(lifelogPath)
    if err != nil {
        return nil, false, err
    }
    
    // Handle null case
    var lifelogs []Lifelog
    if string(data) == "null\n" || string(data) == "null" {
        return make(map[string]bool), false, nil
    }
    
    // Parse JSON
    if err := json.Unmarshal(data, &lifelogs); err != nil {
        return nil, false, fmt.Errorf("failed to parse lifelog: %w", err)
    }
    
    // Handle empty array
    if len(lifelogs) == 0 {
        return make(map[string]bool), false, nil
    }
    
    // Load timezone for conversion
    loc, err := time.LoadLocation(timezone)
    if err != nil {
        return nil, false, fmt.Errorf("invalid timezone: %w", err)
    }
    
    hoursWithAudio := make(map[string]bool)
    hasAnyAudio := false
    
    // Process each lifelog
    for _, lifelog := range lifelogs {
        // Check if lifelog has any blockquotes
        hasBlockquotes := false
        for _, content := range lifelog.Contents {
            if content.Type == "blockquote" {
                hasBlockquotes = true
                break
            }
        }
        
        if !hasBlockquotes {
            continue // Skip lifelogs without audio
        }
        
        hasAnyAudio = true
        
        // Parse start and end times
        // Lifelog times are in the timezone specified (from API)
        startTime, err := time.Parse(time.RFC3339, lifelog.StartTime.Format(time.RFC3339))
        if err != nil {
            // Try parsing as-is (might already be in correct format)
            startTime = lifelog.StartTime
        }
        endTime, err := time.Parse(time.RFC3339, lifelog.EndTime.Format(time.RFC3339))
        if err != nil {
            endTime = lifelog.EndTime
        }
        
        // Convert to UTC for hour calculation
        startUTC := startTime.UTC()
        endUTC := endTime.UTC()
        
        // Generate all hours between start and end (inclusive)
        current := time.Date(
            startUTC.Year(), startUTC.Month(), startUTC.Day(),
            startUTC.Hour(), 0, 0, 0,
            time.UTC,
        )
        endHour := time.Date(
            endUTC.Year(), endUTC.Month(), endUTC.Day(),
            endUTC.Hour(), 0, 0, 0,
            time.UTC,
        )
        
        // Add all hours from start to end (inclusive)
        for current.Before(endHour) || current.Equal(endHour) {
            hourKey := current.Format("2006-01-02 15:00")
            hoursWithAudio[hourKey] = true
            current = current.Add(1 * time.Hour)
        }
    }
    
    return hoursWithAudio, hasAnyAudio, nil
}
```

### 3. Modify Processing Flow

**File**: `onboard/internal/server/handlers.go`

**In the lifelog fetcher goroutine** (after fetching lifelog):

```go
// After successful lifelog fetch:
lifelogPath := filepath.Join(s.outputDir, utcDateStr, "lifelog.json")
hoursWithAudio, hasAudio, err := fetch.ParseLifelogForAudioHours(lifelogPath, job.Timezone)
if err != nil {
    log.Printf("Warning: Failed to parse lifelog for audio hours: %v", err)
    // Fall back to processing all hours (current behavior)
    hoursWithAudio = nil
}

// Mark hours as N/A if they don't have audio
// Note: Set progress to 100% for N/A hours so they count as "done" in progress calculation
// This keeps the progress bar proportional to the full time range
for hourKey, hp := range job.HourProgress {
    // Check if this hour is in the date we just processed
    hourUTC, err := time.Parse("2006-01-02 15:00", hourKey)
    if err != nil {
        continue
    }
    hourInTZ := hourUTC.In(loc)
    if hourInTZ.Format("2006-01-02") == dateInTZ {
        // Check if this hour has audio
        if hoursWithAudio != nil {
            if !hoursWithAudio[hourKey] {
                // No audio for this hour - mark as N/A but count as done (100%) for progress
                s.updateHourStage(job, hourKey, "audio", StageStatusNotAvailable, 100, "")
                s.updateHourStage(job, hourKey, "diarize", StageStatusNotAvailable, 100, "")
                s.updateHourStage(job, hourKey, "elasticsearch", StageStatusNotAvailable, 100, "")
                // Lifelog is already done (we fetched it to determine this)
                continue // Skip sending to audio channel
            }
        }
        // Hour has audio - send to audio channel (existing behavior)
        audioChan <- hour
    }
}
```

### 4. Update UI to Handle N/A Status

**File**: `onboard/templates/index.html`

**In `renderStatusIcon` function**:

```javascript
function renderStatusIcon(status) {
    if (status === 'done') {
        return '<span class="status-icon done">✓</span>';
    } else if (status === 'running') {
        return '<span class="spinner"></span>';
    } else if (status === 'failed') {
        return '<span class="status-icon failed">✗</span>';
    } else if (status === 'skipped') {
        return '<span class="status-icon skipped">✓</span>'; // Darker green
    } else if (status === 'not_available') {
        return '<span class="status-icon not-available">—</span>'; // Dash or N/A
    }
    return ''; // Pending
}
```

**Add CSS**:

```css
.status-icon.not-available {
    color: #999; /* Gray */
    font-weight: normal;
}
```

### 5. Update Progress Calculation

**File**: `onboard/internal/server/handlers.go`

**In `updateHourStage` function**, count N/A hours as "done" (100%) for progress calculation:

**Key Insight**: The progress bar is proportional to the number of hours in the time range. If some hours have no audio, we should count them as "done" (100% progress) for all stages, since we've determined they don't need processing. This keeps the progress bar representing the full time range while the UI still shows N/A status.

**Implementation**: When marking hours as N/A, set all stage progress values to 100:

```go
// When marking hours as N/A (in lifelog fetcher goroutine):
if !hoursWithAudio[hourKey] {
    // No audio for this hour - mark as N/A but count as done for progress
    s.updateHourStage(job, hourKey, "audio", StageStatusNotAvailable, 100, "")
    s.updateHourStage(job, hourKey, "diarize", StageStatusNotAvailable, 100, "")
    s.updateHourStage(job, hourKey, "elasticsearch", StageStatusNotAvailable, 100, "")
    // Lifelog is already done (we fetched it to determine this)
    continue // Skip sending to audio channel
}
```

**No changes needed to `updateHourStage` progress calculation** - it already sums all progress values and divides by count. N/A hours with 100% progress will contribute to the overall progress just like completed hours, which is the desired behavior.

## Benefits

1. **Reduced API Calls**: Only request audio for hours that actually have recordings
2. **Faster Processing**: Skip unnecessary API requests and processing
3. **Better UX**: Clear indication of which hours have no audio (N/A status)
4. **Rate Limit Mitigation**: Fewer API calls = less chance of hitting rate limits

## Edge Cases Handled

- ✅ Null lifelog → All hours marked N/A
- ✅ Empty lifelog array → All hours marked N/A
- ✅ Lifelog with no blockquotes → All hours marked N/A
- ✅ Lifelog spanning multiple hours → All hours marked as having audio
- ✅ Partial hour coverage → Hour marked as having audio if any blockquote exists
- ✅ Parsing errors → Fall back to current behavior (process all hours)

## Testing Recommendations

1. Test with null lifelog file
2. Test with empty array lifelog
3. Test with lifelog spanning multiple hours
4. Test with lifelog that has no blockquotes
5. Test with lifelog that only covers part of an hour
6. Test with multiple lifelogs on the same day
7. Test timezone edge cases (UTC day boundaries)

## Implementation Order

1. Add `StageStatusNotAvailable` constant
2. Create `ParseLifelogForAudioHours` function
3. Modify lifelog fetcher goroutine to parse and filter hours
4. Update UI to display N/A status
5. Update progress calculation to exclude N/A hours
6. Test with various lifelog scenarios

