package sseq

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAddEventEmitsClefCorrelatedLog(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "events.clef")
	if err := Setup(Config{
		Provider:      ProviderFile,
		Application:   "event-app",
		BatchSize:     1,
		FlushInterval: 10 * time.Millisecond,
		File: FileConfig{
			Filename: filename,
			Format:   FileFormatCLEF,
		},
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	defer Shutdown()

	err := Do(context.Background(), "HTTP GET /api/orders", func(ctx context.Context) error {
		ctx, span := Start(ctx, "Load orders")
		defer span.End()
		span.AddEvent("cache.miss", map[string]string{"key": "orders"})
		return nil
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	Shutdown()

	records := readJSONLines(t, filename)
	if len(records) < 3 {
		t.Fatalf("expected root + child span + event, got %d records", len(records))
	}

	foundEvent := false
	for _, record := range records {
		if record["@mt"] != "cache.miss" {
			continue
		}
		foundEvent = true
		if _, exists := record["@st"]; exists {
			t.Fatal("AddEvent CLEF record must not include @st")
		}
		if record["key"] != "orders" {
			t.Fatalf("key = %v", record["key"])
		}
		if record["@tr"] == nil || record["@sp"] == nil {
			t.Fatalf("event missing trace fields: %#v", record)
		}
	}
	if !foundEvent {
		t.Fatal("cache.miss event not found")
	}
}

func TestEventEmitsStandalonePoint(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "point.clef")
	if err := Setup(Config{
		Provider:      ProviderFile,
		Application:   "event-app",
		BatchSize:     1,
		FlushInterval: 10 * time.Millisecond,
		File: FileConfig{
			Filename: filename,
			Format:   FileFormatCLEF,
		},
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	defer Shutdown()

	err := Do(context.Background(), "HTTP POST /api/orders", func(ctx context.Context) error {
		Event(ctx, "order.paid", map[string]string{"amount": "42"})
		return nil
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	Shutdown()

	records := readJSONLines(t, filename)
	foundEvent := false
	for _, record := range records {
		if record["@mt"] != "order.paid" {
			continue
		}
		foundEvent = true
		if _, exists := record["@st"]; exists {
			t.Fatal("Event CLEF record must not include @st")
		}
		if record["amount"] != "42" {
			t.Fatalf("amount = %v", record["amount"])
		}
	}
	if !foundEvent {
		t.Fatal("order.paid event not found")
	}
}

func TestAddEventEmitsAxiomEventsArray(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "events.axiom.ndjson")
	if err := Setup(Config{
		Provider:      ProviderFile,
		Application:   "event-app",
		BatchSize:     1,
		FlushInterval: 10 * time.Millisecond,
		File: FileConfig{
			Filename: filename,
			Format:   FileFormatAxiom,
		},
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	defer Shutdown()

	err := Do(context.Background(), "root", func(ctx context.Context) error {
		ctx, span := Start(ctx, "child")
		defer span.End()
		span.AddEvent("retry", map[string]string{"attempt": "1"})
		return nil
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	Shutdown()

	records := readJSONLines(t, filename)
	found := false
	for _, record := range records {
		if record["name"] != "child" {
			continue
		}
		found = true
		rawEvents, ok := record["events"].([]any)
		if !ok || len(rawEvents) != 1 {
			t.Fatalf("events = %#v", record["events"])
		}
	}
	if !found {
		t.Fatal("child span with events not found")
	}
}

func readJSONLines(t *testing.T, filename string) []map[string]any {
	t.Helper()

	file, err := os.Open(filename)
	if err != nil {
		t.Fatalf("open %s: %v", filename, err)
	}
	defer file.Close()

	var records []map[string]any
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal(line, &record); err != nil {
			t.Fatalf("decode line %q: %v", line, err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan %s: %v", filename, err)
	}
	return records
}
