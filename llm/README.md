# llm

`llm` is an LLM wrapper built on [goai](https://github.com/zendev-sh/goai). It connects to any OpenAI-compatible API and provides a fluent session API for text generation, structured output, and streaming.

## Features

- **Fluent sessions**: Chain `WithSystem`, `AddUser`, `AddAssistant`, and `AddHistory` to build prompts.
- **Text generation**: `Execute` returns plain text responses.
- **Structured output**: `ExecuteTo[T]` parses model output into a typed Go struct.
- **Streaming**: `Stream` returns a channel of text chunks.
- **OpenAI-compatible**: Works with OpenAI, Azure OpenAI, and other compatible endpoints via `LLM_BASE_URL`.
- **Environment config**: Load API key, base URL, and model from environment variables.

## Installation

```bash
go get github.com/SmilingXinyi/gb/llm@latest
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"

	"github.com/SmilingXinyi/gb/llm"
)

type TopicResult struct {
	Topic      string  `json:"topic"`
	Confidence float64 `json:"confidence"`
}

func main() {
	config := llm.DefaultConfig() // reads LLM_API_KEY, LLM_BASE_URL, LLM_MODEL

	// Plain text generation
	reply, err := llm.NewSessionFromConfig(config).
		WithSystem("You are a concise assistant.").
		AddUser("Reply with one short greeting.").
		Execute(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Println("Reply:", reply)

	// Structured output
	result, err := llm.ExecuteTo[TopicResult](context.Background(),
		llm.NewSessionFromConfig(config).
			WithSystem("Classify the user message.").
			AddUser(reply),
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Topic: %s (%.2f)\n", result.Topic, result.Confidence)
}
```

## Configuration

| Variable | Description | Default |
| :--- | :--- | :--- |
| `LLM_API_KEY` | API key for the LLM provider | — |
| `LLM_BASE_URL` | OpenAI-compatible API base URL | `https://api.openai.com/v1` |
| `LLM_MODEL` | Model identifier | `gpt-4o-mini` |

See [config.go](./llm.go) for the full `Config` struct and timeout settings.

## API Overview

| Function / Method | Description |
| :--- | :--- |
| `NewSession(token, baseURL, model)` | Create a session with explicit credentials |
| `NewSessionFromConfig(config)` | Create a session from `Config` |
| `WithSystem(content)` | Set the system instruction |
| `AddUser(content)` / `AddAssistant(content)` | Append messages |
| `AddHistory(messages)` | Inject conversation history |
| `WithTemperature(t)` / `WithMaxOutputTokens(n)` | Adjust sampling parameters |
| `Execute(ctx)` | Generate plain text |
| `ExecuteTo[T](ctx, session)` | Generate structured output into type `T` |
| `Stream(ctx)` | Stream text chunks via a channel |

## Example

```bash
export LLM_API_KEY=sk-...

go run ./examples/basic/
```

## Testing

```bash
cd llm
go test ./...
```
