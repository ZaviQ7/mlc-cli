#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
WHEELS_DIR="${REPO_ROOT}/wheels"

source "$(conda info --base)/etc/profile.d/conda.sh"

# Args
CLI_VENV="${1:-mlc-cli-venv}"

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
# tvm Python package. Reinstall the correct version from the local tvm source.
pip install --force-reinstall -e tvm/3rdparty/tvm-ffi
