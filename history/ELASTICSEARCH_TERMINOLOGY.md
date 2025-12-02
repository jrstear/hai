# Elasticsearch Terminology

## Basic Concepts

### Index ≈ SQL Table
An **index** in Elasticsearch is similar to a **table** in SQL:
- **SQL**: `CREATE TABLE speakers (...)` 
- **Elasticsearch**: `PUT /speakers` (creates an index)

**Our indices:**
- `speakers` - stores speaker records with embeddings
- `recordings` - stores audio file metadata
- `segments` - stores speaker segments (who spoke when)

### Document ≈ SQL Row
A **document** in Elasticsearch is similar to a **row** in SQL:
- **SQL**: One row = one speaker record
- **Elasticsearch**: One document = one speaker record

### Field ≈ SQL Column
A **field** in Elasticsearch is similar to a **column** in SQL:
- **SQL**: `id`, `embedding`, `first_seen`
- **Elasticsearch**: Same field names, but with types (keyword, dense_vector, date)

## Key Differences from SQL

1. **Schema-less (but can have mappings)**: Elasticsearch can infer types, but we define explicit mappings for consistency
2. **Full-text search**: Built-in text analysis and search capabilities
3. **Vector search**: Native support for `dense_vector` fields with kNN (k-nearest neighbors) search
4. **Distributed**: Designed for horizontal scaling across multiple nodes
5. **JSON-based**: All documents are JSON

## Our Schema Mapping

```
SQL Table          →  Elasticsearch Index
─────────────────────────────────────────
speakers           →  speakers
recordings         →  recordings  
segments           →  segments

SQL Row            →  Elasticsearch Document
─────────────────────────────────────────
One speaker        →  One document in speakers index
One recording      →  One document in recordings index
One segment        →  One document in segments index

SQL Column         →  Elasticsearch Field
─────────────────────────────────────────
id (TEXT)          →  id (keyword)
embedding (BLOB)   →  embedding (dense_vector, 256 dims)
first_seen (DATE)  →  first_seen (date)
```

## Example Query Comparison

**SQL:**
```sql
SELECT * FROM speakers WHERE id = 'spkr_abc123';
```

**Elasticsearch:**
```json
GET /speakers/_doc/spkr_abc123
```

**SQL (vector similarity - if supported):**
```sql
SELECT * FROM speakers 
ORDER BY cosine_similarity(embedding, ?) DESC 
LIMIT 10;
```

**Elasticsearch (kNN search):**
```json
POST /speakers/_search
{
  "knn": {
    "field": "embedding",
    "query_vector": [...],
    "k": 10
  }
}
```

## Why We Use Elasticsearch

1. **Vector search**: Native kNN search for speaker matching (much faster than SQL)
2. **Scalability**: Can handle millions of segments across multiple nodes
3. **Full-text search**: Can search lifelog transcripts (future feature)
4. **Flexibility**: Easy to add new fields without migrations
5. **Analytics**: Built-in aggregations for complex queries









