package sseq

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDoRecordsSpanError(t *testing.T) {
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

	if err := Setup(Config{
		Provider:      ProviderHTTP,
		Endpoint:      server.URL,
		BatchSize:     1,
		FlushInterval: time.Hour,
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	t.Cleanup(Shutdown)

	expectedErr := errors.New("payment failed")
	err := Do(context.Background(), "charge payment", func(context.Context) error {
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("Do() error = %v, want %v", err, expectedErr)
	}

	Shutdown()

	if !strings.Contains(receivedBody, `"@l":"Error"`) {
		t.Fatalf("expected error level in payload: %q", receivedBody)
	}
	if !strings.Contains(receivedBody, `"ErrorMessage":"payment failed"`) {
		t.Fatalf("expected error message in payload: %q", receivedBody)
	}
}
