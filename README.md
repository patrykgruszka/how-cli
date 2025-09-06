# `how` - A Simple AI Assistant for your CLI

`how` is a lightweight, cross-platform command-line tool written in Go that helps you remember shell commands. Just ask it `how` to do something, and it will give you the command you need.

It uses the [OpenRouter.ai](https://openrouter.ai) API, which lets you use various LLMs (including free ones) with a single API key.

## Features

- 🚀 **Fast**: Written in Go for optimal performance
- 🌍 **Cross-platform**: Works on Linux, macOS, and Windows
- 🆓 **Free to use**: Uses free OpenRouter models by default
- 🔒 **Secure**: API keys stored locally in your user config directory
- 📦 **Single binary**: No external dependencies required

## Installation

### Prerequisites

- [Go 1.21+](https://golang.org/doc/install) (for building from source)

### Option 1: Quick Install (Recommended)

1. Clone this repository:
   ```bash
   git clone <your-repo-url>
   cd how
   ```

2. Run the installer:
   ```bash
   chmod +x install.sh
   ./install.sh
   ```

### Option 2: Manual Build

1. Clone and build:
   ```bash
   git clone <your-repo-url>
   cd how
   go mod download
   go build -ldflags="-s -w" -o how .
   ```

2. Move to your PATH:
   ```bash
   sudo mv how /usr/local/bin/
   ```

### Option 3: Using Make

```bash
make install
```

### Option 4: Cross-platform Builds

Build for all platforms:
```bash
make cross
```

This creates binaries in the `dist/` directory for:
- Linux (amd64, arm64, 386)
- macOS (amd64, arm64)
- Windows (amd64, 386)

## First-Time Setup

Before using the tool, configure it with your OpenRouter API key:

1. Get a free API key from [openrouter.ai/keys](https://openrouter.ai/keys)
2. Run the setup command:
   ```bash
   how setup
   ```
3. Paste your key when prompted

Your API key will be stored securely in your user's config directory.

## Usage

Simply ask `how` to do something:

```bash
$ how find all files named docker-compose.yml
find . -type f -name "docker-compose.yml"

$ how check memory usage in a human readable format
free -h

$ how count lines in a file
wc -l <filename>

$ how kill process on port 8080
lsof -ti:8080 | xargs kill -9

$ how compress a directory with tar
tar -czf archive.tar.gz directory/
```

### Advanced Usage

Use a different model for a single query:
```bash
how --model "anthropic/claude-3-haiku" list running docker containers
```

Persist a default model to use for all queries (can be overridden by --model):
```bash
how set-model "anthropic/claude-3-haiku"
```

Model selection precedence:
- --model flag
- saved default from config (set via how set-model)
- built-in default: mistralai/mistral-7b-instruct:free

## Configuration

The configuration file is stored at:
- **Linux/macOS**: `~/.config/how/config.yaml`
- **Windows**: `%APPDATA%/how/config.yaml`

Keys used:
- api_key: your OpenRouter API key
- model: your saved default model (optional)

## Development

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make cross

# Install locally
make install

# Clean build artifacts
make clean
```
