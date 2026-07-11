package sender

import (
	"encoding/json"
	"fmt"
	"time"
)

// TimedEvent is a point-in-time annotation attached to a span or emitted standalone.
type TimedEvent struct {
	Name       string
	Time       time.Time
	Attributes map[string]string
}

// SpanEvent describes a completed span or a standalone trace point event.
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
	// Events are span-attached point events (OTel span events / Seq correlated logs).
	Events []TimedEvent
	// Attributes are optional properties for standalone point events.
	Attributes map[string]string
	// PointEvent marks a standalone instantaneous event (no span duration).
	// Seq: CLEF log with @tr/@sp and without @st.
	// Axiom: zero-duration record with parent_span_id linking to the active span.
	PointEvent bool
}

// EncodeSpanEvent serializes a span or point event into one or more CLEF JSON lines.
//
// Seq model (from Seq docs):
//   - @tr marks membership in a trace
//   - @st additionally marks a span
//   - without @st, the record is a correlated log/event on the trace (SerilogTracing
//     maps Activity.Events the same way via ActivityEvents.AsLogEvents)
func EncodeSpanEvent(event SpanEvent) ([]byte, error) {
	records, err := EncodeSpanRecords(event)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("encode clef span: empty payload")
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

// EncodeSpanRecords serializes a span and its attached events into CLEF JSON objects.
func EncodeSpanRecords(event SpanEvent) ([][]byte, error) {
	if event.PointEvent {
		record, err := encodeClefPointEvent(event)
		if err != nil {
			return nil, err
		}
		return [][]byte{record}, nil
	}

	spanRecord, err := encodeClefSpan(event)
	if err != nil {
		return nil, err
	}
	records := [][]byte{spanRecord}

	for _, timedEvent := range event.Events {
		pointRecord, pointErr := encodeClefPointEvent(SpanEvent{
			Name:        timedEvent.Name,
			Application: event.Application,
			TraceID:     event.TraceID,
			SpanID:      event.SpanID,
			EndTime:     timedEvent.Time,
			PointEvent:  true,
			Attributes:  timedEvent.Attributes,
		})
		if pointErr != nil {
			return nil, pointErr
		}
		records = append(records, pointRecord)
	}
	return records, nil
}

// encodeClefSpan serializes a completed span CLEF record (includes @st).
func encodeClefSpan(event SpanEvent) ([]byte, error) {
	clefEvent := map[string]any{
		"@t":  event.EndTime.UTC().Format(time.RFC3339Nano),
		"@st": event.StartTime.UTC().Format(time.RFC3339Nano),
		"@tr": event.TraceID,
		"@sp": event.SpanID,
		"@mt": event.Name,
		"@l":  clefLevel(event),
		"@sk": event.SpanKind,
	}
	if event.ParentID != "" {
		clefEvent["@ps"] = event.ParentID
	}
	if event.Application != "" {
		clefEvent["Application"] = event.Application
	}
	if event.StatusMessage != "" {
		clefEvent["ErrorMessage"] = event.StatusMessage
	}
	if event.HTTPStatusCode > 0 {
		clefEvent["StatusCode"] = event.HTTPStatusCode
	}

	payload, err := json.Marshal(clefEvent)
	if err != nil {
		return nil, fmt.Errorf("marshal clef span: %w", err)
	}
	return payload, nil
}

// encodeClefPointEvent serializes a Seq-correlated log/event without @st.
//
// For standalone Event() calls, ParentID carries the active span id and becomes @sp
// so Seq attaches the log to that span (same model as SerilogTracing Activity events).
func encodeClefPointEvent(event SpanEvent) ([]byte, error) {
	eventTime := event.EndTime
	if eventTime.IsZero() {
		eventTime = event.StartTime
	}

	spanID := event.SpanID
	if event.PointEvent && event.ParentID != "" {
		spanID = event.ParentID
	}

	clefEvent := map[string]any{
		"@t":  eventTime.UTC().Format(time.RFC3339Nano),
		"@tr": event.TraceID,
		"@sp": spanID,
		"@mt": event.Name,
		"@l":  "Information",
	}
	if event.Application != "" {
		clefEvent["Application"] = event.Application
	}
	for key, value := range event.Attributes {
		if key == "" {
			continue
		}
		clefEvent[key] = value
	}

	payload, err := json.Marshal(clefEvent)
	if err != nil {
		return nil, fmt.Errorf("marshal clef point event: %w", err)
	}
	return payload, nil
}

// clefLevel maps span status to a Seq log level.
func clefLevel(event SpanEvent) string {
	if event.HasError {
		return "Error"
	}
	return "Information"
}
