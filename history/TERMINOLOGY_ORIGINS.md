# Terminology Origins Investigation

## Overview

Investigation into the origins of key terminology used in the codebase:
- **"segment"** - Used in our schema and diarization code
- **"blockquote"** - Used by Limitless API for transcribed speech segments

## "Segment" Terminology

### Origin: pyannote.audio Library

**Evidence from codebase:**
- `cmd/diarize/diarize.py` uses `pyannote.audio` library
- Code iterates over diarization results: `for segment, _, label in annotation.itertracks(yield_label=True)`
- The `segment` object has `.start` and `.end` attributes (time boundaries)
- This is the standard terminology from pyannote.audio

**pyannote.audio Usage:**
```python
from pyannote.audio import Pipeline
annotation = diarization.speaker_diarization
for segment, _, label in annotation.itertracks(yield_label=True):
    # segment.start, segment.end are time boundaries
    # label is the speaker ID
```

**Why "segment"?**
- **Standard term in speaker diarization**: "Segment" is the common term in speech processing and diarization literature
- **Time-based division**: A segment represents a contiguous time period
- **Industry standard**: Used by pyannote.audio, which is a widely-used diarization library
- **Clear meaning**: A segment is a portion/division of the audio timeline

**Conclusion:** "Segment" is inherited from pyannote.audio and is standard terminology in speaker diarization. It's appropriate to continue using this term.

## "Blockquote" Terminology

### Origin: Limitless API

**Evidence from lifelog data:**
- Limitless API returns lifelog content with `type: "blockquote"` for transcribed speech
- Blockquotes contain: `content` (transcript text), `speakerName`, timestamps
- The lifelog also has a `markdown` field that formats blockquotes as markdown blockquotes

**Why "blockquote"?**
- **Markdown/HTML terminology**: "Blockquote" is a markdown/HTML element for quoted text
- **Visual representation**: In the lifelog markdown, transcribed speech appears as blockquotes:
  ```markdown
  - Ruth Stearley (11/20/25 9:20 PM): you can't it's not appropriate...
  - Unknown (11/20/25 9:20 PM): I I mean, I can take my my ski gloves.
  ```
- **Semantic meaning**: Blockquotes represent quoted speech (what was said)
- **UI/Display context**: The term reflects how the content is displayed in the Limitless UI

**Alternative theories:**
- Could be because the API returns markdown-formatted content
- Could be a UI/display convention (speech appears as quoted text)
- Could be a historical naming choice from early development

**Conclusion:** "Blockquote" is Limitless-specific terminology, likely chosen because transcribed speech is displayed as markdown blockquotes in their UI. It's not a standard term in speech processing.

## Terminology Comparison

| Term | Source | Domain | Meaning |
|------|--------|--------|---------|
| **Segment** | pyannote.audio | Speech processing / Diarization | Time period during which a single speaker speaks |
| **Blockquote** | Limitless API | UI/Display / Markdown | Transcribed speech segment (displayed as quoted text) |

**Key insight:** Both terms refer to the same concept (a time period with a speaker), but from different perspectives:
- **Segment**: Technical/processing perspective (audio analysis)
- **Blockquote**: UI/display perspective (how it's shown to users)

## Our Usage

### Current Practice

**We use "segment":**
- In our schema: `Segment` struct
- In our code: `segments` array in diarization results
- In our storage: `segments` index in Elasticsearch

**We reference "blockquote":**
- In documentation/comments: "Equivalent to Limitless API's 'blockquote' type"
- When comparing with Limitless data

### Recommendation

**Continue using "segment":**
- ✅ Standard terminology in speech processing
- ✅ Inherited from pyannote.audio (our diarization tool)
- ✅ Technically accurate
- ✅ Consistent with our codebase

**When discussing Limitless data:**
- Use "blockquote" when referring specifically to Limitless API data
- Use "segment" when referring to our own data
- Note the equivalence in documentation

## Conversation Definition

**Updated definition:**
> A **conversation** is a contiguous series of segments/blockquotes by one or more distinct speakers.

**Key aspects:**
- **Contiguous**: Segments are adjacent in time (may have small gaps, but no large breaks)
- **Series**: Multiple segments grouped together
- **One or more speakers**: Can be monolog (1 speaker) or conversation (2+ speakers)
- **Time-based**: Defined by temporal continuity

This definition aligns with:
- How Limitless groups blockquotes into conversations
- How we'll group segments using silence thresholds and speaker changes
- Standard understanding of what constitutes a conversation

## References

- pyannote.audio documentation: Uses "segment" terminology
- Limitless API: Uses "blockquote" for transcribed speech
- Our codebase: Inherits "segment" from pyannote.audio
- Speech processing literature: "Segment" is standard term

