# trace_id

Generate a distributed trace ID from [UUID v7](https://www.rfc-editor.org/rfc/rfc9562) as a 32-character hexadecimal string without dashes.

## Installation

```bash
go get github.com/SmilingXinyi/gb/trace_id@latest
```

## Usage

```go
id, err := trace_id.New() // 019f4b3735f37ca0bdfaad88c25d618d
```

## Example

```bash
go run ./examples/basic/
```

## Test

```bash
cd trace_id
go test ./...
```
