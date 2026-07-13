// Package sseq is a lightweight tracing SDK for Go.
//
// Core flow: Setup → Trace/Start → attributes/events → Shutdown.
// Spans are exported to Seq or Axiom over HTTP, or written to a file for Vector.
package sseq

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/SmilingXinyi/gb/sseq/internal"
	"github.com/SmilingXinyi/gb/sseq/internal/file"
	"github.com/SmilingXinyi/gb/sseq/internal/providers/axiom"
	"github.com/SmilingXinyi/gb/sseq/internal/providers/seq"
	"github.com/SmilingXinyi/gb/sseq/internal/trace"
)

var (
	globalTracer *trace.Tracer
	setupMutex   sync.RWMutex
)

// SetupSeq sends CLEF spans to Seq over HTTP.
func SetupSeq(endpoint, apiKey, application string) error {
	if endpoint == "" {
		return fmt.Errorf("sseq: seq endpoint is required")
	}
	return setup(application, seq.Encoder{}, seq.NewHTTP(endpoint, apiKey))
}

// SetupAxiom sends spans to Axiom over HTTP.
func SetupAxiom(token, dataset, application string) error {
	writer, err := axiom.NewHTTP(token, dataset, "", "")
	if err != nil {
		return fmt.Errorf("sseq: %w", err)
	}
	return setup(application, axiom.Encoder{}, writer)
}

// SetupSeqFile writes CLEF spans to a local file for Vector → Seq.
func SetupSeqFile(filename, application string) error {
	writer, err := file.NewWriter(filename)
	if err != nil {
		return fmt.Errorf("sseq: %w", err)
	}
	return setup(application, seq.Encoder{}, writer)
}

// SetupAxiomFile writes Axiom NDJSON spans to a local file for Vector → Axiom.
func SetupAxiomFile(filename, application string) error {
	writer, err := file.NewWriter(filename)
	if err != nil {
		return fmt.Errorf("sseq: %w", err)
	}
	return setup(application, axiom.Encoder{}, writer)
}

// Shutdown flushes buffered spans and closes the sender.
func Shutdown() {
	setupMutex.Lock()
	defer setupMutex.Unlock()
	if globalTracer != nil {
		_ = globalTracer.Close()
		globalTracer = nil
	}
}

// Trace runs fn inside a named span. kind may be empty for defaults
// (server for roots, internal for children).
func Trace(ctx context.Context, name, kind string, fn func(context.Context) error) error {
	return defaultTracer().Trace(ctx, name, kind, fn)
}

// Start begins a span and returns ctx plus an end function. Call end when done.
func Start(ctx context.Context, name, kind string) (context.Context, func()) {
	ctx, span := defaultTracer().Start(ctx, name, kind)
	return ctx, span.End
}

// Set attaches a key/value attribute to the active span in ctx.
func Set(ctx context.Context, key string, value any) {
	if span := trace.FromContext(ctx); span != nil {
		span.Set(key, value)
	}
}

// Event attaches a named point event to the active span in ctx.
func Event(ctx context.Context, name string, keyValues ...any) {
	span := trace.FromContext(ctx)
	if span == nil {
		return
	}
	span.AddEvent(name, pairsToMap(keyValues...))
}

// Error marks the active span in ctx as failed.
func Error(ctx context.Context, err error) {
	if span := trace.FromContext(ctx); span != nil {
		span.RecordError(err)
	}
}

// IDs returns trace/span identifiers from ctx.
func IDs(ctx context.Context) (traceID, spanID string, ok bool) {
	return trace.IDsFromContext(ctx)
}

// Resume continues a remote trace in ctx for async workers.
func Resume(ctx context.Context, traceID, parentSpanID string) context.Context {
	return trace.Resume(ctx, traceID, parentSpanID)
}

// HTTP wraps an http.Handler and records each request as a server span.
func HTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		spanName := fmt.Sprintf("%s %s", request.Method, request.URL.Path)
		requestContext, span := defaultTracer().Start(request.Context(), spanName, "server")
		span.Set("http.method", request.Method)
		span.Set("http.route", request.URL.Path)
		span.Set("http.target", request.URL.RequestURI())
		defer span.End()

		recorder := &statusRecorder{responseWriter: response, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, request.WithContext(requestContext))
		span.SetHTTPStatus(recorder.statusCode)
	})
}

func setup(application string, encoder ss.Encoder, writer ss.Writer) error {
	setupMutex.Lock()
	defer setupMutex.Unlock()

	if globalTracer != nil {
		_ = globalTracer.Close()
		globalTracer = nil
	}

	globalTracer = &trace.Tracer{
		Application: application,
		Sender: ss.NewSender(ss.BatchConfig{
			BatchSize:     ss.DefaultBatchSize,
			FlushInterval: ss.DefaultFlushInterval,
		}, encoder, writer),
	}
	return nil
}

func defaultTracer() *trace.Tracer {
	setupMutex.RLock()
	tracer := globalTracer
	setupMutex.RUnlock()
	if tracer != nil {
		return tracer
	}
	return &trace.Tracer{}
}

func pairsToMap(keyValues ...any) map[string]any {
	if len(keyValues) == 0 {
		return nil
	}
	result := make(map[string]any, len(keyValues)/2)
	for index := 0; index+1 < len(keyValues); index += 2 {
		key, ok := keyValues[index].(string)
		if !ok || key == "" {
			continue
		}
		result[key] = keyValues[index+1]
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

type statusRecorder struct {
	responseWriter http.ResponseWriter
	statusCode     int
}

func (recorder *statusRecorder) Header() http.Header {
	return recorder.responseWriter.Header()
}

func (recorder *statusRecorder) Write(body []byte) (int, error) {
	return recorder.responseWriter.Write(body)
}

func (recorder *statusRecorder) WriteHeader(statusCode int) {
	recorder.statusCode = statusCode
	recorder.responseWriter.WriteHeader(statusCode)
}
