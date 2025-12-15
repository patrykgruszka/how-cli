# `how` - Lightweight AI assistant for your CLI

Enter **`how`** – the slick, AI-powered sidekick that turns your "WTF do I type?" moments into commands. Built in Go for speed, this cross-platform tool taps into AI to generate commands via top LLMs.

## Features
- Fast and dependency-free.
- Runs on Linux, macOS, Windows.
- Support for **OpenRouter** (default) and **OpenAI**.
- Configurable models and providers.
- Optional command execution with confirmation.
- Saves a local history of generated commands.

## Usage

### Generate a command
Ask away:
```bash
$ how find all files named docker-compose.yml
find . -type f -name "docker-compose.yml"

$ how check memory usage in a human readable format
free -h

$ how kill process on port 8080
lsof -ti:8080 | xargs kill -9
```

**Pro Tip**: Override the model with `--model "google/gemini-2.5-flash"` or set a default via `how set-model`. Defaults to `anthropic/claude-haiku-4.5`.

### Run the generated command (optional)
If you want `how` to execute what it generates:

```bash
$ how --run find all files named docker-compose.yml
find . -type f -name "docker-compose.yml"
# Prompts: Run this command? [y/N]
```

Skip confirmation (use carefully):

```bash
$ how --run --yes kill process on port 8080
lsof -ti:8080 | xargs kill -9
```

### Reuse the last generated command
Print the last generated command:

```bash
$ how last
echo hi
```

Run the last generated command (with confirmation unless `--yes`):

```bash
$ how last --run
$ how last --run --yes
```

## Install

### Download Pre-built Binary (Recommended)
Download the latest release for your platform from [GitHub Releases](https://github.com/patrykgruszka/how-cli/releases):

1. Download the archive for your OS/architecture
2. Extract the binary:
   ```bash
   # Linux/macOS
   tar -xzf how-v*.tar.gz

   # Windows
   # Extract the .zip file
   ```
3. Move to your PATH:
   ```bash
   # Linux/macOS
   sudo mv how /usr/local/bin/

   # Windows
   # Move how.exe to a directory in your PATH
   ```

### Arch Linux (AUR)
For Arch Linux users, install via AUR:
```bash
yay -S how-cli
```

### Build from Source
1. Clone the repo:
   ```bash
   git clone https://github.com/patrykgruszka/how-cli.git
   cd how-cli
   ```
2. Run the installer (handles build and PATH):
   ```bash
   chmod +x install.sh
   ./install.sh
   ```

**Alternatives**: Build manually with `go build` or use `make install`. For cross-platform binaries, run `make cross` (outputs to `dist/`).

## Setup
You need an API key.
1. OpenRouter (Default): Get a key from [openrouter.ai/keys](https://openrouter.ai/keys).
2. OpenAI: Get a key from [platform.openai.com](https://platform.openai.com).

Run setup to select your provider and save your key:
```bash
how setup
```

Paste your key—it's stored in `~/.config/how/config.yaml` (or equivalent on Windows).

## Config & History

### Config
Variables are stored in `~/.config/how/config.yaml`.
- `provider`: `openrouter` (default) or `openai`
- `api_key`: your API key
- `model`: default model ID

### History
Generated commands are saved locally to:
- `~/.config/how/history.jsonl` (or equivalent on Windows)

This is used by `how last`.

## Flags
- `--model`: override the configured/default model for a single invocation
- `--run`: execute the generated command (prompts for confirmation)
- `--yes`: skip confirmation when used with `--run`
- `--debug`: print debug information (provider, endpoint, model, prompt)

