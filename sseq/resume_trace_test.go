package sseq

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestResumeTraceContinuesAsyncWorkerSpan(t *testing.T) {
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

	if err := Setup(Config{
		Endpoint:      server.URL,
		Application:   "unit-test",
		BatchSize:     1,
		FlushInterval: time.Hour,
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	t.Cleanup(Shutdown)

	var producerTraceID string
	var producerSpanID string

	err := Do(context.Background(), "HTTP POST /api/orders", func(requestContext context.Context) error {
		producerTraceID, producerSpanID, _ = TraceFromContext(requestContext)
		if producerTraceID == "" || producerSpanID == "" {
			t.Fatal("expected producer trace identifiers in context")
		}

		workerContext := ResumeTrace(context.Background(), producerTraceID, producerSpanID)
		return Do(workerContext, "Process order async", func(context.Context) error {
			return nil
		})
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	Shutdown()

	mutex.Lock()
	bodies := strings.Join(receivedBodies, "")
	mutex.Unlock()

	producerSpan, found := findClefSpanByName(bodies, "HTTP POST /api/orders")
	if !found {
		t.Fatalf("missing producer span in payload: %q", bodies)
	}
	workerSpan, found := findClefSpanByName(bodies, "Process order async")
	if !found {
		t.Fatalf("missing worker span in payload: %q", bodies)
	}

	if workerSpan.TraceID != producerSpan.TraceID {
		t.Fatalf("worker trace_id = %q, want %q", workerSpan.TraceID, producerSpan.TraceID)
	}
	if workerSpan.ParentID != producerSpan.SpanID {
		t.Fatalf("worker parent id = %q, want %q", workerSpan.ParentID, producerSpan.SpanID)
	}
}

func TestResumeTraceEmptyTraceIDIsNoOp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	if err := Setup(Config{
		Endpoint:      server.URL,
		BatchSize:     1,
		FlushInterval: time.Hour,
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	t.Cleanup(Shutdown)

	resumedContext := ResumeTrace(context.Background(), "", "0123456789abc000")
	_, span := Start(resumedContext, "isolated")
	if span.TraceID() == "" {
		t.Fatal("expected new trace id")
	}
	if span.parentID != "" {
		t.Fatalf("expected root span without parent, got %q", span.parentID)
	}
	span.End()
}

func TestTraceFromContextAfterResumeTrace(t *testing.T) {
	const (
		traceID  = "0123456789abcdef0123456789abcdef"
		parentID = "0123456789abc000"
	)

	resumedContext := ResumeTrace(context.Background(), traceID, parentID)
	gotTraceID, gotSpanID, ok := TraceFromContext(resumedContext)
	if !ok {
		t.Fatal("expected trace identifiers in resumed context")
	}
	if gotTraceID != traceID {
		t.Fatalf("trace_id = %q, want %q", gotTraceID, traceID)
	}
	if gotSpanID != parentID {
		t.Fatalf("span_id = %q, want %q", gotSpanID, parentID)
	}
}

type clefSpanRecord struct {
	Name    string
	TraceID string
	SpanID  string
	ParentID string
}

func findClefSpanByName(payload, name string) (clefSpanRecord, bool) {
	for _, line := range strings.Split(payload, "\n") {
		if line == "" {
			continue
		}
		var clefEvent map[string]any
		if err := json.Unmarshal([]byte(line), &clefEvent); err != nil {
			continue
		}
		if clefEvent["@mt"] != name {
			continue
		}
		return clefSpanRecord{
			Name:     name,
			TraceID:  stringFieldValue(clefEvent["@tr"]),
			SpanID:   stringFieldValue(clefEvent["@sp"]),
			ParentID: stringFieldValue(clefEvent["@ps"]),
		}, true
	}
	return clefSpanRecord{}, false
}

func stringFieldValue(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}
