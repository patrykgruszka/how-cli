# `how` - Lightweight AI assistant for your CLI

Enter **`how`** – the slick, AI-powered sidekick that turns your "WTF do I type?" moments into commands. Built in Go for speed, this cross-platform tool taps OpenRouter.ai's API to generate commands via top LLMs (even free ones!).

## Features
- ⚡ Blazing fast and dependency-free.
- 🌐 Runs on Linux, macOS, Windows.
- 💸 Free models by default—no cost to start.

## 🛠️ Quick Install
1. Clone the repo:
   ```bash
   git clone <your-repo-url>
   cd how
   ```
2. Run the installer (handles build and PATH):
   ```bash
   chmod +x install.sh
   ./install.sh
   ```

**Alternatives**: Build manually with `go build` or use `make install`. For cross-platform binaries, run `make cross` (outputs to `dist/`).

## 🔑 Setup
Grab a free API key from [openrouter.ai/keys](https://openrouter.ai/keys), then:
```bash
how setup
```
Paste your key—it's stored securely in `~/.config/how/config.yaml` (or equivalent on Windows).

## 🎯 Usage
Ask away:
```bash
$ how find all files named docker-compose.yml
find . -type f -name "docker-compose.yml"

$ how check memory usage in a human readable format
free -h

$ how kill process on port 8080
lsof -ti:8080 | xargs kill -9
```

**Pro Tip**: Override the model with `--model "anthropic/claude-3-haiku"` or set a default via `how set-model`. Defaults to `mistralai/mistral-7b-instruct:free`.

## ⚙️ Config & More
Tweak your config file if needed (e.g., for a custom default model). For dev builds or contributions:
- `make build` for local.
