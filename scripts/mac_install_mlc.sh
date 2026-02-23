#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
WHEELS_DIR="${REPO_ROOT}/wheels"

source "$(conda info --base)/etc/profile.d/conda.sh"

# Args
CLI_VENV="${1:-mlc-cli-venv}"
TVM_SOURCE="${2:-bundled}"  # bundled, relax, or custom

if ! conda env list | awk '{print $1}' | grep -qx "${CLI_VENV}"; then
    conda create -n "${CLI_VENV}" -c conda-forge \
        "cmake>=3.24" \
        rust \
        git \
        python=3.13 \
        psutil
fi

conda activate "${CLI_VENV}"

# Verify Python version matches wheel requirement
PYTHON_VERSION=$(python --version | awk '{print $2}' | cut -d. -f1,2)
if [ "$PYTHON_VERSION" != "3.13" ]; then
    echo "Error: mlc-cli-venv has Python $PYTHON_VERSION but wheel requires Python 3.13"
    echo "Recreating environment with correct Python version..."
    conda deactivate
    conda env remove -n "${CLI_VENV}" -y
    conda create -n "${CLI_VENV}" -c conda-forge \
        "cmake>=3.24" \
        rust \
        git \
        python=3.13 \
        psutil -y
    conda activate "${CLI_VENV}"
fi

# Install pre-built MLC wheel from wheels directory
pip install --force-reinstall "${WHEELS_DIR}"/mlc_llm-*.whl
