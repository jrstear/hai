# Containerized Architecture for Multi-User Deployment

## Overview

This document outlines the architecture for containerizing the Hai audio lifelog system to make it shareable and scalable for production use.

## Goals

1. **Shareable**: Others can run it locally with Docker
2. **Web Interface**: Users input API key and date range via browser
3. **Scalable**: Can deploy to AWS for production use
4. **Multi-User**: Support multiple users with data isolation
5. **Production-Ready**: Handle errors, queuing, monitoring

## Architecture Components

### 1. **Web Frontend** (React/Vue/Simple HTML)
- User inputs: Limitless API key, date range
- Job submission and status tracking
- Results visualization (speakers, segments, playback)
- Query interface (search by speaker, time range, etc.)

### 2. **Backend API** (Go)
- REST API for job submission
- User session management
- Job queue management
- Elasticsearch query interface
- Audio file serving (with HTTP Range support)

### 3. **Diarization Worker** (Python)
- Processes audio files using pyannote.audio
- Extracts segments and embeddings
- Indexes results to Elasticsearch
- Handles GPU/CPU processing

### 4. **Elasticsearch** (Container)
- Stores speakers, recordings, segments
- Vector similarity search for speaker matching
- Full-text search (if transcripts added later)

### 5. **Job Queue** (Redis or RabbitMQ)
- Manages diarization jobs
- Handles retries and failures
- Scales workers horizontally

### 6. **Storage** (Volumes)
- Audio files (persistent volume)
- Elasticsearch data (persistent volume)
- Temporary processing files

## Docker Architecture

### Option A: Docker Compose (Recommended for Sharing)

**Best for**: Local development, sharing with others, single-machine deployment

```yaml
# docker-compose.yml
version: '3.8'

services:
  # Web Frontend
  frontend:
    build: ./web
    ports:
      - "3000:3000"
    environment:
      - API_URL=http://backend:8080
    depends_on:
      - backend

  # Backend API
  backend:
    build: ./cmd/api
    ports:
      - "8080:8080"
    environment:
      - ELASTICSEARCH_URL=http://elasticsearch:9200
      - REDIS_URL=redis:6379
      - STORAGE_PATH=/data
    volumes:
      - audio_data:/data/audio
      - es_data:/data/es
    depends_on:
      - elasticsearch
      - redis

  # Diarization Worker
  diarization-worker:
    build: ./cmd/diarize
    environment:
      - ELASTICSEARCH_URL=http://elasticsearch:9200
      - REDIS_URL=redis:6379
      - HF_TOKEN=${HF_TOKEN}
      - STORAGE_PATH=/data
    volumes:
      - audio_data:/data/audio
      - es_data:/data/es
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
    depends_on:
      - elasticsearch
      - redis
      - backend

  # Elasticsearch
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.11.0
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
      - "ES_JAVA_OPTS=-Xms1g -Xmx1g"
    ports:
      - "9200:9200"
    volumes:
      - es_data:/usr/share/elasticsearch/data

  # Redis (Job Queue)
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

volumes:
  audio_data:
  es_data:
  redis_data:
```

### Option B: Kubernetes (For Production/AWS)

**Best for**: Production deployment, horizontal scaling, AWS EKS

```yaml
# Kubernetes manifests
# - frontend-deployment.yaml
# - backend-deployment.yaml
# - diarization-worker-deployment.yaml
# - elasticsearch-statefulset.yaml
# - redis-deployment.yaml
# - ingress.yaml
# - service-accounts.yaml
```

## Multi-User Data Isolation

### Strategy: User-Scoped Indices

Each user gets their own Elasticsearch indices:

```
speakers-{user_id}
recordings-{user_id}
segments-{user_id}
```

**Benefits**:
- Complete data isolation
- Easy user deletion (drop indices)
- Per-user scaling
- Simple querying (always filter by user_id)

**Implementation**:
```go
// Backend API
type UserContext struct {
    UserID string
    ESClient *elasticsearch.Client
}

func (u *UserContext) GetIndex(indexType string) string {
    return fmt.Sprintf("%s-%s", indexType, u.UserID)
}
```

### Alternative: Single Index with User Filtering

```json
// Document structure
{
  "user_id": "user_123",
  "speaker_id": "spkr_abc",
  "embedding": [...],
  ...
}
```

**Query with user filter**:
```json
{
  "query": {
    "bool": {
      "must": [
        { "term": { "user_id": "user_123" } },
        { "knn": { ... } }
      ]
    }
  }
}
```

**Recommendation**: Use **user-scoped indices** for better isolation and performance.

## Job Flow

### 1. User Submits Job

```
User → Frontend → Backend API
  ↓
Backend validates API key
  ↓
Creates job record in Redis
  ↓
Returns job_id to user
```

### 2. Worker Processes Job

```
Worker polls Redis for jobs
  ↓
Fetches audio from Limitless API
  ↓
Saves to /data/audio/{user_id}/{date}/
  ↓
Runs diarization (pyannote)
  ↓
Extracts segments + embeddings
  ↓
Matches speakers (kNN query to ES)
  ↓
Indexes to Elasticsearch
  ↓
Updates job status: "completed"
```

### 3. User Queries Results

```
User → Frontend → Backend API
  ↓
Backend queries Elasticsearch
  ↓
Returns results to frontend
  ↓
Frontend displays/plays audio
```

## Container Details

### Frontend Container

**Technology**: React or Vue.js (or simple HTML + vanilla JS)

**Dockerfile**:
```dockerfile
FROM node:18-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf
EXPOSE 3000
```

**Features**:
- Job submission form
- Job status dashboard
- Results viewer (speakers, segments)
- Audio player with segment highlighting
- Query interface

### Backend API Container

**Technology**: Go (already using Go for ingest)

**Dockerfile**:
```dockerfile
FROM golang:1.21-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api ./cmd/api

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=build /app/api .
EXPOSE 8080
CMD ["./api"]
```

**API Endpoints**:
```
POST   /api/jobs              # Submit new job
GET    /api/jobs/{id}         # Get job status
GET    /api/jobs              # List user's jobs
GET    /api/speakers          # List speakers
GET    /api/segments          # Query segments
GET    /api/audio/{path}      # Serve audio (with Range support)
```

### Diarization Worker Container

**Technology**: Python (pyannote.audio)

**⚠️ Important: M1 Mac Docker Limitation**

**Docker Desktop for Mac runs containers in a Linux VM**, which means:
- ❌ **MPS (Metal Performance Shaders) does NOT work** inside Docker containers on Mac
- ❌ MPS is macOS-specific and requires native macOS, not Linux
- ✅ **CPU fallback works** but is slower (3-5x slower than MPS)

**Solutions for M1 Mac**:

#### Option 1: Hybrid Architecture (Recommended for M1 Mac)

Run diarization **natively on macOS** (outside Docker), other services in Docker:

```yaml
# docker-compose.yml
services:
  # ... other services in Docker ...
  
  # Diarization runs natively (not in Docker)
  # Use a local Python process that connects to Redis/ES
```

**Pros**:
- ✅ Full MPS acceleration (15x faster than real-time)
- ✅ Other services still containerized
- ✅ Easy to share (others can use Docker for everything)

**Cons**:
- ⚠️ Diarization not containerized (but that's OK for sharing)

#### Option 2: CPU-Only Docker (Works but Slower)

Use Docker but accept CPU-only performance:

```dockerfile
FROM python:3.10-slim

# Install system dependencies
RUN apt-get update && apt-get install -y \
    ffmpeg \
    libsndfile1 \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
WORKDIR /app
COPY cmd/diarize/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy worker code
COPY cmd/diarize/worker.py .
COPY cmd/diarize/diarize.py .

# Set environment
ENV PYTHONUNBUFFERED=1
# Force CPU mode (MPS not available in Linux container)
ENV PYTORCH_ENABLE_MPS_FALLBACK=1

CMD ["python", "worker.py"]
```

**Performance**: ~0.02x RTF (50x slower than real-time) vs 0.066x RTF with MPS

#### Option 3: Colima (Experimental)

[Colima](https://github.com/abiosoft/colima) is an alternative to Docker Desktop that might have better hardware access, but MPS support is still not available in Linux containers.

**Recommendation**: Use **Option 1 (Hybrid)** for M1 Mac development/sharing.

**Dockerfile** (for Linux/NVIDIA systems):
```dockerfile
FROM python:3.10-slim

# Install system dependencies
RUN apt-get update && apt-get install -y \
    ffmpeg \
    libsndfile1 \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
WORKDIR /app
COPY cmd/diarize/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy worker code
COPY cmd/diarize/worker.py .
COPY cmd/diarize/diarize.py .

# Set environment
ENV PYTHONUNBUFFERED=1

CMD ["python", "worker.py"]
```

**GPU Support** (for NVIDIA Linux systems):
```dockerfile
# Use nvidia/cuda base image
FROM nvidia/cuda:11.8.0-cudnn8-runtime-ubuntu22.04
# ... rest of setup
```

**Worker Script** (`worker.py`):
```python
import redis
import json
from diarize import run_diarization
from elasticsearch import Elasticsearch

redis_client = redis.Redis(host='redis', port=6379)
es_client = Elasticsearch(['http://elasticsearch:9200'])

def process_job(job_data):
    user_id = job_data['user_id']
    api_key = job_data['api_key']
    start_time = job_data['start_time']
    end_time = job_data['end_time']
    
    # Fetch audio (call Go service or Python)
    # Run diarization
    results = run_diarization(audio_file, hf_token)
    
    # Index to Elasticsearch
    index_speakers(user_id, results, es_client)
    index_segments(user_id, results, es_client)
    
    # Update job status
    redis_client.set(f"job:{job_id}:status", "completed")
```

## M1 Mac Docker Considerations

### The Problem

**Docker Desktop for Mac runs containers in a Linux VM**, which means:
- ❌ **MPS (Metal Performance Shaders) does NOT work** inside Docker containers
- ❌ MPS is macOS-specific and requires native macOS execution
- ✅ CPU fallback works but is **3-5x slower** than MPS

**Your current performance**:
- **With MPS (native macOS)**: 0.066x RTF (15x faster than real-time) ✅
- **With CPU (Docker)**: ~0.02x RTF (50x slower than real-time) ❌

### Solution: Hybrid Architecture for M1 Mac

Run diarization **natively on macOS**, other services in Docker:

```yaml
# docker-compose.m1.yml (for M1 Mac)
version: '3.8'

services:
  # Services that work fine in Docker
  frontend:
    # ... same as before ...
  
  backend:
    # ... same as before ...
  
  elasticsearch:
    # ... same as before ...
  
  redis:
    # ... same as before ...

  # Diarization worker runs NATIVELY (not in Docker)
  # See setup instructions below
```

**Setup for Native Diarization Worker**:

1. **Create a native worker script** (`cmd/diarize/worker-native.py`):
```python
#!/usr/bin/env python3
"""Native macOS worker that uses MPS acceleration"""
import redis
from diarize import run_diarization
# ... worker logic ...
```

2. **Run worker natively** (connects to Docker services):
```bash
# Start Docker services
docker-compose up -d elasticsearch redis backend

# Run worker natively (uses MPS)
conda activate hai
python cmd/diarize/worker-native.py
```

3. **Or use a wrapper script** (`scripts/start-m1.sh`):
```bash
#!/bin/bash
# Start Docker services
docker-compose up -d elasticsearch redis backend frontend

# Wait for services to be ready
sleep 5

# Start native worker
conda activate hai
python cmd/diarize/worker-native.py
```

**Benefits**:
- ✅ Full MPS acceleration (15x faster)
- ✅ Other services still containerized
- ✅ Easy to share (others can use full Docker)

**For Sharing**: Provide both options:
- `docker-compose.yml` - Full Docker (CPU-only, works everywhere)
- `docker-compose.m1.yml` - Hybrid (M1 Mac optimized)
- `scripts/start-m1.sh` - Helper script for M1 Mac users

### Alternative: Accept CPU Performance

If you want everything in Docker for simplicity:
- Use CPU-only Docker container
- Accept slower performance (~50x slower than real-time)
- Fine for testing/sharing, not ideal for production on M1

## Deployment Options

### 1. Local Docker Compose (Sharing)

**Setup**:
```bash
git clone <repo>
cd hai
cp .env.example .env
# Edit .env with HF_TOKEN
docker-compose up
```

**Access**: `http://localhost:3000`

**Pros**:
- Easy to share
- Works on any machine with Docker
- Single command to start

**Cons**:
- Single machine only
- No GPU support on Mac (CPU fallback)
- Limited scaling

### 2. AWS ECS (Production)

**Architecture**:
- ECS Fargate for containers (no EC2 management)
- Application Load Balancer
- RDS for metadata (optional, or use ES)
- S3 for audio file storage
- ECS Service for workers (auto-scaling)

**Setup**:
```bash
# Build and push images
docker build -t hai-backend ./cmd/api
docker tag hai-backend:latest <ecr-repo>/hai-backend:latest
docker push <ecr-repo>/hai-backend:latest

# Deploy with ECS CLI or Terraform
```

**Pros**:
- Auto-scaling
- Managed infrastructure
- GPU support (ECS with EC2 + GPU instances)
- Production-ready

**Cons**:
- More complex setup
- AWS costs
- Requires AWS knowledge

### 3. AWS EKS (Kubernetes)

**Best for**: Large-scale, multi-region, advanced orchestration

**Setup**: Kubernetes manifests + Helm charts

**Pros**:
- Maximum flexibility
- Industry standard
- Great for large teams

**Cons**:
- Steep learning curve
- More operational overhead
- Overkill for small deployments

## Security Considerations

### 1. API Key Storage

**Option A: Per-User Storage (Recommended)**
- Store in encrypted database/Redis
- Never log API keys
- Rotate keys periodically

**Option B: User Provides Each Time**
- Don't store, use session-only
- More secure but less convenient

### 2. Data Isolation

- User-scoped Elasticsearch indices
- File system isolation (`/data/audio/{user_id}/`)
- Backend validates user_id on all requests

### 3. Network Security

- HTTPS in production (TLS termination at load balancer)
- Internal service communication (no external exposure)
- Elasticsearch not exposed externally

### 4. Authentication

**Simple (MVP)**:
- Session-based (cookie)
- User provides API key = authentication

**Production**:
- OAuth2 / JWT tokens
- User accounts with passwords
- API key management UI

## Scaling Considerations

### Horizontal Scaling

**Workers**: Scale diarization workers based on queue depth
```yaml
# docker-compose scale
docker-compose up --scale diarization-worker=5

# Kubernetes
kubectl scale deployment diarization-worker --replicas=10
```

**Backend**: Stateless, scale easily
```yaml
# Load balancer distributes requests
backend:
  deploy:
    replicas: 3
```

**Elasticsearch**: Start single-node, scale to cluster later
```yaml
# Production: 3-node cluster
elasticsearch:
  deploy:
    replicas: 3
```

### Vertical Scaling

**GPU Workers**: Use GPU instances for faster diarization
- AWS: p3.2xlarge (1x V100) or p4d.24xlarge (8x A100)
- Local: NVIDIA GPU with Docker GPU support

**Elasticsearch**: Increase heap size, add nodes
```yaml
environment:
  - "ES_JAVA_OPTS=-Xms4g -Xmx4g"
```

## Development Workflow

### Local Development

```bash
# Start services
docker-compose up -d

# View logs
docker-compose logs -f diarization-worker

# Rebuild after code changes
docker-compose up --build

# Run tests
docker-compose run backend go test ./...
```

### Production Deployment

```bash
# Build images
docker-compose build

# Tag for registry
docker tag hai-backend:latest registry.example.com/hai-backend:v1.0.0

# Push to registry
docker push registry.example.com/hai-backend:v1.0.0

# Deploy (ECS/K8s)
# ... deployment commands
```

## Cost Estimates

### Local Docker (Free)
- Developer time only
- Uses local resources

### AWS ECS (Small Scale)
- Fargate: ~$30-50/month (2 tasks, minimal usage)
- ALB: ~$20/month
- S3: ~$5-10/month (audio storage)
- **Total**: ~$55-80/month

### AWS ECS (Production Scale)
- Fargate: ~$200-500/month (auto-scaling)
- GPU Workers (EC2): ~$500-2000/month (p3.2xlarge)
- ALB: ~$20/month
- S3: ~$50-100/month
- Elasticsearch Service: ~$100-300/month
- **Total**: ~$870-2920/month

## Implementation Phases

### Phase 1: MVP (Docker Compose)
- [ ] Basic web frontend (HTML + JS)
- [ ] Backend API (Go)
- [ ] Diarization worker (Python)
- [ ] Elasticsearch container
- [ ] Redis job queue
- [ ] Basic job submission and status

**Time**: 1-2 weeks

### Phase 2: Multi-User Support
- [ ] User session management
- [ ] User-scoped Elasticsearch indices
- [ ] Data isolation
- [ ] User dashboard

**Time**: 1 week

### Phase 3: Production Features
- [ ] Error handling and retries
- [ ] Monitoring and logging
- [ ] Audio playback with segments
- [ ] Query interface
- [ ] Authentication

**Time**: 2-3 weeks

### Phase 4: AWS Deployment
- [ ] ECS/EKS setup
- [ ] CI/CD pipeline
- [ ] Auto-scaling
- [ ] Monitoring (CloudWatch)
- [ ] Backup and disaster recovery

**Time**: 2-3 weeks

## Recommendation

**Start with Docker Compose** because:
1. ✅ Easy to share (`docker-compose up`)
2. ✅ Works locally for development
3. ✅ Can test multi-container architecture
4. ✅ Easy migration to Kubernetes/ECS later
5. ✅ No cloud costs during development

**Migrate to AWS ECS when**:
- You need production deployment
- You need auto-scaling
- You need GPU instances
- You have multiple users

## Next Steps

1. **Create Dockerfiles** for each component
2. **Set up docker-compose.yml** with all services
3. **Build basic web frontend** (job submission)
4. **Implement backend API** (job management)
5. **Create worker** (process jobs from queue)
6. **Test locally** with Docker Compose
7. **Share** with others for feedback
8. **Plan AWS migration** when ready

