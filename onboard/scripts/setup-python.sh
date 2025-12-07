#!/bin/bash
# Setup Python environment for diarization
# This script sets up conda environment and installs dependencies

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Setting up Python environment for diarization..."

# Check if conda is available
if ! command -v conda &> /dev/null; then
    echo "Error: conda is not installed or not in PATH"
    echo "Install conda: brew install --cask miniconda"
    exit 1
fi

# Check if HF_TOKEN is set
if [ -z "$HF_TOKEN" ]; then
    echo "Warning: HF_TOKEN environment variable is not set"
    echo "Diarization will fail without it."
    echo "Get token from: https://huggingface.co/settings/tokens"
    echo ""
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Create conda environment if it doesn't exist
ENV_NAME="hai"
if ! conda env list | grep -q "^${ENV_NAME} "; then
    echo "Creating conda environment: ${ENV_NAME}"
    conda create -n ${ENV_NAME} python=3.10 -y
else
    echo "Conda environment '${ENV_NAME}' already exists"
fi

# Activate environment and install dependencies
echo "Installing Python dependencies..."
eval "$(conda shell.bash hook)"
conda activate ${ENV_NAME}

# Install dependencies from diarization requirements
DIARIZE_REQ="${PROJECT_ROOT}/../cmd/diarize/requirements.txt"
if [ -f "$DIARIZE_REQ" ]; then
    echo "Installing from: $DIARIZE_REQ"
    pip install -r "$DIARIZE_REQ"
else
    echo "Warning: $DIARIZE_REQ not found, installing default dependencies"
    pip install pyannote.audio torch torchaudio soundfile numpy
fi

# Verify installation
echo ""
echo "Verifying installation..."
python -c "import torch; print(f'PyTorch: {torch.__version__}')"
python -c "import pyannote.audio; print('pyannote.audio: OK')" || {
    echo "Error: pyannote.audio not installed correctly"
    exit 1
}

# Check for MPS (Mac GPU) support
if python -c "import torch; print(torch.backends.mps.is_available())" 2>/dev/null | grep -q "True"; then
    echo "✅ MPS (Metal) acceleration available"
else
    echo "⚠️  MPS acceleration not available (will use CPU)"
fi

echo ""
echo "✅ Python environment setup complete!"
echo ""
echo "To activate the environment manually:"
echo "  conda activate ${ENV_NAME}"
echo ""
echo "Make sure HF_TOKEN is set:"
echo "  export HF_TOKEN='your-token-here'"












