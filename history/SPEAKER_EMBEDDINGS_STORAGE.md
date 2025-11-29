# Speaker Embeddings Storage Organization

## Current Implementation

### File Structure

```
data/audio/
└── 2025/
    └── 11/
        └── 22/
            ├── 15.ogg        # Audio file
            ├── 15.json       # Diarization results (includes embeddings)
            ├── 16.ogg
            ├── 16.json       # (when diarized)
            └── ...
```

**Pattern**: `<audio_basename>.json` stored alongside `<audio_basename>.ogg` in the same directory.

### Storage Format

Embeddings are stored **inline** within the diarization results JSON file:

```json
{
  "audio_file": "/path/to/15.ogg",
  "timestamp": "2025-11-28 17:47:30",
  "audio_duration": 3318.0735,
  "processing_time": 230.77,
  "rtf": 0.070,
  "device": "mps",
  "speaker_count": 5,
  "speakers": ["SPEAKER_00", "SPEAKER_01", "SPEAKER_02", "SPEAKER_03", "SPEAKER_04"],
  "segment_count": 1462,
  "segments": [
    {"speaker": "SPEAKER_02", "start": 0.031, "end": 3.052, "duration": 3.021},
    ...
  ],
  "speaker_embeddings": {
    "SPEAKER_00": [-0.041, -0.121, 0.022, ..., 256 floats total],
    "SPEAKER_01": [0.145, -0.089, 0.203, ..., 256 floats total],
    "SPEAKER_02": [0.092, 0.156, -0.078, ..., 256 floats total],
    "SPEAKER_03": [-0.134, 0.067, 0.145, ..., 256 floats total],
    "SPEAKER_04": [0.078, -0.045, 0.189, ..., 256 floats total]
  }
}
```

### Embedding Structure

- **Format**: Dictionary/object in JSON
- **Keys**: Speaker IDs (`SPEAKER_00`, `SPEAKER_01`, etc.)
- **Values**: Lists of 256 floating-point numbers
- **Data Type**: `float32` (converted to Python `float` for JSON)
- **Storage Size**: ~5KB per file (5 speakers × 256 floats × 4 bytes = 5,120 bytes)

### Characteristics

#### ✅ Current Strengths

1. **Co-location**: Embeddings stored with audio file results
   - Easy to find embeddings for a specific recording
   - Single file contains all information about a recording

2. **Per-file Organization**: Each recording has independent embeddings
   - No dependencies between files
   - Can process files independently

3. **Speaker-scoped**: Embeddings keyed by speaker ID within file
   - Clear mapping from speaker ID to embedding vector
   - Easy to look up embedding for a specific speaker in a recording

4. **Human-readable**: JSON format is easy to inspect and debug
   - Can view embeddings directly in text editor
   - Easy to parse in any language

#### ❌ Current Limitations

1. **No Cross-File Matching**: Each file has independent speaker IDs
   - `SPEAKER_02` in `15.json` ≠ `SPEAKER_02` in `16.json`
   - No way to know if same person appears in multiple files

2. **No Global Speaker Library**: No aggregation across recordings
   - Can't build a database of "known speakers"
   - Can't query "where does this speaker appear?"

3. **No Deduplication**: Same person across files = different IDs
   - If "Jon" appears in both 15.ogg and 16.ogg, he'll have different speaker IDs
   - Requires manual matching or similarity search

4. **Redundant Storage**: Embeddings duplicated in JSON
   - Full embeddings stored even though segments already exist
   - Could be optimized (store once, reference from segments)

5. **No Contact Association**: No mapping to contacts/names
   - Embeddings exist but aren't linked to "Jon", "Ruth", etc.
   - No persistent identity system

## Storage Size Analysis

### Per Recording
- **5 speakers** × **256 floats** × **4 bytes** = **5,120 bytes** (~5 KB)
- Plus metadata, segments (~235 KB for 55-minute recording)
- **Total**: ~240 KB per hour of audio results

### Scaling Estimate
- **1 day** (24 hours): ~5.8 MB embeddings
- **1 week** (168 hours): ~41 MB embeddings  
- **1 month** (720 hours): ~176 MB embeddings
- **1 year** (8,760 hours): ~2.1 GB embeddings

*Note: These are embeddings only. Full JSON files with segments are larger.*

## Future Organization Options

### Option 1: Global Speaker Library (Recommended)

Create a separate database/library of speakers:

```
data/
├── audio/
│   └── 2025/11/22/
│       ├── 15.ogg
│       └── 15.json          # References global speaker IDs
└── speakers/
    ├── library.json         # Global speaker registry
    └── embeddings/
        ├── speaker-abc123.json
        ├── speaker-def456.json
        └── ...
```

**Benefits**:
- Single source of truth for each speaker
- Cross-recording matching built-in
- Can track speaker statistics over time
- Easier contact association

**Structure**:
```json
{
  "speaker_id": "spkr_abc123",
  "first_seen": "2025-11-22T15:00:00Z",
  "last_seen": "2025-11-22T18:00:00Z",
  "recordings": ["15.ogg", "16.ogg"],
  "embedding": [256 floats],
  "contact_id": "contact_xyz789",  // if associated
  "stats": {
    "total_duration": 3600,
    "recording_count": 2
  }
}
```

### Option 2: Database Storage

Use SQLite (or later, Postgres) for structured storage:

```sql
CREATE TABLE speakers (
  id TEXT PRIMARY KEY,
  embedding BLOB,  -- 256 floats as binary
  first_seen TIMESTAMP,
  last_seen TIMESTAMP,
  contact_id TEXT REFERENCES contacts(id)
);

CREATE TABLE speaker_appearances (
  speaker_id TEXT REFERENCES speakers(id),
  audio_file TEXT,
  local_speaker_id TEXT,  -- SPEAKER_00, etc.
  segments JSON,
  FOREIGN KEY (speaker_id) REFERENCES speakers(id)
);
```

**Benefits**:
- Efficient queries ("find all recordings with this speaker")
- Better for large-scale data
- Natural fit for contact association
- Can add indexes for similarity search

### Option 3: Hybrid Approach (Current + Library)

Keep current per-file storage, add global library:

- **Per-file JSON**: Continue storing embeddings inline for quick access
- **Global library**: Build separate index/library for cross-file queries
- **Sync**: Extract embeddings from JSON files into library

**Benefits**:
- Backward compatible
- Fast per-file access
- Enables cross-file queries
- Can migrate gradually

## Recommended Next Steps

1. **Keep current storage** for now (it works, simple, human-readable)

2. **Build speaker library system**:
   - Extract embeddings from JSON files
   - Create global speaker registry
   - Use cosine similarity to match speakers across files
   - Store in SQLite or separate JSON files

3. **Add contact association**:
   - Link speakers to contacts
   - Store in speaker library
   - Update recordings with contact IDs

4. **Consider vector database** (for scale):
   - Use something like ChromaDB, Pinecone, or Qdrant
   - Efficient similarity search
   - Better for "find top 3 similar speakers"

## Accessing Embeddings

### From Python

```python
import json

# Load embeddings from a file
with open('data/audio/2025/11/22/15.json') as f:
    results = json.load(f)
    
embeddings = results['speaker_embeddings']
speaker_02_embedding = embeddings['SPEAKER_02']  # 256-dim vector
```

### Cross-file Matching Example

```python
import numpy as np
from sklearn.metrics.pairwise import cosine_similarity

# Load embeddings from two files
with open('data/audio/2025/11/22/15.json') as f:
    file1 = json.load(f)
with open('data/audio/2025/11/22/16.json') as f:
    file2 = json.load(f)

# Compare speakers using cosine similarity
emb1 = np.array([file1['speaker_embeddings']['SPEAKER_02']])
emb2 = np.array([file2['speaker_embeddings']['SPEAKER_01']])

similarity = cosine_similarity(emb1, emb2)[0][0]
if similarity > 0.8:  # threshold
    print("Likely same speaker!")
```

