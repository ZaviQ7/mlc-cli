#!/usr/bin/env bash
set -e

# Downloads the go binary and
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz

# Remove the existing go binary
sudo rm -rf /usr/local/go

# Install the new go binary
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz

# Add go to the PATH permanently in ~/.bashrc
if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
    echo "" >> ~/.bashrc
    echo "# Go environment variables" >> ~/.bashrc
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    echo "Added Go to PATH in ~/.bashrc"
else
    echo "Go PATH already exists in ~/.bashrc"
fi

# Export for current session
export PATH=$PATH:/usr/local/go/bin

# Verify the installation
go version

# Clean up
rm -f go1.24.0.linux-amd64.tar.gz

# Reload shell
source ~/.bashrc

echo "Go installed successfully and added to ~/.bashrc"