#!/bin/bash

# Build script for cross-platform compilation of 'how' CLI tool

set -e

VERSION=${1:-"dev"}
OUTPUT_DIR="dist"

echo "🚀 Building 'how' CLI tool (version: $VERSION)..."

# Clean previous builds
rm -rf $OUTPUT_DIR
mkdir -p $OUTPUT_DIR

# Build for different platforms
declare -a platforms=(
    "linux/amd64"
    "linux/arm64"
    "linux/386"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/386"
)

for platform in "${platforms[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"
    
    output_name="how"
    if [ $GOOS = "windows" ]; then
        output_name="how.exe"
    fi
    
    output_path="$OUTPUT_DIR/${GOOS}_${GOARCH}/$output_name"
    
    echo "📦 Building for $GOOS/$GOARCH..."
    
    mkdir -p "$OUTPUT_DIR/${GOOS}_${GOARCH}"
    
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="-s -w" -o "$output_path" .
    
    if [ $? -eq 0 ]; then
        echo "✅ Successfully built: $output_path"
    else
        echo "❌ Failed to build for $GOOS/$GOARCH"
        exit 1
    fi
done

echo ""
echo "🎉 All builds completed successfully!"
echo "📂 Binaries are available in the '$OUTPUT_DIR' directory"
echo ""
echo "Available binaries:"
find $OUTPUT_DIR -name "how*" -type f | sort