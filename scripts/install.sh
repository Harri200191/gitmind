#!/bin/bash

# Install script for gitmind
# This is a placeholder - customize as needed for your distribution method

set -euo pipefail

BINARY_NAME="gitmind"
INSTALL_DIR="/usr/local/bin"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is required but not installed."
    echo "Please install Go from https://golang.org/dl/"
    exit 1
fi

# Build the binary
echo "Building $BINARY_NAME..."
make build

# Install the binary
echo "Installing $BINARY_NAME to $INSTALL_DIR..."
sudo install -m 0755 dist/$BINARY_NAME $INSTALL_DIR/$BINARY_NAME

echo "$BINARY_NAME installed successfully!"
echo "You can now run: $BINARY_NAME --help"