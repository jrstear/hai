# Blockquote Boundary Analysis

## Overview

Analysis of blockquote timing data to infer how Limitless determines blockquote boundaries.

## Key Findings

### 1. Blockquotes Are Mostly Sequential

- **192 zero-gap transitions** (back-to-back blockquotes)
- **4 negative gaps** (overlaps) - likely timing approximation/rounding errors
- **99.75% of blockquotes are sequential** (no overlaps)

**Conclusion:** Blockquotes are designed to be non-overlapping segments.

### 2. Gap Distribution Patterns

From analysis of 1,626 gaps across multiple lifelogs:

**Most common gap ranges (same-speaker blockquotes):**
- **1000-1049 ms: 27.5%** (211 gaps) - **PRIMARY THRESHOLD**
- **0-49 ms: 18.1%** (139 gaps) - back-to-back (likely speaker changes)
- **2000-2049 ms: 16.8%** (129 gaps) - **SECONDARY THRESHOLD**
- **3000-3049 ms: 6.0%** (46 gaps)
- **4000-4049 ms: 5.5%** (42 gaps)

**Gap statistics (same-speaker):**
- Min: -4000 ms (overlap, likely error)
- Max: 427,000 ms (71 minutes - very long pause)
- Mean: 7,977 ms (8 seconds)
- **Median: 2,000 ms (2 seconds)**

### 3. Speaker Change vs Same Speaker

**Gaps WITH speaker change:**
- Count: 214 gaps
- Min: -1000 ms (overlap)
- Max: 101,000 ms (101 seconds)
- Mean: 4,140 ms (4.1 seconds)
- **Median: 1,000 ms (1 second)**

**Gaps WITHOUT speaker change (same speaker):**
- Count: 1,412 gaps
- Min: -4000 ms (overlap)
- Max: 427,000 ms (71 minutes)
- Mean: 7,977 ms (8 seconds)
- **Median: 2,000 ms (2 seconds)**

**Key insight:** 
- Speaker changes can happen with **0 ms gap** (immediate turn-taking)
- Same-speaker blockquotes typically have **1-2 second gaps**
- Longer gaps (5+ seconds) are more common for same-speaker pauses

### 4. Inferred Boundary Logic

Based on the data, blockquote boundaries appear to be determined by:

1. **Speaker Change (Primary)**
   - New blockquote starts when speaker changes
   - Can happen with 0 ms gap (immediate turn-taking)
   - Median gap: 1 second (natural pause between speakers)

2. **Silence Threshold (Secondary)**
   - New blockquote starts when silence exceeds threshold
   - **Primary threshold: ~1000 ms (1 second)** - most common gap
   - **Secondary threshold: ~2000 ms (2 seconds)** - longer pauses
   - Applies to same-speaker segments

3. **Combined Logic**
   - If speaker changes → new blockquote (regardless of silence)
   - If same speaker + silence > ~1000-2000 ms → new blockquote
   - If same speaker + silence < ~1000 ms → continue same blockquote

### 5. Common Speech Recognition Thresholds

These thresholds align with standard VAD (Voice Activity Detection) settings:
- **200-300 ms**: Natural pause between words/phrases (too short for segmentation)
- **500-1000 ms**: Pause between sentences (borderline)
- **1000-2000 ms**: Pause between topics/turns (segmentation threshold)
- **2000+ ms**: Significant silence (definite segmentation)

**Limitless appears to use:**
- **~1000 ms** as primary silence threshold (27.5% of gaps)
- **~2000 ms** as secondary threshold (16.8% of gaps)
- **Speaker change** as immediate trigger (can be 0 ms)

## Implications for Our Grouping Logic

When implementing our own segment grouping (for monolog/conversation detection):

1. **Use similar thresholds:**
   - 1000-2000 ms silence threshold for same-speaker segmentation
   - Speaker change = immediate new segment

2. **Handle edge cases:**
   - Very short gaps (< 500 ms) = likely same segment
   - Medium gaps (500-2000 ms) = context-dependent (same topic vs new topic)
   - Long gaps (2000+ ms) = likely new segment

3. **Consider speaker continuity:**
   - Same speaker + short gap = likely same monolog segment
   - Speaker change = new segment (conversation turn or new monolog)

4. **Grouping strategy:**
   - Group consecutive segments by speaker (monolog)
   - Group alternating segments by speakers (conversation)
   - Use silence gaps to determine if segments should be grouped or separated

## Questions for Further Investigation

1. **Are the 4 overlaps real or timing errors?** (likely rounding/approximation)
2. **What determines the 1000 ms vs 2000 ms threshold?** (context, topic change?)
3. **How does Limitless handle overlapping speech?** (does it create separate blockquotes?)
4. **Can we replicate this logic in our diarization grouping?**

## Recommendations

1. **Adopt similar thresholds** for our grouping logic:
   - 1000 ms as primary silence threshold
   - 2000 ms as secondary threshold
   - Speaker change as immediate segment boundary

2. **Test with our diarization data** to see if similar patterns emerge

3. **Consider configurable thresholds** for different use cases (e.g., tighter grouping for conversations, looser for monologs)












