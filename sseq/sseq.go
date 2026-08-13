// Package sseq is a lightweight tracing SDK for Go.
//
// Core flow: Setup → Trace/Start → attributes/events → Shutdown.
// Spans are exported to Seq or Axiom over HTTP, or written to a file for Vector.
package sseq

import (
	"context"
	"fmt"
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
func SetupSeq(endpoint, apiKey, application string, options ...Option) error {
	if endpoint == "" {
		return fmt.Errorf("sseq: seq endpoint is required")
	}
	return setup(application, seq.Encoder{}, seq.NewHTTP(endpoint, apiKey), options...)
}

// SetupAxiom sends spans to Axiom over HTTP.
func SetupAxiom(token, dataset, application string, options ...Option) error {
	writer, err := axiom.NewHTTP(token, dataset, "", "")
	if err != nil {
		return fmt.Errorf("sseq: %w", err)
	}
	return setup(application, axiom.Encoder{}, writer, options...)
}

// SetupSeqFile writes CLEF spans to a local file for Vector → Seq.
func SetupSeqFile(filename, application string, options ...Option) error {
	writer, err := file.NewWriter(filename)
	if err != nil {
		return fmt.Errorf("sseq: %w", err)
	}
	return setup(application, seq.Encoder{}, writer, options...)
}

// SetupAxiomFile writes Axiom NDJSON spans to a local file for Vector → Axiom.
func SetupAxiomFile(filename, application string, options ...Option) error {
	writer, err := file.NewWriter(filename)
	if err != nil {
		return fmt.Errorf("sseq: %w", err)
	}
	return setup(application, axiom.Encoder{}, writer, options...)
}

// Shutdown waits for in-flight spans, flushes buffered events, and closes the sender.
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

// setup replaces the global tracer with a sender built from encoder and writer.
func setup(application string, encoder ss.Encoder, writer ss.Writer, options ...Option) error {
	config := applyOptions(options...)

	setupMutex.Lock()
	defer setupMutex.Unlock()

	if globalTracer != nil {
		_ = globalTracer.Close()
		globalTracer = nil
	}

	sender := ss.NewSender(ss.BatchConfig{
		BatchSize:     config.BatchSize,
		FlushInterval: config.FlushInterval,
		OnError:       config.ErrorHandler,
	}, encoder, writer)

	globalTracer = trace.NewTracer(application, sender, config.ShutdownTimeout, config.ErrorHandler)
	return nil
}

// defaultTracer returns the configured tracer, or a no-op tracer when Setup has not run.
func defaultTracer() *trace.Tracer {
	setupMutex.RLock()
	tracer := globalTracer
	setupMutex.RUnlock()
	if tracer != nil {
		return tracer
	}
	return &trace.Tracer{}
}

// pairsToMap converts alternating key/value arguments into a map.
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
