package axiom

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/SmilingXinyi/gb/sseq/internal"
)

func TestEncodeKeepsSpanWhenAttributeIsNotJSON(t *testing.T) {
	payload, err := Encoder{}.Encode(ss.SpanEvent{
		Name:      "query",
		TraceID:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		SpanID:    "bbbbbbbbbbbbbbbb",
		StartTime: time.Unix(0, 0).UTC(),
		EndTime:   time.Unix(0, 1).UTC(),
		Attributes: map[string]any{
			"user.id": "42",
			"ratio":   math.Inf(1),
		},
		Events: []ss.TimedEvent{{
			Name:       "checkpoint",
			Time:       time.Unix(0, 2).UTC(),
			Attributes: map[string]any{"ch": make(chan int)},
		}},
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	var record map[string]any
	if err := json.Unmarshal(payload, &record); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if record["name"] != "query" {
		t.Fatalf("name = %v", record["name"])
	}
	if record["_time"] != time.Unix(0, 0).UTC().Format(time.RFC3339Nano) {
		t.Fatalf("_time = %v", record["_time"])
	}
	attributes, ok := record["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("attributes = %#v", record["attributes"])
	}
	if attributes["user.id"] != "42" || attributes["ratio"] != "+Inf" {
		t.Fatalf("attributes = %#v", attributes)
	}
	events, ok := record["events"].([]any)
	if !ok || len(events) != 1 {
		t.Fatalf("events = %#v", record["events"])
	}
}
