# OpenRouter CLI

Go CLI for OpenRouter API with interactive TUI chat.

## Development

```bash
go build -o openrouter && go vet ./... && go test ./...
```

## Project Structure

- `main.go` - Entry point
- `cmd/` - Cobra commands
- `internal/api/` - API client (streaming, retry)
- `internal/config/` - Config and session management
- `internal/tui/` - Bubble Tea components
  - `chat/` - Interactive chat
  - `picker/` - Model/session picker
