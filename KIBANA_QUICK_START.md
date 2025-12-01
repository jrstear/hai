# Kibana Quick Start Guide

## Starting Kibana

Kibana starts automatically with Elasticsearch:

```bash
docker-compose --profile elasticsearch up -d
```

Access Kibana at: http://localhost:5601

## Setting Up Index Patterns

**Automated Setup (Recommended):**

Run the setup script to automatically create all index patterns:

```bash
task setup-kibana-index-patterns
```

Or run directly:
```bash
./bin/setup-kibana-index-patterns
```

The script will:
- Wait for Kibana to be ready
- Create all 5 index patterns automatically
- Skip patterns that already exist (idempotent)

**Manual Setup:**

If you prefer to set them up manually, follow the steps below:

### 1. Speakers Index

1. Go to **Stack Management** → **Index Patterns** → **Create index pattern**
2. Index pattern name: `speakers`
3. Time field: `created_at` (or `first_seen`)
4. Click **Create index pattern**

### 2. Recordings Index

1. Create index pattern: `recordings`
2. Time field: `start_time`
3. Click **Create index pattern**

### 3. Segments Index

1. Create index pattern: `segments`
2. Time field: `created_at` (segments don't have absolute timestamps, but `created_at` works for filtering)
3. Click **Create index pattern**

### 4. Lifelogs Index

1. Create index pattern: `lifelogs`
2. Time field: `start_time`
3. Click **Create index pattern**

### 5. Lifelog Blockquotes Index

1. Create index pattern: `lifelog_blockquotes`
2. Time field: `start_time`
3. Click **Create index pattern**

## Useful Queries

### Find All Blockquotes for a Speaker

In **Discover** with `lifelog_blockquotes` index pattern:

```
speaker_id.keyword: "spkr_abc123"
```

### Find Unmapped Blockquotes

```
NOT speaker_id:*
```

### Find Blockquotes by Speaker Name

```
speaker_name.keyword: "Jon Stearley"
```

### Find Segments for a Speaker

In **Discover** with `segments` index pattern:

```
speaker_id.keyword: "spkr_abc123"
```

### Compare Blockquotes and Segments

1. Create a visualization showing blockquotes over time
2. Create another showing segments over time
3. Combine in a dashboard to see overlap

## Creating Visualizations

### Speaker Distribution

1. Go to **Visualize** → **Create visualization** → **Vertical Bar**
2. Select `lifelog_blockquotes` index pattern
3. Y-axis: Count
4. X-axis: Terms aggregation on `speaker_name.keyword`
5. Save as "Blockquotes by Speaker Name"

### Timeline of Blockquotes

1. Create visualization → **Line**
2. Select `lifelog_blockquotes` index pattern
3. X-axis: Date Histogram on `start_time`
4. Split series: Terms on `speaker_name.keyword`
5. Save as "Blockquotes Timeline"

### Mapping Status

1. Create visualization → **Pie Chart**
2. Select `lifelog_blockquotes` index pattern
3. Slice by: Terms on `speaker_id.keyword` (with missing bucket for unmapped)
4. Save as "Mapping Status"

## Reviewing Speaker Mappings

### View All Mapped Blockquotes

1. Go to **Discover**
2. Select `lifelog_blockquotes` index pattern
3. Filter: `speaker_id:*`
4. Add columns: `speaker_name`, `speaker_id`, `content`, `start_time`

### Review Unmapped Blockquotes

1. In **Discover** with `lifelog_blockquotes`
2. Filter: `NOT speaker_id:*`
3. Review why they weren't matched (no segments? low overlap?)

### Compare Speaker Names to IDs

1. Create a data table visualization
2. Group by `speaker_name.keyword`
3. Show unique count of `speaker_id.keyword`
4. This shows which names map to which IDs (and if multiple IDs per name)

## Dev Tools (Advanced Queries)

Access **Dev Tools** (wrench icon) to run Elasticsearch queries directly:

### Get all blockquotes for a speaker
```json
GET /lifelog_blockquotes/_search
{
  "query": {
    "term": {
      "speaker_id.keyword": "spkr_abc123"
    }
  },
  "sort": [
    {
      "start_time": {
        "order": "asc"
      }
    }
  ]
}
```

### Get unmapped blockquotes
```json
GET /lifelog_blockquotes/_search
{
  "query": {
    "bool": {
      "must_not": {
        "exists": {
          "field": "speaker_id"
        }
      }
    }
  }
}
```

### Get blockquotes with their mapped segments
```json
GET /lifelog_blockquotes/_search
{
  "query": {
    "match_all": {}
  },
  "_source": ["id", "speaker_name", "speaker_id", "recording_id", "content", "start_time", "end_time"]
}
```

## Tips

- Use **Saved Searches** to quickly access common queries
- Create **Dashboards** to combine multiple visualizations
- Use **Filters** to narrow down data by time range, speaker, etc.
- Export data using the **Share** button for further analysis

