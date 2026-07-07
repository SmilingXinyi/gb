package sender

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileProviderWritesRotatedSpanFile(t *testing.T) {
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "spans.clef")

	sender, err := NewFile(FileBatchConfig{
		File: FileConfig{
			Filename: filename,
		},
		BatchSize:     1,
		FlushInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewFile() error = %v", err)
	}

	startTime := time.Now().UTC()
	if err := sender.Send(SpanEvent{
		Name:      "file span",
		TraceID:   "0123456789abcdef0123456789abcdef",
		SpanID:    "0123456789abc000",
		SpanKind:  "Server",
		StartTime: startTime,
		EndTime:   startTime,
	}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if err := sender.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	body := string(content)
	if !strings.Contains(body, `"@mt":"file span"`) {
		t.Fatalf("missing span payload in file: %q", body)
	}
}
