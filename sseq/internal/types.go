// Package ss holds shared internal types for sseq.
package ss

import "time"

// TimedEvent is a point-in-time annotation attached to a span.
type TimedEvent struct {
	Name       string
	Time       time.Time
	Attributes map[string]any
}

// SpanEvent is a completed span ready for encoding and delivery.
type SpanEvent struct {
	Name           string
	Application    string
	TraceID        string
	SpanID         string
	ParentID       string
	SpanKind       string
	StartTime      time.Time
	EndTime        time.Time
	HasError       bool
	StatusMessage  string
	HTTPStatusCode int
	Events         []TimedEvent
	Attributes     map[string]any
}

// Encoder serializes span events for a provider.
type Encoder interface {
	Encode(event SpanEvent) ([]byte, error)
}

// Writer delivers encoded payload batches.
type Writer interface {
	WritePayload(payload []byte)
	Close() error
}

// BatchConfig controls asynchronous flush behavior.
type BatchConfig struct {
	BatchSize     int
	FlushInterval time.Duration
}

const (
	DefaultBatchSize     = 20
	DefaultFlushInterval = time.Second
	DefaultHTTPTimeout   = 10 * time.Second
)
