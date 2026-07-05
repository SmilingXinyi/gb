# jv

`jv` is a JSON Schema validator for Go. It supports embedded schemas shipped with the module and ad-hoc schema content passed at runtime.

## Features

- **Embedded schemas**: Built-in schemas under `schemas/` (e.g. `user.json`), loaded via `go:embed`.
- **Custom schemas**: Validate against arbitrary schema content with `ValidateWithSchema`.
- **Reusable validator**: Create a `Validator` instance for repeated validation.
- **Schema discovery**: List and read embedded schema files programmatically.

## Installation

```bash
go get github.com/SmilingXinyi/gb/jv@latest
```

## Quick Start

```go
package main

import (
	"fmt"

	"github.com/SmilingXinyi/gb/jv"
)

func main() {
	// Validate against an embedded schema
	validJSON := `{"name": "Alice", "age": 25, "email": "alice@example.com"}`
	if err := jv.Validate(validJSON, "user.json"); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
	} else {
		fmt.Println("Valid!")
	}

	// Validate against custom schema content
	customSchema := `{
		"type": "object",
		"properties": {
			"score": { "type": "number" }
		}
	}`
	if err := jv.ValidateWithSchema(`{"score": 95.5}`, customSchema); err != nil {
		panic(err)
	}

	// List available embedded schemas
	schemas, _ := jv.ListSchemas()
	fmt.Println("Available schemas:", schemas)
}
```

## API Overview

| Function | Description |
| :--- | :--- |
| `Validate(jsonString, schemaName)` | Convenience validation against an embedded schema |
| `NewValidator()` | Create a reusable validator instance |
| `(*Validator).Validate(jsonString, schemaName)` | Validate using an embedded schema |
| `ValidateWithSchema(jsonString, schemaContent)` | Validate against inline schema content |
| `ListSchemas()` | Return names of all embedded schema files |
| `GetSchema(schemaName)` | Read the raw content of an embedded schema |

## Adding Embedded Schemas

Place JSON Schema files in the `schemas/` directory. They are compiled into the binary via `go:embed` and referenced by filename (e.g. `"user.json"`).

## Example

```bash
go run ./examples/basic/
```

## Testing

```bash
cd jv
go test ./...
```
