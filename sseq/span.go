package sseq

import (
	"context"
	"sync"
	"time"

	"github.com/SmilingXinyi/gb/sseq/internal/sender"
)

const (
	spanKindServer   = "Server"
	spanKindInternal = "Internal"
)

var (
	globalConfig Config
	globalSender *sender.Sender
	setupMutex   sync.RWMutex
)

// Setup initializes the global Seq span sender.
func Setup(config Config) {
	setupMutex.Lock()
	defer setupMutex.Unlock()

	if globalSender != nil {
		_ = globalSender.Close()
		globalSender = nil
	}

	globalConfig = config
	if config.Endpoint == "" {
		return
	}

	globalSender = sender.New(sender.Config{
		Endpoint:      config.Endpoint,
		APIKey:        config.APIKey,
		BatchSize:     config.BatchSize,
		FlushInterval: config.FlushInterval,
	})
}

// Shutdown flushes and closes the global Seq span sender.
func Shutdown() {
	setupMutex.Lock()
	defer setupMutex.Unlock()

	if globalSender != nil {
		_ = globalSender.Close()
		globalSender = nil
	}
}

// Do executes a function within a span and ends the span automatically.
func Do(ctx context.Context, name string, fn func(context.Context) error) error {
	ctx, span := Start(ctx, name)
	defer span.End()
	return fn(ctx)
}

// Start begins a new span and returns an updated context plus the span handle.
func Start(ctx context.Context, name string) (context.Context, *Span) {
	setupMutex.RLock()
	currentConfig := globalConfig
	currentSender := globalSender
	setupMutex.RUnlock()

	span := &Span{
		name:        name,
		application: currentConfig.Application,
		startTime:   time.Now().UTC(),
		sender:      currentSender,
	}

	parentContext, hasParent := activeSpanContext(ctx)
	if hasParent {
		span.traceID = parentContext.traceID
		span.parentID = parentContext.spanID
		span.spanKind = spanKindInternal
	} else {
		traceID, err := newTraceID()
		if err != nil {
			traceID = "00000000000000000000000000000000"
		}
		span.traceID = traceID
		span.spanKind = spanKindServer
	}

	spanID, err := newSpanID()
	if err != nil {
		spanID = "0000000000000000"
	}
	span.spanID = spanID

	childContext := context.WithValue(ctx, activeSpanContextKey, spanContext{
		traceID: span.traceID,
		spanID:  span.spanID,
	})
	return childContext, span
}

// Span represents one operation in a distributed trace.
type Span struct {
	name        string
	application string
	traceID     string
	spanID      string
	parentID    string
	spanKind    string
	startTime   time.Time
	endTime     time.Time
	ended       bool
	sender      *sender.Sender
}

// End completes the span and sends it to Seq.
func (span *Span) End() {
	if span == nil || span.ended {
		return
	}
	span.ended = true
	span.endTime = time.Now().UTC()

	if span.sender == nil {
		return
	}

	_ = span.sender.Send(sender.SpanEvent{
		Name:        span.name,
		Application: span.application,
		TraceID:     span.traceID,
		SpanID:      span.spanID,
		ParentID:    span.parentID,
		SpanKind:    span.spanKind,
		StartTime:   span.startTime,
		EndTime:     span.endTime,
	})
}

// TraceID returns the trace id associated with this span.
func (span *Span) TraceID() string {
	if span == nil {
		return ""
	}
	return span.traceID
}

// SpanID returns the span id associated with this span.
func (span *Span) SpanID() string {
	if span == nil {
		return ""
	}
	return span.spanID
}

// activeSpanContext reads span identifiers stored in the context.
func activeSpanContext(ctx context.Context) (spanContext, bool) {
	if ctx == nil {
		return spanContext{}, false
	}
	value := ctx.Value(activeSpanContextKey)
	spanState, ok := value.(spanContext)
	return spanState, ok
}
