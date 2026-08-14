package seq

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPAcceptsSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Content-Type") != clefContentType {
			t.Errorf("Content-Type = %q", request.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(request.Body)
		if string(body) != "payload\n" {
			t.Errorf("body = %q", body)
		}
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	writer := NewHTTP(server.URL, "")
	if err := writer.WritePayload([]byte("payload\n")); err != nil {
		t.Fatalf("WritePayload() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestHTTPRejectsErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	writer := NewHTTP(server.URL, "secret")
	if err := writer.WritePayload([]byte("payload\n")); err == nil {
		t.Fatal("expected status error")
	}
}
