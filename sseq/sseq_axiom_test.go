package sseq

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSetupAxiomProvider(t *testing.T) {
	var receivedBodies []string

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			response.WriteHeader(http.StatusInternalServerError)
			return
		}
		receivedBodies = append(receivedBodies, string(body))
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	Setup(Config{
		Provider:      ProviderAxiom,
		Application:   "unit-test",
		BatchSize:     1,
		FlushInterval: time.Hour,
		Axiom: AxiomConfig{
			Token:    "test-token",
			Dataset:  "otel-traces",
			Endpoint: server.URL,
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

	bodies := strings.Join(receivedBodies, "")
	for _, name := range []string{"root", "child"} {
		if !strings.Contains(bodies, `"name":"`+name+`"`) {
			t.Fatalf("missing span %q in payload: %q", name, bodies)
		}
	}
	if !strings.Contains(bodies, `"service.name":"unit-test"`) {
		t.Fatalf("missing service.name in payload: %q", bodies)
	}
	if !strings.Contains(bodies, `"parent_span_id"`) {
		t.Fatalf("expected child span parent_span_id in payload: %q", bodies)
	}
}

func TestSetupAxiomVectorFileFormat(t *testing.T) {
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "spans.ndjson")

	Setup(Config{
		Provider:      ProviderFile,
		Application:   "vector-app",
		BatchSize:     1,
		FlushInterval: time.Hour,
		File: FileConfig{
			Filename: filename,
			Format:   FileFormatAxiom,
		},
	})
	t.Cleanup(Shutdown)

	err := Do(context.Background(), "root", func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	Shutdown()

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var axiomEvent map[string]any
	if err := json.Unmarshal(content, &axiomEvent); err != nil {
		t.Fatalf("decode axiom payload: %v", err)
	}
	if axiomEvent["name"] != "root" {
		t.Fatalf("name = %v", axiomEvent["name"])
	}
	if axiomEvent["service.name"] != "vector-app" {
		t.Fatalf("service.name = %v", axiomEvent["service.name"])
	}
	if axiomEvent["_time"] == nil {
		t.Fatal("expected _time field for Vector-compatible ingest")
	}
}
