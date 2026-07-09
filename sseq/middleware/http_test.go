package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SmilingXinyi/gb/sseq"
	"github.com/SmilingXinyi/gb/sseq/middleware"
)

func TestHTTPMiddlewareRecordsSuccess(t *testing.T) {
	var receivedBody string

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			response.WriteHeader(http.StatusInternalServerError)
			return
		}
		receivedBody = string(body)
		response.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	if err := sseq.Setup(sseq.Config{
		Provider:      sseq.ProviderHTTP,
		Endpoint:      server.URL,
		BatchSize:     1,
		FlushInterval: time.Hour,
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	t.Cleanup(sseq.Shutdown)

	handler := middleware.HTTP(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	sseq.Shutdown()

	if !strings.Contains(receivedBody, `"@mt":"GET /api/users"`) {
		t.Fatalf("missing span name in payload: %q", receivedBody)
	}
	if !strings.Contains(receivedBody, `"StatusCode":200`) {
		t.Fatalf("missing HTTP status in payload: %q", receivedBody)
	}
	if strings.Contains(receivedBody, `"@l":"Error"`) {
		t.Fatalf("did not expect error level for 200 response: %q", receivedBody)
	}
}

func TestHTTPMiddlewareRecordsServerError(t *testing.T) {
	var receivedBody string

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			response.WriteHeader(http.StatusInternalServerError)
			return
		}
		receivedBody = string(body)
		response.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	if err := sseq.Setup(sseq.Config{
		Provider:      sseq.ProviderHTTP,
		Endpoint:      server.URL,
		BatchSize:     1,
		FlushInterval: time.Hour,
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	t.Cleanup(sseq.Shutdown)

	handler := middleware.HTTP(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusInternalServerError)
	}))

	request := httptest.NewRequest(http.MethodPost, "/api/orders", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	sseq.Shutdown()

	if !strings.Contains(receivedBody, `"StatusCode":500`) {
		t.Fatalf("missing HTTP 500 status in payload: %q", receivedBody)
	}
	if !strings.Contains(receivedBody, `"@l":"Error"`) {
		t.Fatalf("expected error level for 500 response: %q", receivedBody)
	}
	if !strings.Contains(receivedBody, `"ErrorMessage":"HTTP 500"`) {
		t.Fatalf("expected HTTP 500 error message in payload: %q", receivedBody)
	}
}
