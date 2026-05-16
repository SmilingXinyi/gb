## 项目说明

该项目为 Golang 基础库工具集合，旨在提供高质量、可复用的基础组件。

## 开发指南

### 项目结构规范

项目采用 **Multi-module** 模式管理。每个工具模块位于 `gb/` 的子目录下，并拥有独立的 `go.mod`。

#### 模块目录结构

每个工具模块应遵循以下统一结构：

```text
gb/<tool_name>/
├── go.mod              # 模块定义
├── <tool_name>.go      # 核心 API 入口 (如 Setup, 导出函数)
├── config.go           # 配置结构体定义与默认配置函数
├── writer_*.go         # (可选) 特定功能的实现逻辑 (如 writer_console.go)
├── internal/           # (可选) 内部私有逻辑，不对外暴露
├── examples/           # 示例代码目录
│   └── basic/          # 基础用法示例
│       └── main.go
└── <tool_name>_test.go # 单元测试
```

### 模块开发流程

1.  **创建目录**：在 `gb/` 下创建新工具目录。
2.  **初始化模块**：运行 `go mod init github.com/SmilingXinyi/gb/<tool_name>`。
3.  **加入工作区**：在 `gb/` 根目录运行 `go work use ./<tool_name>`。
4.  **编写代码**：遵循结构规范编写代码、配置、示例和测试。

## 编码指南

### 变量名
- **禁止使用没有通用或具体的含义的变量名进行命名**
- **禁止使用单字母变量名。请使用完整的单词命名，例如用 handler 代替 h，用 response 或 writer 代替 w，用 request 代替 r，用 server 代替 srv**

### 代码注释
- 注释需要编写在定义的上方
- 代码注释使用英文编写
- 代码函数块必须包含注释
