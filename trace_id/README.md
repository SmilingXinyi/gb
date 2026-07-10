# trace_id

Generate distributed trace IDs from [UUID v7](https://www.rfc-editor.org/rfc/rfc9562).

## Installation

```bash
go get github.com/SmilingXinyi/gb/trace_id@latest
```

## Usage

```go
idStr, err := trace_id.NewString() // 019f4b37-35f3-7ca0-bdfa-ad88c25d618d
idHex, err := trace_id.NewHex()    // 019f4b3735f37ca0bdfaad88c25d618d
```

## API

| Function | Returns | Description |
| :--- | :--- | :--- |
| `New()` | `uuid.UUID, error` | UUID v7 |
| `NewString()` | `string, error` | UUID v7 with dashes |
| `NewHex()` | `string, error` | UUID v7 without dashes |
| `MustNew()` | `uuid.UUID` | UUID v7 or panic |
| `MustNewString()` | `string` | With dashes or panic |
| `MustNewHex()` | `string` | Without dashes or panic |

## Example

```bash
go run ./examples/basic/
```

## Test

```bash
cd trace_id
go test ./...
```
