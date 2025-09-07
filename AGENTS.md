# Agent Guidelines for how-cli

## Build/Test/Lint Commands
- **Build**: `make build` or `go build -ldflags="-s -w" -o how .`
- **Test all**: `make test` or `go test -v ./...`
- **Test single**: `go test -v -run TestFunctionName ./...`
- **Lint**: `go vet ./...` and `go fmt ./...`
- **Cross-platform**: `make cross` (outputs to `dist/`)

## Code Style & Conventions
- **Language**: Go 1.25.1, uses Cobra CLI framework and Viper for config
- **Formatting**: Standard `go fmt` - tabs for indentation, no trailing spaces
- **Imports**: Group stdlib, then third-party packages (cobra/viper), blank lines between groups
- **Naming**: CamelCase for public, camelCase for private, descriptive names (e.g. `buildSystemPrompt`, `queryOpenRouter`)
- **Types**: Explicit struct tags for JSON (`json:"field_name"`), interfaces for testability (see `Sys` interface)
- **Error handling**: Return errors up the call stack, use `fmt.Errorf` for wrapping, exit with `os.Exit(1)` at top level
- **Testing**: Use table-driven tests, `httptest` for HTTP mocking, golden files for complex output validation
- **Config**: Use Viper for config management, store in `~/.config/how/config.yaml`

## Project Structure
- Single-package Go CLI tool with main.go containing core logic
- Test files follow `*_test.go` pattern with comprehensive coverage
- No external config rules (no .cursorrules or copilot instructions found)