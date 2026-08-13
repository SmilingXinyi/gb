package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/SmilingXinyi/gb/sseq/internal"
	"github.com/SmilingXinyi/gb/trace_id"
)

type contextKey struct{}

var (
	activeSpanKey  = contextKey{}
	remoteTraceKey = struct{}{}
)

type remoteTrace struct {
	traceID string
	spanID  string
}

const defaultShutdownTimeout = 5 * time.Second

// Tracer owns a sender and creates spans.
type Tracer struct {
	application     string
	sender          *ss.Sender
	shutdownTimeout time.Duration
	errorHandler    func(error)
	lifecycle       sync.RWMutex
	closed          bool
	active          sync.WaitGroup
}

// Span is one operation in a trace.
type Span struct {
	name           string
	application    string
	traceID        string
	spanID         string
	parentID       string
	kind           string
	startTime      time.Time
	endTime        time.Time
	ended          bool
	sender         *ss.Sender
	hasError       bool
	statusMessage  string
	httpStatusCode int
	attributes     map[string]any
	events         []ss.TimedEvent
	mutex          sync.Mutex
	onEnd          func()
	errorHandler   func(error)
}

// NewTracer creates a tracer that exports completed spans through sender.
func NewTracer(application string, sender *ss.Sender, shutdownTimeout time.Duration, errorHandler func(error)) *Tracer {
	if shutdownTimeout <= 0 {
		shutdownTimeout = defaultShutdownTimeout
	}
	return &Tracer{
		application:     application,
		sender:          sender,
		shutdownTimeout: shutdownTimeout,
		errorHandler:    errorHandler,
	}
}

// Start begins a span and stores it in ctx.
func (tracer *Tracer) Start(ctx context.Context, name, kind string) (context.Context, *Span) {
	if tracer == nil {
		return ctx, &Span{name: name, startTime: time.Now().UTC(), kind: defaultKind(kind, false)}
	}

	span := &Span{
		name:         name,
		application:  tracer.application,
		startTime:    time.Now().UTC(),
		kind:         kind,
		errorHandler: tracer.errorHandler,
	}

	tracer.lifecycle.RLock()
	closed := tracer.closed
	if !closed && tracer.sender != nil {
		tracer.active.Add(1)
		span.sender = tracer.sender
		span.onEnd = tracer.active.Done
	}
	tracer.lifecycle.RUnlock()

	parentTraceID, parentSpanID, hasParent := parentFromContext(ctx)
	if hasParent {
		span.traceID = parentTraceID
		span.parentID = parentSpanID
		if span.kind == "" {
			span.kind = "internal"
		}
	} else {
		traceID, err := newTraceID()
		if err != nil {
			traceID = "00000000000000000000000000000000"
		}
		span.traceID = traceID
		if span.kind == "" {
			span.kind = "server"
		}
	}

	spanID, err := newSpanID()
	if err != nil {
		spanID = "0000000000000000"
	}
	span.spanID = spanID
	return contextWithSpan(ctx, span), span
}

// Trace runs fn inside a span.
func (tracer *Tracer) Trace(ctx context.Context, name, kind string, fn func(context.Context) error) error {
	ctx, span := tracer.Start(ctx, name, kind)
	defer span.End()
	if fn == nil {
		return nil
	}
	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
	}
	return err
}

// Close waits for in-flight spans, then flushes the sender.
func (tracer *Tracer) Close() error {
	if tracer == nil {
		return nil
	}

	tracer.lifecycle.Lock()
	if tracer.closed {
		tracer.lifecycle.Unlock()
		return nil
	}
	tracer.closed = true
	tracer.lifecycle.Unlock()

	done := make(chan struct{})
	go func() {
		tracer.active.Wait()
		close(done)
	}()

	timeout := tracer.shutdownTimeout
	if timeout <= 0 {
		timeout = defaultShutdownTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		reportError(tracer.errorHandler, fmt.Errorf("shutdown timed out after %s", timeout))
	}

	if tracer.sender == nil {
		return nil
	}
	err := tracer.sender.Close()
	tracer.sender = nil
	return err
}

// End completes the span and sends it.
func (span *Span) End() {
	if span == nil {
		return
	}

	span.mutex.Lock()
	if span.ended {
		span.mutex.Unlock()
		return
	}
	span.ended = true
	span.endTime = time.Now().UTC()
	event := ss.SpanEvent{
		Name:           span.name,
		Application:    span.application,
		TraceID:        span.traceID,
		SpanID:         span.spanID,
		ParentID:       span.parentID,
		SpanKind:       span.kind,
		StartTime:      span.startTime,
		EndTime:        span.endTime,
		HasError:       span.hasError,
		StatusMessage:  span.statusMessage,
		HTTPStatusCode: span.httpStatusCode,
		Attributes:     cloneMap(span.attributes),
		Events:         append([]ss.TimedEvent(nil), span.events...),
	}
	sender := span.sender
	onEnd := span.onEnd
	span.onEnd = nil
	errorHandler := span.errorHandler
	span.mutex.Unlock()

	defer func() {
		if onEnd != nil {
			onEnd()
		}
	}()

	if sender == nil {
		return
	}
	if err := sender.Send(event); err != nil {
		reportError(errorHandler, fmt.Errorf("send span: %w", err))
	}
}

// Set stores an attribute on the span.
func (span *Span) Set(key string, value any) {
	if span == nil || key == "" {
		return
	}
	span.mutex.Lock()
	defer span.mutex.Unlock()
	if span.ended {
		return
	}
	if span.attributes == nil {
		span.attributes = map[string]any{}
	}
	span.attributes[key] = value
}

// AddEvent attaches a point event to the span.
func (span *Span) AddEvent(name string, attrs map[string]any) {
	if span == nil || name == "" {
		return
	}
	span.mutex.Lock()
	defer span.mutex.Unlock()
	if span.ended {
		return
	}
	span.events = append(span.events, ss.TimedEvent{
		Name:       name,
		Time:       time.Now().UTC(),
		Attributes: cloneMap(attrs),
	})
}

// RecordError marks the span failed.
func (span *Span) RecordError(err error) {
	if span == nil || err == nil {
		return
	}
	span.mutex.Lock()
	defer span.mutex.Unlock()
	if span.ended {
		return
	}
	span.hasError = true
	span.statusMessage = err.Error()
	if span.attributes == nil {
		span.attributes = map[string]any{}
	}
	span.attributes["exception.message"] = err.Error()
	span.attributes["exception.type"] = fmt.Sprintf("%T", err)
}

// SetHTTPStatus records an HTTP status code.
func (span *Span) SetHTTPStatus(statusCode int) {
	if span == nil || statusCode <= 0 {
		return
	}
	span.mutex.Lock()
	defer span.mutex.Unlock()
	if span.ended {
		return
	}
	span.httpStatusCode = statusCode
	if span.attributes == nil {
		span.attributes = map[string]any{}
	}
	span.attributes["http.status_code"] = statusCode
	if statusCode >= 500 {
		span.hasError = true
		if span.statusMessage == "" {
			span.statusMessage = fmt.Sprintf("HTTP %d", statusCode)
		}
	}
}

// FromContext returns the active span in ctx.
func FromContext(ctx context.Context) *Span {
	if ctx == nil {
		return nil
	}
	span, ok := ctx.Value(activeSpanKey).(*Span)
	if !ok {
		return nil
	}
	return span
}

// IDsFromContext returns trace/span ids from ctx.
func IDsFromContext(ctx context.Context) (traceID, spanID string, ok bool) {
	if span := FromContext(ctx); span != nil {
		return span.traceID, span.spanID, span.traceID != ""
	}
	if ctx == nil {
		return "", "", false
	}
	remote, found := ctx.Value(remoteTraceKey).(remoteTrace)
	if !found || remote.traceID == "" {
		return "", "", false
	}
	return remote.traceID, remote.spanID, true
}

// Resume attaches remote trace ids to ctx.
func Resume(ctx context.Context, traceID, parentSpanID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if traceID == "" {
		return ctx
	}
	return context.WithValue(ctx, remoteTraceKey, remoteTrace{traceID: traceID, spanID: parentSpanID})
}

// contextWithSpan stores span as the active span in ctx.
func contextWithSpan(ctx context.Context, span *Span) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, activeSpanKey, span)
}

// parentFromContext returns the parent trace/span ids from ctx.
func parentFromContext(ctx context.Context) (traceID, parentSpanID string, hasParent bool) {
	if span := FromContext(ctx); span != nil && span.traceID != "" {
		return span.traceID, span.spanID, true
	}
	if ctx == nil {
		return "", "", false
	}
	remote, found := ctx.Value(remoteTraceKey).(remoteTrace)
	if found && remote.traceID != "" {
		return remote.traceID, remote.spanID, true
	}
	return "", "", false
}

// defaultKind returns kind, or server/internal when kind is empty.
func defaultKind(kind string, hasParent bool) string {
	if kind != "" {
		return kind
	}
	if hasParent {
		return "internal"
	}
	return "server"
}

// newTraceID creates a 32-character hex trace id.
func newTraceID() (string, error) {
	traceID, err := trace_id.New()
	if err != nil {
		return "", err
	}
	return trace_id.RemoveDashes(traceID), nil
}

// newSpanID creates a 16-character hex span id.
func newSpanID() (string, error) {
	var randomBytes [8]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		return "", fmt.Errorf("generate span id: %w", err)
	}
	return hex.EncodeToString(randomBytes[:]), nil
}

// cloneMap returns a shallow copy of values.
func cloneMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

// reportError invokes handler when both handler and err are non-nil.
func reportError(handler func(error), err error) {
	if handler == nil || err == nil {
		return
	}
	handler(err)
}
