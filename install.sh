#!/bin/bash

echo "🚀 Installing 'how' CLI tool..."

# --- Check for dependencies ---
if ! command -v go &> /dev/null
then
	echo "Error: Go is not installed. Please install it first."
	echo "Visit: https://golang.org/doc/install"
	exit 1
fi

# --- Download dependencies ---
echo "📥 Downloading Go dependencies..."
if ! go mod download; then
	echo "Error: Failed to download dependencies."
	exit 1
fi

# --- Build the application ---
echo "🔨 Building the application..."
if ! go build -ldflags="-s -w" -o how .; then
	echo "Error: Failed to build the application."
	exit 1
fi

# --- Install the binary ---
INSTALL_PATH="/usr/local/bin/how"
echo "📦 Installing binary to $INSTALL_PATH..."

if sudo cp how "$INSTALL_PATH"; then
	echo "✅ 'how' was installed successfully!"
	echo ""
	echo "Next steps:"
	echo "1. Get a free API key from https://openrouter.ai/keys"
	echo "2. Run 'how setup' to configure your key"
	echo "3. You're ready! Try: how list files by size"
	
	# Clean up local binary
	rm -f how
else
	echo "Error: Failed to copy binary to $INSTALL_PATH."
	echo "Please check permissions or try running the script with sudo."
	exit 1
fi