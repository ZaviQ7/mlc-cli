#!/bin/bash
set -e  # Exit on error

source "$(conda info --base)/etc/profile.d/conda.sh"

# Accept parameters
CLI_VENV="${1:-mlc-cli-venv}"
MODEL_URL="${2}"
MODEL_NAME="${3}"
DEVICE="${4:-cuda}"
OVERRIDES="${5}"

# Set CUDA environment variables
DEVICE_FLAG=""
if [ "${DEVICE}" = "cuda" ]; then
    # Dynamically find nvcc to set correct CUDA paths
    if command -v nvcc > /dev/null 2>&1; then
        NVCC_PATH="$(command -v nvcc)"
    elif [[ -x /usr/local/cuda/bin/nvcc ]]; then
        NVCC_PATH="/usr/local/cuda/bin/nvcc"
    else
        echo "Error: nvcc not found. Please install the CUDA toolkit or add it to PATH."
        exit 1
    fi
    NVCC_REAL="$(readlink -f "${NVCC_PATH}" 2>/dev/null || true)"
    NVCC_REAL="${NVCC_REAL:-${NVCC_PATH}}"
    CUDA_BIN_DIR="$(dirname "${NVCC_REAL}")"
    CUDA_HOME="$(dirname "${CUDA_BIN_DIR}")"
    export PATH="${CUDA_BIN_DIR}:${PATH}"
    export CUDACXX="${NVCC_REAL}"
    export CUDA_HOME="${CUDA_HOME}"
    if [[ -d "${CUDA_HOME}/lib64" ]]; then
        export LD_LIBRARY_PATH="${CUDA_HOME}/lib64:${LD_LIBRARY_PATH:-}"
    fi
    DEVICE_FLAG="--device cuda"
elif [ "${DEVICE}" = "cpu" ] || [ "${DEVICE}" = "none" ] || [ -z "${DEVICE}" ]; then
    DEVICE_FLAG=""
else
    DEVICE_FLAG="--device ${DEVICE}"
fi

conda activate ${CLI_VENV}
mkdir -p models
if [ -z "${MODEL_NAME}" ]; then
    if [ -d "models" ]; then
        MODEL_NAME=$(ls -1 models 2>/dev/null | head -n 1)
    fi
fi
MODEL_PATH="models/${MODEL_NAME}"

# Clone model if URL is provided and model doesn't exist
if [ -n "${MODEL_URL}" ]; then
    if [ ! -d "${MODEL_PATH}" ]; then
        echo "Cloning model from HuggingFace..."
        cd models
        git clone ${MODEL_URL}
        cd ${MODEL_NAME}
        git lfs pull
        cd ../..
    else
        echo "Model already exists, skipping download..."
    fi
else
    echo "Using local model: ${MODEL_NAME}"
fi

# Run the model
if [ -z "${MODEL_NAME}" ] || [ ! -d "${MODEL_PATH}" ]; then
    echo "Error: MODEL_NAME not provided or model directory not found: ${MODEL_PATH}"
    echo "Available models:"
    ls -1 models 2>/dev/null || true
    conda deactivate
    exit 1
fi
cd "${MODEL_PATH}"
if [ -n "${OVERRIDES}" ]; then
    if command -v mlc_llm >/dev/null 2>&1; then
        MLC_JIT_POLICY=REDO mlc_llm chat . ${DEVICE_FLAG} --overrides "${OVERRIDES}"
    else
        MLC_JIT_POLICY=REDO python -m mlc_llm chat . ${DEVICE_FLAG} --overrides "${OVERRIDES}"
    fi
else
    if command -v mlc_llm >/dev/null 2>&1; then
        MLC_JIT_POLICY=REDO mlc_llm chat . ${DEVICE_FLAG}
    else
        MLC_JIT_POLICY=REDO python -m mlc_llm chat . ${DEVICE_FLAG}
    fi
fi

conda deactivate
