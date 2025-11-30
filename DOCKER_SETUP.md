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

## Next Steps

Once Elasticsearch is running, you can:
1. Implement the Elasticsearch storage backend (`hai-hh1`)
2. Set up the full Docker Compose stack with all services (`hai-8us`)
3. Test the storage abstraction layer with Elasticsearch

