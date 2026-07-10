# trace_id

`trace_id` generates distributed trace IDs based on [UUID v7](https://www.rfc-editor.org/rfc/rfc9562). UUID v7 is time-ordered and embeds a timestamp, making it suitable for database primary keys and distributed tracing.

## Features

- **Time-ordered IDs**: UUID v7 provides natural sort order by creation time.
- **Multiple formats**: Canonical UUID strings and 32-character lowercase hex for Seq and W3C trace contexts.
- **Parse and validate**: Accept UUID strings or 32-hex input for upstream trace propagation.
- **Simple API**: Typed `ID` value plus error-returning and panic variants.
- **Zero configuration**: No setup required; call and use immediately.

## Installation

```bash
go get github.com/SmilingXinyi/gb/trace_id@latest
```

## Quick Start

```go
package main

import (
	"fmt"

	"github.com/SmilingXinyi/gb/trace_id"
)

func main() {
	// Returns trace_id.ID
	id, err := trace_id.New()
	if err != nil {
		panic(err)
	}
	fmt.Println(id.String()) // canonical UUID
	fmt.Println(id.Hex())    // 32-char hex for Seq / W3C

	// Returns string directly
	idStr, err := trace_id.NewString()
	if err != nil {
		panic(err)
	}
	fmt.Println(idStr)

	hexID, err := trace_id.NewHex()
	if err != nil {
		panic(err)
	}
	fmt.Println(hexID)

	// Parse upstream trace IDs
	parsed, err := trace_id.Parse(hexID)
	if err != nil {
		panic(err)
	}
	fmt.Println(parsed.String())

	// Panic on error (for init or hot paths where failure is unexpected)
	fmt.Println(trace_id.MustNewHex())
}
```

## API Overview

| Function | Returns | Behavior |
| :--- | :--- | :--- |
| `New()` | `ID, error` | Generate a new trace ID |
| `NewString()` | `string, error` | Generate a canonical UUID string |
| `NewHex()` | `string, error` | Generate a 32-character lowercase hex string |
| `MustNew()` | `ID` | Generate or panic |
| `MustNewString()` | `string` | Generate UUID string or panic |
| `MustNewHex()` | `string` | Generate hex string or panic |
| `Parse(raw)` | `ID, error` | Parse UUID or 32-hex input |
| `IsValid(raw)` | `bool` | Validate UUID or 32-hex input |

## Example

```bash
go run ./examples/basic/
```

## Testing

```bash
cd trace_id
go test ./...
```
