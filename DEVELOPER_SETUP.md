# Developer Setup Guide: Clone to Kibana with Data

This guide walks you through setting up the Hai project from scratch, getting data into Elasticsearch, and viewing it in Kibana.

## Prerequisites

- macOS (tested on M1 Mac)
- [Homebrew](https://brew.sh/)
- [Docker Desktop](https://www.docker.com/products/docker-desktop) (for Elasticsearch and Kibana)
- [Go 1.21+](https://go.dev/dl/)
- [Conda](https://docs.conda.io/en/latest/miniconda.html) (for Python environment)

## Step 1: Clone the Repository

```bash
git clone <repository-url>
cd hai
```

## Step 2: Install Prerequisites

### Install Go (if not already installed)

```bash
brew install go
```

### Install Conda (if not already installed)

```bash
brew install --cask miniconda
```

After installation, initialize conda:

```bash
conda init "$(basename "${SHELL}")"
# Restart terminal or run: source ~/.zshrc
```

### Install Task (Task Runner)

```bash
# macOS
brew install go-task/tap/go-task

# Or via Go
go install github.com/go-task/task/v3/cmd/task@latest
```

### Install Docker Desktop

Download and install from: https://www.docker.com/products/docker-desktop

## Step 3: Set Up Python Environment

```bash
# Create conda environment
conda create -n hai python=3.10 -y
conda activate hai

# Install Python dependencies
cd onboard
task setup-python
cd ..
```

This will:
- Create the `hai` conda environment
- Install pyannote.audio and dependencies
- Verify MPS (Metal Performance Shaders) is available for M1 Macs

## Step 4: Configure API Keys

Create a `.env` file in the project root (or set environment variables):

```bash
# Limitless API key (get from https://app.limitless.ai developer settings)
export LIMITLESS_API_KEY="sk-your-key-here"

# Hugging Face token (required for pyannote.audio)
# 1. Create account at https://huggingface.co
# 2. Accept terms for required models:
#    - https://hf.co/pyannote/speaker-diarization-3.1
#    - https://hf.co/pyannote/segmentation-3.0
#    - https://hf.co/pyannote/speaker-diarization-community-1
# 3. Get token from https://huggingface.co/settings/tokens
export HF_TOKEN="hf_your-token-here"

# Elasticsearch URL (optional, defaults to http://localhost:9200)
export ELASTICSEARCH_URL="http://localhost:9200"
```

**Important:** The `.env` file is already in `.gitignore` - never commit it.

## Step 5: Start Elasticsearch and Kibana

```bash
# Start both services
docker-compose --profile elasticsearch up -d

# Wait for services to be ready (about 30-60 seconds)
# Check Elasticsearch health
curl http://localhost:9200/_cluster/health

# Check Kibana status
curl http://localhost:5601/api/status
```

Services will be available at:
- Elasticsearch: http://localhost:9200
- Kibana: http://localhost:5601

## Step 6: Set Up Kibana Index Patterns

```bash
# Build the setup script (if not already built)
task build-setup-kibana

# Run the setup script
task setup-kibana-index-patterns
```

This automatically creates index patterns for:
- `speakers`
- `recordings`
- `segments`
- `lifelogs`
- `lifelog_blockquotes`

## Step 7: Build the Onboarding Server

```bash
cd onboard
task build
cd ..
```

## Step 8: Run the Onboarding Server

```bash
cd onboard
task run
```

This will:
- Start the web server on http://localhost:3000
- Open your browser automatically
- Show the onboarding UI

## Step 9: Process Data via Web UI

1. **Enter API Key**: Your `LIMITLESS_API_KEY` should be pre-filled if set as an environment variable
2. **Select Date Range**: Choose start and end date/time (default: Nov 22, 2025, 2pm-6pm)
3. **Click "Start Processing"**

The server will:
- Download lifelogs for the date range
- Download audio files (in 1-hour chunks)
- Run diarization on each audio file
- Export results to Elasticsearch (if `ELASTICSEARCH_URL` is set)
- Export lifelogs to Elasticsearch
- Show progress in real-time

## Step 10: View Data in Kibana

Once processing completes:

1. **Open Kibana**: http://localhost:5601
2. **Go to Discover**: Click "Discover" in the left sidebar
3. **Select Index Pattern**: Choose one of:
   - `lifelog_blockquotes` - View lifelog transcripts with speaker names
   - `segments` - View diarization segments with speaker IDs
   - `speakers` - View all detected speakers
   - `recordings` - View audio file metadata
   - `lifelogs` - View full lifelog documents

4. **Explore Data**:
   - Use the search bar to filter
   - Click on fields to add them as columns
   - Adjust the time range to see your data

## Step 11: Map Speaker Names (Optional)

After data is loaded, map lifelog speaker names to global speaker IDs:

```bash
cd onboard
task map-speakers CLI_ARGS="-start 2025-11-22 -end 2025-11-23"
```

This will:
- Match lifelog blockquotes with diarization segments by time overlap
- Map speaker names (e.g., "Jon Stearley") to global speaker IDs
- Update the `speaker_id` field in `lifelog_blockquotes`

## Step 12: Review Mappings in Kibana

1. **In Kibana Discover**, select `lifelog_blockquotes` index pattern
2. **Add columns**: `speaker_name`, `speaker_id`, `content`, `start_time`
3. **Filter by speaker**: `speaker_id.keyword: "spkr_abc123"`
4. **Find unmapped**: `NOT speaker_id:*`

## Troubleshooting

### Elasticsearch won't start

- Check Docker Desktop is running
- Check port 9200 is available: `lsof -i :9200`
- Check Docker has enough memory (Elasticsearch needs at least 1GB)
- View logs: `docker-compose --profile elasticsearch logs elasticsearch`

### Kibana won't start

- Wait for Elasticsearch to be fully ready first
- Check port 5601 is available: `lsof -i :5601`
- View logs: `docker-compose --profile elasticsearch logs kibana`

### Python/diarization errors

- Verify conda environment is activated: `conda activate hai`
- Check `HF_TOKEN` is set: `echo $HF_TOKEN`
- Verify pyannote is installed: `python -c "import pyannote.audio; print('OK')"`
- Check MPS is available: `python -c "import torch; print(torch.backends.mps.is_available())"`

### No data in Kibana

- Verify data was exported: Check onboarding server logs for "Loading ... to Elasticsearch"
- Check Elasticsearch indices exist: `curl http://localhost:9200/_cat/indices`
- Verify index patterns are created in Kibana
- Check time range in Kibana matches your data

### Index patterns not created

- Run setup script manually: `./bin/setup-kibana-index-patterns`
- Check Kibana is ready: `curl http://localhost:5601/api/status`
- Create manually via Kibana UI (see `KIBANA_QUICK_START.md`)

## Quick Reference

### Start Everything

```bash
# Terminal 1: Start Elasticsearch and Kibana
docker-compose --profile elasticsearch up -d

# Terminal 2: Start onboarding server
cd onboard && task run
```

### Stop Everything

```bash
# Stop onboarding server: Ctrl+C

# Stop Docker services
docker-compose --profile elasticsearch down
```

### Reset Everything (⚠️ Deletes all data)

```bash
# Stop and remove containers and volumes
docker-compose --profile elasticsearch down -v

# Remove processed data (optional)
rm -rf onboard/data/
```

## Next Steps

- **Explore Data**: Use Kibana Discover to browse your data
- **Create Visualizations**: Build charts and dashboards in Kibana
- **Review Mappings**: Check speaker name mappings and correct any issues
- **Query Data**: Use Kibana Dev Tools to run Elasticsearch queries
- **See**: `KIBANA_QUICK_START.md` for more Kibana usage examples

## Additional Resources

- `SETUP.md` - Detailed setup instructions
- `DOCKER_SETUP.md` - Docker-specific configuration
- `KIBANA_QUICK_START.md` - Kibana usage guide
- `onboard/README.md` - Onboarding server documentation
- `history/` - Planning and design documents

