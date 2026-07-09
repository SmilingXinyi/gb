# sseq

`sseq` sends **span trees** to observability backends for timeline and waterfall visualization. Supported providers:

| Provider | Format | Backend |
|----------|--------|---------|
| `http` | CLEF | [Seq](https://datalust.co/seq) |
| `file` | CLEF or Axiom NDJSON | Local rotated files / Vector pipeline |
| `axiom` | OTel-style NDJSON | [Axiom](https://axiom.co) direct ingest |

It does not write general application logs to console or slog.

## Features

- Span tree ingestion with parent/child relationships
- Simple API: `Do`, `Start`/`End`, `RecordError`, and HTTP middleware
- HTTP middleware records response status codes
- Startup validation via `Setup() error`
- No OpenTelemetry SDK dependency
- Opt-in batching with graceful shutdown

## Installation

```bash
go get github.com/SmilingXinyi/gb/sseq@latest
```

## Quick start (Seq)

```go
if err := sseq.Setup(sseq.DefaultConfig()); err != nil {
    log.Fatal(err)
}
defer sseq.Shutdown()

err := sseq.Do(context.Background(), "HTTP GET /api/users", func(ctx context.Context) error {
    return sseq.Do(ctx, "Query users table", func(ctx context.Context) error {
        return nil
    })
})
```

## Axiom direct ingest

```go
if err := sseq.Setup(sseq.Config{
    Provider:    sseq.ProviderAxiom,
    Application: "my-service",
    Axiom: sseq.AxiomConfig{
        Token:   os.Getenv("AXIOM_TOKEN"),
        Dataset: "av-dataset",
    },
}); err != nil {
    log.Fatal(err)
}
defer sseq.Shutdown()
```

### Axiom time fields

`sseq` derives span timing from `Start`/`End` automatically. The Axiom encoder emits:

| Field | Meaning | Ingest format |
|-------|---------|---------------|
| `_time` | Span **start** time | UTC RFC3339 (`2026-07-09T12:00:00.123456789Z`) |
| `duration` | Elapsed time | **Integer nanoseconds** (30 ms → `30000000`) |

Axiom UI may display milliseconds in waterfall views, but ingest and APL queries use nanoseconds:

```apl
| extend duration_ms = duration / 1000000.0
```

## File provider + Vector

```go
if err := sseq.Setup(sseq.DefaultAxiomVectorFileConfig()); err != nil {
    log.Fatal(err)
}
```

See `examples/axiom-vector/vector.toml` for the Vector sink configuration.

## HTTP middleware

```go
import sseqmiddleware "github.com/SmilingXinyi/gb/sseq/middleware"

http.Handle("/api/", sseqmiddleware.HTTP(apiHandler))
```

Records each request as a root span and attaches the HTTP status code. Responses with status `>= 500` are marked as span errors.

## Provider selection

When `Provider` is empty, auto-detection uses the first match in this order:

1. File — when `File.Filename` is set
2. Axiom — when `Axiom.Token` and `Axiom.Dataset` are set
3. HTTP — when `HTTP.Endpoint` or `Endpoint` is set

If multiple backends are configured, set `Provider` explicitly or `Setup` returns an ambiguous-config error.

## Testing

```bash
cd sseq
go test ./...

# Integration tests (optional)
go test ./... -run Integration
```

Environment variables:

| Variable | Purpose |
|----------|---------|
| `SSEQ_SKIP_INTEGRATION=1` | Skip Docker/live integration tests |
| `AXIOM_TOKEN` / `AXIOM_DATASET` | Axiom integration test credentials |

Seq integration expects local Seq on `http://localhost:5341`.
