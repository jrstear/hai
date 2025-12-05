# export2elastic Module Explanation

## What Goes in `internal/export2elastic/`?

The `export2elastic` module handles **exporting diarization results to Elasticsearch**. It's the bridge between native processing (Python diarization) and containerized storage (Elasticsearch).

## Responsibilities

### 1. Read Diarization Results
- Read JSON output from Python diarization subprocess
- Parse segments, speaker embeddings, metadata
- Handle different result formats/versions

### 2. Speaker Matching
- For each speaker embedding from diarization:
  - Query Elasticsearch using kNN (vector similarity search)
  - Find existing speakers with similar embeddings
  - If match found (similarity > threshold): use existing speaker ID
  - If no match: create new speaker record

### 3. Index to Elasticsearch
- **Speakers**: Create/update speaker documents with embeddings
- **Recordings**: Create recording metadata documents
- **Segments**: Bulk index all segments with speaker/recording references
- Handle bulk operations efficiently (batch inserts)

### 4. Data Transformation
- Transform diarization JSON format to Elasticsearch document format
- Map local speaker IDs (SPEAKER_00) to global speaker IDs
- Calculate and store byte offsets (if available)
- Handle time zone conversions, date formatting

## Code Structure

```
internal/export2elastic/
├── exporter.go          # Main export logic
├── speaker_matcher.go   # Speaker matching via kNN
├── indexer.go           # Bulk indexing operations
├── transformer.go       # Data format transformation
└── types.go             # Type definitions
```

## Example Flow

```go
// exporter.go
func ExportToElasticsearch(
    diarizationResults *DiarizationResult,
    esClient *elasticsearch.Client,
    userID string,
) error {
    // 1. Match speakers
    speakers, err := MatchSpeakers(diarizationResults.SpeakerEmbeddings, esClient, userID)
    
    // 2. Create recording document
    recording := &Recording{
        ID: generateRecordingID(),
        UserID: userID,
        FilePath: diarizationResults.AudioFile,
        StartTime: diarizationResults.StartTime,
        Duration: diarizationResults.Duration,
    }
    
    // 3. Transform segments
    segments := TransformSegments(
        diarizationResults.Segments,
        speakers,
        recording.ID,
    )
    
    // 4. Bulk index
    return BulkIndex(esClient, speakers, recording, segments, userID)
}
```

## Why "export2elastic"?

- **"export"**: We're exporting data FROM diarization results
- **"2elastic"**: TO Elasticsearch
- Clear direction: diarization → Elasticsearch
- Not "import" (which would be Elasticsearch importing, not our code)

## Alternative Names Considered

- `indexer` - Too generic
- `esindexer` - OK but less clear
- `elasticsearch/indexer` - Too verbose
- `export2elastic` - ✅ Clear and descriptive

## Dependencies

- Elasticsearch Go client (`github.com/elastic/go-elasticsearch/v8`)
- Access to diarization JSON results
- User ID for data isolation

## Future Enhancements

- Retry logic for failed index operations
- Progress reporting during bulk operations
- Validation of data before indexing
- Support for updating existing records
- Byte offset calculation integration











