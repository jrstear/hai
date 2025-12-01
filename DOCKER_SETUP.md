# Docker Setup Guide

## Elasticsearch Service

The Elasticsearch service is configured to run in single-node mode for local development.

### Starting Elasticsearch

To start Elasticsearch, use the `elasticsearch` profile:

```bash
docker-compose --profile elasticsearch up -d
```

This will:
- Start Elasticsearch on port 9200
- Create a persistent volume for data (`es_data`)
- Disable security for local development
- Configure JVM heap size to 1GB

### Checking Status

Check if Elasticsearch is healthy:

```bash
curl http://localhost:9200/_cluster/health
```

Or view logs:

```bash
docker-compose --profile elasticsearch logs -f elasticsearch
```

### Stopping Elasticsearch

```bash
docker-compose --profile elasticsearch down
```

To also remove the data volume (⚠️ **WARNING**: This deletes all data):

```bash
docker-compose --profile elasticsearch down -v
```

### Custom Configuration

1. Copy `docker-compose.override.yml.example` to `docker-compose.override.yml`
2. Modify settings as needed (ports, heap size, etc.)
3. The override file is automatically loaded by docker-compose

### Port Conflicts

If port 9200 is already in use, you can override it in `docker-compose.override.yml`:

```yaml
services:
  elasticsearch:
    ports:
      - "9201:9200"
```

Then access Elasticsearch at `http://localhost:9201`.

### Data Persistence

Elasticsearch data is stored in a Docker volume named `es_data`. This persists across container restarts. To view volume information:

```bash
docker volume inspect hai_es_data
```

### Health Checks

The service includes a health check that verifies Elasticsearch is responding. You can check the health status:

```bash
docker-compose --profile elasticsearch ps
```

The health check runs every 10 seconds and allows 30 seconds for initial startup.

## Troubleshooting

### Elasticsearch won't start

1. Check if port 9200 is available:
   ```bash
   lsof -i :9200
   ```

2. Check Docker logs:
   ```bash
   docker-compose --profile elasticsearch logs elasticsearch
   ```

3. Verify Docker has enough memory allocated (Elasticsearch needs at least 1GB)

### Permission errors

If you see permission errors, you may need to adjust the volume permissions:

```bash
sudo chown -R 1000:1000 /var/lib/docker/volumes/hai_es_data/_data
```

### Out of memory

If Elasticsearch crashes due to memory issues, reduce the heap size in `docker-compose.override.yml`:

```yaml
services:
  elasticsearch:
    environment:
      - "ES_JAVA_OPTS=-Xms512m -Xmx512m"
```

## Kibana Service

Kibana is configured to run alongside Elasticsearch for data visualization and analysis.

### Starting Kibana

Kibana starts automatically when you start Elasticsearch with the `elasticsearch` profile:

```bash
docker-compose --profile elasticsearch up -d
```

This will:
- Start Kibana on port 5601
- Connect to Elasticsearch automatically
- Disable security for local development (matches Elasticsearch)

### Accessing Kibana

Open your browser and navigate to:
```
http://localhost:5601
```

### Setting Up Index Patterns

**Automated Setup (Recommended):**

Use the setup script to automatically create all index patterns:

```bash
task setup-kibana-index-patterns
```

Or run directly:
```bash
./bin/setup-kibana-index-patterns
```

This will create index patterns for:
- `speakers` - Time field: `created_at`
- `recordings` - Time field: `start_time`
- `segments` - Time field: `created_at`
- `lifelogs` - Time field: `start_time`
- `lifelog_blockquotes` - Time field: `start_time`

**Manual Setup:**

If you prefer to set them up manually:

1. Go to **Stack Management** → **Index Patterns** → **Create index pattern**
2. Create patterns for each index:
   - `speakers` - Time field: `created_at`
   - `recordings` - Time field: `start_time`
   - `segments` - Time field: `created_at` (or use `start_time` if available)
   - `lifelogs` - Time field: `start_time`
   - `lifelog_blockquotes` - Time field: `start_time`

### Useful Kibana Features

- **Discover**: Browse and search your data
- **Visualize**: Create charts and graphs
- **Dashboard**: Combine multiple visualizations
- **Dev Tools**: Run Elasticsearch queries directly

### Checking Kibana Status

Check if Kibana is healthy:
```bash
curl http://localhost:5601/api/status
```

Or view logs:
```bash
docker-compose --profile elasticsearch logs -f kibana
```

### Port Conflicts

If port 5601 is already in use, you can override it in `docker-compose.override.yml`:

```yaml
services:
  kibana:
    ports:
      - "5602:5601"
```

Then access Kibana at `http://localhost:5602`.

## Next Steps

Once Elasticsearch and Kibana are running, you can:
1. Set up index patterns in Kibana (see above)
2. Explore your data using Kibana Discover
3. Create visualizations and dashboards
4. Review speaker name mappings and correct any issues
5. Query and analyze lifelog blockquotes, segments, and speakers

