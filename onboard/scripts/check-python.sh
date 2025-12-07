#!/bin/bash
# Check if Python environment is set up correctly

set -e

ENV_NAME="hai"

# Check if conda is available
if ! command -v conda &> /dev/null; then
    echo "Error: conda is not installed"
    exit 1
fi

# Check if environment exists
if ! conda env list | grep -q "^${ENV_NAME} "; then
    echo "Error: conda environment '${ENV_NAME}' does not exist"
    echo "Run: task setup-python"
    exit 1
fi

# Check if HF_TOKEN is set
if [ -z "$HF_TOKEN" ]; then
    echo "Error: HF_TOKEN environment variable is not set"
    echo "Set it with: export HF_TOKEN='your-token-here'"
    exit 1
fi

# Check if Python packages are installed
eval "$(conda shell.bash hook)"
conda activate ${ENV_NAME} 2>/dev/null || {
    echo "Error: Failed to activate conda environment"
    exit 1
}

python -c "import pyannote.audio" 2>/dev/null || {
    echo "Error: pyannote.audio not installed"
    echo "Run: task setup-python"
    exit 1
}

echo "✅ Python environment is ready"












