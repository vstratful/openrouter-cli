# OpenRouter CLI

A Go CLI tool for interacting with the OpenRouter API.

## Requirements

- Go 1.25.6+
- `OPENROUTER_API_KEY` environment variable

## Build

```bash
go build -o openrouter
```

## Run

```bash
export OPENROUTER_API_KEY=your-key
./openrouter --model google/gemini-2.5-flash --prompt "Hello"
```

## After Making Changes

Always run these commands after making changes:

```bash
go build -o openrouter && go vet ./... && go test ./...
```

## Project Structure

- `main.go` - Entry point
- `cmd/` - Cobra command definitions (thin wrappers using spf13/cobra)
- `internal/api/` - API client with interface, streaming, retry logic
- `internal/config/` - Configuration and session management
- `internal/tui/` - Bubble Tea TUI components
  - `chat/` - Chat interface (model, update, view, history, autocomplete)
  - `picker/` - Generic picker for models and sessions
