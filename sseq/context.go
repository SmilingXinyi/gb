package sseq

type contextKey struct{}

var activeSpanContextKey = contextKey{}

// spanContext stores trace identifiers propagated through context.Context.
type spanContext struct {
	traceID string
	spanID  string
}
