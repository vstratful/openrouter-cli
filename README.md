# OpenRouter CLI

A terminal-based interface for the [OpenRouter API](https://openrouter.ai/) with interactive chat, image generation, and session management.

## Features

- **Interactive Chat** - Full TUI with markdown rendering, syntax highlighting, and conversation history
- **Single-Turn Mode** - Quick queries directly from the command line
- **Image Generation** - Generate images with configurable aspect ratios and sizes
- **Session Management** - Auto-saved sessions that can be resumed anytime
- **Model Picker** - Browse and filter available models interactively
- **Streaming** - Real-time response streaming with graceful cancellation

## Installation

### From Source

```bash
go install github.com/vstratful/openrouter-cli@latest
```

Or clone and build:

```bash
git clone https://github.com/vstratful/openrouter-cli.git
cd openrouter-cli
go build -o openrouter
```

### Pre-built Binaries

Download from [Releases](https://github.com/vstratful/openrouter-cli/releases) for Linux, macOS, and Windows (AMD64/ARM64).

## Configuration

Set your API key via environment variable:

```bash
export OPENROUTER_API_KEY=sk-or-v1-your-key-here
```

Or create a config file:

| OS      | Path                                             |
|---------|--------------------------------------------------|
| Linux   | `~/.config/openrouter/config.json`               |
| macOS   | `~/Library/Application Support/openrouter/config.json` |
| Windows | `%APPDATA%\openrouter\config.json`               |

```json
{
  "api_key": "sk-or-v1-your-key-here",
  "default_model": "moonshotai/kimi-k2.5",
  "default_image_model": "google/gemini-2.5-flash-image"
}
```

On first run without configuration, you'll be prompted to enter your API key.

## Usage

### Chat

Interactive mode:

```bash
openrouter chat                              # Start with default model
openrouter chat -m anthropic/claude-4.5-sonnet   # Specify a model
```

Single-turn mode:

```bash
openrouter chat -p "Explain Go channels"
openrouter chat -m google/gemini-2.5-flash -p "Hello"
openrouter chat -p "Quick question" --stream=false  # Disable streaming
```

In-chat commands: `/models`, `/resume`, `/new`, `/clear`, `/exit`

### Models

```bash
openrouter models                    # List all models
openrouter models --filter claude    # Filter by name
openrouter models --image-only       # Show image-capable models only
openrouter models --details          # Show pricing and context length
```

### Image Generation

```bash
openrouter image -p "A sunset over mountains" -f sunset.png
openrouter image -p "Portrait" --aspect-ratio 9:16 -f portrait.png
openrouter image -p "Landscape" --size 4K -f hd.png
openrouter image -p "prompt" --base64  # Output raw base64
```

Aspect ratios: `1:1`, `2:3`, `3:2`, `3:4`, `4:3`, `4:5`, `5:4`, `9:16`, `16:9`, `21:9`
Sizes: `1K`, `2K`, `4K`

### Resume Sessions

```bash
openrouter resume          # Pick from saved sessions
openrouter resume --last   # Resume most recent session
openrouter resume <id>     # Resume specific session
```

## For AI Agents

**If you're an AI coding agent (OpenClaw, Claude Code, Cursor, etc.), run this first:**

```bash
openrouter agent-setup
```

This outputs configuration paths, available commands, and usage examples tailored for agent integration. No API key required.

## Keyboard Shortcuts

| Key          | Action                          |
|--------------|---------------------------------|
| `Enter`      | Send message                    |
| `Ctrl+C`     | Quit / Cancel streaming         |
| `Esc`        | Cancel current action           |
| `↑` / `↓`    | Navigate input history          |
| `/`          | Trigger command autocomplete    |

## Development

```bash
# Build
go build -o openrouter

# Run tests
go test ./...

# Lint
go vet ./...

# All at once
go build -o openrouter && go vet ./... && go test ./...
```

## Project Structure

```
├── main.go           # Entry point
├── cmd/              # Cobra commands (chat, models, image, resume, agent-setup)
└── internal/
    ├── api/          # OpenRouter API client, streaming, retry logic
    ├── config/       # Configuration and session management
    └── tui/          # Bubble Tea TUI components
        ├── chat/     # Chat interface
        └── picker/   # Model/session picker
```

## License

MIT
