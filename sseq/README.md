# sseq

Lightweight tracing for Go: create spans and send them to Seq or Axiom.

No OpenTelemetry SDK. No plugin graph. One package surface: `sseq.go`.

| Provider | HTTP | File (Vector) |
|----------|------|----------------|
| Seq | CLEF → Seq ingest | CLEF file → Vector → Seq |
| Axiom | NDJSON → Axiom ingest | NDJSON file → Vector → Axiom |

## Quick start

```go
sseq.SetupSeq("http://localhost:5342/ingest/clef", "", "my-service")
defer sseq.Shutdown()

err := sseq.Trace(ctx, "HTTP GET /api/users", "server", func(ctx context.Context) error {
    sseq.Set(ctx, "http.route", "/api/users")
    return sseq.Trace(ctx, "Query users", "", func(ctx context.Context) error {
        sseq.Set(ctx, "db.system", "postgres")
        return nil
    })
})
```

## Setup

```go
// Seq over HTTP
sseq.SetupSeq(endpoint, apiKey, application)

// Axiom over HTTP
sseq.SetupAxiom(token, dataset, application)

// File for Vector → Seq / Axiom
sseq.SetupSeqFile("spans.clef", application)
sseq.SetupAxiomFile("spans.ndjson", application)
```

## API

| Function | Purpose |
|----------|---------|
| `SetupSeq` / `SetupAxiom` | HTTP export |
| `SetupSeqFile` / `SetupAxiomFile` | File export for Vector |
| `Shutdown` | Flush and close |
| `Trace(ctx, name, kind, fn)` | Run work inside a span |
| `Start(ctx, name, kind)` | Manual span; returns `(ctx, end)` |
| `Set(ctx, key, value)` | Attribute on active span |
| `Event(ctx, name, key, value, ...)` | Point event on active span |
| `Error(ctx, err)` | Mark active span failed |
| `IDs(ctx)` | Read trace/span ids |
| `Resume(ctx, traceID, parentSpanID)` | Continue async work |
| `HTTP(handler)` | HTTP middleware (server spans) |

`kind` may be empty: roots default to `server`, children to `internal`.

## Async

```go
sseq.Trace(ctx, "HTTP POST /orders", "server", func(ctx context.Context) error {
    traceID, spanID, _ := sseq.IDs(ctx)
    return bus.Publish(traceID, spanID, orderID)
})

workerCtx := sseq.Resume(context.Background(), traceID, spanID)
sseq.Trace(workerCtx, "Process order", "consumer", func(context.Context) error {
    return process(orderID)
})
```

## Layout

```text
sseq.go                      # public API only
internal/
  types.go                   # shared structs
  sender.go                  # batch flush
  providers/
    seq/                     # CLEF encode + HTTP
    axiom/                   # Axiom encode + HTTP
  file/                      # rotated file writer (Vector)
  trace/                     # span lifecycle
```

## Testing

```bash
cd sseq
go test ./...
go test ./integration/... -run Integration
```

| Variable | Purpose |
|----------|---------|
| `SSEQ_SKIP_INTEGRATION=1` | Skip live integration tests |
| `AXIOM_TOKEN` / `AXIOM_DATASET` | Axiom credentials |
