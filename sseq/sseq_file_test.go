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

func TestSetupFileProvider(t *testing.T) {
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "spans.clef")

	if err := Setup(Config{
		Provider:      ProviderFile,
		Application:   "unit-test",
		BatchSize:     1,
		FlushInterval: time.Hour,
		File: FileConfig{
			Filename: filename,
		},
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	t.Cleanup(Shutdown)

	err := Do(context.Background(), "root", func(ctx context.Context) error {
		return Do(ctx, "child", func(ctx context.Context) error {
			return nil
		})
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	Shutdown()

	spans, err := readSpanEvents(filename)
	if err != nil {
		t.Fatalf("readSpanEvents() error = %v", err)
	}
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(spans))
	}

	spanByName := make(map[string]map[string]any, len(spans))
	for _, span := range spans {
		name, ok := span["@mt"].(string)
		if !ok || name == "" {
			t.Fatalf("span missing @mt: %+v", span)
		}
		spanByName[name] = span
	}

	root, ok := spanByName["root"]
	if !ok {
		t.Fatal("missing root span")
	}
	child, ok := spanByName["child"]
	if !ok {
		t.Fatal("missing child span")
	}

	if root["Application"] != "unit-test" {
		t.Fatalf("root Application = %v, want unit-test", root["Application"])
	}
	if child["Application"] != "unit-test" {
		t.Fatalf("child Application = %v, want unit-test", child["Application"])
	}
	if root["@tr"] != child["@tr"] {
		t.Fatalf("trace ids differ: root=%v child=%v", root["@tr"], child["@tr"])
	}
	if _, exists := root["@ps"]; exists {
		t.Fatalf("root span should not include @ps")
	}
	if child["@ps"] != root["@sp"] {
		t.Fatalf("child @ps = %v, want %v", child["@ps"], root["@sp"])
	}
	for _, key := range []string{"@t", "@st", "@sp", "@sk"} {
		if root[key] == nil || root[key] == "" {
			t.Fatalf("root span missing %s", key)
		}
		if child[key] == nil || child[key] == "" {
			t.Fatalf("child span missing %s", key)
		}
	}
}

// readSpanEvents reads newline-delimited CLEF span events from a file.
func readSpanEvents(filename string) ([]map[string]any, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var spans []map[string]any
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var span map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &span); err != nil {
			return nil, err
		}
		spans = append(spans, span)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return spans, nil
}
