# Byte Offset Calculation: Efficiency Opportunity Check

## Question
Can pyannote diarization output provide byte offsets directly, avoiding need for separate ffprobe call?

## Analysis

### What pyannote outputs:
From code inspection (`cmd/diarize/diarize.py`):
- `diarization.speaker_diarization` → `Annotation` object
- Segments via `annotation.itertracks(yield_label=True)`:
  - `segment.start` → float (seconds)
  - `segment.end` → float (seconds)
  - `label` → speaker ID (string)

**No byte offset information** in pyannote output.

### Why pyannote doesn't provide byte offsets:
- Works with **decoded audio waveforms** (samples), not raw file bytes
- Audio is loaded via `soundfile` or converted, losing byte-level mapping
- Focus is on speaker separation, not file format details
- Byte offsets are file-format specific (OGG, MP3, etc.)

### Efficiency consideration:
**Current approach**:
1. Run diarization → get timestamps
2. Run ffprobe → get timestamp → byte_offset mapping
3. Match timestamps to byte offsets

**Could we skip ffprobe?**
- ❌ No - pyannote doesn't know byte positions
- ✅ But we could combine into single pass:
  - Parse OGG file structure once
  - Build timestamp → byte_offset map
  - Use for all segments

### Optimization Opportunity:
**Instead of**: ffprobe per segment lookup
**Do**: Parse OGG once, build index, reuse for all segments

Example:
```python
# Build index once
timestamp_to_byte = parse_ogg_index(audio_file)  # {timestamp: byte_offset}

# Use for all segments
for segment in segments:
    start_byte = find_nearest_byte(timestamp_to_byte, segment.start)
    end_byte = find_nearest_byte(timestamp_to_byte, segment.end)
```

**But**: ffprobe already does this efficiently and is standard tool.
- Single command extracts all frames with timestamps + byte positions
- No need to manually parse OGG format
- Already tested and working

## Conclusion

**Answer**: Pyannote **cannot** provide byte offsets directly. It only outputs timestamps.

**Efficiency note**: 
- ffprobe is already efficient (single pass, extracts all frames)
- No significant efficiency gain from trying to get byte offsets from diarization
- Current approach (diarization → timestamps, then ffprobe → byte offsets) is optimal

## Implementation Decision

✅ **Keep current approach**:
1. Diarization produces timestamps (seconds)
2. Run ffprobe to get timestamp → byte_offset mapping
3. Match timestamps to nearest byte offsets
4. Store in segments table

**Error handling**: If ffprobe fails, store NULL for byte offsets, continue processing (don't halt/error).

