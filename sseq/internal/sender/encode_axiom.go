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
//   - events[].timestamp: Unix nanoseconds as an integer (Axiom Trace UI BigInt)
//
// Axiom UI may display durations in milliseconds, but ingest expects nanoseconds.
func EncodeAxiomSpanEvent(event SpanEvent) ([]byte, error) {
	records, err := EncodeAxiomSpanRecords(event)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("encode axiom span: empty payload")
	}
	if len(records) == 1 {
		return records[0], nil
	}

	totalSize := 0
	for _, record := range records {
		totalSize += len(record) + 1
	}
	payload := make([]byte, 0, totalSize)
	for index, record := range records {
		if index > 0 {
			payload = append(payload, '\n')
		}
		payload = append(payload, record...)
	}
	return payload, nil
}

// EncodeAxiomSpanRecords serializes a span or point event into Axiom NDJSON objects.
func EncodeAxiomSpanRecords(event SpanEvent) ([][]byte, error) {
	if event.PointEvent {
		record, err := encodeAxiomPointEvent(event)
		if err != nil {
			return nil, err
		}
		return [][]byte{record}, nil
	}

	record, err := encodeAxiomSpan(event)
	if err != nil {
		return nil, err
	}
	return [][]byte{record}, nil
}

// encodeAxiomSpan serializes a completed span for Axiom ingest.
func encodeAxiomSpan(event SpanEvent) ([]byte, error) {
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
	if len(event.Events) > 0 {
		axiomEvent["events"] = encodeAxiomTimedEvents(event.Events)
	}

	payload, err := json.Marshal(axiomEvent)
	if err != nil {
		return nil, fmt.Errorf("marshal axiom span: %w", err)
	}
	return payload, nil
}

// encodeAxiomPointEvent serializes a standalone zero-duration event for Axiom.
func encodeAxiomPointEvent(event SpanEvent) ([]byte, error) {
	eventTime := event.EndTime
	if eventTime.IsZero() {
		eventTime = event.StartTime
	}
	if eventTime.IsZero() {
		eventTime = time.Now().UTC()
	}

	axiomEvent := map[string]any{
		"_time":       formatAxiomEventTime(eventTime),
		"trace_id":    event.TraceID,
		"span_id":     event.SpanID,
		"name":        event.Name,
		"kind":        "internal",
		"duration":    int64(0),
		"error":       false,
		"status.code": "OK",
		"sseq.event":  true,
	}
	if event.ParentID != "" {
		axiomEvent["parent_span_id"] = event.ParentID
	}
	if event.Application != "" {
		axiomEvent["service.name"] = event.Application
	}
	if len(event.Attributes) > 0 {
		axiomEvent["attributes"] = event.Attributes
	}

	payload, err := json.Marshal(axiomEvent)
	if err != nil {
		return nil, fmt.Errorf("marshal axiom point event: %w", err)
	}
	return payload, nil
}

// encodeAxiomTimedEvents converts attached span events into Axiom/OTel event objects.
// Axiom Trace UI expects event timestamps as Unix nanoseconds (integer BigInt), not RFC3339 strings.
func encodeAxiomTimedEvents(events []TimedEvent) []map[string]any {
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
