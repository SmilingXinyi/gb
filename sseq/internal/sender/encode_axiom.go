package sender

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// EncodeAxiomSpanEvent serializes a span into an Axiom-compatible trace event.
// The payload matches fields expected by the OpenTelemetry Traces dashboard and Vector axiom sink.
func EncodeAxiomSpanEvent(event SpanEvent) ([]byte, error) {
	duration := event.EndTime.Sub(event.StartTime)
	if duration < 0 {
		duration = 0
	}

	axiomEvent := map[string]any{
		"_time":       event.StartTime.UTC().Format(time.RFC3339Nano),
		"trace_id":    event.TraceID,
		"span_id":     event.SpanID,
		"name":        event.Name,
		"kind":        normalizeAxiomSpanKind(event.SpanKind),
		"duration":    duration.Nanoseconds(),
		"error":       false,
		"status.code": "OK",
	}
	if event.ParentID != "" {
		axiomEvent["parent_span_id"] = event.ParentID
	}
	if event.Application != "" {
		axiomEvent["service.name"] = event.Application
	}

	payload, err := json.Marshal(axiomEvent)
	if err != nil {
		return nil, fmt.Errorf("marshal axiom span: %w", err)
	}
	return payload, nil
}

// normalizeAxiomSpanKind maps span kinds to OpenTelemetry semantic values.
func normalizeAxiomSpanKind(spanKind string) string {
	switch strings.ToLower(spanKind) {
	case "server":
		return "server"
	case "client":
		return "client"
	case "producer":
		return "producer"
	case "consumer":
		return "consumer"
	default:
		return "internal"
	}
}
