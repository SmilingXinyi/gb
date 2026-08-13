package seq

import (
	"encoding/json"
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
