# trace_id

Generate distributed trace IDs from [UUID v7](https://www.rfc-editor.org/rfc/rfc9562).

## Installation

```bash
go get github.com/SmilingXinyi/gb/trace_id@v1.0.0
```

## Usage

```go
id, err := trace_id.New()          // 019f4b37-35f3-7ca0-bdfa-ad88c25d618d
hexID := trace_id.RemoveDashes(id) // 019f4b3735f37ca0bdfaad88c25d618d
```

## API

| Function | Description |
| :--- | :--- |
| `New()` | Generate UUID v7 with dashes |
| `RemoveDashes(id)` | Remove dashes from a UUID string |

## Example

```bash
go run ./examples/basic/
```

## Test

```bash
cd trace_id
go test ./...
```
