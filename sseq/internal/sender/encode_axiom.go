package sender

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Axiom encodes span timestamps using OpenTelemetry conventions:
//   - _time: span start timestamp in UTC RFC3339Nano
//   - duration: elapsed time as an integer number of nanoseconds (not milliseconds)
//
// Axiom UI may display durations in milliseconds, but ingest expects nanoseconds.
func EncodeAxiomSpanEvent(event SpanEvent) ([]byte, error) {
	axiomEvent := map[string]any{
		"_time":       formatAxiomEventTime(event.StartTime),
		"trace_id":    event.TraceID,
		"span_id":     event.SpanID,
		"name":        event.Name,
		"kind":        normalizeAxiomSpanKind(event.SpanKind),
		"duration":    axiomDurationNanoseconds(event.StartTime, event.EndTime),
		"error":       event.HasError,
		"status.code": axiomStatusCode(event),
	}
	if event.StatusMessage != "" {
		axiomEvent["status.message"] = event.StatusMessage
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

// formatAxiomEventTime renders the span start time for Axiom ingest.
func formatAxiomEventTime(startTime time.Time) string {
	return startTime.UTC().Format(time.RFC3339Nano)
}

// axiomDurationNanoseconds converts span elapsed time to Axiom's nanosecond integer format.
func axiomDurationNanoseconds(startTime, endTime time.Time) int64 {
	duration := endTime.Sub(startTime)
	if duration < 0 {
		return 0
	}
	return duration.Nanoseconds()
}

// axiomStatusCode maps span status to an OpenTelemetry-style status code.
func axiomStatusCode(event SpanEvent) string {
	if event.HasError {
		return "ERROR"
	}
	return "OK"
}

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
