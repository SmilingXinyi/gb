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

	if axiomEvent["_time"] != startTime.UTC().Format(time.RFC3339Nano) {
		t.Fatalf("_time = %v", axiomEvent["_time"])
	}
	if axiomEvent["_time"] == endTime.UTC().Format(time.RFC3339Nano) {
		t.Fatal("_time must be span start time, not end time")
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
	assertAxiomDurationNanoseconds(t, axiomEvent["duration"], 120*time.Millisecond)
	if axiomEvent["service.name"] != "demo-app" {
		t.Fatalf("service.name = %v", axiomEvent["service.name"])
	}
	if _, exists := axiomEvent["parent_span_id"]; exists {
		t.Fatalf("root span should not include parent_span_id")
	}
}

func TestEncodeAxiomSpanEventDurationNanoseconds(t *testing.T) {
	startTime := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)

	testCases := []struct {
		name     string
		elapsed  time.Duration
		wantNano int64
	}{
		{name: "30ms", elapsed: 30 * time.Millisecond, wantNano: 30_000_000},
		{name: "25ms", elapsed: 25 * time.Millisecond, wantNano: 25_000_000},
		{name: "1ms", elapsed: time.Millisecond, wantNano: 1_000_000},
		{name: "750us", elapsed: 750 * time.Microsecond, wantNano: 750_000},
		{name: "negative clamped to zero", elapsed: -5 * time.Millisecond, wantNano: 0},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			endTime := startTime.Add(testCase.elapsed)

			got := axiomDurationNanoseconds(startTime, endTime)
			if got != testCase.wantNano {
				t.Fatalf("axiomDurationNanoseconds() = %d, want %d", got, testCase.wantNano)
			}

			payload, err := EncodeAxiomSpanEvent(SpanEvent{
				Name:      testCase.name,
				TraceID:   "0123456789abcdef0123456789abcdef",
				SpanID:    "0123456789abc000",
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
			assertAxiomDurationNanoseconds(t, axiomEvent["duration"], time.Duration(testCase.wantNano))
		})
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

func TestEncodeAxiomSpanEventError(t *testing.T) {
	startTime := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	endTime := startTime.Add(30 * time.Millisecond)

	payload, err := EncodeAxiomSpanEvent(SpanEvent{
		Name:          "charge payment",
		TraceID:       "0123456789abcdef0123456789abcdef",
		SpanID:        "0123456789abc001",
		StartTime:     startTime,
		EndTime:       endTime,
		HasError:      true,
		StatusMessage: "payment failed",
	})
	if err != nil {
		t.Fatalf("EncodeAxiomSpanEvent() error = %v", err)
	}

	var axiomEvent map[string]any
	if err := json.Unmarshal(payload, &axiomEvent); err != nil {
		t.Fatalf("decode axiom payload: %v", err)
	}
	if axiomEvent["error"] != true {
		t.Fatalf("error = %v", axiomEvent["error"])
	}
	if axiomEvent["status.code"] != "ERROR" {
		t.Fatalf("status.code = %v", axiomEvent["status.code"])
	}
	if axiomEvent["status.message"] != "payment failed" {
		t.Fatalf("status.message = %v", axiomEvent["status.message"])
	}
}

func TestEncodeAxiomSpanEventWithAttachedEvents(t *testing.T) {
	startTime := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	eventTime := startTime.Add(5 * time.Millisecond)
	endTime := startTime.Add(20 * time.Millisecond)

	payload, err := EncodeAxiomSpanEvent(SpanEvent{
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
		t.Fatalf("EncodeAxiomSpanEvent() error = %v", err)
	}

	var axiomEvent map[string]any
	if err := json.Unmarshal(payload, &axiomEvent); err != nil {
		t.Fatalf("decode axiom payload: %v", err)
	}

	rawEvents, ok := axiomEvent["events"].([]any)
	if !ok || len(rawEvents) != 1 {
		t.Fatalf("events = %#v", axiomEvent["events"])
	}
	eventItem, ok := rawEvents[0].(map[string]any)
	if !ok {
		t.Fatalf("event item type = %T", rawEvents[0])
	}
	if eventItem["name"] != "cache.miss" {
		t.Fatalf("event name = %v", eventItem["name"])
	}
	if eventItem["timestamp"] != eventTime.UTC().Format(time.RFC3339Nano) {
		t.Fatalf("event timestamp = %v", eventItem["timestamp"])
	}
}

func TestEncodeAxiomPointEvent(t *testing.T) {
	eventTime := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)

	payload, err := EncodeAxiomSpanEvent(SpanEvent{
		Name:       "order.paid",
		TraceID:    "0123456789abcdef0123456789abcdef",
		SpanID:     "0123456789abc999",
		ParentID:   "0123456789abc000",
		EndTime:    eventTime,
		PointEvent: true,
		Attributes: map[string]string{"amount": "42"},
	})
	if err != nil {
		t.Fatalf("EncodeAxiomSpanEvent() error = %v", err)
	}

	var axiomEvent map[string]any
	if err := json.Unmarshal(payload, &axiomEvent); err != nil {
		t.Fatalf("decode axiom payload: %v", err)
	}
	if axiomEvent["duration"] != float64(0) {
		t.Fatalf("duration = %v", axiomEvent["duration"])
	}
	if axiomEvent["sseq.event"] != true {
		t.Fatalf("sseq.event = %v", axiomEvent["sseq.event"])
	}
	if axiomEvent["parent_span_id"] != "0123456789abc000" {
		t.Fatalf("parent_span_id = %v", axiomEvent["parent_span_id"])
	}
	if axiomEvent["span_id"] != "0123456789abc999" {
		t.Fatalf("span_id = %v", axiomEvent["span_id"])
	}
}

func assertAxiomDurationNanoseconds(t *testing.T, rawValue any, want time.Duration) {
	t.Helper()

	switch typed := rawValue.(type) {
	case float64:
		if int64(typed) != want.Nanoseconds() {
			t.Fatalf("duration = %v, want %d ns", typed, want.Nanoseconds())
		}
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			t.Fatalf("parse duration number: %v", err)
		}
		if parsed != want.Nanoseconds() {
			t.Fatalf("duration = %d, want %d ns", parsed, want.Nanoseconds())
		}
	case int64:
		if typed != want.Nanoseconds() {
			t.Fatalf("duration = %d, want %d ns", typed, want.Nanoseconds())
		}
	default:
		t.Fatalf("unexpected duration type %T: %v", rawValue, rawValue)
	}
}
