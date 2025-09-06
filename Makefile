# Makefile for 'how' CLI tool

.PHONY: build install clean test run help

# Variables
BINARY_NAME=how
VERSION?=dev
BUILD_DIR=dist
INSTALL_PATH=/usr/local/bin

# Default target
help:
	@echo "Available targets:"
	@echo "  build     - Build the binary for current platform"
	@echo "  install   - Install the binary to $(INSTALL_PATH)"
	@echo "  clean     - Clean build artifacts"
	@echo "  test      - Run tests"
	@echo "  run       - Run the application"
	@echo "  cross     - Build for all platforms"
	@echo "  help      - Show this help message"

# Build for current platform
build:
	@echo "🚀 Building $(BINARY_NAME)..."
	@go build -ldflags="-s -w" -o $(BINARY_NAME) .
	@echo "✅ Build completed: $(BINARY_NAME)"

# Install binary
install: build
	@echo "📦 Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	@sudo cp $(BINARY_NAME) $(INSTALL_PATH)/
	@echo "✅ $(BINARY_NAME) installed successfully!"
	@echo ""
	@echo "Next steps:"
	@echo "1. Get a free API key from https://openrouter.ai/keys"
	@echo "2. Run '$(BINARY_NAME) setup' to configure your key"
	@echo "3. Try: $(BINARY_NAME) list files by size"

# Cross-platform build
cross:
	@chmod +x build.sh
	@./build.sh $(VERSION)

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -f $(BINARY_NAME)
	@rm -rf $(BUILD_DIR)
	@echo "✅ Clean completed"

# Run tests
test:
	@echo "🧪 Running tests..."
	@go test -v ./...

# Run the application
run:
	@go run . $(ARGS)

# Download dependencies
deps:
	@echo "📥 Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✅ Dependencies updated"