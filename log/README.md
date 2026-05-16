# GB Log

`gb/log` 是一个基于 [zerolog](https://github.com/rs/zerolog) 封装的高性能日志工具包。

## 特性

- **结构化日志**：原生支持 JSON 格式。
- **控制台彩色输出**：在开发模式下提供易读的彩色日志。
- **文件自动滚动**：集成 lumberjack，支持按大小、时间自动切割日志文件。
- **调用链追踪**：自动简化调用者路径（相对于项目根目录）。
- **极简 API**：提供全局快捷函数（`Info()`, `Error()` 等）。

## 安装

```bash
go get github.com/SmilingXinyi/gb/log@latest
```

## 快速开始

```go
package main

import (
	"github.com/SmilingXinyi/gb/log"
)

func main() {
	// 1. 初始化配置
	config := log.DefaultConfig()
	config.File.Filename = "app.log" // 设置文件名后将同时输出到文件
	
	// 2. 设置日志
	log.Setup(config)

	// 3. 使用日志
	log.Info().Str("module", "main").Msg("Hello GB Log!")
	
	// 4. 模块化日志
	authLog := log.Module("auth")
	authLog.Debug().Msg("User login attempt")
}
```

## 配置说明

详见 [config.go](./config.go) 中的 `LogConfig` 结构体定义。
