#!/usr/bin/env bash
set -e

source "$(conda info --base)/etc/profile.d/conda.sh"

# Args
CLI_VENV="${1:-mlc-cli-venv}"
MODEL_PATH="$2"
QUANT_TYPE="$3"
DEVICE="${4:-cuda}"
OUTPUT_PATH="$5"

if [ -z "$MODEL_PATH" ] || [ -z "$QUANT_TYPE" ] || [ -z "$OUTPUT_PATH" ]; then
    echo "Usage: $0 <env> <model_path> <quant_type> <device> <output_path>"
    exit 1
fi

# Set CUDA environment variables if using cuda device
if [ "${DEVICE}" = "cuda" ]; then
    if command -v nvcc >/dev/null 2>&1; then
        NVCC_PATH="$(command -v nvcc)"
    elif [[ -x /usr/local/cuda/bin/nvcc ]]; then
        NVCC_PATH="/usr/local/cuda/bin/nvcc"
    elif [[ -x /usr/bin/nvcc ]]; then
        NVCC_PATH="/usr/bin/nvcc"
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
fi

conda activate "${CLI_VENV}"

echo "🔧 Compiling model library..."
echo "   Model:        ${MODEL_PATH}"
echo "   Quantization: ${QUANT_TYPE}"
echo "   Device:       ${DEVICE}"
echo "   Output:       ${OUTPUT_PATH}"

mkdir -p "$(dirname "${OUTPUT_PATH}")"

python -m mlc_llm compile "${MODEL_PATH}" \
    --quantization "${QUANT_TYPE}" \
    --device "${DEVICE}" \
    -o "${OUTPUT_PATH}"

conda deactivate
