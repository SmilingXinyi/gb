package sseq

import (
	"context"
	"fmt"
	"net/http"
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

// Setup initializes the global span sender for the configured provider.
func Setup(config Config) error {
	setupMutex.Lock()
	defer setupMutex.Unlock()

	if globalSender != nil {
		_ = globalSender.Close()
		globalSender = nil
	}

	if err := validateConfig(config); err != nil {
		return err
	}

	builtSender, err := buildSender(config)
	if err != nil {
		return err
	}

	globalConfig = config
	globalSender = builtSender
	warnIgnoredProviderConfigs(config)
	return nil
}

// Shutdown flushes and closes the global span sender.
func Shutdown() {
	setupMutex.Lock()
	defer setupMutex.Unlock()

	if globalSender != nil {
		_ = globalSender.Close()
		globalSender = nil
	}
}

// buildSender creates the configured span sender. The caller must hold setupMutex.
func buildSender(config Config) (*sender.Sender, error) {
	provider := resolveProvider(config)
	switch provider {
	case ProviderHTTP:
		httpConfig := config.HTTP
		if httpConfig.Endpoint == "" {
			httpConfig.Endpoint = config.Endpoint
		}
		if httpConfig.APIKey == "" {
			httpConfig.APIKey = config.APIKey
		}
		return sender.NewHTTP(sender.HTTPBatchConfig{
			Endpoint:      httpConfig.Endpoint,
			APIKey:        httpConfig.APIKey,
			BatchSize:     config.BatchSize,
			FlushInterval: config.FlushInterval,
		}), nil
	case ProviderFile:
		return sender.NewFile(sender.FileBatchConfig{
			File: sender.FileConfig{
				Filename:   config.File.Filename,
				MaxSize:    config.File.MaxSize,
				MaxBackups: config.File.MaxBackups,
				MaxAge:     config.File.MaxAge,
				Compress:   config.File.Compress,
				Format:     sender.FileFormat(config.File.Format),
			},
			BatchSize:     config.BatchSize,
			FlushInterval: config.FlushInterval,
		})
	case ProviderAxiom:
		return sender.NewAxiom(sender.AxiomBatchConfig{
			Axiom: sender.AxiomConfig{
				Token:      config.Axiom.Token,
				Dataset:    config.Axiom.Dataset,
				Domain:     config.Axiom.Domain,
				Endpoint:   config.Axiom.Endpoint,
				HTTPClient: config.Axiom.HTTPClient,
			},
			BatchSize:     config.BatchSize,
			FlushInterval: config.FlushInterval,
		})
	default:
		return nil, fmt.Errorf("sseq: unsupported provider %q", provider)
	}
}

// resolveProvider returns the active provider for the given config.
func resolveProvider(config Config) Provider {
	if config.Provider != "" {
		return config.Provider
	}
	if config.File.Filename != "" {
		return ProviderFile
	}
	if config.Axiom.Token != "" && config.Axiom.Dataset != "" {
		return ProviderAxiom
	}
	if config.HTTP.Endpoint != "" || config.Endpoint != "" {
		return ProviderHTTP
	}
	return ""
}

// Do executes a function within a span and ends the span automatically.
func Do(ctx context.Context, name string, fn func(context.Context) error) error {
	ctx, span := Start(ctx, name)
	defer span.End()
	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
	}
	return err
}

// Event emits a standalone point-in-time event on the active trace.
//
// Backend mapping:
//   - Seq: CLEF log with @tr/@sp and without @st (correlated log on the trace)
//   - Axiom: zero-duration record linked via parent_span_id
//
// If ctx has no active span, Event is a no-op.
func Event(ctx context.Context, name string, attributes map[string]string) {
	if name == "" {
		return
	}

	setupMutex.RLock()
	currentConfig := globalConfig
	currentSender := globalSender
	setupMutex.RUnlock()

	if currentSender == nil {
		return
	}

	spanState, found := activeSpanContext(ctx)
	if !found || spanState.traceID == "" {
		return
	}

	eventTime := time.Now().UTC()
	spanID, err := newSpanID()
	if err != nil {
		spanID = "0000000000000000"
	}

	_ = currentSender.Send(sender.SpanEvent{
		Name:        name,
		Application: currentConfig.Application,
		TraceID:     spanState.traceID,
		SpanID:      spanID,
		ParentID:    spanState.spanID,
		StartTime:   eventTime,
		EndTime:     eventTime,
		Attributes:  cloneAttributes(attributes),
		PointEvent:  true,
	})
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
	name           string
	application    string
	traceID        string
	spanID         string
	parentID       string
	spanKind       string
	startTime      time.Time
	endTime        time.Time
	ended          bool
	sender         *sender.Sender
	hasError       bool
	statusMessage  string
	httpStatusCode int
	events         []sender.TimedEvent
	eventsMutex    sync.Mutex
}

// End completes the span and sends it to the configured provider.
func (span *Span) End() {
	if span == nil || span.ended {
		return
	}
	span.ended = true
	span.endTime = time.Now().UTC()

	if span.sender == nil {
		return
	}

	span.eventsMutex.Lock()
	attachedEvents := append([]sender.TimedEvent(nil), span.events...)
	span.eventsMutex.Unlock()

	_ = span.sender.Send(sender.SpanEvent{
		Name:           span.name,
		Application:    span.application,
		TraceID:        span.traceID,
		SpanID:         span.spanID,
		ParentID:       span.parentID,
		SpanKind:       span.spanKind,
		StartTime:      span.startTime,
		EndTime:        span.endTime,
		HasError:       span.hasError,
		StatusMessage:  span.statusMessage,
		HTTPStatusCode: span.httpStatusCode,
		Events:         attachedEvents,
	})
}

// AddEvent attaches a point-in-time event to this span.
//
// Backend mapping:
//   - Axiom: included in the span's events[] array on End
//   - Seq: emitted as a correlated CLEF log (same @tr/@sp, no @st) when the span ends
//
// This mirrors SerilogTracing's Activity.Events -> log events model for Seq.
func (span *Span) AddEvent(name string, attributes map[string]string) {
	if span == nil || span.ended || name == "" {
		return
	}

	span.eventsMutex.Lock()
	defer span.eventsMutex.Unlock()
	span.events = append(span.events, sender.TimedEvent{
		Name:       name,
		Time:       time.Now().UTC(),
		Attributes: cloneAttributes(attributes),
	})
}

// RecordError marks the span as failed with the given error message.
func (span *Span) RecordError(err error) {
	if span == nil || err == nil {
		return
	}
	span.hasError = true
	span.statusMessage = err.Error()
}

// SetHTTPStatusCode records the HTTP response status for server spans.
func (span *Span) SetHTTPStatusCode(statusCode int) {
	if span == nil || statusCode <= 0 {
		return
	}
	span.httpStatusCode = statusCode
	if statusCode >= http.StatusInternalServerError {
		span.hasError = true
		if span.statusMessage == "" {
			span.statusMessage = fmt.Sprintf("HTTP %d", statusCode)
		}
	}
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

// cloneAttributes copies attribute key/value pairs.
func cloneAttributes(attributes map[string]string) map[string]string {
	if len(attributes) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(attributes))
	for key, value := range attributes {
		cloned[key] = value
	}
	return cloned
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
