# Kibana Automation Options

## Overview

The manual steps in `KIBANA_QUICK_START.md` can be automated using Kibana's Saved Objects API. This allows you to programmatically create index patterns, visualizations, dashboards, and saved searches.

## Kibana Saved Objects API

Kibana stores all configuration (index patterns, visualizations, dashboards, etc.) as "saved objects" in Elasticsearch. These can be created, updated, and managed via REST API.

### Base URL
```
http://localhost:5601/api/saved_objects
```

### Authentication
For local development (security disabled), no authentication needed. For production, you'd need API keys or basic auth.

## Automatable Steps

### 1. Index Patterns

**Manual Step**: Create index patterns via UI

**Automated via API**:
```bash
POST /api/saved_objects/index-pattern/speakers
{
  "attributes": {
    "title": "speakers",
    "timeFieldName": "created_at"
  }
}
```

**For each index pattern**:
- `speakers` (time field: `created_at`)
- `recordings` (time field: `start_time`)
- `segments` (time field: `created_at`)
- `lifelogs` (time field: `start_time`)
- `lifelog_blockquotes` (time field: `start_time`)

### 2. Visualizations

**Manual Step**: Create visualizations via UI

**Automated via API**:
```bash
POST /api/saved_objects/visualization/blockquotes-by-speaker-name
{
  "attributes": {
    "title": "Blockquotes by Speaker Name",
    "visState": {
      "type": "histogram",
      "params": { ... },
      "aggs": [ ... ]
    },
    "kibanaSavedObjectMeta": {
      "searchSourceJSON": { ... }
    }
  }
}
```

**Challenges**:
- Visualization definitions are complex JSON structures
- Need to understand Kibana's internal format
- Easier to create via UI first, then export and reuse

### 3. Dashboards

**Manual Step**: Create dashboards via UI

**Automated via API**:
```bash
POST /api/saved_objects/dashboard/speaker-mapping-dashboard
{
  "attributes": {
    "title": "Speaker Mapping Dashboard",
    "panelsJSON": "[...]",
    "optionsJSON": "{...}",
    "version": 1
  }
}
```

**Challenges**:
- Dashboard JSON is very complex
- References other saved objects (visualizations, searches)
- Best approach: Create in UI, export, then import programmatically

### 4. Saved Searches

**Manual Step**: Save common queries

**Automated via API**:
```bash
POST /api/saved_objects/search/unmapped-blockquotes
{
  "attributes": {
    "title": "Unmapped Blockquotes",
    "kibanaSavedObjectMeta": {
      "searchSourceJSON": {
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
    }
  }
}
```

## Automation Approaches

### Option 1: Init Script (Recommended)

**Approach**: Create a script that runs after Kibana starts to set up index patterns and basic saved objects.

**Implementation**:
- Shell script or Go program that waits for Kibana to be ready
- Uses `curl` or HTTP client to call Kibana API
- Creates all index patterns
- Optionally imports pre-exported visualizations/dashboards

**Pros**:
- ✅ Simple to implement
- ✅ Can be run manually or as part of Docker startup
- ✅ Easy to update

**Cons**:
- ⚠️ Need to handle Kibana startup time
- ⚠️ Need to handle idempotency (don't recreate if exists)

**Location**: `scripts/setup-kibana.sh` or `scripts/setup-kibana.go`

### Option 2: Kibana Init Container

**Approach**: Use a separate container that runs after Kibana starts to configure it.

**Implementation**:
- Add init container to docker-compose.yml
- Runs setup script
- Waits for Kibana health check
- Configures Kibana via API

**Pros**:
- ✅ Automatic on startup
- ✅ Isolated from main Kibana container

**Cons**:
- ⚠️ More complex Docker setup
- ⚠️ Need to handle container orchestration

### Option 3: Kibana Configuration Files

**Approach**: Use Kibana's configuration file to pre-configure some settings.

**Implementation**:
- Mount configuration files into Kibana container
- Use `kibana.yml` for basic settings
- Limited - can't create saved objects this way

**Pros**:
- ✅ Simple for basic config

**Cons**:
- ❌ Can't create index patterns or visualizations
- ❌ Limited to Kibana settings only

### Option 4: Export/Import Saved Objects

**Approach**: Create visualizations/dashboards in UI, export them, then import programmatically.

**Implementation**:
1. Manually create visualizations/dashboards in Kibana UI
2. Export via API: `GET /api/saved_objects/_export`
3. Store exported JSON files in repo
4. Import on setup: `POST /api/saved_objects/_import`

**Pros**:
- ✅ Easy to create complex visualizations (use UI)
- ✅ Version controlled (JSON files in repo)
- ✅ Can be imported programmatically

**Cons**:
- ⚠️ Two-step process (create in UI, then export)
- ⚠️ Need to maintain exported JSON files

### Option 5: Kibana Saved Objects Client Library

**Approach**: Use a Go library or SDK to interact with Kibana API.

**Implementation**:
- Use Go HTTP client or Kibana Go SDK (if exists)
- Programmatically create saved objects
- More type-safe than raw API calls

**Pros**:
- ✅ Type-safe
- ✅ Better error handling
- ✅ Can be integrated into Go codebase

**Cons**:
- ⚠️ Need to find/maintain library
- ⚠️ More complex than simple scripts

## Recommended Approach

**Hybrid: Script + Export/Import**

1. **Index Patterns**: Create via script (simple API calls)
2. **Visualizations/Dashboards**: Create in UI, export, then import via script

**Why**:
- Index patterns are simple and easy to script
- Visualizations are complex - easier to create in UI first
- Best of both worlds: automation + flexibility

## Implementation Details

### Script Structure

```bash
#!/bin/bash
# scripts/setup-kibana.sh

KIBANA_URL="http://localhost:5601"

# Wait for Kibana to be ready
wait_for_kibana() {
  until curl -f "$KIBANA_URL/api/status" > /dev/null 2>&1; do
    echo "Waiting for Kibana..."
    sleep 5
  done
}

# Create index pattern
create_index_pattern() {
  local name=$1
  local time_field=$2
  
  curl -X POST "$KIBANA_URL/api/saved_objects/index-pattern/$name" \
    -H "Content-Type: application/json" \
    -H "kbn-xsrf: true" \
    -d "{
      \"attributes\": {
        \"title\": \"$name\",
        \"timeFieldName\": \"$time_field\"
      }
    }"
}

# Main
wait_for_kibana
create_index_pattern "speakers" "created_at"
create_index_pattern "recordings" "start_time"
# ... etc
```

### Go Implementation

```go
// scripts/setup-kibana.go

package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type IndexPattern struct {
    Attributes struct {
        Title       string `json:"title"`
        TimeFieldName string `json:"timeFieldName"`
    } `json:"attributes"`
}

func createIndexPattern(client *http.Client, kibanaURL, name, timeField string) error {
    pattern := IndexPattern{}
    pattern.Attributes.Title = name
    pattern.Attributes.TimeFieldName = timeField
    
    body, _ := json.Marshal(pattern)
    req, _ := http.NewRequest("POST", 
        fmt.Sprintf("%s/api/saved_objects/index-pattern/%s", kibanaURL, name),
        bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("kbn-xsrf", "true")
    
    resp, err := client.Do(req)
    // Handle response...
    return nil
}
```

## What to Automate

### High Priority (Easy to Automate)
1. ✅ **Index Patterns** - Simple API calls, clear benefit
2. ✅ **Basic Saved Searches** - Simple queries, useful for users

### Medium Priority (Moderate Effort)
3. ⚠️ **Simple Visualizations** - Can script, but complex JSON
4. ⚠️ **Import Pre-exported Dashboards** - Two-step but manageable

### Low Priority (Complex)
5. ❌ **Complex Visualizations** - Better created in UI, then exported
6. ❌ **Custom Dashboards** - Better created in UI, then exported

## Integration Points

### Docker Compose
- Add init script as separate service or entrypoint
- Run after Kibana health check passes

### Taskfile
- Add `task setup-kibana` command
- Can be run manually or as part of startup

### CI/CD
- Could run setup script in CI to verify Kibana configuration
- Export/import saved objects as part of deployment

## Benefits of Automation

1. **Consistency**: Same setup every time
2. **Speed**: No manual clicking through UI
3. **Reproducibility**: Can reset and reconfigure easily
4. **Version Control**: Scripts/configs in git
5. **Documentation**: Scripts serve as documentation

## Drawbacks

1. **Complexity**: More moving parts
2. **Maintenance**: Need to keep scripts in sync with Kibana versions
3. **Flexibility**: Harder to customize per-user
4. **Debugging**: Scripts can fail silently

## Recommendation

**Start Simple**: Automate index patterns only (easy win, high value)

**Then Expand**: Add saved searches and import pre-exported dashboards

**Keep Manual**: Complex visualizations - create in UI, export, then import

This gives you automation for the repetitive stuff while keeping flexibility for the complex stuff.












