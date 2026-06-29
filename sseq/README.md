# sseq

`sseq` sends **span trees only** to [Seq](https://datalust.co/seq) using CLEF over HTTP. It does not write to console, files, or general application logs.

## Features

- Span tree ingestion for Seq timeline visualization
- Simple API: `Do`, `Start`/`End`, and HTTP middleware
- No OpenTelemetry, no zerolog, no slog dependency
- Opt-in batching with graceful shutdown

## Installation

```bash
go get github.com/SmilingXinyi/gb/sseq@latest
```

## Quick start

```go
sseq.Setup(sseq.DefaultConfig())
defer sseq.Shutdown()

err := sseq.Do(context.Background(), "HTTP GET /api/users", func(ctx context.Context) error {
    return sseq.Do(ctx, "Query users table", func(ctx context.Context) error {
        return nil
    })
})
```

## HTTP middleware

```go
http.Handle("/api/", sseqmiddleware.HTTP(apiHandler))
```

## Testing

```bash
cd sseq
go test ./...

# Requires local Seq on http://localhost:5341
go test ./... -run Integration
```

Set `SSEQ_SKIP_INTEGRATION=1` to skip Docker integration tests.
