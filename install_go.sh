#!/usr/bin/env bash
set -e

# Downloads the go binary and
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz

# Remove the existing go binary
sudo rm -rf /usr/local/go

# Install the new go binary
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz

# Add go to the PATH
export PATH=$PATH:/usr/local/go/bin

# Verify the installation
go version

# Clean up
tar -xzf go1.24.0.linux-amd64.tar.gz

echo "Go installed successfully"