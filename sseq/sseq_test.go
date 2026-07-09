package sseq

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDoSpanHierarchy(t *testing.T) {
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

	err := Do(context.Background(), "root", func(ctx context.Context) error {
		return Do(ctx, "child-a", func(ctx context.Context) error {
			return Do(ctx, "child-b", func(ctx context.Context) error {
				return nil
			})
		})
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	Shutdown()

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

func TestStartEndTraceIDStable(t *testing.T) {
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

	rootContext, rootSpan := Start(context.Background(), "root")
	childContext, childSpan := Start(rootContext, "child")
	childSpan.End()
	rootSpan.End()

	if rootSpan.TraceID() == "" || childSpan.TraceID() == "" {
		t.Fatalf("expected non-empty trace ids")
	}
	if rootSpan.TraceID() != childSpan.TraceID() {
		t.Fatalf("trace ids should match across spans")
	}
	if childSpan.parentID != rootSpan.SpanID() {
		t.Fatalf("child parent id = %q, want %q", childSpan.parentID, rootSpan.SpanID())
	}
	_ = childContext
}
