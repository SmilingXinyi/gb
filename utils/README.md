# utils

`utils` provides shared utility functions used across GB modules, including YAML configuration loading and JSON Schema generation for LLM structured output.

## Features

- **YAML loading**: Unmarshal YAML from files or any `io.Reader` into a typed struct.
- **LLM schema builder**: Convert a simplified JSON description into a JSON Schema string optimized for OpenAI strict mode (Structured Outputs).

## Installation

```bash
go get github.com/SmilingXinyi/gb/utils@latest
```

## Quick Start

### YAML loading

```go
package main

import (
	"fmt"

	"github.com/SmilingXinyi/gb/utils"
)

type AppConfig struct {
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`
}

func main() {
	var config AppConfig
	if err := utils.LoadFile("config.yaml", &config); err != nil {
		panic(err)
	}
	fmt.Printf("Server: %s:%d\n", config.Server.Host, config.Server.Port)
}
```

### LLM structured schema generation

`BuildLLMStructure` accepts a simplified JSON description and produces a JSON Schema suitable for OpenAI strict mode:

```go
input := `{
  "topic": "string enum(news,sports,tech)",
  "confidence": "number",
  "tags": ["string"]
}`

schema, err := utils.BuildLLMStructure(input)
if err != nil {
	panic(err)
}
fmt.Println(schema)
```

Supported type hints in the simplified format include `string`, `number`, `integer`, `boolean`, `array`, and `object`. Enum values can be specified as `string enum(a,b,c)`.

## API Overview

| Function | Description |
| :--- | :--- |
| `Load(reader, out)` | Unmarshal YAML from an `io.Reader` into `out` |
| `LoadFile(path, out)` | Unmarshal YAML from a file into `out` |
| `BuildLLMStructure(input)` | Convert a simplified JSON description to a JSON Schema string |

## Example

```bash
go run ./examples/basic/
```

## Testing

```bash
cd utils
go test ./...
```
