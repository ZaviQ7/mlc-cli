#!/usr/bin/env bash
set -e

source "$(conda info --base)/etc/profile.d/conda.sh"

# Args
CLI_VENV="${1:-mlc-cli-venv}"
MODEL_PATH="$2"
QUANT_TYPE="$3"
DEVICE="${4:-metal}"
OUTPUT_PATH="$5"

if [ -z "$MODEL_PATH" ] || [ -z "$QUANT_TYPE" ] || [ -z "$OUTPUT_PATH" ]; then
    echo "Usage: $0 <env> <model_path> <quant_type> <device> <output_path>"
    exit 1
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
