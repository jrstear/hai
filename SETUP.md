# Hai Project Setup Guide

This guide will help you set up the development environment for the Hai Audio Lifelog Processing System.

## Prerequisites

- macOS (tested on M1 Mac)
- [Homebrew](https://brew.sh/) package manager
- Go 1.21+ (for backend services)
- Conda (for Python environment management)

## Initial Setup

### 1. Install Conda

If you don't have conda installed:

```bash
brew install --cask miniconda
```

After installation, initialize conda for your shell:

```bash
conda init "$(basename "${SHELL}")"
```

Then restart your terminal or run:

```bash
source ~/.zshrc  # or ~/.bash_profile for bash
```

### 2. Create Python Environment

Create and activate the `hai` conda environment:

```bash
conda create -n hai python=3.10 -y
conda activate hai
```

### 3. Install Python Dependencies

For diarization (speaker identification):

```bash
cd cmd/diarize
pip install -r requirements.txt
```

### 4. Set Up API Keys

Create a `.env` file in the project root with your API keys:

```bash
# Limitless API key (get from https://app.limitless.ai developer settings)
export LIMITLESS_API_KEY="sk-your-key-here"

# Hugging Face token (required for pyannote.audio)
# 1. Create account at https://huggingface.co
# 2. Accept terms for ALL required models:
#    - https://hf.co/pyannote/speaker-diarization-3.1
#    - https://hf.co/pyannote/segmentation-3.0
# 3. Get token from https://huggingface.co/settings/tokens
export HF_TOKEN="hf_your-token-here"
```

**Important:** Never commit the `.env` file to version control. It's already in `.gitignore`.

## Running the Tools

### Data Ingestion

Fetch audio recordings from Limitless API:

```bash
# Fetch default time range (Nov 22, 2025, 3-7pm MST)
go run cmd/ingest/main.go

# Fetch custom time range
go run cmd/ingest/main.go -start "2025-11-22T10:00:00-07:00" -duration 2h -out data/audio
```

Audio files are saved to `data/audio/YYYY/MM/DD/HH.ogg`

### Diarization Benchmark

Test speaker diarization performance on your M1 Mac:

```bash
conda activate hai
python cmd/diarize/benchmark.py data/audio/2025/11/22/15.ogg
```

This will:
- Load the pyannote speaker diarization model
- Use MPS (Metal Performance Shaders) for M1 acceleration
- Output speaker segments with timestamps
- Report processing time

## Project Structure

```
hai/
├── cmd/
│   ├── ingest/          # Audio ingestion from Limitless API
│   └── diarize/         # Speaker diarization tools
├── data/
│   └── audio/           # Downloaded audio files (not in git)
├── .beads/              # Issue tracking database
├── .env                 # API keys (not in git)
└── AGENTS.md            # AI agent instructions
```

## Using Beads for Task Tracking

This project uses [beads](https://github.com/steveyegge/beads) for issue tracking:

```bash
# See what's ready to work on
bd ready --json

# View all issues
bd list --json

# Show issue details
bd show <issue-id> --json

# Update issue status
bd update <issue-id> --status in_progress --json
```

See [AGENTS.md](AGENTS.md) for detailed workflow.

## Troubleshooting

### Conda not found after installation

Run the conda init command and restart your terminal:

```bash
conda init "$(basename "${SHELL}")"
# Then restart terminal
```

### HF_TOKEN errors

Make sure you've:
1. Created a Hugging Face account
2. Accepted the model terms at https://hf.co/pyannote/speaker-diarization-3.1
3. Added your token to `.env`

### MPS/GPU not detected

The M1 Mac should automatically use MPS acceleration. If it falls back to CPU, check:

```bash
python -c "import torch; print(torch.backends.mps.is_available())"
```

Should return `True` on M1 Macs.
