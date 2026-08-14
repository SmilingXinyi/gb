package seq

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/SmilingXinyi/gb/sseq/internal"
)

// Encoder encodes spans as Seq CLEF JSON lines.
type Encoder struct{}

// Encode serializes a span and attached events into CLEF JSON line(s).
func (Encoder) Encode(event ss.SpanEvent) ([]byte, error) {
	spanRecord, err := encodeSpan(event)
	if err != nil {
		return nil, err
	}
	records := [][]byte{spanRecord}
	for _, timedEvent := range event.Events {
		pointRecord, pointErr := encodePoint(ss.SpanEvent{
			Name:        timedEvent.Name,
			Application: event.Application,
			TraceID:     event.TraceID,
			SpanID:      event.SpanID,
			EndTime:     timedEvent.Time,
			Attributes:  timedEvent.Attributes,
		})
		if pointErr != nil {
			return nil, pointErr
		}
		records = append(records, pointRecord)
	}
	return joinRecords(records)
}

func encodeSpan(event ss.SpanEvent) ([]byte, error) {
	clefEvent := map[string]any{
		"@t":  event.EndTime.UTC().Format(time.RFC3339Nano),
		"@st": event.StartTime.UTC().Format(time.RFC3339Nano),
		"@tr": event.TraceID,
		"@sp": event.SpanID,
		"@mt": event.Name,
		"@l":  level(event),
		"@sk": normalizeKind(event.SpanKind),
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
	applyAttributes(clefEvent, event.Attributes)
	payload, err := json.Marshal(clefEvent)
	if err != nil {
		return nil, fmt.Errorf("marshal clef span: %w", err)
	}
	return payload, nil
}

func encodePoint(event ss.SpanEvent) ([]byte, error) {
	eventTime := event.EndTime
	if eventTime.IsZero() {
		eventTime = event.StartTime
	}
	clefEvent := map[string]any{
		"@t":  eventTime.UTC().Format(time.RFC3339Nano),
		"@tr": event.TraceID,
		"@sp": event.SpanID,
		"@mt": event.Name,
		"@l":  "Information",
	}
	if event.Application != "" {
		clefEvent["Application"] = event.Application
	}
	applyAttributes(clefEvent, event.Attributes)
	payload, err := json.Marshal(clefEvent)
	if err != nil {
		return nil, fmt.Errorf("marshal clef point event: %w", err)
	}
	return payload, nil
}

// applyAttributes copies user attributes, skipping CLEF reserved keys.
func applyAttributes(clefEvent map[string]any, attributes map[string]any) {
	for key, value := range attributes {
		if !isSafeClefAttribute(key) {
			continue
		}
		clefEvent[key] = value
	}
}

// isSafeClefAttribute reports whether key may be written as a CLEF property.
func isSafeClefAttribute(key string) bool {
	if key == "" || strings.HasPrefix(key, "@") {
		return false
	}
	switch key {
	case "Application", "ErrorMessage", "StatusCode":
		return false
	default:
		return true
	}
}

func level(event ss.SpanEvent) string {
	if event.HasError {
		return "Error"
	}
	return "Information"
}

func normalizeKind(spanKind string) string {
	switch strings.ToLower(spanKind) {
	case "server":
		return "Server"
	case "client":
		return "Client"
	case "producer":
		return "Producer"
	case "consumer":
		return "Consumer"
	default:
		return "Internal"
	}
}

func joinRecords(records [][]byte) ([]byte, error) {
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
