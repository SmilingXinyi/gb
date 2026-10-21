package seq

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/SmilingXinyi/gb/sseq/internal"
)

func TestEncodeSkipsReservedClefKeys(t *testing.T) {
	payload, err := Encoder{}.Encode(ss.SpanEvent{
		Name:        "query users",
		Application: "api",
		TraceID:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		SpanID:      "bbbbbbbbbbbbbbbb",
		StartTime:   time.Unix(0, 0).UTC(),
		EndTime:     time.Unix(0, 1).UTC(),
		Attributes: map[string]any{
			"@tr":          "should-not-overwrite",
			"@mt":          "nope",
			"Application":  "attacker",
			"ErrorMessage": "nope",
			"StatusCode":   999,
			"user.id":      "42",
		},
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	var record map[string]any
	if err := json.Unmarshal(payload, &record); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if record["@tr"] != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("@tr = %v", record["@tr"])
	}
	if record["@mt"] != "query users" {
		t.Fatalf("@mt = %v", record["@mt"])
	}
	if record["Application"] != "api" {
		t.Fatalf("Application = %v", record["Application"])
	}
	if _, exists := record["ErrorMessage"]; exists {
		t.Fatalf("ErrorMessage should be absent, got %v", record["ErrorMessage"])
	}
	if record["user.id"] != "42" {
		t.Fatalf("user.id = %v", record["user.id"])
	}
}

func TestEncodePointEventsUseSeparateLines(t *testing.T) {
	payload, err := Encoder{}.Encode(ss.SpanEvent{
		Name:      "root",
		TraceID:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		SpanID:    "bbbbbbbbbbbbbbbb",
		StartTime: time.Unix(0, 0).UTC(),
		EndTime:   time.Unix(0, 1).UTC(),
		Events: []ss.TimedEvent{
			{Name: "cache.miss", Time: time.Unix(0, 2).UTC(), Attributes: map[string]any{"key": "orders"}},
		},
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	lines := strings.Split(string(payload), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d, payload = %q", len(lines), payload)
	}
}

func TestEncodeKeepsSpanWhenOneAttributeIsNotJSON(t *testing.T) {
	payload, err := Encoder{}.Encode(ss.SpanEvent{
		Name:      "query",
		TraceID:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		SpanID:    "bbbbbbbbbbbbbbbb",
		StartTime: time.Unix(0, 0).UTC(),
		EndTime:   time.Unix(0, 1).UTC(),
		Attributes: map[string]any{
			"user.id": "42",
			"ratio":   math.NaN(),
		},
		Events: []ss.TimedEvent{{
			Name:       "checkpoint",
			Time:       time.Unix(0, 2).UTC(),
			Attributes: map[string]any{"ch": make(chan int), "ok": true},
		}},
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	lines := strings.Split(string(payload), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d, payload = %q", len(lines), payload)
	}
	var spanRecord map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &spanRecord); err != nil {
		t.Fatalf("Unmarshal(span) error = %v", err)
	}
	if spanRecord["@mt"] != "query" || spanRecord["user.id"] != "42" || spanRecord["ratio"] != "NaN" {
		t.Fatalf("span record = %#v", spanRecord)
	}
	var eventRecord map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &eventRecord); err != nil {
		t.Fatalf("Unmarshal(event) error = %v", err)
	}
	if eventRecord["@mt"] != "checkpoint" || eventRecord["ok"] != true {
		t.Fatalf("event record = %#v", eventRecord)
	}
	if eventRecord["ch"] != "<unmarshalable chan int>" {
		t.Fatalf("ch = %#v", eventRecord["ch"])
	}
}
