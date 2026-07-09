package sender

// PayloadEncoder serializes span events for a specific delivery backend.
type PayloadEncoder interface {
	Encode(event SpanEvent) ([]byte, error)
}

// PayloadWriter delivers flushed encoded batches to a single backend.
type PayloadWriter interface {
	WritePayload(payload []byte)
	Close() error
}

// ClefEncoder encodes spans as Seq-compatible CLEF JSON lines.
type ClefEncoder struct{}

// Encode serializes a span into a CLEF JSON line.
func (ClefEncoder) Encode(event SpanEvent) ([]byte, error) {
	return EncodeSpanEvent(event)
}

// AxiomEncoder encodes spans as Axiom OpenTelemetry-compatible NDJSON events.
type AxiomEncoder struct{}

// Encode serializes a span into an Axiom trace event JSON line.
func (AxiomEncoder) Encode(event SpanEvent) ([]byte, error) {
	return EncodeAxiomSpanEvent(event)
}
