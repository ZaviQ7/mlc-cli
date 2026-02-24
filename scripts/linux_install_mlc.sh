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

# install MLC Python package
cd mlc-llm/python

pip install -e .
cd ../..

# flashinfer-python==0.4.0 (pulled in by mlc_llm) pins apache-tvm-ffi==0.1.0b15
# which downgrades the version installed by linux_install_tvm.sh and breaks the
# tvm Python package. Reinstall the correct version from the selected TVM source.
if [ "${TVM_SOURCE}" = "relax" ] || [ "${TVM_SOURCE}" = "custom" ]; then
    TVM_DIR="${REPO_ROOT}/tvm"
else
    TVM_DIR="${REPO_ROOT}/mlc-llm/3rdparty/tvm"
fi

TVM_FFI_DIR="${TVM_DIR}/3rdparty/tvm-ffi"
if [ -d "${TVM_FFI_DIR}" ]; then
    pip install --force-reinstall -e "${TVM_FFI_DIR}"
else
    echo "Warning: ${TVM_FFI_DIR} not found, falling back to reinstalling TVM wheel"
    pip install --force-reinstall "${WHEELS_DIR}"/tvm-*.whl
fi
