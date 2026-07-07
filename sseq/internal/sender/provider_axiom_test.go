package sender

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAxiomProviderPostsNDJSON(t *testing.T) {
	var receivedBody string
	var contentType string
	var authorization string

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			response.WriteHeader(http.StatusInternalServerError)
			return
		}
		receivedBody = string(body)
		contentType = request.Header.Get("Content-Type")
		authorization = request.Header.Get("Authorization")
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider, err := NewAxiomProvider(AxiomConfig{
		Token:      "test-token",
		Dataset:    "otel-traces",
		Endpoint:   server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewAxiomProvider() error = %v", err)
	}

	startTime := time.Now().UTC()
	encoded, err := EncodeAxiomSpanEvent(SpanEvent{
		Name:        "root span",
		Application: "unit-test",
		TraceID:     "0123456789abcdef0123456789abcdef",
		SpanID:      "0123456789abc000",
		SpanKind:    "Server",
		StartTime:   startTime,
		EndTime:     startTime.Add(10 * time.Millisecond),
	})
	if err != nil {
		t.Fatalf("EncodeAxiomSpanEvent() error = %v", err)
	}

	provider.WritePayload(append(encoded, '\n'))

	if contentType != axiomIngestContentType {
		t.Fatalf("content type = %q, want %q", contentType, axiomIngestContentType)
	}
	if authorization != "Bearer test-token" {
		t.Fatalf("authorization = %q", authorization)
	}
	if !strings.Contains(receivedBody, `"trace_id":"0123456789abcdef0123456789abcdef"`) {
		t.Fatalf("missing trace payload: %q", receivedBody)
	}
	if !strings.Contains(receivedBody, `"service.name":"unit-test"`) {
		t.Fatalf("missing service.name: %q", receivedBody)
	}
}

func TestNewAxiomProviderRequiresCredentials(t *testing.T) {
	if _, err := NewAxiomProvider(AxiomConfig{}); err == nil {
		t.Fatal("expected error for missing token and dataset")
	}
	if _, err := NewAxiomProvider(AxiomConfig{Token: "token"}); err == nil {
		t.Fatal("expected error for missing dataset")
	}
}
