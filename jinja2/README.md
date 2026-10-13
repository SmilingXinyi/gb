# jinja2

English | [中文](README.zh-CN.md)

Thin Jinja2 renderer for Go. It calls [gonja](https://github.com/nikolalohinski/gonja) and exposes string, file, and writer rendering with ordinary `map[string]any` data.

## Installation

```bash
go get github.com/SmilingXinyi/gb/jinja2@latest
```

## Quick Start

```go
package main

import (
	"fmt"

	"github.com/SmilingXinyi/gb/jinja2"
)

func main() {
	rendered, err := jinja2.Render("Hello {{ name | capitalize }}!", map[string]any{
		"name": "bob",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(rendered) // Hello Bob!
}
```

Compile once when the same template is rendered many times:

```go
template, err := jinja2.Parse("Hello {{ name }}")
if err != nil {
	panic(err)
}
rendered, err := template.Execute(map[string]any{"name": "Ada"})
```

## Configuration

`DefaultConfig` matches gonja defaults. `New` builds an engine that does not change the package-level helpers. `Setup` replaces the engine used by `Render`, `RenderFile`, `Parse`, and `ParseFile`.

```go
config := jinja2.DefaultConfig()
config.StrictUndefined = true
config.TrimBlocks = true
config.SearchPath = "templates"
engine := jinja2.New(config)

rendered, err := engine.RenderFile("page.j2", map[string]any{"name": "world"})
```

Empty delimiter fields keep the gonja defaults, so `jinja2.New(jinja2.Config{StrictUndefined: true})` still uses `{{ }}`.

## API

| Function | Description |
| :--- | :--- |
| `Render(source, data)` | Render a template string with the package-level engine |
| `RenderFile(filePath, data)` | Render a template file with the package-level engine |
| `Parse(source)` / `ParseFile(filePath)` | Compile a template for repeated execution |
| `New(config)` | Create an engine with its own configuration |
| `Setup(config)` | Replace the package-level engine |
| `(*Engine).Render` / `RenderFile` / `RenderBytes` / `Execute` | Render with an engine |
| `(*Template).Execute` / `ExecuteBytes` / `ExecuteWriter` | Execute a compiled template |

## Example

```bash
go run ./examples/basic/
```

## Testing

```bash
cd jinja2
go test ./...
```
