# jinja2

[English](README.md) | 中文

面向 Go 的 Jinja2 渲染器。它直接调用 [gonja](https://github.com/nikolalohinski/gonja)，用普通的 `map[string]any` 渲染字符串、文件，或写入 `io.Writer`。

## 安装

```bash
go get github.com/SmilingXinyi/gb/jinja2@latest
```

## 快速开始

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

同一份模板需要反复渲染时，先编译再执行：

```go
template, err := jinja2.Parse("Hello {{ name }}")
if err != nil {
	panic(err)
}
rendered, err := template.Execute(map[string]any{"name": "Ada"})
```

## 配置

`DefaultConfig` 与 gonja 的默认配置一致。`New` 创建独立引擎，不影响包级函数。`Setup` 会替换 `Render`、`RenderFile`、`Parse` 和 `ParseFile` 使用的引擎。

```go
config := jinja2.DefaultConfig()
config.StrictUndefined = true
config.TrimBlocks = true
config.SearchPath = "templates"
engine := jinja2.New(config)

rendered, err := engine.RenderFile("page.j2", map[string]any{"name": "world"})
```

分隔符字段留空时沿用 gonja 默认值，因此 `jinja2.New(jinja2.Config{StrictUndefined: true})` 仍然使用 `{{ }}`。

## API

| 函数 | 说明 |
| :--- | :--- |
| `Render(source, data)` | 用包级引擎渲染模板字符串 |
| `RenderFile(filePath, data)` | 用包级引擎渲染模板文件 |
| `Parse(source)` / `ParseFile(filePath)` | 编译模板，供重复执行 |
| `New(config)` | 按指定配置创建引擎 |
| `Setup(config)` | 替换包级引擎 |
| `(*Engine).Render` / `RenderFile` / `RenderBytes` / `Execute` | 用指定引擎渲染 |
| `(*Template).Execute` / `ExecuteBytes` / `ExecuteWriter` | 执行已编译的模板 |

## 示例

```bash
go run ./examples/basic/
```

## 测试

```bash
cd jinja2
go test ./...
```
