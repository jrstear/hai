# Test Case: Multi-Speaker Conversation Segment

## Overview
This test case focuses on a 60-second segment (420-480 seconds) from the 3pm hour audio file (`15.ogg`) that exhibits high speaker activity with all 5 detected speakers.

## Audio File Details
- **File**: `data/audio/2025/11/22/15.ogg`
- **Duration**: 3318 seconds (55.3 minutes)
- **Time Range**: 15:00:00 - 15:55:18 MST (Nov 22, 2025)
- **Diarization Results**: `data/audio/2025/11/22/15.json`

## Test Segment
- **Time Window**: 420-480 seconds (7:00 - 8:00 minutes into the hour)
- **Local Time**: ~15:07:00 - 15:08:00 MST
- **UTC Time**: ~22:07:00 - 22:08:00 UTC

## Diarization Characteristics

### Speaker Activity
- **All 5 speakers active**: SPEAKER_00, SPEAKER_01, SPEAKER_02, SPEAKER_03, SPEAKER_04
- **43 speaker changes** in 60 seconds (very high activity)
- **Average segment duration**: ~1.4 seconds per segment

### Speaker Distribution (Overall File)
- SPEAKER_02: 34.5% (1146s) - Primary speaker
- SPEAKER_04: 18.2% (604s)
- SPEAKER_01: 11.2% (372s)
- SPEAKER_03: 9.6% (320s)
- SPEAKER_00: 1.8% (60s) - Rare speaker

## Use Cases

### 1. Speaker Identification Testing
Test speaker identification accuracy with:
- High speaker turn-taking rate
- Multiple simultaneous speakers potentially
- All available speakers in one segment

### 2. Segmentation Quality
Evaluate:
- Segment boundary accuracy
- Overlap detection
- Speaker transition handling

### 3. Comparison with Limitless API
Compare local diarization results with Limitless API lifelog data to:
- Identify speaker name mappings (SPEAKER_XX → "You", "Unknown", etc.)
- Validate segment boundaries
- Assess speaker count accuracy

### 4. Performance Benchmarking
- Processing time for dense conversation
- Accuracy metrics for multi-speaker scenarios
- Edge case handling

## Accessing the Test Segment

### From Diarization Results
```python
import json

with open('data/audio/2025/11/22/15.json') as f:
    results = json.load(f)

# Filter segments in test window
test_segments = [s for s in results['segments'] 
                 if 420 <= s['start'] < 480]
```

### Extract Audio Segment
```bash
ffmpeg -i data/audio/2025/11/22/15.ogg \
  -ss 420 -t 60 \
  test_case_420_480s.ogg
```

## Related Lifelogs
Check `data/lifelogs_2025-11-22.json` for transcripts covering:
- Time: ~15:07 MST
- Likely topics: Various conversations between "You" and "Unknown"

## Next Steps
1. Create automated test that validates speaker detection in this segment
2. Extract and save this segment as a separate test file
3. Create speaker mapping (SPEAKER_XX → names) based on lifelog comparison
4. Use for regression testing when improving diarization pipeline

