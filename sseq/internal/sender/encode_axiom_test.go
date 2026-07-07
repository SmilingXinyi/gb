package sender

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEncodeAxiomSpanEvent(t *testing.T) {
	startTime := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	endTime := startTime.Add(120 * time.Millisecond)

	payload, err := EncodeAxiomSpanEvent(SpanEvent{
		Name:        "HTTP GET /api/users",
		Application: "demo-app",
		TraceID:     "0123456789abcdef0123456789abcdef",
		SpanID:      "0123456789abc000",
		SpanKind:    "Server",
		StartTime:   startTime,
		EndTime:     endTime,
	})
	if err != nil {
		t.Fatalf("EncodeAxiomSpanEvent() error = %v", err)
	}

	var axiomEvent map[string]any
	if err := json.Unmarshal(payload, &axiomEvent); err != nil {
		t.Fatalf("decode axiom payload: %v", err)
	}

	if axiomEvent["_time"] != endTime.UTC().Format(time.RFC3339Nano) {
		t.Fatalf("_time = %v", axiomEvent["_time"])
	}
	if axiomEvent["trace_id"] != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("trace_id = %v", axiomEvent["trace_id"])
	}
	if axiomEvent["span_id"] != "0123456789abc000" {
		t.Fatalf("span_id = %v", axiomEvent["span_id"])
	}
	if axiomEvent["name"] != "HTTP GET /api/users" {
		t.Fatalf("name = %v", axiomEvent["name"])
	}
	if axiomEvent["kind"] != "server" {
		t.Fatalf("kind = %v", axiomEvent["kind"])
	}
	if axiomEvent["duration"] != "120ms" {
		t.Fatalf("duration = %v", axiomEvent["duration"])
	}
	if axiomEvent["service.name"] != "demo-app" {
		t.Fatalf("service.name = %v", axiomEvent["service.name"])
	}
	if _, exists := axiomEvent["parent_span_id"]; exists {
		t.Fatalf("root span should not include parent_span_id")
	}
}

func TestEncodeAxiomSpanEventChild(t *testing.T) {
	startTime := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	endTime := startTime.Add(30 * time.Millisecond)

	payload, err := EncodeAxiomSpanEvent(SpanEvent{
		Name:      "Authenticate user",
		TraceID:   "0123456789abcdef0123456789abcdef",
		SpanID:    "0123456789abc001",
		ParentID:  "0123456789abc000",
		SpanKind:  "Internal",
		StartTime: startTime,
		EndTime:   endTime,
	})
	if err != nil {
		t.Fatalf("EncodeAxiomSpanEvent() error = %v", err)
	}

	var axiomEvent map[string]any
	if err := json.Unmarshal(payload, &axiomEvent); err != nil {
		t.Fatalf("decode axiom payload: %v", err)
	}

	if axiomEvent["parent_span_id"] != "0123456789abc000" {
		t.Fatalf("parent_span_id = %v", axiomEvent["parent_span_id"])
	}
	if axiomEvent["kind"] != "internal" {
		t.Fatalf("kind = %v", axiomEvent["kind"])
	}
}

func TestFormatAxiomDuration(t *testing.T) {
	testCases := []struct {
		duration time.Duration
		want     string
	}{
		{500 * time.Nanosecond, "500ns"},
		{750 * time.Microsecond, "750us"},
		{120 * time.Millisecond, "120ms"},
		{1500 * time.Millisecond, "1.500s"},
	}

	for _, testCase := range testCases {
		if got := formatAxiomDuration(testCase.duration); got != testCase.want {
			t.Fatalf("formatAxiomDuration(%v) = %q, want %q", testCase.duration, got, testCase.want)
		}
	}
}
