# oss

`oss` is a unified object storage abstraction layer. It exposes a single `Storage` interface so you can switch cloud providers without changing business code.

## Features

- **Provider-agnostic API**: Upload, download, list, stat, sign URLs, and copy objects through one interface.
- **Pluggable adapters**: Register providers via blank imports; swap by changing the provider name.
- **Implemented providers**: Baidu BOS, Tencent COS.
- **Skeleton providers**: Aliyun OSS, AWS S3 (not yet implemented).

## Directory Structure

```text
oss/
├── oss.go              # Provider enum and Storage interface
├── config.go           # Config struct
├── object.go           # Shared types (PutOptions, ObjectMeta, ListResult, etc.)
├── errors.go           # Common error types
├── client.go           # New() factory and Register() mechanism
├── baidu/              # Baidu BOS adapter
├── aliyun/             # Aliyun OSS adapter (skeleton)
├── tencent/            # Tencent COS adapter
├── s3/                 # AWS S3 / S3-compatible adapter (skeleton)
└── examples/
    ├── basic/          # Basic usage example
    ├── baidu/          # Baidu runnable example
    └── tencent/        # Tencent runnable example
```

## Installation

```bash
go get github.com/SmilingXinyi/gb/oss@latest
```

## Quick Start

```go
import (
    "context"

    "github.com/SmilingXinyi/gb/oss"
    _ "github.com/SmilingXinyi/gb/oss/baidu" // register Baidu provider
)

// Option 1: type-safe provider enum (recommended)
client, err := oss.New(oss.ProviderBaidu, oss.Config{
    AccessKey: "your-ak",
    SecretKey: "your-sk",
    Region:    "bj",
    Bucket:    "my-bucket",
})

// Option 2: string provider name
// client, err := oss.New("baidu", oss.Config{...})

ctx := context.Background()

// Upload
err = client.Put(ctx, "", "path/to/file.txt", reader, size, nil)

// Download
readCloser, err := client.Get(ctx, "", "path/to/file.txt")
defer readCloser.Close()

// Metadata
meta, err := client.Stat(ctx, "", "path/to/file.txt")

// List
result, err := client.List(ctx, "", "prefix/", &oss.ListOptions{Delimiter: "/"})

// Pre-signed URL
url, err := client.SignURL(ctx, "", "path/to/file.txt", "GET", 3600)

// Server-side copy
err = client.Copy(ctx, "src-bucket", "src/key", "dst-bucket", "dst/key")

// Delete
err = client.Delete(ctx, "", "path/to/file.txt")
```

To switch providers, change the blank import and the first argument to `oss.New`; all other calls stay the same.

## Provider Configuration

### Baidu BOS

| Field | Description | Example |
| :--- | :--- | :--- |
| `AccessKey` | BOS access key | `AKXXXXXXXXXX` |
| `SecretKey` | BOS secret key | `SKXXXXXXXXXX` |
| `Region` | Region prefix used to derive the endpoint | `bj`, `gz`, `su` |
| `Endpoint` | Explicit endpoint (overrides Region) | `bj.bcebos.com` |
| `Bucket` | Default bucket (optional) | `my-bucket` |
| `Token` | STS temporary token (optional) | — |

Region to endpoint mapping (defaults to `bj` when empty):

| Region | Endpoint |
| :--- | :--- |
| `bj` | `bj.bcebos.com` (Beijing) |
| `gz` | `gz.bcebos.com` (Guangzhou) |
| `su` | `su.bcebos.com` (Suzhou) |
| `hkg` | `hkg.bcebos.com` (Hong Kong) |
| `fwh` | `fwh.bcebos.com` (Wuhan) |

### Tencent COS

| Field | Description | Example |
| :--- | :--- | :--- |
| `AccessKey` | SecretId | `AKIDXXXXXXXXXX` |
| `SecretKey` | SecretKey | `XXXXXXXXXX` |
| `Region` | Region | `ap-guangzhou`, `ap-shanghai` |
| `Endpoint` | Bucket URL (`https://{bucket}.cos.{region}.myqcloud.com`) | — |
| `Bucket` | Default bucket (`{name}-{appid}`) | `my-bucket-1250000000` |
| `Token` | SessionToken (optional) | — |

## Testing

### Baidu BOS

**Unit tests (no credentials required)**

```bash
go test ./baidu/ -run "TestNewClient"
```

**Integration tests**

```bash
cp baidu/.env.example baidu/.env   # fill in real credentials
go test ./baidu/ -v -run "TestIntegration"
go test ./baidu/ -v -run "TestIntegration_PutAndGet"  # single test
```

**Example**

```bash
go run ./examples/baidu/
```

### Tencent COS

**Unit tests (no credentials required)**

```bash
go test ./tencent/ -v -run "TestIsNotFound"
```

**Integration tests**

```bash
cp tencent/.env.example tencent/.env
go test ./tencent/ -v -run "TestIntegration"
```

**Example**

```bash
go run ./examples/tencent/
```

### Aliyun OSS / AWS S3

Adapters are not yet implemented; integration tests will be added when they are ready.

> **Note**: Place each provider's `.env` file in its adapter directory:
> - Baidu BOS: `baidu/.env`
> - Tencent COS: `tencent/.env`
>
> Integration tests are skipped automatically when `.env` is missing; unit tests always run.
