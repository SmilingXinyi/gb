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

// Tracer owns a sender and creates spans.
type Tracer struct {
	Application string
	Sender      *ss.Sender
}

// Span is one operation in a trace.
type Span struct {
	Name           string
	Application    string
	TraceID        string
	SpanID         string
	ParentID       string
	Kind           string
	StartTime      time.Time
	EndTime        time.Time
	Ended          bool
	Sender         *ss.Sender
	HasError       bool
	StatusMessage  string
	HTTPStatusCode int
	Attributes     map[string]any
	Events         []ss.TimedEvent
	Mutex          sync.Mutex
}

// Start begins a span and stores it in ctx.
func (tracer *Tracer) Start(ctx context.Context, name, kind string) (context.Context, *Span) {
	if tracer == nil {
		return ctx, &Span{Name: name, StartTime: time.Now().UTC(), Kind: defaultKind(kind, false)}
	}

	span := &Span{
		Name:        name,
		Application: tracer.Application,
		StartTime:   time.Now().UTC(),
		Sender:      tracer.Sender,
		Kind:        kind,
	}

	parentTraceID, parentSpanID, hasParent := parentFromContext(ctx)
	if hasParent {
		span.TraceID = parentTraceID
		span.ParentID = parentSpanID
		if span.Kind == "" {
			span.Kind = "internal"
		}
	} else {
		traceID, err := newTraceID()
		if err != nil {
			traceID = "00000000000000000000000000000000"
		}
		span.TraceID = traceID
		if span.Kind == "" {
			span.Kind = "server"
		}
	}

	spanID, err := newSpanID()
	if err != nil {
		spanID = "0000000000000000"
	}
	span.SpanID = spanID
	return contextWithSpan(ctx, span), span
}

// Trace runs fn inside a span.
func (tracer *Tracer) Trace(ctx context.Context, name, kind string, fn func(context.Context) error) error {
	ctx, span := tracer.Start(ctx, name, kind)
	defer span.End()
	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
	}
	return err
}

// Close flushes the sender.
func (tracer *Tracer) Close() error {
	if tracer == nil || tracer.Sender == nil {
		return nil
	}
	err := tracer.Sender.Close()
	tracer.Sender = nil
	return err
}

// End completes the span and sends it.
func (span *Span) End() {
	if span == nil {
		return
	}
	span.Mutex.Lock()
	defer span.Mutex.Unlock()
	if span.Ended {
		return
	}
	span.Ended = true
	span.EndTime = time.Now().UTC()
	if span.Sender == nil {
		return
	}
	_ = span.Sender.Send(ss.SpanEvent{
		Name:           span.Name,
		Application:    span.Application,
		TraceID:        span.TraceID,
		SpanID:         span.SpanID,
		ParentID:       span.ParentID,
		SpanKind:       span.Kind,
		StartTime:      span.StartTime,
		EndTime:        span.EndTime,
		HasError:       span.HasError,
		StatusMessage:  span.StatusMessage,
		HTTPStatusCode: span.HTTPStatusCode,
		Attributes:     cloneMap(span.Attributes),
		Events:         append([]ss.TimedEvent(nil), span.Events...),
	})
}

// Set stores an attribute on the span.
func (span *Span) Set(key string, value any) {
	if span == nil || key == "" {
		return
	}
	span.Mutex.Lock()
	defer span.Mutex.Unlock()
	if span.Ended {
		return
	}
	if span.Attributes == nil {
		span.Attributes = map[string]any{}
	}
	span.Attributes[key] = value
}

// AddEvent attaches a point event to the span.
func (span *Span) AddEvent(name string, attrs map[string]any) {
	if span == nil || name == "" {
		return
	}
	span.Mutex.Lock()
	defer span.Mutex.Unlock()
	if span.Ended {
		return
	}
	span.Events = append(span.Events, ss.TimedEvent{
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
	span.Mutex.Lock()
	defer span.Mutex.Unlock()
	if span.Ended {
		return
	}
	span.HasError = true
	span.StatusMessage = err.Error()
	if span.Attributes == nil {
		span.Attributes = map[string]any{}
	}
	span.Attributes["exception.message"] = err.Error()
	span.Attributes["exception.type"] = fmt.Sprintf("%T", err)
}

// SetHTTPStatus records an HTTP status code.
func (span *Span) SetHTTPStatus(statusCode int) {
	if span == nil || statusCode <= 0 {
		return
	}
	span.Mutex.Lock()
	defer span.Mutex.Unlock()
	if span.Ended {
		return
	}
	span.HTTPStatusCode = statusCode
	if span.Attributes == nil {
		span.Attributes = map[string]any{}
	}
	span.Attributes["http.status_code"] = statusCode
	if statusCode >= 500 {
		span.HasError = true
		if span.StatusMessage == "" {
			span.StatusMessage = fmt.Sprintf("HTTP %d", statusCode)
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
		return span.TraceID, span.SpanID, span.TraceID != ""
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

func contextWithSpan(ctx context.Context, span *Span) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, activeSpanKey, span)
}

func parentFromContext(ctx context.Context) (traceID, parentSpanID string, hasParent bool) {
	if span := FromContext(ctx); span != nil && span.TraceID != "" {
		return span.TraceID, span.SpanID, true
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

func defaultKind(kind string, hasParent bool) string {
	if kind != "" {
		return kind
	}
	if hasParent {
		return "internal"
	}
	return "server"
}

func newTraceID() (string, error) {
	traceID, err := trace_id.New()
	if err != nil {
		return "", err
	}
	return trace_id.RemoveDashes(traceID), nil
}

func newSpanID() (string, error) {
	var randomBytes [8]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		return "", fmt.Errorf("generate span id: %w", err)
	}
	return hex.EncodeToString(randomBytes[:]), nil
}

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
