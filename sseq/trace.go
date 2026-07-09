package sseq

import "context"

// ResumeTrace attaches remote trace identifiers to ctx so subsequent Start/Do calls
// continue an existing trace instead of starting a new one.
//
// parentSpanID is the producer span that enqueued async work. It becomes the parent
// of the next span created on the resumed context. Pass an empty parentSpanID to
// attach new spans to the trace without a parent link.
func ResumeTrace(ctx context.Context, traceID, parentSpanID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if traceID == "" {
		return ctx
	}

	return context.WithValue(ctx, activeSpanContextKey, spanContext{
		traceID: traceID,
		spanID:  parentSpanID,
	})
}

// TraceFromContext returns trace identifiers stored in ctx by Start or ResumeTrace.
// The returned spanID is the active span id used as the parent for the next span.
func TraceFromContext(ctx context.Context) (traceID string, spanID string, ok bool) {
	spanState, found := activeSpanContext(ctx)
	if !found || spanState.traceID == "" {
		return "", "", false
	}
	return spanState.traceID, spanState.spanID, true
}
