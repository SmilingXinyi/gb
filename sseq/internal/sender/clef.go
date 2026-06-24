package sender

import (
	"encoding/json"
	"fmt"
	"time"
)

// SpanEvent describes a completed span to send to Seq as a CLEF event.
type SpanEvent struct {
	Name        string
	Application string
	TraceID     string
	SpanID      string
	ParentID    string
	SpanKind    string
	StartTime   time.Time
	EndTime     time.Time
}

// EncodeSpanEvent serializes a span into a Seq-compatible CLEF JSON line.
func EncodeSpanEvent(event SpanEvent) ([]byte, error) {
	clefEvent := map[string]any{
		"@t":  event.EndTime.UTC().Format(time.RFC3339Nano),
		"@st": event.StartTime.UTC().Format(time.RFC3339Nano),
		"@tr": event.TraceID,
		"@sp": event.SpanID,
		"@mt": event.Name,
		"@l":  "Information",
		"@sk": event.SpanKind,
	}
	if event.ParentID != "" {
		clefEvent["@ps"] = event.ParentID
	}
	if event.Application != "" {
		clefEvent["Application"] = event.Application
	}

	payload, err := json.Marshal(clefEvent)
	if err != nil {
		return nil, fmt.Errorf("marshal clef span: %w", err)
	}
	return payload, nil
}
