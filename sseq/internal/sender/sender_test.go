package sender

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestCloseWaitsForInFlightPost(t *testing.T) {
	var requestStarted atomic.Bool
	requestFinished := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = io.ReadAll(request.Body)
		requestStarted.Store(true)
		<-requestFinished
		response.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	sender := New(Config{
		Endpoint:      server.URL,
		BatchSize:     1,
		FlushInterval: time.Hour,
		HTTPClient:    server.Client(),
	})

	startTime := time.Now().UTC()
	if err := sender.Send(SpanEvent{
		Name:      "slow span",
		TraceID:   "0123456789abcdef0123456789abcdef",
		SpanID:    "0123456789abc000",
		SpanKind:  "Server",
		StartTime: startTime,
		EndTime:   startTime,
	}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	closeDone := make(chan struct{})
	go func() {
		if err := sender.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
		close(closeDone)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for !requestStarted.Load() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !requestStarted.Load() {
		t.Fatal("expected HTTP request to start before Close returns")
	}

	select {
	case <-closeDone:
		t.Fatal("Close returned before HTTP request finished")
	default:
	}

	close(requestFinished)

	select {
	case <-closeDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not return after HTTP request finished")
	}
}
