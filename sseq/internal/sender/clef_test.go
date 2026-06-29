package sender

import (
	"encoding/json"
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
