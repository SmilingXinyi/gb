# sseq

Lightweight tracing for Go: create spans and send them to Seq, Axiom, or any OTLP/HTTP backend such as Jaeger, Tempo, or the OpenTelemetry Collector.

No OpenTelemetry SDK. No plugin graph. Public API lives in `sseq.go`, `config.go`, and `http.go`.

| Provider | HTTP | File (Vector) |
|----------|------|----------------|
| Seq | CLEF → Seq ingest | CLEF file → Vector → Seq |
| Axiom | NDJSON → Axiom ingest | NDJSON file → Vector → Axiom |
| OTLP | OTLP/HTTP JSON → Jaeger, Tempo, Collector | — |

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

// Axiom over HTTP. Domain defaults to api.axiom.co.
sseq.SetupAxiom(token, dataset, application)
sseq.SetupAxiom(token, dataset, application, sseq.WithAxiomDomain("api.eu.axiom.co"))
sseq.SetupAxiom(token, dataset, application, sseq.WithAxiomEndpoint("https://example/ingest"))

// OTLP/HTTP JSON. The endpoint is the full traces URL.
// See examples/jaeger. The official Jaeger binary listens on 127.0.0.1.
sseq.SetupOTLP("http://localhost:4318/v1/traces", "my-service")
sseq.SetupOTLP("http://localhost:4318/v1/traces", "my-service",
    sseq.WithHeaders(map[string]string{"Authorization": "Bearer token"}),
    sseq.WithResourceAttributes(map[string]any{"deployment.environment": "prod"}),
)

// File for Vector → Seq / Axiom
sseq.SetupSeqFile("spans.clef", application)
sseq.SetupAxiomFile("spans.ndjson", application)

// Optional exporter settings
sseq.SetupSeq(endpoint, apiKey, application,
    sseq.WithBatchSize(50),
    sseq.WithFlushInterval(2*time.Second),
    sseq.WithShutdownTimeout(3*time.Second),
    sseq.WithWriteQueueSize(32),
    sseq.WithWriteAttempts(3),
    sseq.WithErrorHandler(func(err error) { log.Println(err) }),
)
```

## API

| Function | Purpose |
|----------|---------|
| `SetupSeq` / `SetupAxiom` / `SetupOTLP` | HTTP export |
| `SetupSeqFile` / `SetupAxiomFile` | File export for Vector |
| `WithHeaders` | Extra HTTP headers for OTLP, such as authorization |
| `WithResourceAttributes` | Resource attributes on OTLP spans |
| `WithBatchSize` / `WithFlushInterval` | Flush tuning |
| `WithWriteQueueSize` | Queued batches before new ones are dropped |
| `WithWriteAttempts` | Export attempts per batch; further retries stop after Shutdown |
| `WithAxiomDomain` / `WithAxiomEndpoint` | Axiom region host or full ingest URL |
| `WithShutdownTimeout` | How long Shutdown waits for in-flight spans |
| `WithErrorHandler` | Export and shutdown errors; nil silences logs |
| `Shutdown` | Wait, flush, and close (returns error) |
| `Trace(ctx, name, kind, fn)` | Run work inside a span |
| `Start(ctx, name, kind)` | Manual span; returns `(ctx, end)` |
| `Set(ctx, key, value)` | Attribute on active span |
| `Event(ctx, name, key, value, ...)` | Point event on active span |
| `Error(ctx, err)` | Mark active span failed |
| `IDs(ctx)` | Read trace/span ids |
| `Resume(ctx, traceID, parentSpanID)` | Continue async work. Invalid ids are ignored |
| `HTTP(handler)` | HTTP middleware (server spans, W3C traceparent) |
| `Inject` / `Extract` | W3C `traceparent` for outbound / inbound HTTP |

`kind` may be empty: roots default to `server`, children to `internal`.

Attribute values that `encoding/json` cannot marshal, such as `NaN` or a channel, are replaced with a placeholder. The rest of the span is still exported.

OTLP attributes are AnyValues: `stringValue`, `intValue` (a decimal string), `doubleValue`, `boolValue`, and `arrayValue`. A value that does not fit one of those forms is sent as `fmt.Sprint` text, and that one value does not drop the span. `traceId` is 32 hexadecimal characters and `spanId` is 16. A root span omits `parentSpanId`. Span kind is an OTLP number: server 2, client 3, producer 4, consumer 5, and internal or any other kind 1. A failed span sets `status.code` to 2, copies `status.message`, and adds an `exception` event.

A batch is one `POST` with `Content-Type: application/json`. Protobuf and gzip are not used. A non-2xx response, or a 200 whose `partialSuccess.rejectedSpans` is greater than zero, is reported to `WithErrorHandler` with the status code and the start of the response body. Jaeger answers a fully accepted export with `{"partialSuccess":{}}`, which is success.

The HTTP middleware names spans after the method only (`GET`, `POST`) and records `http.method`, `http.target`, and `http.host`. `http.target` is the path only; the query string is omitted. Set `http.route` yourself when you have a low-cardinality template.

Incoming W3C `traceparent` flags are stored on the span and written back by `Inject`. Locally created traces use flags `01`. `Resume` does not receive flags, so async work resumed from ids alone is marked sampled.

## Timestamps

Seq, Axiom, and OTLP each keep that backend's own trace time:

| Provider | Span timestamp | Start | Point events |
|----------|----------------|-------|--------------|
| Seq CLEF | `@t` is the end time | `@st` is the start time | `@t` is the event time |
| Axiom | `_time` is the start time | `duration` is nanoseconds from start to end | `timestamp` is Unix nanoseconds |
| OTLP | `endTimeUnixNano` | `startTimeUnixNano` as a decimal string | `timeUnixNano` as a decimal string |

`Shutdown` returns an error from the final flush/close. It releases its lock before waiting, so an in-flight span can still finish. Spans started after `Shutdown` begins are not exported. `defer sseq.Shutdown()` still compiles; check the error when you need it. `WithErrorHandler(nil)` silences export logs.

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
sseq.go                      # Setup / Trace / Start / Shutdown
config.go                    # Config and Setup options
http.go                      # HTTP middleware and W3C traceparent
internal/
  types.go                   # shared structs
  sender.go                  # batch flush
  providers/
    seq/                     # CLEF encode + HTTP
    axiom/                   # Axiom encode + HTTP
    otlp/                    # OTLP/HTTP JSON encode + HTTP
  file/                      # rotated file writer (Vector)
  trace/                     # span lifecycle
```

## Testing

```bash
cd sseq
go test ./...
go test ./integration/... -run Integration
```

### Seq Docker (integration)

```bash
docker run -d --name sseq-seq \
  -e ACCEPT_EULA=Y \
  -e SEQ_FIRSTRUN_ADMINUSERNAME=admin \
  -e SEQ_FIRSTRUN_ADMINPASSWORD='Admin123456!' \
  -p 5341:80 \
  -p 5342:5341 \
  datalust/seq:latest
```

Then open `http://localhost:5341` and run:

```bash
go test ./integration/... -run TestIntegrationSpanTreeWithSeqDocker -v
```

### Jaeger (integration)

OTLP/HTTP JSON. The test sends a parent span and a failing child, then reads them from `/api/v3/traces/{traceId}`. It skips when Jaeger is not running.

```bash
# Official binary. OTLP listens on 127.0.0.1:4318 unless JAEGER_LISTEN_HOST is set.
./jaeger

# Or Docker:
docker run -d --name sseq-jaeger \
  -p 16686:16686 \
  -p 4318:4318 \
  cr.jaegertracing.io/jaegertracing/jaeger:2.22.0
```

```bash
go test ./integration/... -run TestIntegrationOTLPJaeger -v
```

| Variable | Purpose |
|----------|---------|
| `SSEQ_SKIP_INTEGRATION=1` | Skip live integration tests |
| `AXIOM_TOKEN` / `AXIOM_DATASET` | Axiom credentials |
| `SSEQ_JAEGER_OTLP_URL` | OTLP traces URL. Default `http://127.0.0.1:4318/v1/traces` |
| `SSEQ_JAEGER_QUERY_URL` | Jaeger query URL. Default `http://127.0.0.1:16686` |
