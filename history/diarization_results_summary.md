# Diarization Results Summary

## Audio File: `15.ogg` (3pm hour, Nov 22, 2025)

**Location**: `data/audio/2025/11/22/15.ogg`  
**Results**: `data/audio/2025/11/22/15.json` (206KB)

### Performance
- **Duration**: 3,318 seconds (55.3 minutes)
- **Processing Time**: 219.3 seconds (3.7 minutes)
- **RTF**: 0.066x (**15.1x faster than real-time**)
- **Device**: MPS (Metal Performance Shaders) acceleration on M1 Mac

### Speakers Detected
5 speakers identified:
- SPEAKER_00, SPEAKER_01, SPEAKER_02, SPEAKER_03, SPEAKER_04

### Speaker Statistics

| Speaker | Duration | Percentage | Segments | Notes |
|---------|----------|------------|----------|-------|
| SPEAKER_02 | 1,146.0s | 34.5% | 471 | Primary speaker |
| SPEAKER_04 | 604.1s | 18.2% | 374 | Secondary speaker |
| SPEAKER_01 | 372.4s | 11.2% | 337 | Active participant |
| SPEAKER_03 | 320.0s | 9.6% | 184 | Active participant |
| SPEAKER_00 | 60.1s | 1.8% | 96 | Rare speaker |

**Total Segments**: 1,462

## Recommended Test Case: 420-480s Window

### Why This Segment?
- **Highest speaker activity**: All 5 speakers active
- **43 speaker changes** in 60 seconds
- **Dense conversation**: Rapid turn-taking
- **Good for testing**: Speaker identification, segmentation accuracy, overlap detection

### Segment Details
- **Time Window**: 420-480 seconds (7:00-8:00 minutes into audio)
- **Local Time**: ~15:07:00-15:08:00 MST
- **Segments**: 51 segments in 60 seconds
- **Speakers Present**: All 5 (SPEAKER_00, 01, 02, 03, 04)

### Example Segments
```
[420.7s-424.5s] SPEAKER_02 (3.78s)
[424.8s-425.2s] SPEAKER_00 (0.41s)  ← Rare speaker
[425.5s-427.8s] SPEAKER_02 (2.30s)
[440.0s-440.1s] SPEAKER_04 (0.03s)  ← Very short segment
[445.3s-450.0s] SPEAKER_04 (4.74s)
[447.6s-447.6s] SPEAKER_03 (0.03s)  ← Overlap?
[448.0s-449.7s] SPEAKER_03 (1.75s)
```

### Use Cases for This Test Segment
1. **Speaker Identification Testing**: Validate accuracy with all speakers
2. **Segmentation Quality**: Test boundary detection and overlap handling
3. **Comparison with Limitless API**: Match SPEAKER_XX IDs to named speakers
4. **Performance Benchmarking**: Measure accuracy in dense conversation scenarios
5. **Edge Case Handling**: Very short segments, rapid transitions, overlaps

## Related Multi-Speaker Lifelogs

From `data/lifelogs_2025-11-22.json`, the 3pm hour contains:

1. **14:43-15:02**: "Jon Stearley's new recording device and AI discussion" (2 speakers, 1091s)
2. **15:02-15:09**: "Conversation about food, exercise, and AI coaching" (2 speakers, 438s)
3. **15:09-15:15**: "Jon Stearley discusses AI-powered audio modification" (2 speakers, 359s)
4. **15:15-15:22**: "Jon Stearley and Unknown discuss a variety of topics" (2 speakers, 449s)
5. **15:36-15:46**: "Discussion about audiobooks" (2 speakers, 583s)
6. **15:46-16:01**: "Discussion about biographies, AI, and personal memoirs" (2 speakers, 925s)

**Note**: Lifelogs show 2 speakers ("You" and "Unknown"), but diarization detected 5 speakers. This suggests:
- Multiple "Unknown" speakers might be different people
- Background speakers or cross-talk
- Different speakers in the same conversation identified by diarization but not labeled in lifelogs

## Next Steps

1. ✅ Diarization results cached (`15.json`)
2. ✅ Test case identified (420-480s window)
3. 🔲 Extract audio segment as separate test file
4. 🔲 Create speaker mapping (SPEAKER_XX → names) based on lifelog comparison
5. 🔲 Build automated tests using this segment
6. 🔲 Compare with Limitless API speaker identification

## Files Created
- `data/audio/2025/11/22/15.json` - Full diarization results
- `cmd/diarize/test_case_420_480s.md` - Detailed test case documentation
- This summary document

