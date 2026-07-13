package axiom

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/SmilingXinyi/gb/sseq/internal"
)

// Encoder encodes spans as Axiom NDJSON events.
type Encoder struct{}

// Encode serializes a span into an Axiom trace event JSON line.
func (Encoder) Encode(event ss.SpanEvent) ([]byte, error) {
	axiomEvent := map[string]any{
		"_time":       event.StartTime.UTC().Format(time.RFC3339Nano),
		"trace_id":    event.TraceID,
		"span_id":     event.SpanID,
		"name":        event.Name,
		"kind":        normalizeKind(event.SpanKind),
		"duration":    durationNanoseconds(event.StartTime, event.EndTime),
		"error":       event.HasError,
		"status.code": statusCode(event),
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
	if event.HTTPStatusCode > 0 {
		axiomEvent["http.status_code"] = event.HTTPStatusCode
	}
	if len(event.Attributes) > 0 {
		axiomEvent["attributes"] = event.Attributes
	}
	if len(event.Events) > 0 {
		axiomEvent["events"] = encodeEvents(event.Events)
	}

	payload, err := json.Marshal(axiomEvent)
	if err != nil {
		return nil, fmt.Errorf("marshal axiom span: %w", err)
	}
	return payload, nil
}

func encodeEvents(events []ss.TimedEvent) []map[string]any {
	encoded := make([]map[string]any, 0, len(events))
	for _, timedEvent := range events {
		item := map[string]any{
			"name":      timedEvent.Name,
			"timestamp": timedEvent.Time.UTC().UnixNano(),
		}
		if len(timedEvent.Attributes) > 0 {
			item["attributes"] = timedEvent.Attributes
		}
		encoded = append(encoded, item)
	}
	return encoded
}

func durationNanoseconds(startTime, endTime time.Time) int64 {
	duration := endTime.Sub(startTime)
	if duration < 0 {
		return 0
	}
	return duration.Nanoseconds()
}

func statusCode(event ss.SpanEvent) string {
	if event.HasError {
		return "ERROR"
	}
	return "OK"
}

func normalizeKind(spanKind string) string {
	switch strings.ToLower(spanKind) {
	case "server", "client", "producer", "consumer":
		return strings.ToLower(spanKind)
	default:
		return "internal"
	}
}
