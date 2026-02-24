#!/usr/bin/env bash
set -e  # Exit on error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
WHEELS_DIR="${REPO_ROOT}/wheels"
CUDA_COMPUTE_CAPABILITY="${1:-86}"
TVM_SOURCE="${2:-bundled}"  # bundled or custom
BUILD_WHEELS="${3:-y}"

source "$(conda info --base)/etc/profile.d/conda.sh"

conda env remove -y -n tvm-build-venv || true
conda create -y -n tvm-build-venv -c conda-forge \
    "llvmdev=19" \
    "cmake>=3.24" \
    git \
    zstd \
    python=3.13

conda activate tvm-build-venv

# Set library path for linker to find zstd and libxml2
export LD_LIBRARY_PATH="$CONDA_PREFIX/lib:${LD_LIBRARY_PATH:-}"

# Determine TVM directory based on source selection
if [ "${TVM_SOURCE}" = "custom" ]; then
    TVM_DIR="${REPO_ROOT}/tvm"
    echo "Using custom TVM from ${TVM_DIR}"
    if [ ! -d "${TVM_DIR}" ]; then
        echo "Error: Custom TVM directory not found at ${TVM_DIR}"
        echo "Please clone TVM to ${TVM_DIR} or select bundled TVM option"
        exit 1
    fi
elif [ "${TVM_SOURCE}" = "relax" ]; then
    TVM_DIR="${REPO_ROOT}/tvm"
    echo "Using mlc-ai/relax (mlc branch) at ${TVM_DIR}"
    if [ ! -d "${TVM_DIR}" ]; then
        echo "Cloning mlc-ai/relax on mlc branch..."
        git clone --recursive -b mlc https://github.com/mlc-ai/relax.git "${TVM_DIR}"
    elif [ "$(git -C "${TVM_DIR}" rev-parse --abbrev-ref HEAD)" != "mlc" ]; then
        echo "Switching TVM to mlc branch (mlc-ai/relax)..."
        git -C "${TVM_DIR}" remote set-url origin https://github.com/mlc-ai/relax.git
        git -C "${TVM_DIR}" fetch origin mlc
        git -C "${TVM_DIR}" checkout mlc
        git -C "${TVM_DIR}" submodule update --init --recursive
    else
        echo "TVM is already on mlc branch."
    fi
else
    # Clone mlc-llm if it doesn't exist (for bundled TVM)
    MLC_LLM_DIR="${REPO_ROOT}/mlc-llm"
    if [ ! -d "${MLC_LLM_DIR}" ]; then
        echo "mlc-llm not found, cloning from https://github.com/mlc-ai/mlc-llm..."
        git clone --recursive https://github.com/mlc-ai/mlc-llm "${MLC_LLM_DIR}"
    fi
    
    TVM_DIR="${REPO_ROOT}/mlc-llm/3rdparty/tvm"
    echo "Using bundled TVM from ${TVM_DIR}"
    if [ ! -d "${TVM_DIR}" ]; then
        echo "Error: Bundled TVM directory not found at ${TVM_DIR}"
        echo "Initializing submodules..."
        git -C "${MLC_LLM_DIR}" submodule update --init --recursive
    fi
fi

cd "${TVM_DIR}"
# create the build directory
rm -rf build && mkdir build && cd build
# specify build requirements in `config.cmake`
cp ../cmake/config.cmake .

# controls default compilation flags (use sed to replace existing values)
sed -i 's/set(CMAKE_BUILD_TYPE .*/set(CMAKE_BUILD_TYPE Release)/' config.cmake
# LLVM is a must dependency
sed -i 's|set(USE_LLVM .*)|set(USE_LLVM "llvm-config --ignore-libllvm --link-static")|' config.cmake
sed -i 's/set(HIDE_PRIVATE_SYMBOLS .*/set(HIDE_PRIVATE_SYMBOLS ON)/' config.cmake
# GPU SDKs - enable Metal for macOS
sed -i 's/set(USE_CUDA .*/set(USE_CUDA ON)/' config.cmake
sed -i 's/set(USE_ROCM .*/set(USE_ROCM OFF)/' config.cmake
sed -i 's/set(USE_METAL .*/set(USE_METAL OFF)/' config.cmake
sed -i 's/set(USE_VULKAN .*/set(USE_VULKAN OFF)/' config.cmake
sed -i 's/set(USE_OPENCL .*/set(USE_OPENCL OFF)/' config.cmake

# Below are options for CUDA, turn on if needed
# CUDA_ARCH is the cuda compute capability of your GPU.
# Examples: 89 for 4090, 90a for H100/H200, 100a for B200.
# Reference: https://developer.nvidia.com/cuda-gpus
sed -i "s/set(CMAKE_CUDA_ARCHITECTURES .*/set(CMAKE_CUDA_ARCHITECTURES ${CUDA_COMPUTE_CAPABILITY})/" config.cmake
sed -i 's/set(USE_CUBLAS .*/set(USE_CUBLAS ON)/' config.cmake
sed -i 's/set(USE_CUTLASS .*/set(USE_CUTLASS OFF)/' config.cmake # known compile failures on CUDA 12+
sed -i 's/set(USE_THRUST .*/set(USE_THRUST OFF)/' config.cmake # known CHECK macro errors
sed -i 's/set(USE_NVTX .*/set(USE_NVTX OFF)/' config.cmake

cmake ..
make -j$(nproc)
cd ..

if [ "${BUILD_WHEELS}" = "y" ]; then
    # Build wheel and copy to wheels directory
    mkdir -p "${WHEELS_DIR}"

    # Clean CMake cache and Makefiles to avoid Make/Ninja conflict
    # but keep the compiled libraries in build/
    cd build
    rm -f Makefile CMakeCache.txt cmake_install.cmake
    rm -rf CMakeFiles
    cd ..

    # Build TVM wheel from the tvm root directory (where pyproject.toml is)
    python -m pip install build
    python -m build --wheel --outdir "${WHEELS_DIR}"

    echo "TVM wheel created in ${WHEELS_DIR}"
else
    echo "Skipping TVM wheel build."
fi

