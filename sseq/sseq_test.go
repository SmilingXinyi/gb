package sseq_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/SmilingXinyi/gb/sseq"
)

func TestTraceHierarchy(t *testing.T) {
	var receivedBodies []string
	var mutex sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			response.WriteHeader(http.StatusInternalServerError)
			return
		}
		mutex.Lock()
		receivedBodies = append(receivedBodies, string(body))
		mutex.Unlock()
		response.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	if err := sseq.SetupSeq(server.URL, "", "unit-test"); err != nil {
		t.Fatalf("SetupSeq() error = %v", err)
	}
	t.Cleanup(sseq.Shutdown)

	err := sseq.Trace(context.Background(), "root", "server", func(ctx context.Context) error {
		return sseq.Trace(ctx, "child-a", "", func(ctx context.Context) error {
			return sseq.Trace(ctx, "child-b", "", func(context.Context) error {
				return nil
			})
		})
	})
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}
	sseq.Shutdown()

	mutex.Lock()
	bodies := strings.Join(receivedBodies, "")
	mutex.Unlock()

	for _, name := range []string{"root", "child-a", "child-b"} {
		if !strings.Contains(bodies, `"@mt":"`+name+`"`) {
			t.Fatalf("missing span %q in payloads: %q", name, bodies)
		}
	}
	if !strings.Contains(bodies, `"@ps"`) {
		t.Fatalf("expected child spans with parent id, body=%q", bodies)
	}
}

func TestTraceRecordsError(t *testing.T) {
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		receivedBody = string(body)
		response.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	if err := sseq.SetupSeq(server.URL, "", ""); err != nil {
		t.Fatalf("SetupSeq() error = %v", err)
	}
	t.Cleanup(sseq.Shutdown)

	expectedErr := errors.New("payment failed")
	err := sseq.Trace(context.Background(), "charge", "", func(context.Context) error {
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("Trace() error = %v", err)
	}
	sseq.Shutdown()

	if !strings.Contains(receivedBody, `"@l":"Error"`) {
		t.Fatalf("expected error level: %q", receivedBody)
	}
}

func TestSeqFileAndAttributes(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "spans.clef")
	if err := sseq.SetupSeqFile(filename, "file-app"); err != nil {
		t.Fatalf("SetupSeqFile() error = %v", err)
	}
	t.Cleanup(sseq.Shutdown)

	err := sseq.Trace(context.Background(), "root", "server", func(ctx context.Context) error {
		sseq.Set(ctx, "user.id", "42")
		sseq.Event(ctx, "cache.miss", "key", "orders")
		return nil
	})
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}
	sseq.Shutdown()

	records := readJSONLines(t, filename)
	if len(records) < 2 {
		t.Fatalf("expected span + event, got %d", len(records))
	}
}

func TestResume(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "resume.clef")
	if err := sseq.SetupSeqFile(filename, ""); err != nil {
		t.Fatalf("SetupSeqFile() error = %v", err)
	}
	t.Cleanup(sseq.Shutdown)

	var producerTraceID, producerSpanID string
	err := sseq.Trace(context.Background(), "producer", "server", func(ctx context.Context) error {
		producerTraceID, producerSpanID, _ = sseq.IDs(ctx)
		workerCtx := sseq.Resume(context.Background(), producerTraceID, producerSpanID)
		return sseq.Trace(workerCtx, "consumer", "consumer", func(context.Context) error {
			return nil
		})
	})
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}
	sseq.Shutdown()

	if producerTraceID == "" {
		t.Fatal("expected producer trace id")
	}
}

func TestHTTPMiddleware(t *testing.T) {
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		receivedBody = string(body)
		response.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	if err := sseq.SetupSeq(server.URL, "", ""); err != nil {
		t.Fatalf("SetupSeq() error = %v", err)
	}
	t.Cleanup(sseq.Shutdown)

	handler := sseq.HTTP(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusOK)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/users", nil))
	sseq.Shutdown()

	if !strings.Contains(receivedBody, `"@mt":"GET /api/users"`) {
		t.Fatalf("missing span: %q", receivedBody)
	}
	if !strings.Contains(receivedBody, `"StatusCode":200`) {
		t.Fatalf("missing status: %q", receivedBody)
	}
}

func TestSetupRequiresValues(t *testing.T) {
	if err := sseq.SetupSeq("", "", ""); err == nil {
		t.Fatal("expected SetupSeq error")
	}
	if err := sseq.SetupAxiom("", "dataset", ""); err == nil {
		t.Fatal("expected SetupAxiom error")
	}
}

func readJSONLines(t *testing.T, filename string) []map[string]any {
	t.Helper()
	file, err := os.Open(filename)
	if err != nil {
		t.Fatalf("open: %v", err)
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
			t.Fatalf("decode: %v", err)
		}
		records = append(records, record)
	}
	return records
}