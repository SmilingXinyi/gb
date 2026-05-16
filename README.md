# GB (Go Base)

GB 是一个高质量、可复用的 Golang 基础库工具集合。它采用 **Multi-module** 模式管理，每个工具包都是一个独立的模块，具有极轻的依赖负担。

## 工具索引

| 工具包 | 说明 | 状态 |
| :--- | :--- | :--- |
| [log](./log) | 基于 zerolog 的高性能、结构化日志工具，支持控制台彩色输出与文件自动滚动。 | ✅ 稳定 |

## 开发指南

项目使用 Go Workspaces (`go.work`) 进行本地开发管理。

### 本地开发 (Go Workspaces)

如果你在开发其他项目（如 `voice-utils`）的同时需要修改 `gb` 中的工具，建议在它们的共同父目录下创建 `go.work`：

```bash
go work init ./gb/log ./your-project
```

这样你对 `gb/log` 的任何修改都会立即在 `your-project` 中生效，无需 `replace` 指令。

## 如何发布

由于采用了多模块结构，发布时需要为每个模块打上带有路径前缀的标签：

```bash
# 发布 log 模块的 v1.0.0 版本
git tag log/v1.0.0
git push origin log/v1.0.0
```

## 如何安装

你可以根据需要只安装特定的工具包，而不会引入其他无关的依赖。

```bash
go get github.com/SmilingXinyi/gb/log@latest
```
