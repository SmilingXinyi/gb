package file

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriterCloseFlushesAndRejectsLaterWrites(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "spans.clef")
	writer, err := NewWriter(filename)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}

	if err := writer.WritePayload([]byte("hello\n")); err != nil {
		t.Fatalf("WritePayload() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}

	contents, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(contents) != "hello\n" {
		t.Fatalf("file contents = %q", contents)
	}

	if err := writer.WritePayload([]byte("late\n")); err == nil {
		t.Fatal("expected WritePayload error after Close")
	}
}

func TestNewWriterRequiresFilename(t *testing.T) {
	if _, err := NewWriter(""); err == nil {
		t.Fatal("expected error")
	}
}
