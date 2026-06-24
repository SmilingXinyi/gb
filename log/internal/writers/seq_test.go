package writers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSeqWriterWriteAndClose(t *testing.T) {
	var requestBody string
	var requestHeaders http.Header
	var mutex sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			response.WriteHeader(http.StatusInternalServerError)
			return
		}

		mutex.Lock()
		requestBody = string(body)
		requestHeaders = request.Header.Clone()
		mutex.Unlock()

		response.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	writer := NewSeqWriter(SeqWriterConfig{
		Endpoint:      server.URL,
		Application:   "demo-app",
		BatchSize:     1,
		FlushInterval: time.Hour,
		HTTPClient:    server.Client(),
	})
	defer writer.Close()

	payload := []byte(`{"level":"info","time":"2026-06-24T12:00:00Z","message":"hello","module":"auth"}`)
	if _, err := writer.Write(payload); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		mutex.Lock()
		received := requestBody
		mutex.Unlock()
		if received != "" || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	mutex.Lock()
	receivedBody := requestBody
	receivedHeaders := requestHeaders
	mutex.Unlock()

	if !strings.Contains(receivedBody, `"@mt":"hello"`) {
		t.Fatalf("request body missing message: %q", receivedBody)
	}
	if !strings.Contains(receivedBody, `"Application":"demo-app"`) {
		t.Fatalf("request body missing application: %q", receivedBody)
	}
	if receivedHeaders.Get("Content-Type") != seqCLEFContentType {
		t.Fatalf("content type = %q, want %q", receivedHeaders.Get("Content-Type"), seqCLEFContentType)
	}
}

func TestSeqWriterUsesAPIKeyHeader(t *testing.T) {
	var apiKeyHeader string
	var mutex sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		mutex.Lock()
		apiKeyHeader = request.Header.Get("X-Seq-ApiKey")
		mutex.Unlock()
		response.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	writer := NewSeqWriter(SeqWriterConfig{
		Endpoint:      server.URL,
		APIKey:        "secret-key",
		BatchSize:     1,
		FlushInterval: time.Hour,
		HTTPClient:    server.Client(),
	})
	defer writer.Close()

	if _, err := writer.Write([]byte(`{"level":"info","time":"2026-06-24T12:00:00Z","message":"hello"}`)); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		mutex.Lock()
		received := apiKeyHeader
		mutex.Unlock()
		if received != "" || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	mutex.Lock()
	receivedKey := apiKeyHeader
	mutex.Unlock()

	if receivedKey != "secret-key" {
		t.Fatalf("api key header = %q, want secret-key", receivedKey)
	}
}
