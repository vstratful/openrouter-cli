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

## Project Structure

- `main.go` - Entry point
- `cmd/` - Cobra command definitions (uses spf13/cobra)
