#!/usr/bin/env bash
set -e  # Exit on error

# Args
BUILD_VENV="${1:-mlc-build-venv}"
CUDA="${2:-n}"
ROCM="${3:-n}"
VULKAN="${4:-n}"
METAL="${5:-y}"
OPEN_CL="${6:-n}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
WHEELS_DIR="${REPO_ROOT}/wheels"
# MLC-LLM should use its bundled TVM submodule (3rdparty/tvm), not standalone TVM
TVM_SOURCE_DIR=""

source "$(conda info --base)/etc/profile.d/conda.sh"

# create the conda environment with build dependency
conda create -y -n "${BUILD_VENV}" -c conda-forge \
    "cmake>=3.24" \
    rust \
    git \
    zstd \
    python=3.13
# enter the build environment
conda activate "${BUILD_VENV}"

# Set library path for cmake to find zstd
export DYLD_LIBRARY_PATH="$CONDA_PREFIX/lib:$DYLD_LIBRARY_PATH"

# Ensure root tvm is on the mlc branch (mlc-ai/relax, compatible with mlc-llm)
if [ ! -d "${TVM_SOURCE_DIR}" ]; then
    echo "Cloning TVM (mlc-ai/relax) on mlc branch..."
    git clone --recursive -b mlc https://github.com/mlc-ai/relax.git "${TVM_SOURCE_DIR}"
elif [ "$(git -C "${TVM_SOURCE_DIR}" rev-parse --abbrev-ref HEAD)" != "mlc" ]; then
    echo "Switching root tvm to mlc branch (mlc-ai/relax)..."
    git -C "${TVM_SOURCE_DIR}" remote set-url origin https://github.com/mlc-ai/relax.git
    git -C "${TVM_SOURCE_DIR}" fetch origin mlc
    git -C "${TVM_SOURCE_DIR}" checkout mlc
    git -C "${TVM_SOURCE_DIR}" submodule update --init --recursive
else
    echo "Root tvm is already on mlc branch."
fi

# Check if mlc-llm directory exists
# Set macOS deployment target to current OS version
export MACOSX_DEPLOYMENT_TARGET=$(sw_vers -productVersion | cut -d. -f1)

# clone from GitHub (or use existing)
if [ ! -d "mlc-llm" ]; then
    echo "Cloning mlc-llm..."
    git clone --recursive https://github.com/mlc-ai/mlc-llm.git
else
    echo "mlc-llm directory already exists, skipping clone."
fi
cd mlc-llm/

# create build directory
mkdir -p build && cd build

# generate build configuration
printf "%s\n%s\n%s\n%s\n%s\n%s\n\n\n" \
    "${TVM_SOURCE_DIR}" \
    "${CUDA}" \
    "${ROCM}" \
    "${VULKAN}" \
    "${METAL}" \
    "${OPEN_CL}" \
    | python3 ../cmake/gen_cmake_config.py

# build mlc_llm libraries
cmake .. -DCMAKE_POLICY_VERSION_MINIMUM=3.5 && make -j4
cd ..

# Build wheel and copy to wheels directory
mkdir -p "${WHEELS_DIR}"

cd python
python -m pip install build
python -m build --wheel --outdir "${WHEELS_DIR}"
cd ..

echo "MLC-LLM wheel created in ${WHEELS_DIR}"

conda deactivate

