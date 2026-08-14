package sseq_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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
	t.Cleanup(func() { _ = sseq.Shutdown() })

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
	t.Cleanup(func() { _ = sseq.Shutdown() })

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
	t.Cleanup(func() { _ = sseq.Shutdown() })

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
	t.Cleanup(func() { _ = sseq.Shutdown() })

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
	t.Cleanup(func() { _ = sseq.Shutdown() })

	handler := sseq.HTTP(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusOK)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/users", nil))
	sseq.Shutdown()

	if !strings.Contains(receivedBody, `"@mt":"GET"`) {
		t.Fatalf("missing span: %q", receivedBody)
	}
	if strings.Contains(receivedBody, `"@mt":"GET /api/users"`) {
		t.Fatalf("span name should not include the raw path: %q", receivedBody)
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

func TestReservedClefAttributesAreNotOverwritten(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "reserved.clef")
	if err := sseq.SetupSeqFile(filename, "file-app"); err != nil {
		t.Fatalf("SetupSeqFile() error = %v", err)
	}
	t.Cleanup(func() { _ = sseq.Shutdown() })

	err := sseq.Trace(context.Background(), "query users", "server", func(ctx context.Context) error {
		sseq.Set(ctx, "@tr", "should-not-overwrite")
		sseq.Set(ctx, "@mt", "nope")
		sseq.Set(ctx, "user.id", "42")
		return nil
	})
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}
	sseq.Shutdown()

	records := readJSONLines(t, filename)
	if len(records) == 0 {
		t.Fatal("expected span record")
	}
	if records[0]["@mt"] != "query users" {
		t.Fatalf("@mt = %v", records[0]["@mt"])
	}
	if records[0]["@tr"] == "should-not-overwrite" {
		t.Fatal("trace id was overwritten by attribute")
	}
	if records[0]["user.id"] != "42" {
		t.Fatalf("user.id = %v", records[0]["user.id"])
	}
}

func TestSeqHTTPAcceptsStatusOK(t *testing.T) {
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		receivedBody = string(body)
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var exportErrors []error
	var mutex sync.Mutex
	if err := sseq.SetupSeq(server.URL, "", "ok-status",
		sseq.WithBatchSize(1),
		sseq.WithErrorHandler(func(err error) {
			mutex.Lock()
			exportErrors = append(exportErrors, err)
			mutex.Unlock()
		}),
	); err != nil {
		t.Fatalf("SetupSeq() error = %v", err)
	}
	t.Cleanup(func() { _ = sseq.Shutdown() })

	if err := sseq.Trace(context.Background(), "ok", "", func(context.Context) error {
		return nil
	}); err != nil {
		t.Fatalf("Trace() error = %v", err)
	}
	sseq.Shutdown()

	mutex.Lock()
	defer mutex.Unlock()
	if len(exportErrors) != 0 {
		t.Fatalf("unexpected export errors: %v", exportErrors)
	}
	if !strings.Contains(receivedBody, `"@mt":"ok"`) {
		t.Fatalf("missing span: %q", receivedBody)
	}
}

func TestShutdownWaitsForInFlightSpan(t *testing.T) {
	var receivedBody string
	var mutex sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		mutex.Lock()
		receivedBody += string(body)
		mutex.Unlock()
		response.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	if err := sseq.SetupSeq(server.URL, "", "drain", sseq.WithBatchSize(1)); err != nil {
		t.Fatalf("SetupSeq() error = %v", err)
	}
	t.Cleanup(func() { _ = sseq.Shutdown() })

	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- sseq.Trace(context.Background(), "slow", "", func(context.Context) error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started

	shutdownDone := make(chan struct{})
	go func() {
		sseq.Shutdown()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		t.Fatal("Shutdown returned before in-flight span ended")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatalf("Trace() error = %v", err)
	}
	select {
	case <-shutdownDone:
	case <-time.After(time.Second):
		t.Fatal("Shutdown did not return after span ended")
	}

	mutex.Lock()
	body := receivedBody
	mutex.Unlock()
	if !strings.Contains(body, `"@mt":"slow"`) {
		t.Fatalf("missing span after shutdown: %q", body)
	}
}

func TestHTTPMiddlewarePanicMarksError(t *testing.T) {
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		receivedBody = string(body)
		response.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	if err := sseq.SetupSeq(server.URL, "", "panic", sseq.WithBatchSize(1)); err != nil {
		t.Fatalf("SetupSeq() error = %v", err)
	}
	t.Cleanup(func() { _ = sseq.Shutdown() })

	handler := sseq.HTTP(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	func() {
		defer func() {
			if recover() == nil {
				t.Error("expected panic to propagate")
			}
		}()
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/panic", nil))
	}()
	sseq.Shutdown()

	if !strings.Contains(receivedBody, `"@l":"Error"`) {
		t.Fatalf("expected error span: %q", receivedBody)
	}
	if !strings.Contains(receivedBody, "boom") {
		t.Fatalf("expected panic message: %q", receivedBody)
	}
}

func TestHTTPMiddlewareUnwrapAndHijack(t *testing.T) {
	inner := &hijackableWriter{ResponseWriter: httptest.NewRecorder()}
	handler := sseq.HTTP(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		unwrapper, ok := response.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			t.Error("ResponseWriter does not implement Unwrap")
			return
		}
		if unwrapper.Unwrap() != inner {
			t.Error("Unwrap() did not return the original writer")
		}
		hijacker, ok := response.(http.Hijacker)
		if !ok {
			t.Error("ResponseWriter does not implement http.Hijacker")
			return
		}
		_, _, err := hijacker.Hijack()
		if err == nil {
			t.Error("expected hijack error from inner writer")
		}
		if !inner.hijacked {
			t.Error("inner Hijack was not called")
		}
	}))
	handler.ServeHTTP(inner, httptest.NewRequest(http.MethodGet, "/ws", nil))
}

func TestInjectExtractTraceparent(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "traceparent.clef")
	if err := sseq.SetupSeqFile(filename, "traceparent"); err != nil {
		t.Fatalf("SetupSeqFile() error = %v", err)
	}
	t.Cleanup(func() { _ = sseq.Shutdown() })

	header := make(http.Header)
	err := sseq.Trace(context.Background(), "outbound", "client", func(ctx context.Context) error {
		sseq.Inject(ctx, header)
		traceID, spanID, ok := sseq.IDs(ctx)
		if !ok {
			t.Fatal("expected span ids")
		}
		extractedTraceID, extractedSpanID, extracted := sseq.Extract(header)
		if !extracted {
			t.Fatal("expected extract success")
		}
		if extractedTraceID != strings.ToLower(traceID) || extractedSpanID != strings.ToLower(spanID) {
			t.Fatalf("extracted (%s, %s) want (%s, %s)", extractedTraceID, extractedSpanID, traceID, spanID)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}

	if _, _, ok := sseq.Extract(http.Header{}); ok {
		t.Fatal("expected empty header extract to fail")
	}
}

func TestHTTPMiddlewareContinuesRemoteTrace(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "remote.clef")
	if err := sseq.SetupSeqFile(filename, "remote"); err != nil {
		t.Fatalf("SetupSeqFile() error = %v", err)
	}
	t.Cleanup(func() { _ = sseq.Shutdown() })

	remoteTraceID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	remoteSpanID := "bbbbbbbbbbbbbbbb"
	request := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	request.Header.Set("traceparent", "00-"+remoteTraceID+"-"+remoteSpanID+"-01")

	handler := sseq.HTTP(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		traceID, _, ok := sseq.IDs(request.Context())
		if !ok || traceID != remoteTraceID {
			t.Errorf("trace id = %q, ok = %v", traceID, ok)
		}
		response.WriteHeader(http.StatusNoContent)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), request)
	sseq.Shutdown()

	records := readJSONLines(t, filename)
	if len(records) == 0 {
		t.Fatal("expected span record")
	}
	if records[0]["@tr"] != remoteTraceID {
		t.Fatalf("@tr = %v", records[0]["@tr"])
	}
	if records[0]["@ps"] != remoteSpanID {
		t.Fatalf("@ps = %v", records[0]["@ps"])
	}
	if _, exists := records[0]["http.route"]; exists {
		t.Fatal("http.route should not be set from the raw path")
	}
	if records[0]["http.target"] != "/users/42" {
		t.Fatalf("http.target = %v", records[0]["http.target"])
	}
	if records[0]["@mt"] != "GET" {
		t.Fatalf("@mt = %v, want GET", records[0]["@mt"])
	}
}

func TestTracePanicMarksError(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "trace-panic.clef")
	if err := sseq.SetupSeqFile(filename, "panic"); err != nil {
		t.Fatalf("SetupSeqFile() error = %v", err)
	}
	t.Cleanup(func() { _ = sseq.Shutdown() })

	func() {
		defer func() {
			if recover() == nil {
				t.Error("expected panic to propagate")
			}
		}()
		_ = sseq.Trace(context.Background(), "work", "", func(context.Context) error {
			panic("boom")
		})
	}()
	if err := sseq.Shutdown(); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	records := readJSONLines(t, filename)
	if len(records) == 0 {
		t.Fatal("expected span record")
	}
	if records[0]["@l"] != "Error" {
		t.Fatalf("level = %v", records[0]["@l"])
	}
	if records[0]["ErrorMessage"] != "panic: boom" {
		t.Fatalf("ErrorMessage = %v", records[0]["ErrorMessage"])
	}
}

func TestWithErrorHandlerNilSilences(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "silent.clef")
	if err := sseq.SetupSeqFile(filename, "silent", sseq.WithErrorHandler(nil)); err != nil {
		t.Fatalf("SetupSeqFile() error = %v", err)
	}
	t.Cleanup(func() { _ = sseq.Shutdown() })
	if err := sseq.Trace(context.Background(), "ok", "", func(context.Context) error {
		return nil
	}); err != nil {
		t.Fatalf("Trace() error = %v", err)
	}
}

func TestHTTPMiddlewareHidesHijackerWhenUnsupported(t *testing.T) {
	handler := sseq.HTTP(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if _, ok := response.(http.Hijacker); ok {
			t.Error("wrapper should not implement http.Hijacker when the inner writer does not")
		}
		if _, ok := response.(http.Flusher); !ok {
			t.Error("wrapper should implement http.Flusher when the inner writer does")
		}
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

type hijackableWriter struct {
	http.ResponseWriter
	hijacked bool
}

func (writer *hijackableWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	writer.hijacked = true
	return nil, nil, fmt.Errorf("hijack not connected")
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
