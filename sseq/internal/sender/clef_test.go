package sender

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestEncodeSpanEvent(t *testing.T) {
	startTime := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	endTime := startTime.Add(120 * time.Millisecond)

	payload, err := EncodeSpanEvent(SpanEvent{
		Name:        "HTTP GET /api/users",
		Application: "demo-app",
		TraceID:     "0123456789abcdef0123456789abcdef",
		SpanID:      "0123456789abc000",
		ParentID:    "",
		SpanKind:    "Server",
		StartTime:   startTime,
		EndTime:     endTime,
	})
	if err != nil {
		t.Fatalf("EncodeSpanEvent() error = %v", err)
	}

	var clefEvent map[string]any
	if err := json.Unmarshal(payload, &clefEvent); err != nil {
		t.Fatalf("decode clef payload: %v", err)
	}

	if clefEvent["@tr"] != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("@tr = %v", clefEvent["@tr"])
	}
	if clefEvent["@sp"] != "0123456789abc000" {
		t.Fatalf("@sp = %v", clefEvent["@sp"])
	}
	if clefEvent["@mt"] != "HTTP GET /api/users" {
		t.Fatalf("@mt = %v", clefEvent["@mt"])
	}
	if clefEvent["@sk"] != "Server" {
		t.Fatalf("@sk = %v", clefEvent["@sk"])
	}
	if _, exists := clefEvent["@ps"]; exists {
		t.Fatalf("root span should not include @ps")
	}
	if _, exists := clefEvent["@st"]; !exists {
		t.Fatal("span record must include @st")
	}
}

func TestEncodeSpanEventChild(t *testing.T) {
	startTime := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	endTime := startTime.Add(30 * time.Millisecond)

	payload, err := EncodeSpanEvent(SpanEvent{
		Name:      "Authenticate user",
		TraceID:   "0123456789abcdef0123456789abcdef",
		SpanID:    "0123456789abc001",
		ParentID:  "0123456789abc000",
		SpanKind:  "Internal",
		StartTime: startTime,
		EndTime:   endTime,
	})
	if err != nil {
		t.Fatalf("EncodeSpanEvent() error = %v", err)
	}

	var clefEvent map[string]any
	if err := json.Unmarshal(payload, &clefEvent); err != nil {
		t.Fatalf("decode clef payload: %v", err)
	}

	if clefEvent["@ps"] != "0123456789abc000" {
		t.Fatalf("@ps = %v", clefEvent["@ps"])
	}
}

func TestEncodeSpanEventWithAttachedEvents(t *testing.T) {
	startTime := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	eventTime := startTime.Add(10 * time.Millisecond)
	endTime := startTime.Add(30 * time.Millisecond)

	payload, err := EncodeSpanEvent(SpanEvent{
		Name:      "HTTP GET /api/orders",
		TraceID:   "0123456789abcdef0123456789abcdef",
		SpanID:    "0123456789abc000",
		SpanKind:  "Server",
		StartTime: startTime,
		EndTime:   endTime,
		Events: []TimedEvent{
			{
				Name: "cache.miss",
				Time: eventTime,
				Attributes: map[string]string{
					"key": "orders",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("EncodeSpanEvent() error = %v", err)
	}

	lines := strings.Split(string(payload), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected span + event lines, got %d: %q", len(lines), payload)
	}

	var spanRecord map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &spanRecord); err != nil {
		t.Fatalf("decode span record: %v", err)
	}
	if _, exists := spanRecord["@st"]; !exists {
		t.Fatal("first record must be a span with @st")
	}

	var eventRecord map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &eventRecord); err != nil {
		t.Fatalf("decode event record: %v", err)
	}
	if _, exists := eventRecord["@st"]; exists {
		t.Fatal("attached event must not include @st")
	}
	if eventRecord["@sp"] != "0123456789abc000" {
		t.Fatalf("@sp = %v", eventRecord["@sp"])
	}
	if eventRecord["@mt"] != "cache.miss" {
		t.Fatalf("@mt = %v", eventRecord["@mt"])
	}
	if eventRecord["key"] != "orders" {
		t.Fatalf("attribute key = %v", eventRecord["key"])
	}
}

func TestEncodeSpanPointEvent(t *testing.T) {
	eventTime := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)

	payload, err := EncodeSpanEvent(SpanEvent{
		Name:       "order.paid",
		TraceID:    "0123456789abcdef0123456789abcdef",
		SpanID:     "0123456789abc999",
		ParentID:   "0123456789abc000",
		EndTime:    eventTime,
		PointEvent: true,
		Attributes: map[string]string{"amount": "42"},
	})
	if err != nil {
		t.Fatalf("EncodeSpanEvent() error = %v", err)
	}

	var eventRecord map[string]any
	if err := json.Unmarshal(payload, &eventRecord); err != nil {
		t.Fatalf("decode point event: %v", err)
	}
	if _, exists := eventRecord["@st"]; exists {
		t.Fatal("point event must not include @st")
	}
	if eventRecord["@sp"] != "0123456789abc000" {
		t.Fatalf("@sp = %v, want owning span id", eventRecord["@sp"])
	}
	if eventRecord["amount"] != "42" {
		t.Fatalf("amount = %v", eventRecord["amount"])
	}
}
