# trace_id

`trace_id` generates distributed trace IDs based on [UUID v7](https://www.rfc-editor.org/rfc/rfc9562). UUID v7 is time-ordered and embeds a timestamp, making it suitable for database primary keys and distributed tracing.

## Features

- **Time-ordered IDs**: UUID v7 provides natural sort order by creation time.
- **Simple API**: Four functions covering error-returning and panic variants.
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
	// Returns uuid.UUID
	id, err := trace_id.New()
	if err != nil {
		panic(err)
	}
	fmt.Println(id.String())

	// Returns string directly
	idStr, err := trace_id.NewString()
	if err != nil {
		panic(err)
	}
	fmt.Println(idStr)

	// Panic on error (for init or hot paths where failure is unexpected)
	fmt.Println(trace_id.MustNewString())
}
```

## API Overview

| Function | Returns | Behavior |
| :--- | :--- | :--- |
| `New()` | `uuid.UUID, error` | Generate a new trace ID |
| `NewString()` | `string, error` | Generate a new trace ID as a string |
| `MustNew()` | `uuid.UUID` | Generate or panic |
| `MustNewString()` | `string` | Generate string or panic |

## Example

```bash
go run ./examples/basic/
```

## Testing

```bash
cd trace_id
go test ./...
```
