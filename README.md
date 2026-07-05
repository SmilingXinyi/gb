# GB (Go Base)

GB is a collection of high-quality, reusable Golang base libraries and utilities. It uses a **multi-module** layout: each tool is its own module with minimal dependency overhead.

## Package index

| Package | Description | Status |
| :--- | :--- | :--- |
| [log](./log) | High-performance structured logging built on zerolog, with colored console output and automatic log file rotation. | ✅ Stable |
| [sseq](./sseq) | Seq-only span tree ingestion via CLEF (no console, no OTel). | 🆕 New |
| [llm](./llm) | LLM wrapper built on [goai](https://github.com/zendev-sh/goai): fluent sessions, `Execute` / `ExecuteTo` / `Stream`. | ✅ Stable |
| [jv](./jv) | JSON Schema validator with embedded and external schema support. | ✅ Stable |
| [trace_id](./trace_id) | Distributed trace ID generator based on UUID v7. | ✅ Stable |
| [utils](./utils) | Shared utilities such as YAML loading and LLM structured schema generation. | ✅ Stable |
| [gsc](./gsc) | Google Search Console API: search analytics, URL inspection, site and sitemap management. | ✅ Stable |
| [oss](./oss) | Unified object storage abstraction for Baidu BOS, Tencent COS, and other providers. | ✅ Stable |

## Development

The project is managed locally with Go Workspaces (`go.work`).

### Local development (Go Workspaces)

If you are working on another project and need to change tools in `gb`, create a `go.work` file in their common parent directory:

```bash
go work init ./gb/log ./your-project
```

Changes under `gb/log` (or other `gb` modules) take effect in `your-project` immediately, without `replace` directives.

## Releasing

Because of the multi-module layout, each module must be tagged with a path-prefixed version tag:

```bash
# Release log module v1.0.0
git tag log/v1.0.0
git push origin log/v1.0.0
```

## Installation

Install only the packages you need; unrelated modules are not pulled in as dependencies.

```bash
go get github.com/SmilingXinyi/gb/log@latest
```
