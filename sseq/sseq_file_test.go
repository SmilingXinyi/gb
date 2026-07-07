package sseq

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSetupFileProvider(t *testing.T) {
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "spans.clef")

	Setup(Config{
		Provider:      ProviderFile,
		Application:   "unit-test",
		BatchSize:     1,
		FlushInterval: time.Hour,
		File: FileConfig{
			Filename: filename,
		},
	})
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

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	body := string(content)
	for _, name := range []string{"root", "child"} {
		if !strings.Contains(body, `"@mt":"`+name+`"`) {
			t.Fatalf("missing span %q in file: %q", name, body)
		}
	}
}
