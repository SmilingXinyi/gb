# GB Log

`gb/log` is a high-performance logging package based on [zerolog](https://github.com/rs/zerolog).

## Features

- **Structured Logging**: Native JSON support.
- **Colorful Console Output**: Easy-to-read colored logs in development mode.
- **Log Rotation**: Integrated with lumberjack for automatic log file rotation based on size and time.
- **Seq Ingestion**: Optional CLEF writer for sending structured logs to [Seq](https://datalust.co/seq).
- **Caller Tracking**: Automatically simplifies caller paths (relative to the project root).
- **Minimalist API**: Provides global shortcut functions (`Info()`, `Error()`, etc.).

## Installation

```bash
go get github.com/SmilingXinyi/gb/log@latest
```

## Quick Start

```go
package main

import (
	"github.com/SmilingXinyi/gb/log"
)

func main() {
	// 1. Initialize configuration
	config := log.DefaultConfig()
	config.File.Filename = "app.log" // Output to both console and file
	
	// 2. Setup logging
	log.Setup(config)

	// 3. Use logging
	log.Info().Str("module", "main").Msg("Hello GB Log!")
	
	// 4. Modular logging
	authLog := log.Module("auth")
	authLog.Debug().Msg("User login attempt")
}
```

### Seq Output

Seq output is disabled by default. Enable it in configuration:

```go
import "time"

config := log.DefaultConfig()
config.Seq = log.SeqConfig{
	Enabled:       true,
	Endpoint:      "http://localhost:5342/ingest/clef",
	Level:         "info",
	Application:   "my-service",
	BatchSize:     50,
	FlushInterval: 2 * time.Second,
}
log.Setup(config)
defer log.Shutdown()
```

See [examples/seq-basic](./examples/seq-basic/main.go) for a complete example.

## Configuration

See the `LogConfig` struct definition in [config.go](./config.go) for details.
